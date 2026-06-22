package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/actionlab-ai/aisphere-hub/backend/internal/httputil"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/model"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/ports"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/sandbox"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/skillzip"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/store"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/versioning"
)

type Service struct {
	store        store.Backend
	objectStore  ports.ObjectStore
	objectPrefix string
	author       string
	runtimeCache ports.RuntimeCache
	lockManager  ports.LockManager
	lockTTL      time.Duration
	lockWait     time.Duration
	cacheTTL     CacheTTL
	sandboxMgr   sandbox.Manager
}

type CacheTTL struct {
	Route   time.Duration
	Version time.Duration
	Group   time.Duration
}

func New(st store.Backend, author string) *Service {
	if author == "" {
		author = "-"
	}
	return &Service{store: st, author: author, lockTTL: 30 * time.Second, lockWait: 2 * time.Second, cacheTTL: CacheTTL{Route: 5 * time.Minute, Version: 10 * time.Minute, Group: time.Minute}}
}

func (s *Service) WithRuntimeCache(c ports.RuntimeCache) *Service {
	s.runtimeCache = c
	return s
}

func (s *Service) WithLockManager(l ports.LockManager, ttl, wait time.Duration) *Service {
	s.lockManager = l
	if ttl > 0 {
		s.lockTTL = ttl
	}
	if wait > 0 {
		s.lockWait = wait
	}
	return s
}

func (s *Service) WithCacheTTL(ttl CacheTTL) *Service {
	if ttl.Route > 0 {
		s.cacheTTL.Route = ttl.Route
	}
	if ttl.Version > 0 {
		s.cacheTTL.Version = ttl.Version
	}
	if ttl.Group > 0 {
		s.cacheTTL.Group = ttl.Group
	}
	return s
}

func (s *Service) WithObjectStore(os ports.ObjectStore, prefix string) *Service {
	s.objectStore = os
	s.objectPrefix = strings.Trim(strings.TrimSpace(prefix), "/")
	return s
}

func (s *Service) WithSandboxManager(m sandbox.Manager) *Service {
	s.sandboxMgr = m
	return s
}

func (s *Service) SandboxManager() sandbox.Manager {
	return s.sandboxMgr
}

func (s *Service) skillLockKey(namespaceID, skillName string) string {
	return "aihub:lock:skill:" + strings.TrimSpace(skillName)
}

func (s *Service) groupLockKey(namespaceID, groupName string) string {
	return "aihub:lock:skillset:" + strings.TrimSpace(groupName)
}

func (s *Service) withSkillWriteLock(namespaceID, skillName string, fn func() error) error {
	if s.lockManager == nil || strings.TrimSpace(skillName) == "" {
		return fn()
	}
	return s.lockManager.WithLock(context.Background(), s.skillLockKey(namespaceID, skillName), s.lockTTL, fn)
}

func (s *Service) withGroupWriteLock(namespaceID, groupName string, fn func() error) error {
	if s.lockManager == nil || strings.TrimSpace(groupName) == "" {
		return fn()
	}
	return s.lockManager.WithLock(context.Background(), s.groupLockKey(namespaceID, groupName), s.lockTTL, fn)
}

func (s *Service) waitSkillUnlocked(ctx context.Context, namespaceID, skillName string) {
	if s.lockManager != nil && strings.TrimSpace(skillName) != "" {
		_ = s.lockManager.Wait(ctx, s.skillLockKey(namespaceID, skillName), s.lockWait)
	}
}

func (s *Service) waitGroupUnlocked(ctx context.Context, namespaceID, groupName string) {
	if s.lockManager != nil && strings.TrimSpace(groupName) != "" {
		_ = s.lockManager.Wait(ctx, s.groupLockKey(namespaceID, groupName), s.lockWait)
	}
}

func (s *Service) UploadSkillFromZip(namespaceID string, zipBytes []byte, overwrite bool, targetVersion, commitMsg string) (string, error) {
	var name string
	err := s.store.WithWrite(func() error {
		skill, err := skillzip.ParseSkillFromZip(zipBytes, namespaceID)
		if err != nil {
			return err
		}
		if err := validateName(skill.Name); err != nil {
			return err
		}
		version, err := s.resolveUploadVersion(skill.SkillMD, zipBytes, targetVersion)
		if err != nil {
			return err
		}
		name = skill.Name
		return s.doUploadSingle(namespaceID, skill, version, overwrite, commitMsg)
	})
	return name, err
}

func (s *Service) BatchUploadSkillsFromZip(namespaceID string, zipBytes []byte, overwrite bool) (model.BatchUploadResult, error) {
	result := model.BatchUploadResult{Succeeded: []string{}, Failed: []model.FailedItem{}}
	parsed, err := skillzip.ParseMultipleSkillsFromZip(zipBytes, namespaceID)
	if err != nil {
		return result, err
	}
	for _, f := range parsed.Failures {
		result.Failed = append(result.Failed, model.FailedItem{Name: f.Folder, Reason: f.Reason})
	}
	_ = s.store.WithWrite(func() error {
		for _, skill := range parsed.Skills {
			name := skill.Name
			version, err := s.resolveUploadVersion(skill.SkillMD, nil, "")
			if err == nil {
				err = s.doUploadSingle(namespaceID, skill, version, overwrite, "")
			}
			if err != nil {
				result.Failed = append(result.Failed, model.FailedItem{Name: nameOrUnknown(name), Reason: err.Error()})
				continue
			}
			result.Succeeded = append(result.Succeeded, name)
		}
		return nil
	})
	return result, nil
}

func (s *Service) doUploadSingle(namespaceID string, skill model.Skill, uploadVersion string, overwrite bool, commitMsg string) error {
	rec, err := s.store.Load(namespaceID, skill.Name)
	if err != nil {
		return err
	}
	if rec == nil {
		return s.createDraftWithSkill(namespaceID, skill, uploadVersion, nil, true, commitMsg)
	}
	if overwrite {
		if rec.EditingVersion != "" {
			return s.overwriteEditingDraft(rec, skill, rec.EditingVersion, commitMsg)
		}
		newVersion := s.resolveFinalUploadVersion(rec, uploadVersion)
		return s.createDraftWithSkill(namespaceID, skill, newVersion, rec, false, commitMsg)
	}
	if rec.EditingVersion != "" || rec.ReviewingVersion != "" {
		return httputil.Conflict("A working version already exists; use overwrite=true or submit/publish/delete draft first")
	}
	newVersion := s.resolveFinalUploadVersion(rec, uploadVersion)
	return s.createDraftWithSkill(namespaceID, skill, newVersion, rec, false, commitMsg)
}

func (s *Service) GetSkillDetail(namespaceID, skillName string) (model.SkillMeta, error) {
	var out model.SkillMeta
	err := s.store.WithRead(func() error {
		rec, err := s.require(namespaceID, skillName)
		if err != nil {
			return err
		}
		out = rec.Meta()
		sort.Slice(out.Versions, func(i, j int) bool { return versioning.Compare(out.Versions[i].Version, out.Versions[j].Version) > 0 })
		return nil
	})
	return out, err
}

func (s *Service) ListSkillVersionFiles(namespaceID, skillName, version string) (model.SkillVersionFileList, error) {
	out := model.SkillVersionFileList{NamespaceID: namespaceID, SkillName: skillName, Version: version, Files: []model.SkillFileInfo{}}
	err := s.store.WithRead(func() error {
		rec, err := s.require(namespaceID, skillName)
		if err != nil {
			return err
		}
		v := rec.Versions[version]
		if version == "" {
			return httputil.BadRequest("Version is required")
		}
		if v == nil {
			return httputil.NotFound("Skill version not found: " + skillName + "@" + version)
		}
		out.Files = skillFileInfos(v.Skill)
		return nil
	})
	return out, err
}

func (s *Service) GetSkillVersionFile(namespaceID, skillName, version, filePath string) (model.SkillVersionFileContent, error) {
	out := model.SkillVersionFileContent{NamespaceID: namespaceID, SkillName: skillName, Version: version, Path: strings.TrimPrefix(filePath, "/")}
	err := s.store.WithRead(func() error {
		rec, err := s.require(namespaceID, skillName)
		if err != nil {
			return err
		}
		v := rec.Versions[version]
		if version == "" {
			return httputil.BadRequest("Version is required")
		}
		if v == nil {
			return httputil.NotFound("Skill version not found: " + skillName + "@" + version)
		}
		content, binary, err := skillFileContent(v.Skill, out.Path)
		if err != nil {
			return err
		}
		out.Content = content
		out.Binary = binary
		return nil
	})
	return out, err
}

func (s *Service) CompareSkillVersions(namespaceID, skillName, baseVersion, targetVersion string) (model.SkillVersionCompare, error) {
	out := model.SkillVersionCompare{NamespaceID: namespaceID, SkillName: skillName, BaseVersion: baseVersion, TargetVersion: targetVersion}
	err := s.store.WithRead(func() error {
		rec, err := s.require(namespaceID, skillName)
		if err != nil {
			return err
		}
		base := rec.Versions[baseVersion]
		target := rec.Versions[targetVersion]
		if baseVersion == "" || targetVersion == "" {
			return httputil.BadRequest("baseVersion and targetVersion are required")
		}
		if base == nil {
			return httputil.NotFound("Base version not found: " + baseVersion)
		}
		if target == nil {
			return httputil.NotFound("Target version not found: " + targetVersion)
		}
		out.BaseSkillMD = base.Skill.SkillMD
		out.TargetSkillMD = target.Skill.SkillMD
		out.BaseFiles = skillFileInfos(base.Skill)
		out.TargetFiles = skillFileInfos(target.Skill)
		return nil
	})
	return out, err
}

func skillFileInfos(skill model.Skill) []model.SkillFileInfo {
	out := []model.SkillFileInfo{{Path: skillzip.SkillMDFile, Name: skillzip.SkillMDFile, Type: "markdown", Size: int64(len([]byte(skill.SkillMD)))}}
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
		p := r.Name
		if r.Type != "" {
			p = strings.Trim(r.Type, "/") + "/" + r.Name
		}
		binary := r.Metadata != nil && fmt.Sprint(r.Metadata[skillzip.EncodingKey]) == skillzip.EncodingBase64
		size := int64(len([]byte(r.Content)))
		out = append(out, model.SkillFileInfo{Path: p, Name: r.Name, Type: r.Type, Size: size, Binary: binary})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

func skillFileContent(skill model.Skill, filePath string) (string, bool, error) {
	filePath = strings.TrimPrefix(strings.TrimSpace(filePath), "/")
	if filePath == "" || filePath == skillzip.SkillMDFile {
		return skill.SkillMD, false, nil
	}
	for _, r := range skill.Resource {
		if r == nil {
			continue
		}
		p := r.Name
		if r.Type != "" {
			p = strings.Trim(r.Type, "/") + "/" + r.Name
		}
		if p == filePath {
			if r.Metadata != nil && fmt.Sprint(r.Metadata[skillzip.EncodingKey]) == skillzip.EncodingBase64 {
				return "", true, nil
			}
			return r.Content, false, nil
		}
	}
	return "", false, httputil.NotFound("file not found: " + filePath)
}

func skillZipDigest(skill model.Skill) (string, int64) {
	zipBytes, err := skillzip.ToZipBytes(skill)
	if err != nil {
		return "", 0
	}
	sum := sha256.Sum256(zipBytes)
	return hex.EncodeToString(sum[:]), int64(len(zipBytes))
}

func versionRevision(skillName string, v *model.VersionRecord) string {
	if v == nil {
		return ""
	}
	seed := fmt.Sprintf("%s|%s|%s|%s|%d", skillName, v.Version, v.MD5, v.SHA256, v.UpdateTime)
	sum := sha256.Sum256([]byte(seed))
	return hex.EncodeToString(sum[:])[:16]
}

func (s *Service) GetSkillVersion(namespaceID, skillName, version string) (model.Skill, error) {
	var skill model.Skill
	err := s.store.WithRead(func() error {
		rec, err := s.require(namespaceID, skillName)
		if err != nil {
			return err
		}
		v := rec.Versions[version]
		if version == "" {
			return httputil.BadRequest("Version is required for skill version detail")
		}
		if v == nil {
			return httputil.NotFound("Skill version not found: " + skillName + "@" + version)
		}
		skill = v.Skill
		return nil
	})
	return skill, err
}

func (s *Service) GetSkillVersionRecord(namespaceID, skillName, version string) (*model.VersionRecord, error) {
	var out *model.VersionRecord
	err := s.store.WithRead(func() error {
		rec, err := s.require(namespaceID, skillName)
		if err != nil {
			return err
		}
		v := rec.Versions[version]
		if version == "" {
			return httputil.BadRequest("Version is required")
		}
		if v == nil {
			return httputil.NotFound("Skill version not found: " + skillName + "@" + version)
		}
		copy := *v
		copy.Skill = model.Skill{}
		out = &copy
		return nil
	})
	return out, err
}

func (s *Service) DownloadSkillVersion(namespaceID, skillName, version string) (model.Skill, []byte, string, error) {
	var skill model.Skill
	var zipBytes []byte
	var md5 string
	err := s.store.WithWrite(func() error {
		rec, err := s.require(namespaceID, skillName)
		if err != nil {
			return err
		}
		v := rec.Versions[version]
		if version == "" {
			return httputil.BadRequest("Version is required")
		}
		if v == nil {
			return httputil.NotFound("Skill version not found: " + skillName + "@" + version)
		}
		v.DownloadCount++
		rec.DownloadCount++
		rec.UpdateTime = model.NowMillis()
		if err := s.store.Save(rec); err != nil {
			return err
		}
		skill = v.Skill
		md5 = v.MD5
		zipBytes, err = s.loadZipBytes(context.Background(), v)
		return err
	})
	return skill, zipBytes, md5, err
}

func (s *Service) DeleteSkill(namespaceID, skillName string) error {
	return s.withSkillWriteLock(namespaceID, skillName, func() error {
		return s.store.WithWrite(func() error {
			if err := s.store.Delete(namespaceID, skillName); err != nil {
				return err
			}
			s.invalidateSkill(namespaceID, skillName)
			s.emitCatalogEvent(model.CatalogEventSkillDeleted, "skill", skillName, "", "", "", nil)
			return nil
		})
	})
}

func (s *Service) ListSkills(namespaceID, skillName, search, orderBy, owner, scope, bizTag, groupName string, pageNo, pageSize int) (model.Page, error) {
	if pageNo <= 0 {
		pageNo = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}
	var page model.Page
	err := s.store.WithRead(func() error {
		recs, err := s.store.List(namespaceID)
		if err != nil {
			return err
		}
		allowedByGroup := map[string]bool{}
		if groupName != "" {
			groups, err := s.store.ListGroups(namespaceID)
			if err != nil {
				return err
			}
			foundGroup := false
			for _, g := range groups {
				if strings.EqualFold(g.Name, groupName) {
					foundGroup = true
					for _, m := range g.Members {
						allowedByGroup[m.SkillName] = true
					}
					break
				}
			}
			if !foundGroup {
				recs = nil
			}
		}
		items := make([]model.SkillSummary, 0)
		for _, rec := range recs {
			if groupName != "" && !allowedByGroup[rec.Name] {
				continue
			}
			if skillName != "" {
				if strings.EqualFold(search, "accurate") {
					if rec.Name != skillName {
						continue
					}
				} else if !strings.Contains(strings.ToLower(rec.Name), strings.ToLower(skillName)) {
					continue
				}
			}
			if owner != "" && rec.Owner != owner {
				continue
			}
			if scope != "" && !strings.EqualFold(rec.Scope, scope) {
				continue
			}
			if bizTag != "" && !strings.Contains(rec.BizTags, bizTag) {
				continue
			}
			items = append(items, rec.Summary())
		}
		sortSummaries(items, orderBy)
		total := len(items)
		start := (pageNo - 1) * pageSize
		if start > total {
			start = total
		}
		end := start + pageSize
		if end > total {
			end = total
		}
		pages := 0
		if total > 0 {
			pages = (total + pageSize - 1) / pageSize
		}
		page = model.Page{PageNumber: pageNo, PagesAvailable: pages, TotalCount: total, PageItems: items[start:end]}
		return nil
	})
	return page, err
}

func (s *Service) CreateDraft(namespaceID, name, basedOnVersion, targetVersion string, initial *model.Skill, commitMsg string) (string, error) {
	var created string
	err := s.store.WithWrite(func() error {
		rec, err := s.store.Load(namespaceID, name)
		if err != nil {
			return err
		}
		if rec == nil {
			if basedOnVersion != "" {
				return httputil.NotFound("Skill not found: " + name + ", cannot use basedOnVersion for a brand-new skill")
			}
			if initial == nil {
				return httputil.BadRequest("skillCard is required when creating a brand-new skill draft")
			}
			initial.NamespaceID = namespaceID
			initial.Name = name
			if err := ensureSkillValid(*initial); err != nil {
				return err
			}
			version := ""
			if targetVersion == "" {
				var er error
				version, er = s.resolveUploadVersion(initial.SkillMD, nil, "")
				if er != nil {
					return er
				}
			} else {
				var er error
				version, er = s.resolveSpecifiedDraftVersion(nil, targetVersion, "", "")
				if er != nil {
					return er
				}
			}
			created = version
			return s.createDraftWithSkill(namespaceID, *initial, version, nil, true, commitMsg)
		}
		if rec.EditingVersion != "" || rec.ReviewingVersion != "" {
			return httputil.Conflict("A working version already exists; create draft is not allowed")
		}
		base := s.resolveBaseVersion(rec, basedOnVersion)
		newVersion, err := s.resolveSpecifiedDraftVersion(rec, targetVersion, basedOnVersion, base)
		if err != nil {
			return err
		}
		if base == "" {
			if initial == nil {
				return httputil.BadRequest("skillCard is required when no published version exists to fork from")
			}
			initial.NamespaceID = namespaceID
			initial.Name = name
			if err := ensureSkillValid(*initial); err != nil {
				return err
			}
			created = newVersion
			return s.createDraftWithSkill(namespaceID, *initial, newVersion, rec, false, commitMsg)
		}
		if initial != nil {
			return httputil.BadRequest("skillCard must not be set when creating a draft from an existing version; omit it to fork, then use PUT /draft to edit")
		}
		baseRow := rec.Versions[base]
		if baseRow == nil {
			return httputil.NotFound("Base version not found: " + base)
		}
		created = newVersion
		return s.createDraftWithSkill(namespaceID, baseRow.Skill, newVersion, rec, false, commitMsg)
	})
	return created, err
}

func (s *Service) UpdateDraft(namespaceID string, draftSkill model.Skill, commitMsg string) error {
	return s.store.WithWrite(func() error {
		if err := ensureSkillValid(draftSkill); err != nil {
			return err
		}
		rec, err := s.require(namespaceID, draftSkill.Name)
		if err != nil {
			return err
		}
		if rec.EditingVersion == "" {
			return httputil.NotFound("No editing draft exists for skill: " + draftSkill.Name)
		}
		return s.overwriteEditingDraft(rec, draftSkill, rec.EditingVersion, commitMsg)
	})
}

func (s *Service) DeleteDraft(namespaceID, name string) error {
	return s.store.WithWrite(func() error {
		rec, err := s.require(namespaceID, name)
		if err != nil {
			return err
		}
		editing := rec.EditingVersion
		if editing == "" {
			return nil
		}
		delete(rec.Versions, editing)
		rec.EditingVersion = ""
		rec.UpdateTime = model.NowMillis()
		return s.saveSkillAndInvalidate(rec)
	})
}

func (s *Service) Submit(namespaceID, name, version string) (string, error) {
	var target string
	err := s.store.WithWrite(func() error {
		rec, err := s.require(namespaceID, name)
		if err != nil {
			return err
		}
		target = resolveSubmitTarget(rec, version)
		if target == "" {
			return httputil.BadRequest("No draft or reviewing version exists for skill: " + name)
		}
		v := rec.Versions[target]
		if v == nil {
			return httputil.NotFound("Skill version not found: " + name + "@" + target)
		}
		if v.Status == model.VersionStatusDraft {
			v.Status = model.VersionStatusReviewing
			rec.EditingVersion = ""
			rec.ReviewingVersion = target
		}
		// Nacos behavior when no publish pipeline is configured: submit publishes directly.
		return s.publishLocked(rec, target, true, false)
	})
	return target, err
}

func (s *Service) Publish(namespaceID, name, version string, updateLatest bool, force bool) error {
	return s.store.WithWrite(func() error {
		rec, err := s.require(namespaceID, name)
		if err != nil {
			return err
		}
		return s.publishLocked(rec, version, updateLatest, force)
	})
}

func (s *Service) Redraft(namespaceID, name, version string) error {
	return s.store.WithWrite(func() error {
		rec, err := s.require(namespaceID, name)
		if err != nil {
			return err
		}
		if rec.EditingVersion != "" || rec.ReviewingVersion != "" {
			return httputil.Conflict("A working version already exists")
		}
		v := rec.Versions[version]
		if v == nil {
			return httputil.NotFound("Skill version not found: " + name + "@" + version)
		}
		v.Status = model.VersionStatusDraft
		v.UpdateTime = model.NowMillis()
		rec.EditingVersion = version
		rec.UpdateTime = v.UpdateTime
		return s.saveSkillAndInvalidate(rec)
	})
}

func (s *Service) UpdateLabels(namespaceID, name string, labels map[string]string) error {
	return s.store.WithWrite(func() error {
		rec, err := s.require(namespaceID, name)
		if err != nil {
			return err
		}
		for label, version := range labels {
			if strings.TrimSpace(label) == "" || strings.TrimSpace(version) == "" {
				return httputil.BadRequest("labels must be non-empty")
			}
			v := rec.Versions[version]
			if v == nil {
				return httputil.NotFound("label references missing version: " + version)
			}
		}
		rec.Labels = labels
		rec.UpdateTime = model.NowMillis()
		return s.saveSkillAndInvalidate(rec)
	})
}

func (s *Service) UpdateBizTags(namespaceID, name, bizTags string) error {
	return s.store.WithWrite(func() error {
		rec, err := s.require(namespaceID, name)
		if err != nil {
			return err
		}
		rec.BizTags = bizTags
		rec.UpdateTime = model.NowMillis()
		return s.saveSkillAndInvalidate(rec)
	})
}

func (s *Service) UpdateMetadata(namespaceID, name string, metadata model.Skill) error {
	return s.store.WithWrite(func() error {
		rec, err := s.require(namespaceID, name)
		if err != nil {
			return err
		}
		metadata.Name = name
		metadata.NamespaceID = namespaceID
		rec.SkillSet = metadata.SkillSet
		rec.Groups = metadata.Groups
		rec.Keywords = metadata.Keywords
		rec.ModelName = metadata.ModelName
		rec.ModelDescription = metadata.ModelDescription
		rec.MatchHint = metadata.MatchHint
		rec.Activation = metadata.Activation
		rec.Priority = metadata.Priority
		rec.UpdateTime = model.NowMillis()
		return s.saveSkillAndInvalidate(rec)
	})
}

func (s *Service) UpdateScope(namespaceID, name, scope string) error {
	return s.store.WithWrite(func() error {
		if scope == "" {
			scope = model.ScopePublic
		}
		scope = strings.ToUpper(scope)
		if scope != model.ScopePublic && scope != model.ScopePrivate {
			return httputil.BadRequest("scope must be PUBLIC or PRIVATE")
		}
		rec, err := s.require(namespaceID, name)
		if err != nil {
			return err
		}
		rec.Scope = scope
		rec.UpdateTime = model.NowMillis()
		return s.saveSkillAndInvalidate(rec)
	})
}

func (s *Service) ChangeOnlineStatus(namespaceID, name, scope, version string, online bool) error {
	return s.store.WithWrite(func() error {
		rec, err := s.require(namespaceID, name)
		if err != nil {
			return err
		}
		skillScope := strings.EqualFold(scope, "skill") || version == ""
		if skillScope {
			if online {
				rec.Status = model.MetaStatusEnable
			} else {
				rec.Status = model.MetaStatusDisable
			}
			rec.UpdateTime = model.NowMillis()
			return s.saveSkillAndInvalidate(rec)
		}
		v := rec.Versions[version]
		if v == nil {
			return httputil.NotFound("Skill version not found: " + name + "@" + version)
		}
		if online {
			v.Status = model.VersionStatusOnline
		} else {
			v.Status = model.VersionStatusOffline
		}
		v.UpdateTime = model.NowMillis()
		rec.UpdateTime = v.UpdateTime
		return s.saveSkillAndInvalidate(rec)
	})
}

func (s *Service) QuerySkill(namespaceID, name, version, label, clientMD5 string) (skill model.Skill, zipBytes []byte, md5, resolvedVersion string, notModified bool, err error) {
	if strings.TrimSpace(name) == "" {
		err = httputil.BadRequest("name is required")
		return
	}
	if label == "" {
		label = model.LabelLatest
	}
	ctx := context.Background()
	s.waitSkillUnlocked(ctx, namespaceID, name)

	// Cache-aside fast path. Cache only stores routable online versions; any admin write
	// invalidates route/version keys after the source-of-truth write succeeds.
	if s.runtimeCache != nil {
		if version == "" {
			if cachedVersion, ok, er := s.runtimeCache.GetRoute(ctx, namespaceID, "skill", name, label); er == nil && ok && cachedVersion != "" {
				resolvedVersion = cachedVersion
			}
		} else {
			resolvedVersion = version
		}
		if resolvedVersion != "" {
			var cached model.VersionRecord
			if ok, er := s.runtimeCache.GetVersionMeta(ctx, namespaceID, "skill", name, resolvedVersion, &cached); er == nil && ok && cached.Status == model.VersionStatusOnline {
				md5 = cached.MD5
				if clientMD5 != "" && strings.EqualFold(clientMD5, md5) {
					notModified = true
					_ = s.runtimeCache.IncrementDownload(ctx, namespaceID, "skill", name, resolvedVersion, 1)
					return
				}
				skill = cached.Skill
				zipBytes, err = s.loadZipBytes(ctx, &cached)
				if err == nil {
					_ = s.runtimeCache.IncrementDownload(ctx, namespaceID, "skill", name, resolvedVersion, 1)
					return
				}
				// Object cache metadata can outlive an object-store hiccup; fall through to DB
				// to rebuild from source metadata / fallback payload and refresh the cache.
			}
		}
	}

	err = s.store.WithRead(func() error {
		rec, er := s.require(namespaceID, name)
		if er != nil {
			return er
		}
		if rec.Status != model.MetaStatusEnable {
			return httputil.NotFound("Skill is disabled: " + name)
		}
		resolvedVersion = s.resolveRuntimeVersion(rec, version, label)
		if resolvedVersion == "" {
			return httputil.NotFound("No online version resolved for skill: " + name)
		}
		v := rec.Versions[resolvedVersion]
		if v == nil || v.Status != model.VersionStatusOnline {
			return httputil.NotFound("Skill version is not online: " + name + "@" + resolvedVersion)
		}
		md5 = v.MD5
		if s.runtimeCache != nil {
			if version == "" {
				_ = s.runtimeCache.SetRoute(ctx, namespaceID, "skill", name, label, resolvedVersion, s.cacheTTL.Route)
			}
			_ = s.runtimeCache.SetVersionMeta(ctx, namespaceID, "skill", name, resolvedVersion, v, s.cacheTTL.Version)
		}
		if clientMD5 != "" && strings.EqualFold(clientMD5, md5) {
			notModified = true
			return nil
		}
		skill = v.Skill
		zipBytes, er = s.loadZipBytes(ctx, v)
		return er
	})
	if err != nil {
		return
	}
	if s.runtimeCache != nil && resolvedVersion != "" {
		_ = s.runtimeCache.IncrementDownload(ctx, namespaceID, "skill", name, resolvedVersion, 1)
	} else if resolvedVersion != "" {
		// No external cache/counter is configured, so keep the old local-store behavior:
		// update download count synchronously in the source of truth.
		_ = s.store.WithWrite(func() error {
			rec, er := s.require(namespaceID, name)
			if er != nil {
				return er
			}
			v := rec.Versions[resolvedVersion]
			if v != nil {
				v.DownloadCount++
			}
			rec.DownloadCount++
			rec.UpdateTime = model.NowMillis()
			return s.store.Save(rec)
		})
	}
	return
}

func (s *Service) SearchPublic(namespaceID, q string, limit int, sourceBase string) ([]map[string]interface{}, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	var out []map[string]interface{}
	err := s.store.WithRead(func() error {
		recs, err := s.store.List(namespaceID)
		if err != nil {
			return err
		}
		q = strings.ToLower(q)
		for _, rec := range recs {
			if !exportable(rec) {
				continue
			}
			if q != "" && !strings.Contains(strings.ToLower(rec.Name+" "+rec.Description+" "+strings.Join(rec.Keywords, " ")), q) {
				continue
			}
			out = append(out, map[string]interface{}{"id": rec.Name, "name": rec.Name, "installs": rec.DownloadCount, "source": strings.TrimRight(sourceBase, "/") + "/.well-known/agent-skills/" + rec.Name + ".zip"})
			if len(out) >= limit {
				break
			}
		}
		return nil
	})
	return out, err
}

func (s *Service) WellKnownIndex(namespaceID, base string) (map[string]interface{}, error) {
	idx := map[string]interface{}{"$schema": "https://www.agentcommunity.org/schemas/skills-index-v0.json", "skills": []interface{}{}}
	err := s.store.WithRead(func() error {
		recs, err := s.store.List(namespaceID)
		if err != nil {
			return err
		}
		items := []interface{}{}
		for _, rec := range recs {
			if !exportable(rec) {
				continue
			}
			ver := rec.Labels[model.LabelLatest]
			if ver == "" {
				ver = s.resolveRuntimeVersion(rec, "", model.LabelLatest)
			}
			v := rec.Versions[ver]
			if v == nil {
				continue
			}
			items = append(items, map[string]interface{}{"name": rec.Name, "type": "skill", "description": rec.Description, "url": strings.TrimRight(base, "/") + "/" + rec.Name + ".zip", "digest": v.MD5, "version": ver, "files": v.Files})
		}
		idx["skills"] = items
		return nil
	})
	return idx, err
}

func (s *Service) RegistrySkill(namespaceID, skillName string) (model.Skill, []byte, string, error) {
	var skill model.Skill
	var zipBytes []byte
	var md5 string
	err := s.store.WithRead(func() error {
		rec, err := s.require(namespaceID, skillName)
		if err != nil {
			return err
		}
		if !exportable(rec) {
			return httputil.NotFound("Skill not found: " + skillName)
		}
		ver := s.resolveRuntimeVersion(rec, "", model.LabelLatest)
		v := rec.Versions[ver]
		if v == nil {
			return httputil.NotFound("Skill not found: " + skillName)
		}
		skill = v.Skill
		md5 = v.MD5
		var er error
		zipBytes, er = skillzip.ToZipBytes(skill)
		return er
	})
	return skill, zipBytes, md5, err
}

func (s *Service) RegistryFile(namespaceID, skillName, filePath string) (string, error) {
	skill, _, _, err := s.RegistrySkill(namespaceID, skillName)
	if err != nil {
		return "", err
	}
	filePath = strings.TrimPrefix(filePath, "/")
	if filePath == "" || filePath == skillzip.SkillMDFile {
		return skill.SkillMD, nil
	}
	for _, r := range skill.Resource {
		if r == nil {
			continue
		}
		p := r.Name
		if r.Type != "" {
			p = strings.Trim(r.Type, "/") + "/" + r.Name
		}
		if p == filePath {
			if r.Metadata != nil && fmt.Sprint(r.Metadata[skillzip.EncodingKey]) == skillzip.EncodingBase64 {
				return "", httputil.BadRequest("resource is binary/base64; download the zip archive instead")
			}
			return r.Content, nil
		}
	}
	return "", httputil.NotFound("file not found: " + filePath)
}

func (s *Service) createDraftWithSkill(namespaceID string, skill model.Skill, version string, existed *model.SkillRecord, isNew bool, commitMsg string) error {
	if err := ensureSkillValid(skill); err != nil {
		return err
	}
	if !versioning.Supported(version) {
		return httputil.BadRequest("Invalid version: " + version + ", expected x.y.z or vN")
	}
	skill.NamespaceID = namespaceID
	skill.Name = strings.TrimSpace(skill.Name)
	files := skillzip.FileList(skill)
	md5 := skillzip.ContentMD5(skill)
	sha, size := skillZipDigest(skill)
	now := model.NowMillis()
	v := &model.VersionRecord{Version: version, Status: model.VersionStatusDraft, Author: s.author, CommitMsg: commitMsg, CreateTime: now, UpdateTime: now, MD5: md5, SHA256: sha, SizeBytes: size, Files: files, Skill: skill}
	v.Revision = versionRevision(skill.Name, v)
	if err := s.persistVersionObjects(context.Background(), namespaceID, skill.Name, version, skill, v); err != nil {
		return err
	}
	rec := existed
	if rec == nil || isNew {
		rec = model.NewSkillRecord(namespaceID, skill, s.author, "local")
	} else {
		rec.ApplySkillMetadata(skill)
	}
	rec.Versions[version] = v
	rec.EditingVersion = version
	rec.ReviewingVersion = ""
	rec.UpdateTime = now
	return s.saveSkillAndInvalidate(rec)
}

func (s *Service) overwriteEditingDraft(rec *model.SkillRecord, skill model.Skill, editing, commitMsg string) error {
	if err := ensureSkillValid(skill); err != nil {
		return err
	}
	v := rec.Versions[editing]
	if v == nil || v.Status != model.VersionStatusDraft {
		return httputil.NotFound("Draft version not found: " + editing)
	}
	skill.NamespaceID = rec.NamespaceID
	skill.Name = rec.Name
	v.Skill = skill
	v.Files = skillzip.FileList(skill)
	v.MD5 = skillzip.ContentMD5(skill)
	v.SHA256, v.SizeBytes = skillZipDigest(skill)
	v.Revision = versionRevision(rec.Name, v)
	if err := s.persistVersionObjects(context.Background(), rec.NamespaceID, rec.Name, editing, skill, v); err != nil {
		return err
	}
	v.UpdateTime = model.NowMillis()
	if commitMsg != "" {
		v.CommitMsg = commitMsg
	}
	rec.ApplySkillMetadata(skill)
	rec.EditingVersion = editing
	return s.saveSkillAndInvalidate(rec)
}

func (s *Service) publishLocked(rec *model.SkillRecord, version string, updateLatest bool, force bool) error {
	if version == "" {
		return httputil.BadRequest("Version is required")
	}
	v := rec.Versions[version]
	if v == nil {
		return httputil.NotFound("Skill version not found: " + rec.Name + "@" + version)
	}
	if !force && v.Status != model.VersionStatusReviewing && v.Status != model.VersionStatusReviewed && v.Status != model.VersionStatusDraft && v.Status != model.VersionStatusOnline {
		return httputil.Conflict("Version status cannot be published: " + v.Status)
	}
	v.Status = model.VersionStatusOnline
	v.UpdateTime = model.NowMillis()
	v.Revision = versionRevision(rec.Name, v)
	if rec.EditingVersion == version {
		rec.EditingVersion = ""
	}
	if rec.ReviewingVersion == version {
		rec.ReviewingVersion = ""
	}
	if rec.Labels == nil {
		rec.Labels = map[string]string{}
	}
	if updateLatest {
		rec.Labels[model.LabelLatest] = version
	}
	rec.Status = model.MetaStatusEnable
	rec.UpdateTime = v.UpdateTime
	if err := s.saveSkillAndInvalidate(rec); err != nil {
		return err
	}
	s.emitCatalogEvent(model.CatalogEventSkillPublished, "skill", rec.Name, rec.SkillSet, version, v.Revision, map[string]interface{}{"version": version, "latest": rec.Labels[model.LabelLatest]})
	s.notifySubscribers(rec.NamespaceID, model.SubscriptionTargetSkill, rec.Name, "skill.published", "Skill published", fmt.Sprintf("%s@%s has been published", rec.Name, version), map[string]interface{}{"version": version})
	return nil
}

func (s *Service) require(namespaceID, name string) (*model.SkillRecord, error) {
	rec, err := s.store.Load(namespaceID, name)
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, httputil.NotFound("Skill not found: " + name)
	}
	return rec, nil
}

func (s *Service) resolveUploadVersion(skillMD string, zipBytes []byte, targetVersion string) (string, error) {
	candidate := skillzip.ResolveVersionFromSkillMD(skillMD)
	if candidate == "" {
		candidate = skillzip.ResolveVersionFromZip(zipBytes)
	}
	if candidate == "" {
		candidate = strings.TrimSpace(targetVersion)
	}
	if candidate == "" {
		candidate = skillzip.DefaultInitialVersion
	}
	if !versioning.Supported(candidate) {
		return "", httputil.BadRequest("Invalid version: '" + candidate + "', expected x.y.z or vN")
	}
	return candidate, nil
}

func (s *Service) resolveFinalUploadVersion(rec *model.SkillRecord, candidate string) string {
	if rec == nil {
		return candidate
	}
	versions := versionsOf(rec)
	if !contains(versions, candidate) {
		return candidate
	}
	if max := versioning.MaxSemver(versions); max != "" {
		return versioning.NextPatch(max)
	}
	if maxV := versioning.MaxVNumber(versions); maxV > 0 {
		return fmt.Sprintf("v%d", maxV+1)
	}
	return versioning.NextPatch(candidate)
}

func (s *Service) resolveSpecifiedDraftVersion(rec *model.SkillRecord, targetVersion, basedOnVersion, baseVersion string) (string, error) {
	if targetVersion == "" {
		if rec == nil {
			return skillzip.DefaultInitialVersion, nil
		}
		return s.nextDraftVersion(rec), nil
	}
	candidate := strings.TrimSpace(targetVersion)
	if !versioning.Supported(candidate) {
		return "", httputil.BadRequest("Invalid targetVersion format: " + candidate + ", expected x.y.z or vN")
	}
	if rec != nil {
		if rec.Versions[candidate] != nil {
			return "", httputil.Conflict("targetVersion already exists: " + candidate)
		}
		if basedOnVersion != "" && baseVersion != "" && !versioning.Greater(candidate, baseVersion) {
			return "", httputil.BadRequest("targetVersion must be greater than basedOnVersion, basedOnVersion=" + baseVersion + ", targetVersion=" + candidate)
		}
	}
	return candidate, nil
}

func (s *Service) nextDraftVersion(rec *model.SkillRecord) string {
	versions := versionsOf(rec)
	if max := versioning.MaxSemver(versions); max != "" {
		return versioning.NextPatch(max)
	}
	if maxV := versioning.MaxVNumber(versions); maxV > 0 {
		return fmt.Sprintf("v%d", maxV+1)
	}
	return skillzip.DefaultInitialVersion
}

func (s *Service) resolveBaseVersion(rec *model.SkillRecord, basedOnVersion string) string {
	if basedOnVersion != "" {
		return basedOnVersion
	}
	if rec.Labels != nil && rec.Labels[model.LabelLatest] != "" {
		return rec.Labels[model.LabelLatest]
	}
	versions := versionsOf(rec)
	if max := versioning.MaxSemver(versions); max != "" {
		return max
	}
	if maxV := versioning.MaxVNumber(versions); maxV > 0 {
		return fmt.Sprintf("v%d", maxV)
	}
	return ""
}

func (s *Service) resolveRuntimeVersion(rec *model.SkillRecord, version, label string) string {
	if version != "" {
		return version
	}
	if label != "" && rec.Labels != nil && rec.Labels[label] != "" {
		return rec.Labels[label]
	}
	if rec.Labels != nil && rec.Labels[model.LabelLatest] != "" {
		return rec.Labels[model.LabelLatest]
	}
	var candidates []string
	for ver, v := range rec.Versions {
		if v.Status == model.VersionStatusOnline {
			candidates = append(candidates, ver)
		}
	}
	versioning.SortDesc(candidates)
	if len(candidates) > 0 {
		return candidates[0]
	}
	return ""
}

func resolveSubmitTarget(rec *model.SkillRecord, version string) string {
	if version != "" {
		return version
	}
	if rec.EditingVersion != "" {
		return rec.EditingVersion
	}
	if rec.ReviewingVersion != "" {
		return rec.ReviewingVersion
	}
	return ""
}

func ensureSkillValid(skill model.Skill) error {
	if strings.TrimSpace(skill.Name) == "" {
		return httputil.BadRequest("Skill name is required")
	}
	if strings.TrimSpace(skill.Description) == "" {
		return httputil.BadRequest("Skill description is required")
	}
	if strings.TrimSpace(skill.SkillMD) == "" {
		return httputil.BadRequest("Skill markdown body is required")
	}
	return nil
}
func validateName(name string) error {
	if strings.TrimSpace(name) == "" {
		return httputil.BadRequest("Skill name is required")
	}
	if strings.ContainsAny(name, "/\\ ") {
		return httputil.BadRequest("Skill name cannot contain slash, backslash, or space")
	}
	return nil
}
func versionsOf(rec *model.SkillRecord) []string {
	out := []string{}
	if rec != nil {
		for v := range rec.Versions {
			out = append(out, v)
		}
	}
	return out
}
func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}
func nameOrUnknown(s string) string {
	if s == "" {
		return "unknown"
	}
	return s
}
func exportable(rec *model.SkillRecord) bool {
	return rec != nil && rec.Status == model.MetaStatusEnable && rec.Scope == model.ScopePublic && rec.OnlineCount() > 0
}

func (s *Service) persistVersionObjects(ctx context.Context, namespaceID, skillName, version string, skill model.Skill, v *model.VersionRecord) error {
	if s.objectStore == nil {
		return nil
	}
	zipBytes, err := skillzip.ToZipBytes(skill)
	if err != nil {
		return err
	}
	base := s.objectKey(namespaceID, skillName, version)
	zipKey := base + "/skill.zip"
	info, err := s.objectStore.Put(ctx, zipKey, bytes.NewReader(zipBytes), ports.PutOptions{ContentType: "application/zip"})
	if err != nil {
		return err
	}
	skillMdKey := base + "/SKILL.md"
	_, _ = s.objectStore.Put(ctx, skillMdKey, strings.NewReader(skill.SkillMD), ports.PutOptions{ContentType: "text/markdown; charset=utf-8"})
	manifest := model.SkillIndexManifest{Labels: map[string]string{model.LabelLatest: version}, Versions: map[string][]string{version: skillzip.FileList(skill)}}
	mb, _ := json.MarshalIndent(manifest, "", "  ")
	manifestKey := base + "/manifest.json"
	_, _ = s.objectStore.Put(ctx, manifestKey, bytes.NewReader(mb), ports.PutOptions{ContentType: "application/json"})
	v.Storage = &model.StorageDescriptor{Provider: "object", ObjectKey: zipKey, SkillMdObjectKey: skillMdKey, ManifestObjectKey: manifestKey, ContentMD5: info.ContentMD5, SHA256: v.SHA256, SizeBytes: info.Size}
	if v.SizeBytes == 0 {
		v.SizeBytes = info.Size
	}
	if v.MD5 == "" && info.ContentMD5 != "" {
		v.MD5 = info.ContentMD5
	}
	return nil
}

func (s *Service) loadZipBytes(ctx context.Context, v *model.VersionRecord) ([]byte, error) {
	if s.objectStore != nil && v != nil && v.Storage != nil && v.Storage.ObjectKey != "" {
		rc, _, err := s.objectStore.Get(ctx, v.Storage.ObjectKey)
		if err == nil {
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}
	return skillzip.ToZipBytes(v.Skill)
}

func (s *Service) objectKey(namespaceID, skillName, version string) string {
	parts := []string{s.objectPrefix, "skills", skillName, "versions", version}
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.Trim(p, "/")
		if p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, "/")
}

func (s *Service) AppendCatalogEvent(e *model.CatalogEvent) error {
	if e == nil {
		return nil
	}
	if e.App == "" {
		e.App = model.DefaultApp
	}
	if e.CreatedAt == 0 {
		e.CreatedAt = model.NowMillis()
	}
	return s.store.AppendCatalogEvent(e)
}

func (s *Service) ListCatalogEvents(q model.CatalogEventQuery) ([]*model.CatalogEvent, int64, error) {
	if q.App == "" {
		q.App = model.DefaultApp
	}
	return s.store.ListCatalogEvents(q)
}

func (s *Service) emitCatalogEvent(eventType, resourceType, resourceID, skillset, version, revision string, payload map[string]interface{}) {
	if strings.TrimSpace(resourceID) == "" {
		resourceID = "*"
	}
	objType := resourceType
	if objType == "" {
		objType = "resource"
	}
	_ = s.store.AppendCatalogEvent(&model.CatalogEvent{
		App:          model.DefaultApp,
		EventType:    eventType,
		Object:       model.DefaultApp + ":" + objType + ":" + resourceID,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		SkillSetName: skillset,
		Version:      version,
		Revision:     revision,
		Payload:      payload,
		CreatedAt:    model.NowMillis(),
	})
}

func sortSummaries(items []model.SkillSummary, orderBy string) {
	switch strings.ToLower(orderBy) {
	case "name", "name_asc":
		sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	case "download", "downloadcount", "download_count":
		sort.Slice(items, func(i, j int) bool { return items[i].DownloadCount > items[j].DownloadCount })
	default:
		sort.Slice(items, func(i, j int) bool {
			if items[i].UpdateTime == nil || items[j].UpdateTime == nil {
				return items[i].Name < items[j].Name
			}
			return *items[i].UpdateTime > *items[j].UpdateTime
		})
	}
}

func ParseSkillCard(skillCard string, namespaceID, fallbackName string) (*model.Skill, error) {
	skillCard = strings.TrimSpace(skillCard)
	if skillCard == "" {
		return nil, nil
	}
	var skill model.Skill
	if strings.HasPrefix(skillCard, "{") {
		if err := json.Unmarshal([]byte(skillCard), &skill); err != nil {
			return nil, httputil.BadRequest("invalid skillCard json: " + err.Error())
		}
		if skill.SkillMD != "" && (skill.Name == "" || skill.Description == "") {
			parsed, err := skillzip.ParseSkillMarkdown(skill.SkillMD, namespaceID)
			if err != nil {
				return nil, err
			}
			if skill.Name == "" {
				skill.Name = parsed.Name
			}
			if skill.Description == "" {
				skill.Description = parsed.Description
			}
			if len(skill.Groups) == 0 {
				skill.Groups = parsed.Groups
			}
			if len(skill.Keywords) == 0 {
				skill.Keywords = parsed.Keywords
			}
		}
	} else {
		parsed, err := skillzip.ParseSkillMarkdown(skillCard, namespaceID)
		if err != nil {
			return nil, err
		}
		skill = parsed
	}
	if skill.Name == "" {
		skill.Name = fallbackName
	}
	if skill.NamespaceID == "" {
		skill.NamespaceID = namespaceID
	}
	return &skill, nil
}

func (s *Service) ListGroups(namespaceID, q string, pageNo, pageSize int) (model.Page, error) {
	if pageNo <= 0 {
		pageNo = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	var page model.Page
	err := s.store.WithRead(func() error {
		groups, err := s.store.ListGroups(namespaceID)
		if err != nil {
			return err
		}
		items := make([]*model.SkillGroup, 0, len(groups))
		q = strings.ToLower(strings.TrimSpace(q))
		for _, g := range groups {
			if q != "" && !strings.Contains(strings.ToLower(g.Name+" "+g.DisplayName+" "+g.Description), q) {
				continue
			}
			items = append(items, g)
		}
		total := len(items)
		start := (pageNo - 1) * pageSize
		if start > total {
			start = total
		}
		end := start + pageSize
		if end > total {
			end = total
		}
		pages := 0
		if total > 0 {
			pages = (total + pageSize - 1) / pageSize
		}
		page = model.Page{PageNumber: pageNo, PagesAvailable: pages, TotalCount: total, PageItems: items[start:end]}
		return nil
	})
	return page, err
}

func (s *Service) GetGroup(namespaceID, name string) (*model.SkillGroup, error) {
	var out *model.SkillGroup
	err := s.store.WithRead(func() error {
		g, err := s.store.LoadGroup(namespaceID, name)
		if err != nil {
			return err
		}
		if g == nil {
			return httputil.NotFound("Skill group not found: " + name)
		}
		out = g
		return nil
	})
	return out, err
}

func (s *Service) SaveGroup(namespaceID string, g model.SkillGroup) (*model.SkillGroup, error) {
	var out *model.SkillGroup
	err := s.store.WithWrite(func() error {
		if strings.TrimSpace(g.Name) == "" {
			return httputil.BadRequest("group name is required")
		}
		existed, err := s.store.LoadGroup(namespaceID, g.Name)
		if err != nil {
			return err
		}
		if existed == nil {
			existed = model.NewSkillGroup(namespaceID, g.Name, g.DisplayName, g.Description, s.author, g.Scope)
		} else {
			existed.DisplayName = g.DisplayName
			existed.Description = g.Description
			if g.Scope != "" {
				existed.Scope = g.Scope
			}
			existed.UpdateTime = model.NowMillis()
		}
		if g.Owner != "" {
			existed.Owner = g.Owner
		}
		if g.Labels != nil {
			existed.Labels = g.Labels
		}
		if g.Metadata != nil {
			existed.Metadata = g.Metadata
		}
		if g.Members != nil {
			existed.Members = normalizeMembers(g.Members)
		}
		if err := s.validateGroupMembers(namespaceID, existed.Members); err != nil {
			return err
		}
		if err := s.store.SaveGroup(existed); err != nil {
			return err
		}
		out = existed
		if err := s.invalidateGroup(namespaceID, existed.Name); err != nil {
			return err
		}
		s.emitCatalogEvent(model.CatalogEventSkillSetUpdated, "skillset", existed.Name, existed.Name, "", fmt.Sprintf("%d", existed.UpdateTime), map[string]interface{}{"memberCount": len(existed.Members)})
		return nil
	})
	return out, err
}

func (s *Service) DeleteGroup(namespaceID, name string) error {
	return s.store.WithWrite(func() error {
		if err := s.store.DeleteGroup(namespaceID, name); err != nil {
			return err
		}
		if err := s.invalidateGroup(namespaceID, name); err != nil {
			return err
		}
		s.emitCatalogEvent(model.CatalogEventSkillSetDeleted, "skillset", name, name, "", "", nil)
		return nil
	})
}

func (s *Service) BindGroupMember(namespaceID, groupName string, member model.SkillGroupMember) (*model.SkillGroup, error) {
	var out *model.SkillGroup
	err := s.store.WithWrite(func() error {
		g, err := s.store.LoadGroup(namespaceID, groupName)
		if err != nil {
			return err
		}
		if g == nil {
			return httputil.NotFound("Skill group not found: " + groupName)
		}
		if member.SkillName == "" {
			return httputil.BadRequest("skillName is required")
		}
		if member.Version == "" && member.Label == "" {
			member.Label = model.LabelLatest
		}
		if err := s.validateGroupMembers(namespaceID, []model.SkillGroupMember{member}); err != nil {
			return err
		}
		matched := false
		for i := range g.Members {
			if g.Members[i].SkillName == member.SkillName {
				g.Members[i] = member
				matched = true
				break
			}
		}
		if !matched {
			g.Members = append(g.Members, member)
		}
		g.Members = normalizeMembers(g.Members)
		g.UpdateTime = model.NowMillis()
		if err := s.store.SaveGroup(g); err != nil {
			return err
		}
		out = g
		if err := s.invalidateGroup(namespaceID, groupName); err != nil {
			return err
		}
		s.emitCatalogEvent(model.CatalogEventSkillSetUpdated, "skillset", groupName, groupName, member.Version, fmt.Sprintf("%d", g.UpdateTime), map[string]interface{}{"member": member.SkillName, "op": "bind"})
		return nil
	})
	return out, err
}

func (s *Service) UnbindGroupMember(namespaceID, groupName, skillName string) (*model.SkillGroup, error) {
	var out *model.SkillGroup
	err := s.store.WithWrite(func() error {
		g, err := s.store.LoadGroup(namespaceID, groupName)
		if err != nil {
			return err
		}
		if g == nil {
			return httputil.NotFound("Skill group not found: " + groupName)
		}
		members := make([]model.SkillGroupMember, 0, len(g.Members))
		for _, m := range g.Members {
			if m.SkillName != skillName {
				members = append(members, m)
			}
		}
		g.Members = normalizeMembers(members)
		g.UpdateTime = model.NowMillis()
		if err := s.store.SaveGroup(g); err != nil {
			return err
		}
		out = g
		if err := s.invalidateGroup(namespaceID, groupName); err != nil {
			return err
		}
		s.emitCatalogEvent(model.CatalogEventSkillSetUpdated, "skillset", groupName, groupName, "", fmt.Sprintf("%d", g.UpdateTime), map[string]interface{}{"member": skillName, "op": "unbind"})
		return nil
	})
	return out, err
}

func (s *Service) ResolveGroupManifest(namespaceID, groupName, label string) (model.SkillGroupManifest, error) {
	if label == "" {
		label = model.LabelLatest
	}
	ctx := context.Background()
	s.waitGroupUnlocked(ctx, namespaceID, groupName)
	var manifest model.SkillGroupManifest
	if s.runtimeCache != nil {
		if ok, err := s.runtimeCache.GetGroupManifest(ctx, namespaceID, groupName, label, &manifest); err == nil && ok {
			return manifest, nil
		}
	}
	err := s.store.WithRead(func() error {
		g, err := s.store.LoadGroup(namespaceID, groupName)
		if err != nil {
			return err
		}
		if g == nil {
			return httputil.NotFound("Skill group not found: " + groupName)
		}
		manifest = model.SkillGroupManifest{Name: g.Name, Label: label, Version: g.Labels[label], Members: []model.ResolvedSkillRef{}, UpdateTime: g.UpdateTime}
		for _, m := range normalizeMembers(g.Members) {
			rec, err := s.require(namespaceID, m.SkillName)
			if err != nil {
				return err
			}
			ver := m.Version
			if ver == "" {
				ver = s.resolveRuntimeVersion(rec, "", m.Label)
			}
			if ver == "" {
				return httputil.NotFound("No routable version for group member: " + m.SkillName)
			}
			vr := rec.Versions[ver]
			if vr == nil {
				return httputil.NotFound("Missing version for group member: " + m.SkillName + "@" + ver)
			}
			manifest.Members = append(manifest.Members, model.ResolvedSkillRef{Name: m.SkillName, Version: ver, Label: m.Label, MD5: vr.MD5, Required: m.Required, Order: m.Order})
		}
		return nil
	})
	if err == nil && s.runtimeCache != nil {
		_ = s.runtimeCache.SetGroupManifest(ctx, namespaceID, groupName, label, manifest, s.cacheTTL.Group)
	}
	return manifest, err
}

func (s *Service) validateGroupMembers(namespaceID string, members []model.SkillGroupMember) error {
	for _, m := range members {
		if strings.TrimSpace(m.SkillName) == "" {
			return httputil.BadRequest("group member skillName is required")
		}
		rec, err := s.store.Load(namespaceID, m.SkillName)
		if err != nil {
			return err
		}
		if rec == nil {
			return httputil.NotFound("group member skill not found: " + m.SkillName)
		}
		if m.Version != "" && rec.Versions[m.Version] == nil {
			return httputil.NotFound("group member version not found: " + m.SkillName + "@" + m.Version)
		}
	}
	return nil
}

func normalizeMembers(in []model.SkillGroupMember) []model.SkillGroupMember {
	out := append([]model.SkillGroupMember(nil), in...)
	for i := range out {
		if out[i].Label == "" && out[i].Version == "" {
			out[i].Label = model.LabelLatest
		}
		if out[i].Order == 0 {
			out[i].Order = i + 1
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Order < out[j].Order })
	return out
}

func (s *Service) invalidateGroup(namespaceID, groupName string) error {
	if s.runtimeCache == nil {
		return nil
	}
	return s.runtimeCache.DeleteGroupManifests(context.Background(), namespaceID, groupName)
}

func (s *Service) invalidateSkill(namespaceID, skillName string, versions ...string) {
	if s.runtimeCache == nil {
		return
	}
	ctx := context.Background()
	_ = s.runtimeCache.DeleteRoutes(ctx, namespaceID, "skill", skillName)
	_ = s.runtimeCache.DeleteVersionMeta(ctx, namespaceID, "skill", skillName, versions...)
	// Group manifests contain resolved member versions/md5. When a member skill changes
	// labels/status/version metadata, every group referencing that skill must be evicted.
	groups, err := s.store.ListGroups(namespaceID)
	if err != nil {
		return
	}
	for _, g := range groups {
		for _, m := range g.Members {
			if m.SkillName == skillName {
				_ = s.runtimeCache.DeleteGroupManifests(ctx, namespaceID, g.Name)
				break
			}
		}
	}
}

func (s *Service) saveSkillAndInvalidate(rec *model.SkillRecord) error {
	return s.withSkillWriteLock(rec.NamespaceID, rec.Name, func() error {
		if err := s.store.Save(rec); err != nil {
			return err
		}
		s.invalidateSkill(rec.NamespaceID, rec.Name)
		s.emitCatalogEvent(model.CatalogEventSkillUpdated, "skill", rec.Name, rec.SkillSet, rec.Labels[model.LabelLatest], fmt.Sprintf("%d", rec.UpdateTime), map[string]interface{}{"skillSet": rec.SkillSet, "status": rec.Status})
		return nil
	})
}

// SubmitSkillProposal is the safe write entry for Agent self-improvement. It
// creates a proposal + overlay only. It intentionally does not mutate skill
// route/version caches because no runtime-routable skill changed yet.
func (s *Service) SubmitSkillProposal(p model.SkillProposal) (*model.SkillProposal, error) {
	var out *model.SkillProposal
	err := s.store.WithWrite(func() error {
		if strings.TrimSpace(p.NamespaceID) == "" {
			p.NamespaceID = model.DefaultNamespace
		}
		if strings.TrimSpace(p.SkillName) == "" {
			return httputil.BadRequest("skillName is required")
		}
		rec, err := s.require(p.NamespaceID, p.SkillName)
		if err != nil {
			return err
		}
		if p.BaseVersion == "" {
			p.BaseVersion = s.resolveRuntimeVersion(rec, "", model.LabelLatest)
		}
		if p.BaseVersion == "" || rec.Versions[p.BaseVersion] == nil {
			return httputil.NotFound("base version not found for proposal")
		}
		if p.ProposalID == "" {
			p.ProposalID = "sp_" + randHex(12)
		}
		if p.ProposalType == "" {
			p.ProposalType = "delta"
		}
		p.Status = model.ProposalStatusSubmitted
		now := model.NowMillis()
		p.CreateTime = now
		p.UpdateTime = now
		p.OverlayRef = fmt.Sprintf("skill-overlay://%s/%s/%s", p.NamespaceID, p.SkillName, p.ProposalID)
		if p.CandidateVersion == "" {
			p.CandidateVersion = versioning.NextPatch(p.BaseVersion) + "-candidate.1"
		}
		overlay := &model.SkillOverlay{OverlayRef: p.OverlayRef, NamespaceID: p.NamespaceID, SkillName: p.SkillName, BaseVersion: p.BaseVersion, ProposalID: p.ProposalID, Overlay: map[string]interface{}{"delta": p.Delta, "evidence": p.Evidence, "reason": p.Reason}, Status: model.OverlayStatusActive, CreateTime: now}
		if err := s.store.SaveProposal(&p); err != nil {
			return err
		}
		if err := s.store.SaveOverlay(overlay); err != nil {
			return err
		}
		out = &p
		return nil
	})
	return out, err
}

func (s *Service) GetSkillProposal(proposalID string) (*model.SkillProposal, error) {
	var out *model.SkillProposal
	err := s.store.WithRead(func() error {
		p, err := s.store.LoadProposal(proposalID)
		if err != nil {
			return err
		}
		if p == nil {
			return httputil.NotFound("skill proposal not found: " + proposalID)
		}
		out = p
		return nil
	})
	return out, err
}

func (s *Service) ListSkillProposals(q model.ProposalQuery) (model.Page, error) {
	if q.PageNo <= 0 {
		q.PageNo = 1
	}
	if q.PageSize <= 0 {
		q.PageSize = 20
	}
	var page model.Page
	err := s.store.WithRead(func() error {
		items, total64, err := s.store.ListProposals(q)
		if err != nil {
			return err
		}
		total := int(total64)
		start := (q.PageNo - 1) * q.PageSize
		if start > total {
			start = total
		}
		end := start + q.PageSize
		if end > total {
			end = total
		}
		pages := 0
		if total > 0 {
			pages = (total + q.PageSize - 1) / q.PageSize
		}
		page = model.Page{PageNumber: q.PageNo, PagesAvailable: pages, TotalCount: total, PageItems: items[start:end]}
		return nil
	})
	return page, err
}

func (s *Service) GetSkillOverlay(overlayRef string) (*model.SkillOverlay, error) {
	var out *model.SkillOverlay
	err := s.store.WithRead(func() error {
		o, err := s.store.LoadOverlay(overlayRef)
		if err != nil {
			return err
		}
		if o == nil {
			return httputil.NotFound("skill overlay not found")
		}
		out = o
		return nil
	})
	return out, err
}

func (s *Service) ValidateSkillProposal(proposalID string) (*model.ProposalValidation, error) {
	var out *model.ProposalValidation
	err := s.store.WithWrite(func() error {
		p, err := s.store.LoadProposal(proposalID)
		if err != nil {
			return err
		}
		if p == nil {
			return httputil.NotFound("skill proposal not found: " + proposalID)
		}
		p.Status = model.ProposalStatusValidating
		p.UpdateTime = model.NowMillis()
		check := map[string]interface{}{"format": "passed", "safety": "not_implemented", "conflict": "not_implemented"}
		if len(p.Delta) == 0 {
			check["format"] = "warning_empty_delta"
		}
		v := &model.ProposalValidation{ProposalID: proposalID, ValidationStatus: "passed", Score: 0.80, CheckResult: check, TestResult: map[string]interface{}{"mode": "stub"}, CreateTime: model.NowMillis()}
		if err := s.store.SaveProposal(p); err != nil {
			return err
		}
		if err := s.store.SaveProposalValidation(v); err != nil {
			return err
		}
		out = v
		return nil
	})
	return out, err
}

func (s *Service) ApproveSkillProposal(proposalID string, opts model.ProposalApproveOptions) (*model.SkillProposal, error) {
	var out *model.SkillProposal
	err := s.store.WithWrite(func() error {
		p, err := s.store.LoadProposal(proposalID)
		if err != nil {
			return err
		}
		if p == nil {
			return httputil.NotFound("skill proposal not found: " + proposalID)
		}
		rec, err := s.require(p.NamespaceID, p.SkillName)
		if err != nil {
			return err
		}
		base := rec.Versions[p.BaseVersion]
		if base == nil {
			return httputil.NotFound("base version not found: " + p.BaseVersion)
		}
		target := opts.TargetVersion
		if target == "" {
			target = versioning.NextPatch(p.BaseVersion)
		}
		if rec.Versions[target] != nil {
			return httputil.Conflict("target version already exists: " + target)
		}
		merged := base.Skill
		merged.SkillMD = applyProposalDeltaToSkillMD(base.Skill.SkillMD, p)
		md5 := skillzip.ContentMD5(merged)
		now := model.NowMillis()
		status := model.VersionStatusReviewed
		if opts.Publish {
			status = model.VersionStatusOnline
		}
		if opts.Online {
			status = model.VersionStatusOnline
		}
		vr := &model.VersionRecord{Version: target, Status: status, Author: firstNonEmptyStr(opts.Reviewer, p.Source.AgentID, s.author), CommitMsg: "promoted from proposal " + p.ProposalID, CreateTime: now, UpdateTime: now, MD5: md5, Files: []string{"SKILL.md"}, Skill: merged}
		rec.Versions[target] = vr
		rec.UpdateTime = now
		if opts.Label != "" {
			if rec.Labels == nil {
				rec.Labels = map[string]string{}
			}
			rec.Labels[opts.Label] = target
		}
		p.Status = model.ProposalStatusPromoted
		p.CandidateVersion = target
		p.UpdateTime = now
		if err := s.store.Save(rec); err != nil {
			return err
		}
		if err := s.store.SaveProposal(p); err != nil {
			return err
		}
		s.invalidateSkill(p.NamespaceID, p.SkillName, target)
		out = p
		return nil
	})
	return out, err
}

func (s *Service) RejectSkillProposal(proposalID, reviewer, reason string) (*model.SkillProposal, error) {
	var out *model.SkillProposal
	err := s.store.WithWrite(func() error {
		p, err := s.store.LoadProposal(proposalID)
		if err != nil {
			return err
		}
		if p == nil {
			return httputil.NotFound("skill proposal not found: " + proposalID)
		}
		p.Status = model.ProposalStatusRejected
		p.UpdateTime = model.NowMillis()
		if p.Evidence == nil {
			p.Evidence = map[string]interface{}{}
		}
		p.Evidence["rejectReviewer"] = reviewer
		p.Evidence["rejectReason"] = reason
		if err := s.store.SaveProposal(p); err != nil {
			return err
		}
		out = p
		return nil
	})
	return out, err
}

func applyProposalDeltaToSkillMD(base string, p *model.SkillProposal) string {
	var b strings.Builder
	b.WriteString(strings.TrimRight(base, "\n"))
	b.WriteString("\n\n---\n\n")
	b.WriteString("## Agent Proposal: ")
	b.WriteString(p.ProposalID)
	b.WriteString("\n\n")
	if p.Reason != "" {
		b.WriteString("### Reason\n\n" + p.Reason + "\n\n")
	}
	if len(p.Delta) > 0 {
		pretty, _ := json.MarshalIndent(p.Delta, "", "  ")
		b.WriteString("### Delta\n\n```json\n")
		b.Write(pretty)
		b.WriteString("\n```\n")
	}
	return b.String()
}

func randHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

func firstNonEmptyStr(vs ...string) string {
	for _, v := range vs {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return "-"
}

func (s *Service) ListNamespaces() ([]*model.NamespaceInfo, error) { return s.store.ListNamespaces() }
func (s *Service) SaveNamespace(ns model.NamespaceInfo, operator string) (*model.NamespaceInfo, error) {
	if ns.NamespaceID == "" {
		ns.NamespaceID = model.DefaultNamespace
	}
	if ns.DisplayName == "" {
		ns.DisplayName = ns.NamespaceID
	}
	if ns.Visibility == "" {
		ns.Visibility = model.ScopePrivate
	}
	if ns.Owner == "" {
		ns.Owner = operator
	}
	if ns.Metadata == nil {
		ns.Metadata = map[string]interface{}{}
	}
	if err := s.store.SaveNamespace(&ns); err != nil {
		return nil, err
	}
	_ = s.AppendAudit(model.AuditLog{NamespaceID: ns.NamespaceID, ResourceType: "namespace", ResourceName: ns.NamespaceID, Action: "namespace.save", Operator: operator, Detail: map[string]interface{}{"visibility": ns.Visibility}})
	return &ns, nil
}
func (s *Service) GetNamespace(namespaceID string) (*model.NamespaceInfo, error) {
	return s.store.LoadNamespace(namespaceID)
}
func (s *Service) ListNamespaceMembers(namespaceID string, pageNo, pageSize int) (model.Page, error) {
	items, total, err := s.store.ListNamespaceMembers(model.NamespaceMemberQuery{NamespaceID: namespaceID, PageNo: pageNo, PageSize: pageSize})
	if err != nil {
		return model.Page{}, err
	}
	return model.Page{PageNumber: pageNo, PagesAvailable: 1, TotalCount: int(total), PageItems: items}, nil
}
func (s *Service) SaveNamespaceMember(m model.NamespaceMember, operator string) (*model.NamespaceMember, error) {
	if m.NamespaceID == "" {
		m.NamespaceID = model.DefaultNamespace
	}
	if m.SubjectID == "" {
		return nil, httputil.BadRequest("subjectId is required")
	}
	if len(m.Roles) == 0 {
		m.Roles = []string{model.AccessRoleViewer}
	}
	if err := s.store.SaveNamespaceMember(&m); err != nil {
		return nil, err
	}
	_ = s.AppendAudit(model.AuditLog{NamespaceID: m.NamespaceID, ResourceType: "namespace_member", ResourceName: m.SubjectID, Action: "namespace.member.save", Operator: operator, Detail: map[string]interface{}{"roles": m.Roles}})
	return &m, nil
}
func (s *Service) DeleteNamespaceMember(namespaceID, subjectID, operator string) error {
	if err := s.store.DeleteNamespaceMember(namespaceID, subjectID); err != nil {
		return err
	}
	_ = s.AppendAudit(model.AuditLog{NamespaceID: namespaceID, ResourceType: "namespace_member", ResourceName: subjectID, Action: "namespace.member.delete", Operator: operator})
	return nil
}

func (s *Service) SetSkillStar(namespaceID, skillName, subjectID string, starred bool) (*model.SkillSocialStats, error) {
	if subjectID == "" {
		subjectID = "anonymous"
	}
	if err := s.store.SetStar(namespaceID, skillName, subjectID, starred); err != nil {
		return nil, err
	}
	return s.store.GetSkillSocialStats(namespaceID, skillName, subjectID)
}
func (s *Service) RateSkill(namespaceID, skillName, subjectID string, rating int, comment string) (*model.SkillSocialStats, error) {
	if subjectID == "" {
		subjectID = "anonymous"
	}
	if err := s.store.SetRating(&model.RatingRecord{NamespaceID: namespaceID, SkillName: skillName, SubjectID: subjectID, Rating: rating, Comment: comment}); err != nil {
		return nil, err
	}
	return s.store.GetSkillSocialStats(namespaceID, skillName, subjectID)
}
func (s *Service) Subscribe(namespaceID, targetType, targetName, subjectID string, subscribed bool) error {
	if subjectID == "" {
		subjectID = "anonymous"
	}
	return s.store.SetSubscription(namespaceID, targetType, targetName, subjectID, subscribed)
}
func (s *Service) GetSkillSocialStats(namespaceID, skillName, subjectID string) (*model.SkillSocialStats, error) {
	return s.store.GetSkillSocialStats(namespaceID, skillName, subjectID)
}

func (s *Service) AppendAudit(l model.AuditLog) error {
	if l.CreateTime == 0 {
		l.CreateTime = model.NowMillis()
	}
	if l.Operator == "" {
		l.Operator = s.author
	}
	return s.store.AppendAudit(&l)
}
func (s *Service) ListAuditLogs(q model.AuditQuery) (model.Page, error) {
	items, total, err := s.store.ListAuditLogs(q)
	if err != nil {
		return model.Page{}, err
	}
	return model.Page{PageNumber: q.PageNo, PagesAvailable: 1, TotalCount: int(total), PageItems: items}, nil
}

func (s *Service) CreateToken(req model.TokenCreateRequest, operator string) (*model.TokenInfo, error) {
	if req.Name == "" {
		return nil, httputil.BadRequest("name is required")
	}
	if req.SubjectID == "" {
		return nil, httputil.BadRequest("subjectId is required")
	}
	tok := make([]byte, 24)
	_, _ = rand.Read(tok)
	secret := "skh_" + hex.EncodeToString(tok)
	keyID := "key_" + hex.EncodeToString(tok[:8])
	h := sha256.Sum256([]byte(secret))
	t := &model.TokenInfo{KeyID: keyID, Name: req.Name, SubjectID: req.SubjectID, SubjectType: req.SubjectType, Roles: req.Roles, Permissions: req.Permissions, Namespaces: req.Namespaces, Status: "active", ExpiresAt: req.ExpiresAt, CreateTime: model.NowMillis(), Token: secret, TokenHash: hex.EncodeToString(h[:])}
	if err := s.store.SaveToken(t); err != nil {
		return nil, err
	}
	_ = s.AppendAudit(model.AuditLog{ResourceType: "token", ResourceName: keyID, Action: "token.create", Operator: operator, Detail: map[string]interface{}{"subjectId": req.SubjectID}})
	return t, nil
}
func (s *Service) ListTokens(subjectID string) ([]*model.TokenInfo, error) {
	return s.store.ListTokens(subjectID)
}
func (s *Service) DeleteToken(keyID, operator string) error {
	if err := s.store.DeleteToken(keyID); err != nil {
		return err
	}
	_ = s.AppendAudit(model.AuditLog{ResourceType: "token", ResourceName: keyID, Action: "token.delete", Operator: operator})
	return nil
}
func (s *Service) SaveIdempotency(r *model.IdempotencyRecord) error {
	return s.store.SaveIdempotency(r)
}
func (s *Service) LoadIdempotency(key string) (*model.IdempotencyRecord, error) {
	return s.store.LoadIdempotency(key)
}

func (s *Service) AppendNotification(n *model.Notification) error {
	if n == nil {
		return nil
	}
	if n.ID == "" {
		n.ID = "ntf_" + randomHex(8)
	}
	if n.CreateTime == 0 {
		n.CreateTime = model.NowMillis()
	}
	return s.store.AppendNotification(n)
}
func (s *Service) ListNotifications(q model.NotificationQuery) (model.Page, error) {
	items, total, err := s.store.ListNotifications(q)
	if err != nil {
		return model.Page{}, err
	}
	return model.Page{PageNumber: q.PageNo, PagesAvailable: 1, TotalCount: int(total), PageItems: items}, nil
}
func (s *Service) MarkNotificationRead(subjectID, notificationID string) error {
	return s.store.MarkNotificationRead(subjectID, notificationID)
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *Service) notifySubscribers(namespaceID, targetType, targetName, eventType, title, message string, payload map[string]interface{}) {
	subs, err := s.store.ListSubscribers(namespaceID, targetType, targetName)
	if err != nil {
		return
	}
	for _, subject := range subs {
		_ = s.AppendNotification(&model.Notification{NamespaceID: namespaceID, SubjectID: subject, TargetType: targetType, TargetName: targetName, EventType: eventType, Title: title, Message: message, Payload: payload})
	}
}
