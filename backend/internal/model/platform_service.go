package model

const (
	ServiceKindAgent    = "agent"
	ServiceKindSkill    = "skill"
	ServiceKindSkillSet = "skillset"
	ServiceKindTool     = "tool"
	ServiceKindMCP      = "mcp"
	ServiceKindWorkflow = "workflow"

	ServiceProviderHub      = "hub"
	ServiceProviderInline   = "inline"
	ServiceProviderExternal = "external"

	ServiceReloadPinned      = "pinned"
	ServiceReloadFollowLabel = "follow_label"
	ServiceReloadLive        = "live"
)

// AgentServiceRef is the platform-level dependency contract used by AgentKit.
// Agent / Skill / SkillSet / Tool / MCP are all runtime services. Hub is only
// one provider implementation that resolves these references into immutable
// runtime manifests.
type AgentServiceRef struct {
	Kind      string                 `json:"kind"`                // agent / skill / skillset / tool / mcp / workflow
	Name      string                 `json:"name"`                // stable service id within its kind
	Alias     string                 `json:"alias,omitempty"`     // runtime name exposed to the agent
	Provider  string                 `json:"provider,omitempty"`  // hub / inline / external
	Version   string                 `json:"version,omitempty"`   // pinned version
	Label     string                 `json:"label,omitempty"`     // latest / stable / gray
	Required  bool                   `json:"required"`            // fail resolve when dependency is unavailable
	Reload    string                 `json:"reload,omitempty"`    // pinned / follow_label / live
	MountPath string                 `json:"mountPath,omitempty"` // optional sandbox mount path
	Runtime   map[string]interface{} `json:"runtime,omitempty"`   // inline/external runtime descriptor, no secrets
	Config    map[string]interface{} `json:"config,omitempty"`    // non-secret runtime config
	Metadata  map[string]interface{} `json:"metadata,omitempty"`  // tags for UI/debug
	DependsOn []AgentServiceRef      `json:"dependsOn,omitempty"` // nested runtime dependencies
}

// RuntimeServiceManifest is the resolved, permission-checked snapshot item
// consumed by AgentKit runtime. It is intentionally generic so the runtime can
// materialize any component through the same service dispatcher.
type RuntimeServiceManifest struct {
	Kind        string                   `json:"kind"`
	Name        string                   `json:"name"`
	Alias       string                   `json:"alias,omitempty"`
	Provider    string                   `json:"provider"`
	Object      string                   `json:"object,omitempty"`
	Version     string                   `json:"version,omitempty"`
	Label       string                   `json:"label,omitempty"`
	Revision    string                   `json:"revision,omitempty"`
	Status      string                   `json:"status,omitempty"`
	Required    bool                     `json:"required"`
	Reload      string                   `json:"reload"`
	MountPath   string                   `json:"mountPath,omitempty"`
	ChangeToken string                   `json:"changeToken,omitempty"`
	SnapshotID  string                   `json:"snapshotId,omitempty"`
	Runtime     map[string]interface{}   `json:"runtime,omitempty"`
	Execution   map[string]interface{}   `json:"execution,omitempty"`
	Config      map[string]interface{}   `json:"config,omitempty"`
	Payload     map[string]interface{}   `json:"payload,omitempty"`
	Metadata    map[string]interface{}   `json:"metadata,omitempty"`
	DependsOn   []RuntimeServiceManifest `json:"dependsOn,omitempty"`
}
