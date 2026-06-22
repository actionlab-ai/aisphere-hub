package skillzip

import (
	"archive/zip"
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/actionlab-ai/aisphere-hub/backend/internal/httputil"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/model"
)

const (
	SkillMDFile           = "SKILL.md"
	EncodingKey           = "encoding"
	EncodingBase64        = "base64"
	DefaultInitialVersion = "0.0.1"
	MaxUploadBytes        = 20 * 1024 * 1024
	MaxEntries            = 500
	MaxUncompressedBytes  = 50 * 1024 * 1024
)

var frontMatterRe = regexp.MustCompile(`(?s)^---\s*\r?\n(.*?)\r?\n---\s*(?:\r?\n|$)(.*)$`)

type EntryData struct {
	Name string
	Data []byte
}
type ParseFailure struct {
	Folder string
	Reason string
}
type MultiParseResult struct {
	Skills   []model.Skill
	Failures []ParseFailure
}

func ParseSkillFromZip(zipBytes []byte, namespaceID string) (model.Skill, error) {
	entries, err := unzip(zipBytes)
	if err != nil {
		return model.Skill{}, err
	}
	skillEntry := findSkillMD(entries)
	if skillEntry == nil {
		return model.Skill{}, httputil.BadRequest("SKILL.md file not found in zip")
	}
	content := stripBOM(string(skillEntry.Data))
	skill, err := ParseSkillMarkdown(content, namespaceID)
	if err != nil {
		return model.Skill{}, err
	}
	skill.Resource = parseResources(entries, skill.Name, skillEntry.Name)
	return skill, nil
}

func ParseMultipleSkillsFromZip(zipBytes []byte, namespaceID string) (MultiParseResult, error) {
	entries, err := unzip(zipBytes)
	if err != nil {
		return MultiParseResult{}, err
	}
	var skillEntries []*EntryData
	for i := range entries {
		name := entries[i].Name
		if isMacMetadata(name) {
			continue
		}
		if name == SkillMDFile || strings.HasSuffix(name, "/"+SkillMDFile) {
			skillEntries = append(skillEntries, &entries[i])
		}
	}
	if len(skillEntries) == 0 {
		return MultiParseResult{}, httputil.BadRequest("SKILL.md file not found in zip")
	}
	for _, e := range skillEntries {
		if e.Name == SkillMDFile {
			s, err := ParseSkillFromZip(zipBytes, namespaceID)
			if err != nil {
				return MultiParseResult{}, err
			}
			return MultiParseResult{Skills: []model.Skill{s}}, nil
		}
	}
	res := MultiParseResult{}
	prefixes := map[string]bool{}
	for _, e := range skillEntries {
		prefixes[getPrefix(e.Name)] = true
	}
	for _, e := range skillEntries {
		prefix := getPrefix(e.Name)
		content := stripBOM(string(e.Data))
		s, err := ParseSkillMarkdown(content, namespaceID)
		if err != nil {
			res.Failures = append(res.Failures, ParseFailure{Folder: folderName(prefix), Reason: err.Error()})
			continue
		}
		scoped := filterPrefix(entries, prefix)
		s.Resource = parseResources(scoped, s.Name, SkillMDFile)
		res.Skills = append(res.Skills, s)
	}
	// warn peer dirs without SKILL.md, same broad behavior as Nacos.
	for _, e := range entries {
		parts := strings.Split(strings.Trim(e.Name, "/"), "/")
		if len(parts) < 2 {
			continue
		}
		p := parts[0] + "/"
		if prefixes[p] || strings.HasPrefix(parts[0], ".") || parts[0] == "__MACOSX" || parts[0] == "node_modules" {
			continue
		}
		prefixes[p] = true
		res.Failures = append(res.Failures, ParseFailure{Folder: parts[0], Reason: "SKILL.md not found in this folder, skipped"})
	}
	if len(res.Skills) == 0 {
		return res, httputil.BadRequest("No valid skills found in zip")
	}
	return res, nil
}

func ParseSkillMarkdown(markdownContent, namespaceID string) (model.Skill, error) {
	m := frontMatterRe.FindStringSubmatch(markdownContent)
	if len(m) != 3 {
		return model.Skill{}, httputil.BadRequest("SKILL.md must contain YAML front matter (---)")
	}
	yaml := ParseYamlFrontMatter(m[1])
	name := strings.TrimSpace(yaml["name"])
	desc := strings.TrimSpace(yaml["description"])
	if name == "" {
		return model.Skill{}, httputil.BadRequest("Skill name is required in YAML front matter")
	}
	if desc == "" {
		return model.Skill{}, httputil.BadRequest("Skill description is required in YAML front matter")
	}
	if strings.TrimSpace(m[2]) == "" {
		return model.Skill{}, httputil.BadRequest("Skill markdown body is required")
	}
	skill := model.Skill{SkillBase: model.SkillBase{NamespaceID: namespaceID, Name: name, Description: desc}, SkillMD: markdownContent, Resource: map[string]*model.SkillResource{}}
	ApplyFrontMatter(&skill, yaml)
	return skill, nil
}

func ParseYamlFrontMatter(yamlContent string) map[string]string {
	result := map[string]string{}
	var currentKey string
	for _, raw := range strings.Split(yamlContent, "\n") {
		line := strings.TrimRight(raw, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') && currentKey != "" {
			nested := strings.TrimSpace(line)
			if i := strings.Index(nested, ":"); i > 0 {
				k := strings.TrimSpace(nested[:i])
				v := parseScalar(strings.TrimSpace(nested[i+1:]))
				result[currentKey+"."+k] = v
			}
			continue
		}
		if i := strings.Index(line, ":"); i > 0 {
			k := strings.TrimSpace(line[:i])
			v := parseScalar(strings.TrimSpace(line[i+1:]))
			result[k] = v
			currentKey = k
		} else {
			currentKey = ""
		}
	}
	return result
}

func ApplyFrontMatter(skill *model.Skill, yaml map[string]string) {
	if skill == nil {
		return
	}
	if v := first(yaml, "skillSet", "metadata.skillSet"); v != "" {
		skill.SkillSet = v
	}
	skill.Groups = parseList(first(yaml, "groups", "metadata.groups"))
	skill.Keywords = parseList(first(yaml, "keywords", "metadata.keywords"))
	if v := first(yaml, "modelName", "metadata.modelName"); v != "" {
		skill.ModelName = v
	}
	if v := first(yaml, "modelDescription", "metadata.modelDescription"); v != "" {
		skill.ModelDescription = v
	}
	if v := first(yaml, "matchHint", "metadata.matchHint"); v != "" {
		skill.MatchHint = v
	}
	if v := first(yaml, "activation", "metadata.activation"); v != "" {
		skill.Activation = v
	}
	if v := first(yaml, "priority", "metadata.priority"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			skill.Priority = &i
		}
	}
}

func ResolveVersionFromSkillMD(skillMD string) string {
	yaml := ParseYamlFrontMatterFromMarkdown(skillMD)
	return first(yaml, "version", "metadata.version")
}

func ParseYamlFrontMatterFromMarkdown(markdown string) map[string]string {
	m := frontMatterRe.FindStringSubmatch(markdown)
	if len(m) != 3 {
		return map[string]string{}
	}
	return ParseYamlFrontMatter(m[1])
}

func ResolveVersionFromZip(zipBytes []byte) string {
	entries, err := unzip(zipBytes)
	if err != nil {
		return ""
	}
	skillEntry := findSkillMD(entries)
	if skillEntry == nil {
		return ""
	}
	yaml := ParseYamlFrontMatterFromMarkdown(string(skillEntry.Data))
	if v := first(yaml, "version", "metadata.version"); v != "" {
		return v
	}
	metaPath := path.Join(path.Dir(skillEntry.Name), "_meta.json")
	if path.Dir(skillEntry.Name) == "." {
		metaPath = "_meta.json"
	}
	for _, e := range entries {
		if e.Name == metaPath {
			var m map[string]interface{}
			if json.Unmarshal(e.Data, &m) == nil && m["version"] != nil {
				return fmt.Sprint(m["version"])
			}
		}
	}
	return ""
}

func ToZipBytes(skill model.Skill) ([]byte, error) {
	if strings.TrimSpace(skill.Name) == "" {
		return nil, httputil.BadRequest("Skill name cannot be blank")
	}
	buf := bytes.NewBuffer(nil)
	zw := zip.NewWriter(buf)
	if err := writeZipEntry(zw, skill.Name+"/"+SkillMDFile, []byte(skill.SkillMD)); err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(skill.Resource))
	for k := range skill.Resource {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		r := skill.Resource[k]
		if r == nil || r.Name == "" {
			continue
		}
		entryPath := skill.Name + "/"
		if r.Type != "" {
			entryPath += strings.Trim(r.Type, "/") + "/"
		}
		entryPath += r.Name
		var b []byte
		if r.Metadata != nil && fmt.Sprint(r.Metadata[EncodingKey]) == EncodingBase64 {
			decoded, err := base64.StdEncoding.DecodeString(r.Content)
			if err != nil {
				return nil, err
			}
			b = decoded
		} else {
			b = []byte(r.Content)
		}
		if err := writeZipEntry(zw, entryPath, b); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func ContentMD5(skill model.Skill) string {
	b, _ := ToZipBytes(skill)
	sum := md5.Sum(b)
	return hex.EncodeToString(sum[:])
}

func FileList(skill model.Skill) []string {
	files := []string{SkillMDFile}
	for _, r := range skill.Resource {
		if r == nil || r.Name == "" {
			continue
		}
		p := r.Name
		if r.Type != "" {
			p = strings.Trim(r.Type, "/") + "/" + r.Name
		}
		files = append(files, p)
	}
	sort.Strings(files)
	return files
}

func ResourceID(typ, name string) string {
	if typ != "" {
		return typ + "::" + name
	}
	return name
}

func unzip(zipBytes []byte) ([]EntryData, error) {
	if len(zipBytes) == 0 {
		return nil, httputil.BadRequest("Skill zip file is empty")
	}
	if len(zipBytes) > MaxUploadBytes {
		return nil, httputil.BadRequest("Skill zip size must not exceed 20MB")
	}
	zr, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		return nil, httputil.BadRequest("Failed to parse zip file: " + err.Error())
	}
	if len(zr.File) > MaxEntries {
		return nil, httputil.BadRequest("ZIP file contains too many entries")
	}
	var out []EntryData
	var total int64
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		name := strings.ReplaceAll(f.Name, "\\", "/")
		if !safePath(name) {
			return nil, httputil.BadRequest("Path traversal detected: " + name)
		}
		if strings.Contains(name, "__MACOSX") || isMacMetadata(name) {
			continue
		}
		r, err := f.Open()
		if err != nil {
			return nil, err
		}
		b, err := io.ReadAll(io.LimitReader(r, MaxUncompressedBytes+1))
		_ = r.Close()
		if err != nil {
			return nil, err
		}
		total += int64(len(b))
		if total > MaxUncompressedBytes {
			return nil, httputil.BadRequest("ZIP decompressed size exceeds limit 50MB")
		}
		out = append(out, EntryData{Name: name, Data: b})
	}
	return out, nil
}

func findSkillMD(entries []EntryData) *EntryData {
	var firstNested *EntryData
	for i := range entries {
		if entries[i].Name == SkillMDFile {
			return &entries[i]
		}
		if firstNested == nil && strings.HasSuffix(entries[i].Name, "/"+SkillMDFile) {
			firstNested = &entries[i]
		}
	}
	return firstNested
}

func parseResources(entries []EntryData, skillName, descriptorPath string) map[string]*model.SkillResource {
	resources := map[string]*model.SkillResource{}
	for _, e := range entries {
		itemName := e.Name
		if itemName == descriptorPath || strings.HasSuffix(itemName, "/") || isMacMetadata(itemName) {
			continue
		}
		parts := strings.Split(itemName, "/")
		var typ, resourceName string
		if len(parts) == 1 {
			typ = ""
			resourceName = parts[0]
		} else if len(parts) == 2 && parts[0] == skillName {
			typ = ""
			resourceName = parts[1]
		} else if len(parts) >= 3 && parts[0] == skillName {
			typ = strings.Join(parts[1:len(parts)-1], "/")
			resourceName = parts[len(parts)-1]
		} else if len(parts) >= 2 {
			typ = strings.Join(parts[:len(parts)-1], "/")
			resourceName = parts[len(parts)-1]
		} else {
			continue
		}
		content := string(e.Data)
		metadata := map[string]interface{}{}
		if isBinary(resourceName) {
			content = base64.StdEncoding.EncodeToString(e.Data)
			metadata[EncodingKey] = EncodingBase64
		}
		r := &model.SkillResource{Name: resourceName, Type: typ, Content: content}
		if len(metadata) > 0 {
			r.Metadata = metadata
		}
		resources[ResourceID(typ, resourceName)] = r
	}
	return resources
}

func writeZipEntry(zw *zip.Writer, name string, data []byte) error {
	if !safePath(name) {
		return httputil.BadRequest("unsafe zip entry path: " + name)
	}
	w, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func safePath(p string) bool {
	return p != "" && !strings.HasPrefix(p, "/") && !strings.HasPrefix(p, "\\") && !strings.Contains(p, "..")
}
func getPrefix(p string) string {
	if i := strings.LastIndex(p, "/"); i >= 0 {
		return p[:i+1]
	}
	return ""
}
func filterPrefix(entries []EntryData, prefix string) []EntryData {
	if prefix == "" {
		return entries
	}
	out := []EntryData{}
	for _, e := range entries {
		if strings.HasPrefix(e.Name, prefix) {
			n := e.Name[len(prefix):]
			if n != "" {
				out = append(out, EntryData{Name: n, Data: e.Data})
			}
		}
	}
	return out
}
func folderName(prefix string) string {
	prefix = strings.Trim(prefix, "/")
	if prefix == "" {
		return "unknown"
	}
	parts := strings.Split(prefix, "/")
	return parts[len(parts)-1]
}
func stripBOM(s string) string { return strings.TrimPrefix(s, "\ufeff") }
func parseScalar(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && ((s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'')) {
		return strings.ReplaceAll(s[1:len(s)-1], `\"`, `"`)
	}
	return s
}
func first(m map[string]string, keys ...string) string {
	for _, k := range keys {
		if strings.TrimSpace(m[k]) != "" {
			return strings.TrimSpace(m[k])
		}
	}
	return ""
}
func parseList(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	s = strings.TrimPrefix(strings.TrimSuffix(s, "]"), "[")
	parts := strings.Split(s, ",")
	out := []string{}
	for _, p := range parts {
		p = strings.Trim(strings.TrimSpace(p), `"'`)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
func isMacMetadata(itemName string) bool {
	base := path.Base(itemName)
	return strings.HasPrefix(base, "._")
}
func isBinary(name string) bool {
	ext := strings.ToLower(path.Ext(name))
	switch ext {
	case ".md", ".txt", ".json", ".yaml", ".yml", ".xml", ".csv", ".sh", ".py", ".go", ".js", ".ts", ".html", ".css", ".properties", ".conf", ".ini":
		return false
	default:
		return ext != ""
	}
}
