package model

// ModelProfile is the Hub-side declaration of a model runtime profile.
// It intentionally stores product/runtime metadata but never stores real upstream API keys.
// Real provider keys live in aisphere-gateway Secret/env config.
type ModelProfile struct {
	NamespaceID   string                 `json:"-"`
	ID            string                 `json:"id"`
	Version       string                 `json:"version,omitempty"`
	Status        string                 `json:"status,omitempty"`
	DisplayName   string                 `json:"displayName,omitempty"`
	Description   string                 `json:"description,omitempty"`
	Provider      string                 `json:"provider,omitempty"`
	APIFormat     string                 `json:"apiFormat,omitempty"`
	Endpoint      string                 `json:"endpoint,omitempty"`
	Model         string                 `json:"model,omitempty"`
	UpstreamModel string                 `json:"upstreamModel,omitempty"`
	UpstreamPath  string                 `json:"upstreamPath,omitempty"`
	SecretRef     string                 `json:"secretRef,omitempty"`
	Headers       map[string]string      `json:"headers,omitempty"`
	AllowedTools  []string               `json:"allowedTools,omitempty"`
	Limits        ModelProfileLimits     `json:"limits,omitempty"`
	Reasoning     map[string]interface{} `json:"reasoning,omitempty"`
	Labels        map[string]string      `json:"labels,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	CreateTime    int64                  `json:"createTime,omitempty"`
	UpdateTime    int64                  `json:"updateTime,omitempty"`
}

type ModelProfileLimits struct {
	MaxInputTokens  int64 `json:"maxInputTokens,omitempty"`
	MaxOutputTokens int64 `json:"maxOutputTokens,omitempty"`
}
