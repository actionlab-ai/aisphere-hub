package sandbox

// Config is the runtime-only sandbox manager configuration. It mirrors the
// public AIHub config shape but deliberately avoids importing the config package
// so the sandbox executor can be reused by AgentKit runtime without a Hub
// dependency.
type Config struct {
	Enabled bool
	Driver  string

	Namespace            string
	CreateNamespace      bool
	APIServer            string
	Kubeconfig           string
	Token                string
	TokenFile            string
	CAFile               string
	Insecure             bool
	ServiceAccount       string
	RuntimeClassName     string
	NetworkPolicyEnabled bool
	DefaultNetworkMode   string
	DefaultEgressCIDRs   []string

	Image              string
	ImagePullPolicy    string
	WorkspaceMountPath string
	StorageClass       string
	WorkspaceSize      string
	ToolPort           int
	BrowserPort        int
	VNCOrWebPort       int
	DefaultCPU         string
	DefaultMemory      string
	MaxCPU             string
	MaxMemory          string
	IdleTTLSeconds     int
}
