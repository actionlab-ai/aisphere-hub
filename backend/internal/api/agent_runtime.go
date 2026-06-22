package api

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/actionlab-ai/aisphere-hub/backend/internal/httputil"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/model"
	"github.com/gin-gonic/gin"
)

type resolveAgentRuntimeRequest struct {
	RuntimeID string `json:"runtimeId"`
	SessionID string `json:"sessionId"`
	Version   string `json:"version,omitempty"`
	Label     string `json:"label,omitempty"`
	Policy    string `json:"policy,omitempty"`
}

type agentRuntimeSnapshot struct {
	SnapshotID    string                         `json:"snapshotId"`
	RuntimeID     string                         `json:"runtimeId"`
	SessionID     string                         `json:"sessionId"`
	AgentID       string                         `json:"agentId"`
	AgentVersion  string                         `json:"agentVersion"`
	AgentRevision string                         `json:"agentRevision"`
	GeneratedAt   string                         `json:"generatedAt"`
	Policy        string                         `json:"policy"`
	Definition    model.AgentDefinition          `json:"definition"`
	Sandbox       model.AgentSandboxRef          `json:"sandbox,omitempty"`
	Services      []model.RuntimeServiceManifest `json:"services"`
	Skills        []catalogSkillSnapshotItem     `json:"skills"`
	Tools         []toolRuntimeSnapshotItem      `json:"tools,omitempty"`
	ChangeToken   string                         `json:"changeToken"`
}

func (h *Handler) resolveAgentRuntime(c *gin.Context) {
	id := strings.TrimSpace(c.Param("agentId"))
	if allowed, access := h.catalogCan(c, "agent", id, "run"); !allowed {
		forbiddenAgent(c, id, "run", access)
		return
	}
	var req resolveAgentRuntimeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.Fail(c, httputil.BadRequest(err.Error()))
		return
	}
	if strings.TrimSpace(req.RuntimeID) == "" {
		req.RuntimeID = principalID(c)
	}
	if strings.TrimSpace(req.SessionID) == "" {
		req.SessionID = "sess_" + time.Now().UTC().Format("20060102150405.000000000")
	}
	if strings.TrimSpace(req.Policy) == "" {
		req.Policy = "pinned_authorized"
	}
	record, err := h.svc.GetAgent(aihubNS(), id)
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	if strings.EqualFold(record.Status, model.MetaStatusDisable) {
		httputil.Fail(c, httputil.Conflict("agent is disabled: "+id))
		return
	}
	version := strings.TrimSpace(req.Version)
	if version == "" && strings.TrimSpace(req.Label) != "" && record.Labels != nil {
		version = record.Labels[strings.TrimSpace(req.Label)]
	}
	if version == "" {
		version = record.LatestVersion
	}
	agentVersion := record.Versions[version]
	if agentVersion == nil {
		httputil.Fail(c, httputil.NotFound("agent version not found: "+version))
		return
	}
	skills, err := h.resolveAgentSkills(c, agentVersion.Definition)
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	tools, err := h.resolveAgentTools(c, agentVersion.Definition)
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	services, err := h.resolveAgentPlatformServices(c, agentVersion.Definition)
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	changeToken := serviceSnapshotRevision(id+"@"+version, services)
	revision := shortHash(fmt.Sprintf("%s|%s|%s|%s|%s|%s", id, version, agentVersion.Revision, snapshotRevision(id+"@"+version, skills), toolSnapshotRevision(id+"@"+version, tools), changeToken))
	c.JSON(http.StatusOK, agentRuntimeSnapshot{
		SnapshotID: "agent_snap_" + shortHash(req.RuntimeID+"|"+req.SessionID+"|"+revision),
		RuntimeID:  req.RuntimeID, SessionID: req.SessionID, AgentID: id,
		AgentVersion: version, AgentRevision: agentVersion.Revision,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339), Policy: req.Policy,
		Definition: agentVersion.Definition, Sandbox: agentVersion.Definition.Sandbox, Services: services, Skills: skills, Tools: tools, ChangeToken: changeToken,
	})
}

func (h *Handler) resolveAgentSkills(c *gin.Context, definition model.AgentDefinition) ([]catalogSkillSnapshotItem, error) {
	refs := map[string]model.AgentSkillRef{}
	for _, svc := range canonicalAgentServiceRefs(definition) {
		switch svc.Kind {
		case model.ServiceKindSkillSet:
			name := strings.TrimSpace(svc.Name)
			if name == "" {
				continue
			}
			allowed, _ := h.catalogCan(c, "skillset", name, "read")
			if !allowed {
				if svc.Required {
					return nil, forbiddenAgentDependency("skillset", name, "read")
				}
				continue
			}
			manifest, err := h.svc.ResolveGroupManifest(aihubNS(), name, firstNonEmptyRuntime(svc.Label, model.LabelLatest))
			if err != nil {
				if svc.Required {
					return nil, err
				}
				continue
			}
			for _, member := range manifest.Members {
				if member.Name == "" {
					continue
				}
				refs[member.Name] = model.AgentSkillRef{Name: member.Name, Version: member.Version, Label: member.Label, Required: member.Required}
			}
		case model.ServiceKindSkill:
			if name := strings.TrimSpace(svc.Name); name != "" {
				refs[name] = model.AgentSkillRef{Name: name, Version: svc.Version, Label: svc.Label, Required: svc.Required}
			}
		}
	}
	names := make([]string, 0, len(refs))
	for name := range refs {
		names = append(names, name)
	}
	sort.Strings(names)
	items := make([]catalogSkillSnapshotItem, 0, len(names))
	for _, name := range names {
		item, err := h.resolveAgentSkill(c, refs[name])
		if err != nil {
			if refs[name].Required {
				return nil, err
			}
			continue
		}
		items = append(items, *item)
	}
	return items, nil
}

func (h *Handler) resolveAgentTools(c *gin.Context, definition model.AgentDefinition) ([]toolRuntimeSnapshotItem, error) {
	refs := map[string]model.AgentToolRef{}
	for _, svc := range canonicalAgentServiceRefs(definition) {
		if svc.Kind != model.ServiceKindTool {
			continue
		}
		if name := strings.TrimSpace(svc.Name); name != "" {
			refs[name] = model.AgentToolRef{Name: name, Version: svc.Version, Label: svc.Label, Required: svc.Required}
		}
	}
	names := make([]string, 0, len(refs))
	for name := range refs {
		names = append(names, name)
	}
	sort.Strings(names)
	items := make([]toolRuntimeSnapshotItem, 0, len(names))
	for _, name := range names {
		item, err := h.resolveAgentTool(c, refs[name])
		if err != nil {
			if refs[name].Required {
				return nil, err
			}
			continue
		}
		items = append(items, *item)
	}
	return items, nil
}

func (h *Handler) resolveAgentTool(c *gin.Context, ref model.AgentToolRef) (*toolRuntimeSnapshotItem, error) {
	name := strings.TrimSpace(ref.Name)
	if allowed, _ := h.catalogCan(c, "tool", name, "read"); !allowed {
		return nil, forbiddenAgentDependency("tool", name, "read")
	}
	if allowed, _ := h.catalogCan(c, "tool", name, "run"); !allowed {
		return nil, forbiddenAgentDependency("tool", name, "run")
	}
	rec, err := h.svc.GetTool(aihubNS(), name)
	if err != nil {
		return nil, err
	}
	if strings.EqualFold(rec.Status, model.MetaStatusDisable) {
		return nil, httputil.Conflict("tool is disabled: " + name)
	}
	version := strings.TrimSpace(ref.Version)
	if version == "" && strings.TrimSpace(ref.Label) != "" && rec.Labels != nil {
		version = rec.Labels[ref.Label]
	}
	if version == "" {
		version = rec.LatestVersion
	}
	vr := rec.Versions[version]
	if vr == nil {
		return nil, httputil.NotFound("tool version not found: " + version)
	}
	return &toolRuntimeSnapshotItem{
		Name: name, Version: version, Revision: vr.Revision, Object: h.catalogObject("tool", name), Status: rec.Status,
		Runtime: vr.Definition.Runtime, Execution: vr.Definition.Execution, InputSchema: vr.Definition.InputSchema, OutputSchema: vr.Definition.OutputSchema,
		TimeoutMillis: vr.Definition.TimeoutMillis, Retry: vr.Definition.Retry, Metadata: vr.Definition.Metadata,
	}, nil
}

func toolSnapshotRevision(name string, items []toolRuntimeSnapshotItem) string {
	parts := make([]string, 0, len(items)+1)
	parts = append(parts, name)
	for _, item := range items {
		parts = append(parts, item.Name+"@"+item.Version+":"+item.Revision)
	}
	return shortHash(strings.Join(parts, "|"))
}

func (h *Handler) resolveAgentSkill(c *gin.Context, ref model.AgentSkillRef) (*catalogSkillSnapshotItem, error) {
	name := strings.TrimSpace(ref.Name)
	if allowed, _ := h.catalogCan(c, "skill", name, "read"); !allowed {
		return nil, forbiddenAgentDependency("skill", name, "read")
	}
	if allowed, _ := h.catalogCan(c, "skill", name, "download"); !allowed {
		return nil, forbiddenAgentDependency("skill", name, "download")
	}
	meta, err := h.svc.GetSkillDetail(aihubNS(), name)
	if err != nil {
		return nil, err
	}
	version := strings.TrimSpace(ref.Version)
	if version == "" && strings.TrimSpace(ref.Label) != "" && meta.Labels != nil {
		version = meta.Labels[ref.Label]
	}
	if version == "" {
		version = latestCatalogVersion(meta)
	}
	if version == "" {
		return nil, httputil.NotFound("no runnable version for skill: " + name)
	}
	entry, err := h.svc.GetSkillVersionRecord(aihubNS(), name, version)
	if err != nil {
		return nil, err
	}
	return &catalogSkillSnapshotItem{
		Name: name, Version: version, Revision: skillVersionRevision(name, entry), Object: h.catalogObject("skill", name),
		SHA256: versionSHA256(entry), MD5: entry.MD5, Size: versionSize(entry),
		DownloadURL: fmt.Sprintf("/v3/aihub/catalog/skills/%s/versions/%s/download", name, version),
	}, nil
}

func forbiddenAgentDependency(resourceType, resourceID, action string) error {
	return &httputil.AppError{Status: http.StatusForbidden, Code: http.StatusForbidden, Message: "forbidden: object=aihub:" + resourceType + ":" + resourceID + " action=" + action}
}
