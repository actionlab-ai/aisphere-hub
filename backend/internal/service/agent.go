package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/actionlab-ai/aisphere-hub/backend/internal/httputil"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/model"
)

func (s *Service) CreateAgent(namespaceID string, req model.AgentUpsertRequest, ownerSubject string) (*model.AgentRecord, error) {
	if err := validateAgentRequest(req, false); err != nil {
		return nil, err
	}
	var created *model.AgentRecord
	err := s.store.WithWrite(func() error {
		existing, err := s.store.LoadAgent(namespaceID, req.ID)
		if err != nil {
			return err
		}
		if existing != nil {
			return httputil.Conflict("agent already exists: " + req.ID)
		}
		now := model.NowMillis()
		version := firstNonEmptyAgent(req.Version, "1.0.0")
		status, err := normalizedResourceStatus(req.Status)
		if err != nil {
			return err
		}
		definition := normalizedAgentDefinition(req.Definition)
		created = &model.AgentRecord{
			NamespaceID: namespaceID, App: model.DefaultApp, OwnerSubject: strings.TrimSpace(ownerSubject),
			ID: req.ID, DisplayName: strings.TrimSpace(req.DisplayName), Description: strings.TrimSpace(req.Description),
			Status: status, Scope: firstNonEmptyAgent(req.Scope, model.ScopePrivate),
			Labels: normalizedResourceLabels(req.Labels, version), LatestVersion: version, CreateTime: now, UpdateTime: now,
			Versions: map[string]*model.AgentVersionRecord{version: newAgentVersion(version, definition, ownerSubject, req.CommitMsg, now)},
		}
		if err := s.store.SaveAgent(created); err != nil {
			return err
		}
		s.emitCatalogEvent(model.CatalogEventAgentUpdated, "agent", created.ID, "", version, created.Versions[version].Revision, map[string]interface{}{"status": created.Status, "latest": created.LatestVersion})
		return nil
	})
	return created, err
}

func (s *Service) GetAgent(namespaceID, id string) (*model.AgentRecord, error) {
	if strings.TrimSpace(id) == "" {
		return nil, httputil.BadRequest("agent id is required")
	}
	rec, err := s.store.LoadAgent(namespaceID, strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, httputil.NotFound("agent not found: " + id)
	}
	return rec, nil
}

func (s *Service) ListAgents(namespaceID string) ([]*model.AgentRecord, error) {
	return s.store.ListAgents(namespaceID)
}

func (s *Service) UpdateAgent(namespaceID, id string, req model.AgentUpsertRequest, operator string) (*model.AgentRecord, error) {
	id = strings.TrimSpace(id)
	if req.ID != "" && req.ID != id {
		return nil, httputil.BadRequest("agent id cannot be changed")
	}
	req.ID = id
	if err := validateAgentRequest(req, false); err != nil {
		return nil, err
	}
	var updated *model.AgentRecord
	err := s.store.WithWrite(func() error {
		rec, err := s.store.LoadAgent(namespaceID, id)
		if err != nil {
			return err
		}
		if rec == nil {
			return httputil.NotFound("agent not found: " + id)
		}
		version := firstNonEmptyAgent(req.Version, nextAgentVersion(rec.LatestVersion))
		if rec.Versions[version] != nil {
			return httputil.Conflict("agent version already exists: " + version)
		}
		rec.DisplayName = strings.TrimSpace(req.DisplayName)
		rec.Description = strings.TrimSpace(req.Description)
		if strings.TrimSpace(req.Status) != "" {
			status, err := normalizedResourceStatus(req.Status)
			if err != nil {
				return err
			}
			rec.Status = status
		}
		if strings.TrimSpace(req.Scope) != "" {
			rec.Scope = strings.TrimSpace(req.Scope)
		}
		now := model.NowMillis()
		rec.UpdateTime = now
		rec.LatestVersion = version
		rec.Labels = mergeResourceLabels(rec.Labels, req.Labels, version)
		rec.Versions[version] = newAgentVersion(version, normalizedAgentDefinition(req.Definition), operator, req.CommitMsg, now)
		if err := s.store.SaveAgent(rec); err != nil {
			return err
		}
		s.emitCatalogEvent(model.CatalogEventAgentUpdated, "agent", rec.ID, "", version, rec.Versions[version].Revision, map[string]interface{}{"status": rec.Status, "latest": rec.LatestVersion})
		updated = rec
		return nil
	})
	return updated, err
}

func (s *Service) DeleteAgent(namespaceID, id string) error {
	if strings.TrimSpace(id) == "" {
		return httputil.BadRequest("agent id is required")
	}
	return s.store.WithWrite(func() error {
		rec, err := s.store.LoadAgent(namespaceID, strings.TrimSpace(id))
		if err != nil {
			return err
		}
		if rec == nil {
			return httputil.NotFound("agent not found: " + id)
		}
		if err := s.store.DeleteAgent(namespaceID, id); err != nil {
			return err
		}
		s.emitCatalogEvent(model.CatalogEventAgentDeleted, "agent", id, "", rec.LatestVersion, "", nil)
		return nil
	})
}

func newAgentVersion(version string, definition model.AgentDefinition, author, commitMsg string, now int64) *model.AgentVersionRecord {
	b, _ := json.Marshal(definition)
	sum := sha256.Sum256(b)
	digest := hex.EncodeToString(sum[:])
	return &model.AgentVersionRecord{Version: version, Revision: digest[:16], SHA256: digest, Author: strings.TrimSpace(author), CommitMsg: commitMsg, CreateTime: now, Definition: definition}
}

func validateAgentRequest(req model.AgentUpsertRequest, allowEmptyDefinition bool) error {
	if strings.TrimSpace(req.ID) == "" {
		return httputil.BadRequest("agent id is required")
	}
	for _, r := range req.ID {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_') {
			return httputil.BadRequest("agent id may contain only letters, numbers, hyphen, and underscore")
		}
	}
	if strings.TrimSpace(req.Version) != "" && !isAgentVersion(req.Version) {
		return httputil.BadRequest("agent version must use numeric semantic version format")
	}
	if allowEmptyDefinition && len(req.Definition.Files) == 0 {
		return nil
	}
	definition := normalizedAgentDefinition(req.Definition)
	if definition.EntryPoint == "" {
		return httputil.BadRequest("agent definition entryPoint is required")
	}
	if _, ok := definition.Files[definition.EntryPoint]; !ok {
		return httputil.BadRequest("agent definition must include entryPoint file")
	}
	for file := range definition.Files {
		if !isSafeAgentFilePath(file) {
			return httputil.BadRequest("agent definition contains an invalid file path")
		}
	}
	for _, svc := range definition.Services {
		if strings.TrimSpace(svc.Kind) == "" || strings.TrimSpace(svc.Name) == "" {
			return httputil.BadRequest("agent service ref requires kind and name")
		}
		if svc.MountPath != "" && !isSafeAgentFilePath(svc.MountPath) {
			return httputil.BadRequest("agent service ref contains an invalid mountPath")
		}
	}
	return nil
}

func normalizedAgentDefinition(in model.AgentDefinition) model.AgentDefinition {
	out := model.AgentDefinition{
		EntryPoint: strings.TrimPrefix(strings.TrimSpace(in.EntryPoint), "./"),
		Files:      map[string]string{},
		Services:   normalizeAgentServiceRefs(in.Services),
		Skills:     append([]model.AgentSkillRef(nil), in.Skills...),
		SkillSets:  append([]model.AgentSkillSetRef(nil), in.SkillSets...),
		Tools:      append([]model.AgentToolRef(nil), in.Tools...),
	}
	for file, content := range in.Files {
		out.Files[strings.TrimPrefix(strings.ReplaceAll(file, "\\", "/"), "./")] = content
	}
	sort.Slice(out.Skills, func(i, j int) bool { return out.Skills[i].Name < out.Skills[j].Name })
	sort.Slice(out.SkillSets, func(i, j int) bool { return out.SkillSets[i].Name < out.SkillSets[j].Name })
	sort.Slice(out.Tools, func(i, j int) bool { return out.Tools[i].Name < out.Tools[j].Name })
	sort.Slice(out.Services, func(i, j int) bool {
		if out.Services[i].Kind == out.Services[j].Kind {
			return out.Services[i].Name < out.Services[j].Name
		}
		return out.Services[i].Kind < out.Services[j].Kind
	})
	return out
}

func normalizeAgentServiceRefs(in []model.AgentServiceRef) []model.AgentServiceRef {
	out := make([]model.AgentServiceRef, 0, len(in))
	for _, ref := range in {
		ref.Kind = normalizeServiceKind(ref.Kind)
		ref.Name = strings.TrimSpace(ref.Name)
		ref.Alias = strings.TrimSpace(ref.Alias)
		ref.Provider = normalizeServiceProvider(ref.Provider)
		ref.Version = strings.TrimSpace(ref.Version)
		ref.Label = strings.TrimSpace(ref.Label)
		ref.Reload = normalizeServiceReload(ref.Reload, ref.Version, ref.Label)
		ref.MountPath = strings.TrimPrefix(strings.TrimSpace(strings.ReplaceAll(ref.MountPath, "\\", "/")), "./")
		ref.DependsOn = normalizeAgentServiceRefs(ref.DependsOn)
		if ref.Kind == "" || ref.Name == "" {
			continue
		}
		out = append(out, ref)
	}
	return out
}

func normalizeServiceKind(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "", "service":
		return ""
	case "skill", "skills":
		return model.ServiceKindSkill
	case "skillset", "skill-set", "skill_group", "skill-group", "group":
		return model.ServiceKindSkillSet
	case "tool", "tools":
		return model.ServiceKindTool
	case "mcp", "mcp_server", "mcp-server":
		return model.ServiceKindMCP
	case "agent", "agents":
		return model.ServiceKindAgent
	case "workflow", "workflows":
		return model.ServiceKindWorkflow
	default:
		return strings.ToLower(strings.TrimSpace(kind))
	}
}

func normalizeServiceProvider(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "", "hub", "aihub":
		return model.ServiceProviderHub
	case "inline", "local":
		return model.ServiceProviderInline
	case "external", "remote":
		return model.ServiceProviderExternal
	default:
		return strings.ToLower(strings.TrimSpace(provider))
	}
}

func normalizeServiceReload(reload, version, label string) string {
	switch strings.ToLower(strings.TrimSpace(reload)) {
	case model.ServiceReloadPinned, model.ServiceReloadFollowLabel, model.ServiceReloadLive:
		return strings.ToLower(strings.TrimSpace(reload))
	}
	if strings.TrimSpace(version) != "" {
		return model.ServiceReloadPinned
	}
	if strings.TrimSpace(label) != "" {
		return model.ServiceReloadFollowLabel
	}
	return model.ServiceReloadLive
}

func isSafeAgentFilePath(file string) bool {
	file = strings.TrimSpace(strings.ReplaceAll(file, "\\", "/"))
	return file != "" && !strings.HasPrefix(file, "/") && path.Clean(file) == file && !strings.HasPrefix(file, "../") && file != ".."
}

func isAgentVersion(v string) bool {
	parts := strings.Split(strings.TrimSpace(v), ".")
	if len(parts) != 3 {
		return false
	}
	for _, part := range parts {
		if _, err := strconv.Atoi(part); err != nil {
			return false
		}
	}
	return true
}

func nextAgentVersion(current string) string {
	parts := strings.Split(current, ".")
	if len(parts) != 3 {
		return "1.0.0"
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return "1.0.0"
	}
	return parts[0] + "." + parts[1] + "." + strconv.Itoa(patch+1)
}

func firstNonEmptyAgent(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
