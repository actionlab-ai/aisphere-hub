package model

// SandboxToolSchema is the model-facing tool contract exported by the sandbox
// tool server. It is safe to pass name/description/inputSchema to the LLM
// context. Runtime execution details such as Pod DNS, PVC names, secrets and
// Kubernetes policy must stay outside model context.
type SandboxToolSchema struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	InputSchema map[string]interface{} `json:"inputSchema,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type SandboxToolListResponse struct {
	SandboxID string              `json:"sandboxId,omitempty"`
	Endpoint  string              `json:"endpoint,omitempty"`
	Tools     []SandboxToolSchema `json:"tools"`
	// ModelTools is the safe, model-facing function schema list. It keeps the
	// original platform tool id in x_aisphere.toolId for ToolGateway dispatch.
	ModelTools []map[string]interface{} `json:"modelTools,omitempty"`
}

// SandboxToolCallRequest is the runtime-facing request used by AgentKit
// ToolGateway. ToolGateway can call the sandbox service directly, or call Hub's
// proxy endpoint for development/debugging.
type SandboxToolCallRequest struct {
	Tool          string                 `json:"tool"`
	Input         map[string]interface{} `json:"input,omitempty"`
	TraceID       string                 `json:"traceId,omitempty"`
	RunID         string                 `json:"runId,omitempty"`
	Attempt       int                    `json:"attempt,omitempty"`
	TimeoutMillis int64                  `json:"timeoutMillis,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

type SandboxToolError struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

type SandboxToolCallResult struct {
	OK             bool                   `json:"ok"`
	SandboxID      string                 `json:"sandboxId,omitempty"`
	Tool           string                 `json:"tool"`
	Result         map[string]interface{} `json:"result,omitempty"`
	Error          *SandboxToolError      `json:"error,omitempty"`
	Trace          string                 `json:"trace,omitempty"`
	TraceID        string                 `json:"traceId,omitempty"`
	DurationMillis int64                  `json:"durationMillis,omitempty"`
	Raw            map[string]interface{} `json:"raw,omitempty"`
}
