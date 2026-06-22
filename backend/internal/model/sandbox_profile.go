package model

type SandboxProfile struct {
	NamespaceID     string                     `json:"-"`
	ID              string                     `json:"id"`
	Version         string                     `json:"version,omitempty"`
	Status          string                     `json:"status,omitempty"`
	DisplayName     string                     `json:"displayName,omitempty"`
	Description     string                     `json:"description,omitempty"`
	Driver          string                     `json:"driver,omitempty"`
	TemplateRef     string                     `json:"templateRef,omitempty"`
	WarmPoolRef     string                     `json:"warmPoolRef,omitempty"`
	ClusterSelector map[string]string          `json:"clusterSelector,omitempty"`
	Network         SandboxProfileNetwork      `json:"network,omitempty"`
	Workspace       SandboxProfileWorkspace    `json:"workspace,omitempty"`
	Resources       map[string]string          `json:"resources,omitempty"`
	Capabilities    SandboxProfileCapabilities `json:"capabilities,omitempty"`
	BuiltinTools    []string                   `json:"builtinTools,omitempty"`
	Labels          map[string]string          `json:"labels,omitempty"`
	Metadata        map[string]interface{}     `json:"metadata,omitempty"`
	CreateTime      int64                      `json:"createTime,omitempty"`
	UpdateTime      int64                      `json:"updateTime,omitempty"`
}

type SandboxProfileNetwork struct {
	Mode        string   `json:"mode,omitempty"`
	EgressCIDRs []string `json:"egressCidrs,omitempty"`
}
type SandboxProfileWorkspace struct {
	Size      string `json:"size,omitempty"`
	MountPath string `json:"mountPath,omitempty"`
	Reuse     string `json:"reuse,omitempty"`
}
type SandboxProfileCapabilities struct {
	Browser     bool `json:"browser"`
	Shell       bool `json:"shell"`
	MCP         bool `json:"mcp"`
	Online      bool `json:"online"`
	CustomTools bool `json:"customTools"`
}

type SandboxPolicy struct {
	NamespaceID     string                 `json:"-"`
	ID              string                 `json:"id"`
	TargetType      string                 `json:"targetType"`
	TargetID        string                 `json:"targetId"`
	AllowedProfiles []string               `json:"allowedProfiles,omitempty"`
	Limits          SandboxPolicyLimits    `json:"limits,omitempty"`
	Labels          map[string]string      `json:"labels,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	CreateTime      int64                  `json:"createTime,omitempty"`
	UpdateTime      int64                  `json:"updateTime,omitempty"`
}

type SandboxPolicyLimits struct {
	MaxConcurrentSandboxes int    `json:"maxConcurrentSandboxes,omitempty"`
	MaxCPU                 string `json:"maxCpu,omitempty"`
	MaxMemory              string `json:"maxMemory,omitempty"`
	MaxWorkspaceTotal      string `json:"maxWorkspaceTotal,omitempty"`
	AllowOnline            bool   `json:"allowOnline"`
	AllowShell             bool   `json:"allowShell"`
	AllowBrowser           bool   `json:"allowBrowser"`
	AllowCustomImage       bool   `json:"allowCustomImage"`
}
