package model

const (
	// ToolExecutionPlacementSandbox means the tool implementation is launched
	// inside the per-session sandbox. This is the default for file/project/shell
	// and package based tools because filesystem access must be scoped to the
	// session workspace.
	ToolExecutionPlacementSandbox = "sandbox"
	// ToolExecutionPlacementRuntime is reserved for trusted AgentKit platform
	// tools that do not need tenant filesystem access. Avoid it for user/project
	// data plane tools.
	ToolExecutionPlacementRuntime = "runtime"
	// ToolExecutionPlacementRemote means AgentKit calls an external service or
	// remote MCP server through the tool gateway.
	ToolExecutionPlacementRemote = "remote"
	// ToolExecutionPlacementHub is for control-plane tools implemented by Hub
	// APIs, such as catalog or grant operations. These should be rare.
	ToolExecutionPlacementHub = "hub"

	ToolFilesystemReadonly  = "readonly"
	ToolFilesystemWorkspace = "workspace"
	ToolFilesystemNone      = "none"

	ToolNetworkNone       = "none"
	ToolNetworkEgress     = "egress"
	ToolNetworkRestricted = "restricted"
)

// ToolDefinition is the immutable Hub-side definition of a tool that can be
// resolved into a runtime manifest for AgentKit. It intentionally stores only
// non-secret routing/config metadata; credentials must be resolved by runtime
// secret providers or MCP server configuration.
type ToolDefinition struct {
	Runtime       ToolRuntimeDefinition   `json:"runtime"`
	Execution     ToolExecutionDefinition `json:"execution,omitempty"`
	InputSchema   map[string]interface{}  `json:"inputSchema,omitempty"`
	OutputSchema  map[string]interface{}  `json:"outputSchema,omitempty"`
	TimeoutMillis int64                   `json:"timeoutMillis,omitempty"`
	Retry         *ToolRetryPolicy        `json:"retry,omitempty"`
	Metadata      map[string]interface{}  `json:"metadata,omitempty"`
}

type ToolRuntimeDefinition struct {
	Type          string                 `json:"type"` // builtin / mcp / openapi / http / function
	Server        string                 `json:"server,omitempty"`
	Name          string                 `json:"name,omitempty"`
	URL           string                 `json:"url,omitempty"`
	Method        string                 `json:"method,omitempty"`
	Package       string                 `json:"package,omitempty"`
	EntryPoint    string                 `json:"entryPoint,omitempty"`
	Headers       map[string]string      `json:"headers,omitempty"`
	Config        map[string]interface{} `json:"config,omitempty"`
	CredentialRef string                 `json:"credentialRef,omitempty"`
	Description   string                 `json:"description,omitempty"`
}

type ToolExecutionDefinition struct {
	// Placement decides where the tool implementation runs. Hub only stores the
	// declaration; AgentKit Tool Gateway performs the actual placement.
	Placement string `json:"placement,omitempty"` // sandbox / runtime / remote / hub
	// Runner describes how the sandbox/runtime starts the tool: builtin, mcp,
	// stdio, http, container, wasm, python, binary.
	Runner string `json:"runner,omitempty"`
	// Image is used when Runner=container. Store image refs, never image bytes.
	Image string `json:"image,omitempty"`
	// Command and Args are used by stdio/binary/container runners.
	Command string   `json:"command,omitempty"`
	Args    []string `json:"args,omitempty"`
	// WorkingDir is interpreted inside the sandbox/container.
	WorkingDir string `json:"workingDir,omitempty"`
	// Filesystem controls workspace access. Defaults are derived from placement.
	Filesystem string `json:"filesystem,omitempty"` // none / readonly / workspace
	Network    string `json:"network,omitempty"`    // none / restricted / egress
	// Mounts are logical sandbox mounts. Host paths are intentionally not allowed
	// in Hub definitions; runtime maps logical refs to real sandbox paths.
	Mounts []ToolMountDefinition `json:"mounts,omitempty"`
	// Env may contain non-secret env values only. SecretRefs are resolved by the
	// runtime secret broker and injected into the sandbox or remote connector.
	Env        map[string]string `json:"env,omitempty"`
	SecretRefs []string          `json:"secretRefs,omitempty"`
	// Host allow/deny lists are enforced by the tool gateway for remote/http/mcp.
	AllowHosts []string            `json:"allowHosts,omitempty"`
	DenyHosts  []string            `json:"denyHosts,omitempty"`
	Resources  *ToolResourceLimits `json:"resources,omitempty"`
	// Capabilities are human-readable and policy-readable claims, for example:
	// filesystem.read, filesystem.write, network.egress, process.exec.
	Capabilities []string `json:"capabilities,omitempty"`
}

type ToolMountDefinition struct {
	Name      string `json:"name"`
	Ref       string `json:"ref"`            // workspace://, artifact://, skill://, secret:// (reference only)
	MountPath string `json:"mountPath"`      // path inside sandbox
	Mode      string `json:"mode,omitempty"` // ro / rw
}

type ToolResourceLimits struct {
	CPU            string `json:"cpu,omitempty"`
	Memory         string `json:"memory,omitempty"`
	TimeoutMillis  int64  `json:"timeoutMillis,omitempty"`
	MaxOutputBytes int64  `json:"maxOutputBytes,omitempty"`
}

type ToolRetryPolicy struct {
	MaxAttempts       int      `json:"maxAttempts,omitempty"`
	BackoffMillis     int64    `json:"backoffMillis,omitempty"`
	RetryOnErrorCodes []string `json:"retryOnErrorCodes,omitempty"`
}

type ToolVersionRecord struct {
	Version    string         `json:"version"`
	Revision   string         `json:"revision"`
	SHA256     string         `json:"sha256"`
	Author     string         `json:"author,omitempty"`
	CommitMsg  string         `json:"commitMsg,omitempty"`
	CreateTime int64          `json:"createTime"`
	Definition ToolDefinition `json:"definition"`
}

type ToolRecord struct {
	NamespaceID   string                        `json:"-"`
	App           string                        `json:"app,omitempty"`
	OrgID         string                        `json:"orgId,omitempty"`
	ProjectID     string                        `json:"projectId,omitempty"`
	OwnerSubject  string                        `json:"ownerSubject,omitempty"`
	ID            string                        `json:"id"`
	DisplayName   string                        `json:"displayName,omitempty"`
	Description   string                        `json:"description,omitempty"`
	Status        string                        `json:"status"`
	Scope         string                        `json:"scope,omitempty"`
	Labels        map[string]string             `json:"labels,omitempty"`
	LatestVersion string                        `json:"latestVersion"`
	Versions      map[string]*ToolVersionRecord `json:"versions"`
	CreateTime    int64                         `json:"createTime"`
	UpdateTime    int64                         `json:"updateTime"`
}

type ToolUpsertRequest struct {
	ID          string            `json:"id,omitempty"`
	DisplayName string            `json:"displayName,omitempty"`
	Description string            `json:"description,omitempty"`
	Status      string            `json:"status,omitempty"`
	Scope       string            `json:"scope,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Version     string            `json:"version,omitempty"`
	CommitMsg   string            `json:"commitMsg,omitempty"`
	Definition  ToolDefinition    `json:"definition"`
}

type ToolFailureReportRequest struct {
	ToolID         string                 `json:"toolId"`
	ToolVersion    string                 `json:"toolVersion,omitempty"`
	AgentID        string                 `json:"agentId,omitempty"`
	AgentVersion   string                 `json:"agentVersion,omitempty"`
	RuntimeID      string                 `json:"runtimeId,omitempty"`
	SessionID      string                 `json:"sessionId,omitempty"`
	RunID          string                 `json:"runId,omitempty"`
	TraceID        string                 `json:"traceId,omitempty"`
	SnapshotID     string                 `json:"snapshotId,omitempty"`
	Attempt        int                    `json:"attempt,omitempty"`
	ErrorCode      string                 `json:"errorCode,omitempty"`
	ErrorMessage   string                 `json:"errorMessage"`
	Retryable      bool                   `json:"retryable,omitempty"`
	InputDigest    string                 `json:"inputDigest,omitempty"`
	InputPreview   string                 `json:"inputPreview,omitempty"`
	DurationMillis int64                  `json:"durationMillis,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

type ToolFailureRecord struct {
	ID             string                 `json:"id"`
	App            string                 `json:"app,omitempty"`
	NamespaceID    string                 `json:"namespaceId,omitempty"`
	Object         string                 `json:"object,omitempty"`
	ToolID         string                 `json:"toolId"`
	ToolVersion    string                 `json:"toolVersion,omitempty"`
	AgentID        string                 `json:"agentId,omitempty"`
	AgentVersion   string                 `json:"agentVersion,omitempty"`
	RuntimeID      string                 `json:"runtimeId,omitempty"`
	SessionID      string                 `json:"sessionId,omitempty"`
	RunID          string                 `json:"runId,omitempty"`
	TraceID        string                 `json:"traceId,omitempty"`
	SnapshotID     string                 `json:"snapshotId,omitempty"`
	Attempt        int                    `json:"attempt,omitempty"`
	ErrorCode      string                 `json:"errorCode,omitempty"`
	ErrorMessage   string                 `json:"errorMessage"`
	Retryable      bool                   `json:"retryable,omitempty"`
	InputDigest    string                 `json:"inputDigest,omitempty"`
	InputPreview   string                 `json:"inputPreview,omitempty"`
	DurationMillis int64                  `json:"durationMillis,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	Reporter       string                 `json:"reporter,omitempty"`
	CreateTime     int64                  `json:"createTime"`
}

type ToolFailureQuery struct {
	ToolID     string
	AgentID    string
	RuntimeID  string
	SessionID  string
	RunID      string
	TraceID    string
	SnapshotID string
	Limit      int
}
