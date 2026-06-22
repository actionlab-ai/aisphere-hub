package model

// AgentDefinition is the immutable runtime payload for one Agent version.
// Files are relative to the Agent package root and must include EntryPoint.
type AgentDefinition struct {
	EntryPoint string            `json:"entryPoint"`
	Files      map[string]string `json:"files"`
	// Services is the canonical AgentKit dependency list. Legacy fields below
	// are still accepted and are normalized into service manifests during
	// runtime resolve.
	Sandbox   AgentSandboxRef    `json:"sandbox,omitempty"`
	Model     AgentModelRef      `json:"model,omitempty"`
	Services  []AgentServiceRef  `json:"services,omitempty"`
	Skills    []AgentSkillRef    `json:"skills,omitempty"`
	SkillSets []AgentSkillSetRef `json:"skillSets,omitempty"`
	Tools     []AgentToolRef     `json:"tools,omitempty"`
}

type AgentModelRef struct {
	Profile string `json:"profile,omitempty"`
	Model   string `json:"model,omitempty"`
}

type AgentSandboxRef struct {
	Profile     string `json:"profile,omitempty"`
	Reuse       string `json:"reuse,omitempty"`
	TemplateRef string `json:"templateRef,omitempty"`
	WarmPoolRef string `json:"warmPoolRef,omitempty"`
}

type AgentSkillRef struct {
	Name     string `json:"name"`
	Version  string `json:"version,omitempty"`
	Label    string `json:"label,omitempty"`
	Required bool   `json:"required"`
}

type AgentSkillSetRef struct {
	Name     string `json:"name"`
	Required bool   `json:"required"`
}

type AgentToolRef struct {
	Name     string `json:"name"`
	Version  string `json:"version,omitempty"`
	Label    string `json:"label,omitempty"`
	Required bool   `json:"required"`
}

type AgentVersionRecord struct {
	Version    string          `json:"version"`
	Revision   string          `json:"revision"`
	SHA256     string          `json:"sha256"`
	Author     string          `json:"author,omitempty"`
	CommitMsg  string          `json:"commitMsg,omitempty"`
	CreateTime int64           `json:"createTime"`
	Definition AgentDefinition `json:"definition"`
}

type AgentRecord struct {
	NamespaceID   string                         `json:"-"`
	App           string                         `json:"app,omitempty"`
	OrgID         string                         `json:"orgId,omitempty"`
	ProjectID     string                         `json:"projectId,omitempty"`
	OwnerSubject  string                         `json:"ownerSubject,omitempty"`
	ID            string                         `json:"id"`
	DisplayName   string                         `json:"displayName,omitempty"`
	Description   string                         `json:"description,omitempty"`
	Status        string                         `json:"status"`
	Scope         string                         `json:"scope,omitempty"`
	Labels        map[string]string              `json:"labels,omitempty"`
	LatestVersion string                         `json:"latestVersion"`
	Versions      map[string]*AgentVersionRecord `json:"versions"`
	CreateTime    int64                          `json:"createTime"`
	UpdateTime    int64                          `json:"updateTime"`
}

type AgentUpsertRequest struct {
	ID          string            `json:"id,omitempty"`
	DisplayName string            `json:"displayName,omitempty"`
	Description string            `json:"description,omitempty"`
	Status      string            `json:"status,omitempty"`
	Scope       string            `json:"scope,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Version     string            `json:"version,omitempty"`
	CommitMsg   string            `json:"commitMsg,omitempty"`
	Definition  AgentDefinition   `json:"definition"`
}
