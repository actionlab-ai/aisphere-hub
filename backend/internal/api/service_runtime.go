package api

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/actionlab-ai/aisphere-hub/backend/internal/httputil"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/model"
	"github.com/gin-gonic/gin"
)

func (h *Handler) resolveAgentPlatformServices(c *gin.Context, definition model.AgentDefinition) ([]model.RuntimeServiceManifest, error) {
	refs := canonicalAgentServiceRefs(definition)
	items := make([]model.RuntimeServiceManifest, 0, len(refs))
	for _, ref := range refs {
		item, err := h.resolvePlatformService(c, ref)
		if err != nil {
			if ref.Required {
				return nil, err
			}
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func canonicalAgentServiceRefs(definition model.AgentDefinition) []model.AgentServiceRef {
	refs := make([]model.AgentServiceRef, 0, len(definition.Services)+len(definition.Skills)+len(definition.SkillSets)+len(definition.Tools))
	seen := map[string]int{}
	add := func(ref model.AgentServiceRef) {
		ref.Kind = normalizeRuntimeServiceKind(ref.Kind)
		ref.Name = strings.TrimSpace(ref.Name)
		ref.Provider = normalizeRuntimeServiceProvider(ref.Provider)
		ref.Reload = normalizeRuntimeServiceReload(ref.Reload, ref.Version, ref.Label)
		if ref.Kind == "" || ref.Name == "" {
			return
		}
		key := ref.Kind + ":" + ref.Name
		if ref.Alias != "" {
			key += ":" + ref.Alias
		}
		if idx, ok := seen[key]; ok {
			// Explicit services win over legacy fields, while later explicit refs may
			// override earlier ones to make local JSON editing predictable.
			refs[idx] = ref
			return
		}
		seen[key] = len(refs)
		refs = append(refs, ref)
	}
	for _, skillset := range definition.SkillSets {
		add(model.AgentServiceRef{Kind: model.ServiceKindSkillSet, Name: skillset.Name, Required: skillset.Required, Provider: model.ServiceProviderHub})
	}
	for _, skill := range definition.Skills {
		add(model.AgentServiceRef{Kind: model.ServiceKindSkill, Name: skill.Name, Version: skill.Version, Label: skill.Label, Required: skill.Required, Provider: model.ServiceProviderHub})
	}
	for _, tool := range definition.Tools {
		add(model.AgentServiceRef{Kind: model.ServiceKindTool, Name: tool.Name, Version: tool.Version, Label: tool.Label, Required: tool.Required, Provider: model.ServiceProviderHub})
	}
	for _, svc := range definition.Services {
		add(svc)
	}
	sort.Slice(refs, func(i, j int) bool {
		if refs[i].Kind == refs[j].Kind {
			return refs[i].Name < refs[j].Name
		}
		return refs[i].Kind < refs[j].Kind
	})
	return refs
}

func (h *Handler) resolvePlatformService(c *gin.Context, ref model.AgentServiceRef) (model.RuntimeServiceManifest, error) {
	ref.Kind = normalizeRuntimeServiceKind(ref.Kind)
	ref.Provider = normalizeRuntimeServiceProvider(ref.Provider)
	ref.Reload = normalizeRuntimeServiceReload(ref.Reload, ref.Version, ref.Label)
	base := model.RuntimeServiceManifest{
		Kind: ref.Kind, Name: strings.TrimSpace(ref.Name), Alias: strings.TrimSpace(ref.Alias), Provider: ref.Provider,
		Label: strings.TrimSpace(ref.Label), Required: ref.Required, Reload: ref.Reload, MountPath: strings.TrimSpace(ref.MountPath),
		Runtime: ref.Runtime, Config: ref.Config, Metadata: ref.Metadata,
	}
	if base.Alias == "" {
		base.Alias = base.Name
	}
	if base.Provider != model.ServiceProviderHub {
		base.Status = model.MetaStatusEnable
		base.ChangeToken = runtimeServiceChangeToken(base.Kind, base.Name, base.Version, base.Revision, base.Reload)
		base.SnapshotID = runtimeServiceSnapshotID(base.Kind, base.Name, base.Version, base.Revision)
		return h.resolveNestedRuntimeServices(c, base, ref.DependsOn)
	}
	switch base.Kind {
	case model.ServiceKindSkill:
		item, err := h.resolveAgentSkill(c, model.AgentSkillRef{Name: ref.Name, Version: ref.Version, Label: ref.Label, Required: ref.Required})
		if err != nil {
			return base, err
		}
		base.Object, base.Version, base.Revision = item.Object, item.Version, item.Revision
		base.Status = model.MetaStatusEnable
		base.Payload = map[string]interface{}{"downloadUrl": item.DownloadURL, "sha256": item.SHA256, "md5": item.MD5, "size": item.Size}
	case model.ServiceKindSkillSet:
		if allowed, _ := h.catalogCan(c, "skillset", ref.Name, "read"); !allowed {
			return base, forbiddenAgentDependency("skillset", ref.Name, "read")
		}
		manifest, err := h.svc.ResolveGroupManifest(aihubNS(), ref.Name, firstNonEmptyRuntime(ref.Label, model.LabelLatest))
		if err != nil {
			return base, err
		}
		base.Object = h.catalogObject("skillset", ref.Name)
		base.Version = firstNonEmptyRuntime(ref.Version, ref.Label, model.LabelLatest)
		base.Revision = shortHash(fmt.Sprintf("%s|%s|%d|%v", manifest.Name, manifest.Version, manifest.UpdateTime, manifest.Members))
		base.Status = model.MetaStatusEnable
		base.Payload = map[string]interface{}{"members": manifest.Members}
	case model.ServiceKindTool:
		item, err := h.resolveAgentTool(c, model.AgentToolRef{Name: ref.Name, Version: ref.Version, Label: ref.Label, Required: ref.Required})
		if err != nil {
			return base, err
		}
		base.Object, base.Version, base.Revision, base.Status = item.Object, item.Version, item.Revision, item.Status
		base.Runtime = toolRuntimeToMap(item.Runtime)
		base.Execution = toolExecutionToMap(item.Execution)
		base.Payload = map[string]interface{}{"inputSchema": item.InputSchema, "outputSchema": item.OutputSchema, "timeoutMillis": item.TimeoutMillis, "retry": item.Retry, "metadata": item.Metadata}
	case model.ServiceKindAgent:
		if allowed, _ := h.catalogCan(c, "agent", ref.Name, "run"); !allowed {
			return base, forbiddenAgentDependency("agent", ref.Name, "run")
		}
		rec, err := h.svc.GetAgent(aihubNS(), ref.Name)
		if err != nil {
			return base, err
		}
		if strings.EqualFold(rec.Status, model.MetaStatusDisable) {
			return base, httputil.Conflict("agent service is disabled: " + ref.Name)
		}
		version := selectRuntimeVersion(ref.Version, ref.Label, rec.Labels, rec.LatestVersion)
		vr := rec.Versions[version]
		if vr == nil {
			return base, httputil.NotFound("agent service version not found: " + version)
		}
		base.Object, base.Version, base.Revision, base.Status = h.catalogObject("agent", ref.Name), version, vr.Revision, rec.Status
		base.Payload = map[string]interface{}{"entryPoint": vr.Definition.EntryPoint, "files": vr.Definition.Files}
	case model.ServiceKindMCP:
		// MCP servers are services too. In this cut Hub acts as a permissioned
		// descriptor provider; concrete connection/secret resolution remains in
		// AgentKit runtime and sandbox policy.
		if ref.Provider == model.ServiceProviderHub {
			if allowed, _ := h.catalogCan(c, "mcp", ref.Name, "run"); !allowed {
				return base, forbiddenAgentDependency("mcp", ref.Name, "run")
			}
			base.Object = h.catalogObject("mcp", ref.Name)
		}
		base.Status = model.MetaStatusEnable
		base.Version = firstNonEmptyRuntime(ref.Version, ref.Label)
	default:
		return base, httputil.BadRequest("unsupported service kind: " + base.Kind)
	}
	base.ChangeToken = runtimeServiceChangeToken(base.Kind, base.Name, base.Version, base.Revision, base.Reload)
	base.SnapshotID = runtimeServiceSnapshotID(base.Kind, base.Name, base.Version, base.Revision)
	return h.resolveNestedRuntimeServices(c, base, ref.DependsOn)
}

func (h *Handler) resolveNestedRuntimeServices(c *gin.Context, base model.RuntimeServiceManifest, refs []model.AgentServiceRef) (model.RuntimeServiceManifest, error) {
	for _, dep := range refs {
		item, err := h.resolvePlatformService(c, dep)
		if err != nil {
			if dep.Required {
				return base, err
			}
			continue
		}
		base.DependsOn = append(base.DependsOn, item)
	}
	return base, nil
}

func normalizeRuntimeServiceKind(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
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

func normalizeRuntimeServiceProvider(provider string) string {
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

func normalizeRuntimeServiceReload(reload, version, label string) string {
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

func selectRuntimeVersion(version, label string, labels map[string]string, latest string) string {
	version = strings.TrimSpace(version)
	if version != "" {
		return version
	}
	label = strings.TrimSpace(label)
	if label != "" && labels != nil && strings.TrimSpace(labels[label]) != "" {
		return strings.TrimSpace(labels[label])
	}
	return strings.TrimSpace(latest)
}

func runtimeServiceChangeToken(kind, name, version, revision, reload string) string {
	return strings.Join([]string{kind, name, version, revision, reload}, ":")
}

func runtimeServiceSnapshotID(kind, name, version, revision string) string {
	return "svc_snap_" + shortHash(strings.Join([]string{kind, name, version, revision, time.Now().UTC().Format(time.RFC3339Nano)}, "|"))
}

func toolRuntimeToMap(rt model.ToolRuntimeDefinition) map[string]interface{} {
	out := map[string]interface{}{"type": rt.Type}
	if rt.Server != "" {
		out["server"] = rt.Server
	}
	if rt.Name != "" {
		out["name"] = rt.Name
	}
	if rt.URL != "" {
		out["url"] = rt.URL
	}
	if rt.Method != "" {
		out["method"] = rt.Method
	}
	if rt.Package != "" {
		out["package"] = rt.Package
	}
	if rt.EntryPoint != "" {
		out["entryPoint"] = rt.EntryPoint
	}
	if len(rt.Headers) > 0 {
		out["headers"] = rt.Headers
	}
	if len(rt.Config) > 0 {
		out["config"] = rt.Config
	}
	if rt.CredentialRef != "" {
		out["credentialRef"] = rt.CredentialRef
	}
	if rt.Description != "" {
		out["description"] = rt.Description
	}
	return out
}

func toolExecutionToMap(exec model.ToolExecutionDefinition) map[string]interface{} {
	out := map[string]interface{}{}
	if exec.Placement != "" {
		out["placement"] = exec.Placement
	}
	if exec.Runner != "" {
		out["runner"] = exec.Runner
	}
	if exec.Image != "" {
		out["image"] = exec.Image
	}
	if exec.Command != "" {
		out["command"] = exec.Command
	}
	if len(exec.Args) > 0 {
		out["args"] = exec.Args
	}
	if exec.WorkingDir != "" {
		out["workingDir"] = exec.WorkingDir
	}
	if exec.Filesystem != "" {
		out["filesystem"] = exec.Filesystem
	}
	if exec.Network != "" {
		out["network"] = exec.Network
	}
	if len(exec.Mounts) > 0 {
		out["mounts"] = exec.Mounts
	}
	if len(exec.Env) > 0 {
		out["env"] = exec.Env
	}
	if len(exec.SecretRefs) > 0 {
		out["secretRefs"] = exec.SecretRefs
	}
	if len(exec.AllowHosts) > 0 {
		out["allowHosts"] = exec.AllowHosts
	}
	if len(exec.DenyHosts) > 0 {
		out["denyHosts"] = exec.DenyHosts
	}
	if exec.Resources != nil {
		out["resources"] = exec.Resources
	}
	if len(exec.Capabilities) > 0 {
		out["capabilities"] = exec.Capabilities
	}
	return out
}

func serviceSnapshotRevision(name string, items []model.RuntimeServiceManifest) string {
	parts := make([]string, 0, len(items)+1)
	parts = append(parts, name)
	for _, item := range items {
		parts = append(parts, fmt.Sprintf("%s:%s@%s:%s:%s", item.Kind, item.Name, item.Version, item.Revision, item.ChangeToken))
	}
	return shortHash(strings.Join(parts, "|"))
}

func firstNonEmptyRuntime(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

type resolveRuntimeServicesRequest struct {
	RuntimeID string                  `json:"runtimeId"`
	SessionID string                  `json:"sessionId"`
	Services  []model.AgentServiceRef `json:"services"`
}

type runtimeServicesSnapshot struct {
	SnapshotID  string                         `json:"snapshotId"`
	RuntimeID   string                         `json:"runtimeId"`
	SessionID   string                         `json:"sessionId"`
	GeneratedAt string                         `json:"generatedAt"`
	ChangeToken string                         `json:"changeToken"`
	Services    []model.RuntimeServiceManifest `json:"services"`
}

func (h *Handler) resolveRuntimeServices(c *gin.Context) {
	var req resolveRuntimeServicesRequest
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
	items := make([]model.RuntimeServiceManifest, 0, len(req.Services))
	for _, ref := range req.Services {
		item, err := h.resolvePlatformService(c, ref)
		if err != nil {
			if ref.Required {
				httputil.Fail(c, err)
				return
			}
			continue
		}
		items = append(items, item)
	}
	changeToken := serviceSnapshotRevision("adhoc", items)
	c.JSON(200, runtimeServicesSnapshot{
		SnapshotID:  "services_snap_" + shortHash(req.RuntimeID+"|"+req.SessionID+"|"+changeToken),
		RuntimeID:   req.RuntimeID,
		SessionID:   req.SessionID,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		ChangeToken: changeToken,
		Services:    items,
	})
}
