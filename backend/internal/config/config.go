package config

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Addr   string `yaml:"addr"`
		Author string `yaml:"author"`
	} `yaml:"server"`

	Database struct {
		Provider   string `yaml:"provider"` // postgres / local / memory
		DSN        string `yaml:"dsn"`
		AutoCreate bool   `yaml:"autoCreate"`
		Charset    string `yaml:"charset"`
		Collation  string `yaml:"collation"`
		Local      struct {
			Root string `yaml:"root"`
		} `yaml:"local"`
	} `yaml:"database"`

	Migration struct {
		Enabled bool   `yaml:"enabled"`
		Dir     string `yaml:"dir"`
	} `yaml:"migration"`

	Cache struct {
		Provider string `yaml:"provider"` // redis / memory / noop
		Redis    struct {
			Mode     string   `yaml:"mode"` // single / cluster
			Addrs    []string `yaml:"addrs"`
			Username string   `yaml:"username"`
			Password string   `yaml:"password"`
			DB       int      `yaml:"db"`
			Prefix   string   `yaml:"prefix"`
		} `yaml:"redis"`
	} `yaml:"cache"`

	ObjectStore struct {
		Provider string `yaml:"provider"` // s3 / minio / local / none
		Prefix   string `yaml:"prefix"`
		Local    struct {
			Root string `yaml:"root"`
		} `yaml:"local"`
		S3 struct {
			Endpoint       string `yaml:"endpoint"`
			Region         string `yaml:"region"`
			AccessKey      string `yaml:"accessKey"`
			SecretKey      string `yaml:"secretKey"`
			Bucket         string `yaml:"bucket"`
			UseSSL         bool   `yaml:"useSSL"`
			ForcePathStyle bool   `yaml:"forcePathStyle"`
			Prefix         string `yaml:"prefix"`
		} `yaml:"s3"`
	} `yaml:"objectStore"`

	Runtime struct {
		RouteCacheTTLSeconds   int `yaml:"routeCacheTTLSeconds"`
		VersionCacheTTLSeconds int `yaml:"versionCacheTTLSeconds"`
		GroupCacheTTLSeconds   int `yaml:"groupCacheTTLSeconds"`
	} `yaml:"runtime"`

	Sandbox SandboxConfig `yaml:"sandbox"`

	Review struct {
		Enabled bool `yaml:"enabled"`
	} `yaml:"review"`

	Auth     AuthConfig     `yaml:"auth"`
	Authz    AuthzConfig    `yaml:"authz"`
	AISphere AISphereConfig `yaml:"aisphereAuth"`
	Lock     LockConfig     `yaml:"lock"`
	Web      WebConfig      `yaml:"web"`
	Ops      OpsConfig      `yaml:"ops"`
}

// AISphereConfig wires AIHub into the platform aisphere-auth service.
// The block is optional: when AISphere.Enabled is false, AIHub keeps
// running with its existing local / OIDC / casdoor-remote auth stack
// untouched. When enabled, AIHub registers an extra AuthProvider that
// trusts the AI Sphere session cookie, and an extra Authorizer that
// delegates /authz/check to aisphere-auth.
//
// Both additions are COMPLEMENTARY. They never replace existing providers
// automatically; the operator still chooses which authz.Provider to use
// (the new value "aisphere-auth" is added alongside the existing
// "static / store / casbin / casdoor-remote" options).
type AISphereConfig struct {
	Enabled            bool   `yaml:"enabled"`
	Endpoint           string `yaml:"endpoint"`
	ServiceToken       string `yaml:"serviceToken"`
	ServiceTokenHeader string `yaml:"serviceTokenHeader"`
	CookieName         string `yaml:"cookieName"`
	App                string `yaml:"app"`
	HTTPTimeoutSeconds int    `yaml:"httpTimeoutSeconds"`
	CacheTTLSeconds    int    `yaml:"cacheTTLSeconds"`
	FailClosed         bool   `yaml:"failClosed"`
}

// SandboxConfig configures the AgentKit sandbox execution plane. The first
// production driver is Kubernetes, implemented with direct Kubernetes API
// calls instead of a Kubernetes MCP server because sandbox lifecycle is
// platform infrastructure, not an agent tool.
type SandboxConfig struct {
	Enabled bool   `yaml:"enabled"`
	Driver  string `yaml:"driver"` // kubernetes / noop

	Kubernetes struct {
		Namespace            string `yaml:"namespace"`
		CreateNamespace      bool   `yaml:"createNamespace"`
		APIServer            string `yaml:"apiServer"`
		Kubeconfig           string `yaml:"kubeconfig"`
		Token                string `yaml:"token"`
		TokenFile            string `yaml:"tokenFile"`
		CAFile               string `yaml:"caFile"`
		Insecure             bool   `yaml:"insecure"`
		ServiceAccount       string `yaml:"serviceAccount"`
		RuntimeClassName     string `yaml:"runtimeClassName"`
		NetworkPolicyEnabled bool   `yaml:"networkPolicyEnabled"`
	} `yaml:"kubernetes"`

	DefaultNetworkMode string   `yaml:"defaultNetworkMode"`
	DefaultEgressCIDRs []string `yaml:"defaultEgressCidrs"`

	Image              string `yaml:"image"`
	ImagePullPolicy    string `yaml:"imagePullPolicy"`
	WorkspaceMountPath string `yaml:"workspaceMountPath"`
	StorageClass       string `yaml:"storageClass"`
	WorkspaceSize      string `yaml:"workspaceSize"`
	ToolPort           int    `yaml:"toolPort"`
	BrowserPort        int    `yaml:"browserPort"`
	VNCOrWebPort       int    `yaml:"vncOrWebPort"`
	DefaultCPU         string `yaml:"defaultCpu"`
	DefaultMemory      string `yaml:"defaultMemory"`
	MaxCPU             string `yaml:"maxCpu"`
	MaxMemory          string `yaml:"maxMemory"`
	IdleTTLSeconds     int    `yaml:"idleTtlSeconds"`
}

type OpsConfig struct {
	MetricsEnabled bool `yaml:"metricsEnabled"`
	RateLimit      struct {
		Enabled             bool `yaml:"enabled"`
		WriteLimitPerMinute int  `yaml:"writeLimitPerMinute"`
	} `yaml:"rateLimit"`
	Idempotency struct {
		Enabled    bool `yaml:"enabled"`
		TTLSeconds int  `yaml:"ttlSeconds"`
	} `yaml:"idempotency"`
	Audit struct {
		Enabled bool `yaml:"enabled"`
	} `yaml:"audit"`
}

type WebConfig struct {
	Enabled       bool   `yaml:"enabled"`
	Dir           string `yaml:"dir"`
	RoutePrefix   string `yaml:"routePrefix"`
	IndexFallback bool   `yaml:"indexFallback"`
}

type AuthConfig struct {
	Enabled         bool                 `yaml:"enabled"`
	Mode            string               `yaml:"mode"` // local / external / mixed
	AllowAnonymous  bool                 `yaml:"allowAnonymous"`
	AllowPublicRead bool                 `yaml:"allowPublicRead"`
	Local           LocalAuthConfig      `yaml:"local"`
	APIKeys         []APIKeyAuth         `yaml:"apiKeys"` // legacy bootstrap api keys
	Providers       []AuthProviderConfig `yaml:"providers"`
	RoleMappings    []RoleMappingConfig  `yaml:"roleMappings"`
}

type LocalAuthConfig struct {
	Enabled               bool            `yaml:"enabled"`
	Issuer                string          `yaml:"issuer"`
	SigningSecret         string          `yaml:"signingSecret"`
	AccessTokenTTLSeconds int             `yaml:"accessTokenTTLSeconds"`
	RefreshTTLSeconds     int             `yaml:"refreshTokenTTLSeconds"`
	AccountFile           string          `yaml:"accountFile"`
	AutoCreateBootstrap   bool            `yaml:"autoCreateBootstrap"`
	SetupEnabled          bool            `yaml:"setupEnabled"`
	SetupToken            string          `yaml:"setupToken"`
	Users                 []LocalUserAuth `yaml:"users"`
}

type LocalUserAuth struct {
	Username     string   `yaml:"username"`
	Password     string   `yaml:"password"`
	PasswordHash string   `yaml:"passwordHash"`
	SubjectID    string   `yaml:"subjectId"`
	SubjectType  string   `yaml:"subjectType"`
	DisplayName  string   `yaml:"displayName"`
	Email        string   `yaml:"email"`
	Organization string   `yaml:"organization"`
	Roles        []string `yaml:"roles"`
	Permissions  []string `yaml:"permissions"`
	Namespaces   []string `yaml:"namespaces"`
	Disabled     bool     `yaml:"disabled"`
}

type AuthProviderConfig struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"` // api_key / jwt / oidc / introspection

	// Common JWT/OIDC fields.
	Issuer        string   `yaml:"issuer"`
	Audience      string   `yaml:"audience"`
	JWKSURL       string   `yaml:"jwksUrl"`
	PublicKeyFile string   `yaml:"publicKeyFile"`
	Scopes        []string `yaml:"scopes"`

	// OIDC authorization-code config for Admin UI integrations.
	ClientID     string `yaml:"clientId"`
	ClientSecret string `yaml:"clientSecret"`
	RedirectURL  string `yaml:"redirectUrl"`

	// OAuth2 token introspection config for opaque tokens.
	IntrospectionURL string `yaml:"introspectionUrl"`

	// API key config.
	Header string       `yaml:"header"`
	Keys   []APIKeyAuth `yaml:"keys"`

	ClaimMapping ClaimMappingConfig `yaml:"claimMapping"`
}

type ClaimMappingConfig struct {
	Subject      string `yaml:"subject"`
	SubjectType  string `yaml:"subjectType"`
	Username     string `yaml:"username"`
	Email        string `yaml:"email"`
	Groups       string `yaml:"groups"`
	Roles        string `yaml:"roles"`
	Organization string `yaml:"organization"`
}

type RoleMappingConfig struct {
	Provider      string   `yaml:"provider"`
	ExternalGroup string   `yaml:"externalGroup"`
	ExternalRole  string   `yaml:"externalRole"`
	SubjectType   string   `yaml:"subjectType"`
	InternalRoles []string `yaml:"internalRoles"`
	Permissions   []string `yaml:"permissions"`
	Namespaces    []string `yaml:"namespaces"`
}

type APIKeyAuth struct {
	Name         string   `yaml:"name"`
	SubjectID    string   `yaml:"subjectId"`
	SubjectType  string   `yaml:"subjectType"` // human / organization / agent / service
	Organization string   `yaml:"organization"`
	Token        string   `yaml:"token"`
	Roles        []string `yaml:"roles"`
	Permissions  []string `yaml:"permissions"`
	Namespaces   []string `yaml:"namespaces"` // public, dev, prod or *
}

// AuthzConfig controls AIHub resource authorization.
// Recommended production mode is provider=casdoor-remote: Casdoor is the internal IAM/permission center;
// AIHub only calls Casdoor Exposed Casbin APIs to decide allow/deny.
// provider=casbin is kept as a local/demo fallback.
// provider=aisphere-auth delegates the decision to the platform aisphere-auth
// service. It is an ADDITIONAL option — selecting it never disables the
// existing casdoor-remote / casbin code paths; operators can switch back at
// any time.
type AuthzConfig struct {
	Provider    string             `yaml:"provider"` // static / store / casbin / casdoor-remote / aisphere-auth
	Model       string             `yaml:"model"`
	PolicyStore string             `yaml:"policyStore"` // file / postgres, only used by provider=casbin
	PolicyFile  string             `yaml:"policyFile"`
	AutoSave    bool               `yaml:"autoSave"`
	Casdoor     CasdoorAuthzConfig `yaml:"casdoor"`
}

// CasdoorAuthzConfig is used when authz.provider=casdoor-remote.
// It maps AIHub authorization requests to Casdoor's exposed Casbin APIs.
type CasdoorAuthzConfig struct {
	Endpoint        string `yaml:"endpoint"`
	Owner           string `yaml:"owner"`
	Application     string `yaml:"application"`
	Permission      string `yaml:"permission"`
	PermissionID    string `yaml:"permissionId"`
	Model           string `yaml:"model"`
	ModelID         string `yaml:"modelId"`
	ResourceID      string `yaml:"resourceId"`
	EnforcerID      string `yaml:"enforcerId"`
	ClientID        string `yaml:"clientId"`
	ClientSecret    string `yaml:"clientSecret"`
	SubjectFormat   string `yaml:"subjectFormat"` // casdoor / principal / external
	CacheTTLSeconds int    `yaml:"cacheTTLSeconds"`
	FailClosed      bool   `yaml:"failClosed"`
}

type LockConfig struct {
	Enabled        bool   `yaml:"enabled"`
	Provider       string `yaml:"provider"` // redis / memory / noop
	TTLSeconds     int    `yaml:"ttlSeconds"`
	WaitSeconds    int    `yaml:"waitSeconds"`
	UseRedisConfig bool   `yaml:"useRedisConfig"`
}

func Default() Config {
	var c Config
	c.Server.Addr = ":8848"
	c.Server.Author = "-"
	c.Database.Provider = "local"
	c.Database.AutoCreate = true
	c.Database.Charset = "utf8mb4"
	c.Database.Collation = "utf8mb4_unicode_ci"
	c.Database.Local.Root = "./data/aihub"
	c.Migration.Enabled = true
	c.Migration.Dir = "./migrations"
	c.Cache.Provider = "memory"
	c.Cache.Redis.Mode = "single"
	c.Cache.Redis.Addrs = []string{"127.0.0.1:6379"}
	c.Cache.Redis.Prefix = "aihub"
	c.ObjectStore.Provider = "local"
	c.ObjectStore.Local.Root = "./data/objects"
	c.ObjectStore.S3.Endpoint = "127.0.0.1:9000"
	c.ObjectStore.S3.Region = "us-east-1"
	c.ObjectStore.S3.AccessKey = "minioadmin"
	c.ObjectStore.S3.SecretKey = "minioadmin"
	c.ObjectStore.S3.Bucket = "aihub"
	c.Runtime.RouteCacheTTLSeconds = 300
	c.Runtime.VersionCacheTTLSeconds = 600
	c.Runtime.GroupCacheTTLSeconds = 60
	c.Sandbox.Enabled = false
	c.Sandbox.Driver = "kubernetes"
	c.Sandbox.Kubernetes.Namespace = "aisphere-sandbox"
	c.Sandbox.Kubernetes.CreateNamespace = true
	c.Sandbox.Kubernetes.ServiceAccount = "default"
	c.Sandbox.Kubernetes.NetworkPolicyEnabled = true
	c.Sandbox.DefaultNetworkMode = "offline"
	c.Sandbox.Image = "registry.local/aisphere/agentkit-sandbox:latest"
	c.Sandbox.ImagePullPolicy = "IfNotPresent"
	c.Sandbox.WorkspaceMountPath = "/workspace"
	c.Sandbox.WorkspaceSize = "10Gi"
	c.Sandbox.ToolPort = 18081
	c.Sandbox.BrowserPort = 9222
	c.Sandbox.VNCOrWebPort = 6080
	c.Sandbox.DefaultCPU = "500m"
	c.Sandbox.DefaultMemory = "1Gi"
	c.Sandbox.MaxCPU = "2"
	c.Sandbox.MaxMemory = "4Gi"
	c.Sandbox.IdleTTLSeconds = 3600
	c.Lock.Enabled = true
	c.Lock.Provider = "memory"
	c.Lock.TTLSeconds = 30
	c.Lock.WaitSeconds = 2
	c.Lock.UseRedisConfig = true
	c.Auth.AllowPublicRead = true
	c.Auth.Mode = "mixed"
	c.Auth.Local.Enabled = true
	c.Auth.Local.Issuer = "aihub-local"
	c.Auth.Local.SigningSecret = "CHANGE_ME_LOCAL_JWT_SECRET"
	c.Auth.Local.AccessTokenTTLSeconds = 3600
	c.Auth.Local.RefreshTTLSeconds = 604800
	c.Auth.Local.AccountFile = "./data/iam/local_accounts.json"
	c.Auth.Local.AutoCreateBootstrap = false
	c.Auth.Local.SetupEnabled = true
	c.Auth.Providers = []AuthProviderConfig{}
	c.Authz.Provider = "static"
	c.Authz.Model = "./configs/casbin/model.conf"
	c.Authz.PolicyStore = "file"
	c.Authz.PolicyFile = "./configs/casbin/policy.csv"
	c.Authz.AutoSave = true
	// aisphere-auth integration is OFF by default. Operators opt in by
	// flipping aisphereAuth.enabled=true and pointing endpoint at the
	// platform aisphere-auth service.
	c.AISphere.Enabled = false
	c.AISphere.Endpoint = "http://aisphere-auth:18080"
	c.AISphere.CookieName = "aisphere_session"
	c.AISphere.App = "aihub"
	c.AISphere.HTTPTimeoutSeconds = 10
	c.AISphere.CacheTTLSeconds = 5
	c.AISphere.FailClosed = true
	c.Web.Enabled = true
	c.Web.Dir = "../front/dist"
	c.Web.RoutePrefix = "/ui"
	c.Web.IndexFallback = true
	c.Ops.MetricsEnabled = true
	c.Ops.RateLimit.Enabled = true
	c.Ops.RateLimit.WriteLimitPerMinute = 120
	c.Ops.Idempotency.Enabled = true
	c.Ops.Idempotency.TTLSeconds = 3600
	c.Ops.Audit.Enabled = true
	return c
}

func LoadFromFlags() (Config, string, error) {
	defaultPath := os.Getenv("SKILLHUB_CONFIG")
	if defaultPath == "" {
		defaultPath = "config.yaml"
	}
	path := flag.String("config", defaultPath, "aihub config file path")
	flag.Parse()
	c, err := Load(*path)
	return c, *path, err
}

func Load(path string) (Config, error) {
	c := Default()
	if path != "" {
		b, err := os.ReadFile(path)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) || path != "config.yaml" {
				return c, fmt.Errorf("read config %s: %w", path, err)
			}
		} else {
			b = []byte(os.ExpandEnv(string(b)))
			if err := yaml.Unmarshal(b, &c); err != nil {
				return c, fmt.Errorf("parse config %s: %w", path, err)
			}
		}
	}
	applyEnvOverrides(&c)
	normalize(&c)
	return c, nil
}

func normalize(c *Config) {
	c.Server.Addr = firstNonEmpty(c.Server.Addr, ":8848")
	c.Server.Author = firstNonEmpty(c.Server.Author, "-")
	c.Database.Provider = strings.ToLower(firstNonEmpty(c.Database.Provider, "local"))
	c.Database.Charset = firstNonEmpty(c.Database.Charset, "utf8mb4")
	c.Database.Collation = firstNonEmpty(c.Database.Collation, "utf8mb4_unicode_ci")
	c.Database.Local.Root = firstNonEmpty(c.Database.Local.Root, "./data/aihub")
	c.Migration.Dir = firstNonEmpty(c.Migration.Dir, "./migrations")
	c.Cache.Provider = strings.ToLower(firstNonEmpty(c.Cache.Provider, "memory"))
	c.Cache.Redis.Mode = strings.ToLower(firstNonEmpty(c.Cache.Redis.Mode, "single"))
	if len(c.Cache.Redis.Addrs) == 0 {
		c.Cache.Redis.Addrs = []string{"127.0.0.1:6379"}
	}
	c.Cache.Redis.Prefix = strings.Trim(c.Cache.Redis.Prefix, ":")
	c.ObjectStore.Provider = strings.ToLower(firstNonEmpty(c.ObjectStore.Provider, "local"))
	c.ObjectStore.Local.Root = firstNonEmpty(c.ObjectStore.Local.Root, "./data/objects")
	c.ObjectStore.S3.Endpoint = firstNonEmpty(c.ObjectStore.S3.Endpoint, "127.0.0.1:9000")
	c.ObjectStore.S3.Region = firstNonEmpty(c.ObjectStore.S3.Region, "us-east-1")
	c.ObjectStore.S3.Bucket = firstNonEmpty(c.ObjectStore.S3.Bucket, "aihub")
	if c.ObjectStore.Prefix == "" && c.ObjectStore.S3.Prefix != "" {
		c.ObjectStore.Prefix = c.ObjectStore.S3.Prefix
	}
	c.ObjectStore.Prefix = strings.Trim(c.ObjectStore.Prefix, "/")
	if c.Runtime.RouteCacheTTLSeconds <= 0 {
		c.Runtime.RouteCacheTTLSeconds = 300
	}
	if c.Runtime.VersionCacheTTLSeconds <= 0 {
		c.Runtime.VersionCacheTTLSeconds = 600
	}
	if c.Runtime.GroupCacheTTLSeconds <= 0 {
		c.Runtime.GroupCacheTTLSeconds = 60
	}
	c.Sandbox.Driver = strings.ToLower(firstNonEmpty(c.Sandbox.Driver, "kubernetes"))
	c.Sandbox.Kubernetes.Namespace = firstNonEmpty(c.Sandbox.Kubernetes.Namespace, "aisphere-sandbox")
	c.Sandbox.Kubernetes.ServiceAccount = firstNonEmpty(c.Sandbox.Kubernetes.ServiceAccount, "default")
	c.Sandbox.DefaultNetworkMode = strings.ToLower(firstNonEmpty(c.Sandbox.DefaultNetworkMode, "offline"))
	switch c.Sandbox.DefaultNetworkMode {
	case "online", "restricted", "offline":
	default:
		c.Sandbox.DefaultNetworkMode = "offline"
	}
	c.Sandbox.Image = firstNonEmpty(c.Sandbox.Image, "registry.local/aisphere/agentkit-sandbox:latest")
	c.Sandbox.ImagePullPolicy = firstNonEmpty(c.Sandbox.ImagePullPolicy, "IfNotPresent")
	c.Sandbox.WorkspaceMountPath = firstNonEmpty(c.Sandbox.WorkspaceMountPath, "/workspace")
	c.Sandbox.WorkspaceSize = firstNonEmpty(c.Sandbox.WorkspaceSize, "10Gi")
	if c.Sandbox.ToolPort <= 0 {
		c.Sandbox.ToolPort = 18081
	}
	if c.Sandbox.BrowserPort <= 0 {
		c.Sandbox.BrowserPort = 9222
	}
	if c.Sandbox.VNCOrWebPort <= 0 {
		c.Sandbox.VNCOrWebPort = 6080
	}
	c.Sandbox.DefaultCPU = firstNonEmpty(c.Sandbox.DefaultCPU, "500m")
	c.Sandbox.DefaultMemory = firstNonEmpty(c.Sandbox.DefaultMemory, "1Gi")
	c.Sandbox.MaxCPU = firstNonEmpty(c.Sandbox.MaxCPU, "2")
	c.Sandbox.MaxMemory = firstNonEmpty(c.Sandbox.MaxMemory, "4Gi")
	if c.Sandbox.IdleTTLSeconds <= 0 {
		c.Sandbox.IdleTTLSeconds = 3600
	}
	c.Auth.Mode = strings.ToLower(firstNonEmpty(c.Auth.Mode, "mixed"))
	c.Auth.Local.Issuer = firstNonEmpty(c.Auth.Local.Issuer, "aihub-local")
	c.Auth.Local.SigningSecret = firstNonEmpty(c.Auth.Local.SigningSecret, "CHANGE_ME_LOCAL_JWT_SECRET")
	if c.Auth.Local.AccessTokenTTLSeconds <= 0 {
		c.Auth.Local.AccessTokenTTLSeconds = 3600
	}
	if c.Auth.Local.RefreshTTLSeconds <= 0 {
		c.Auth.Local.RefreshTTLSeconds = 604800
	}
	c.Auth.Local.AccountFile = firstNonEmpty(c.Auth.Local.AccountFile, "./data/iam/local_accounts.json")
	c.Lock.Provider = strings.ToLower(firstNonEmpty(c.Lock.Provider, "memory"))
	c.Web.Dir = firstNonEmpty(c.Web.Dir, "../front/dist")
	c.Web.RoutePrefix = firstNonEmpty(c.Web.RoutePrefix, "/ui")
	if !strings.HasPrefix(c.Web.RoutePrefix, "/") {
		c.Web.RoutePrefix = "/" + c.Web.RoutePrefix
	}
	c.Web.RoutePrefix = strings.TrimRight(c.Web.RoutePrefix, "/")
	if c.Lock.TTLSeconds <= 0 {
		c.Lock.TTLSeconds = 30
	}
	if c.Lock.WaitSeconds <= 0 {
		c.Lock.WaitSeconds = 2
	}

	for i := range c.Auth.Providers {
		c.Auth.Providers[i].Type = strings.ToLower(firstNonEmpty(c.Auth.Providers[i].Type, "api_key"))
		c.Auth.Providers[i].Name = firstNonEmpty(c.Auth.Providers[i].Name, c.Auth.Providers[i].Type)
		if c.Auth.Providers[i].Header == "" && (c.Auth.Providers[i].Type == "api_key" || c.Auth.Providers[i].Type == "apikey") {
			c.Auth.Providers[i].Header = "X-API-Key"
		}
	}
}

func applyEnvOverrides(c *Config) {
	setStr(&c.Server.Addr, "SKILLHUB_ADDR")
	setStr(&c.Server.Author, "SKILLHUB_AUTHOR")
	setStr(&c.Database.Provider, "SKILLHUB_STORE")
	setStr(&c.Database.DSN, "SKILLHUB_MYSQL_DSN")
	setBool(&c.Database.AutoCreate, "SKILLHUB_DB_AUTO_CREATE")
	setStr(&c.Database.Charset, "SKILLHUB_DB_CHARSET")
	setStr(&c.Database.Collation, "SKILLHUB_DB_COLLATION")
	setStr(&c.Database.Local.Root, "SKILLHUB_DATA_DIR")
	setBool(&c.Migration.Enabled, "SKILLHUB_MIGRATION_ENABLED")
	setStr(&c.Migration.Dir, "SKILLHUB_MIGRATION_DIR")
	setStr(&c.Cache.Provider, "SKILLHUB_CACHE")
	setStr(&c.Cache.Redis.Mode, "SKILLHUB_REDIS_MODE")
	if v := os.Getenv("SKILLHUB_REDIS_ADDRS"); v != "" {
		c.Cache.Redis.Addrs = splitCSV(v)
	}
	setStr(&c.Cache.Redis.Username, "SKILLHUB_REDIS_USERNAME")
	setStr(&c.Cache.Redis.Password, "SKILLHUB_REDIS_PASSWORD")
	setInt(&c.Cache.Redis.DB, "SKILLHUB_REDIS_DB")
	setStr(&c.Cache.Redis.Prefix, "SKILLHUB_REDIS_PREFIX")
	setStr(&c.ObjectStore.Provider, "SKILLHUB_OBJECT_STORE")
	setStr(&c.ObjectStore.Prefix, "SKILLHUB_OBJECT_PREFIX")
	setStr(&c.ObjectStore.Local.Root, "SKILLHUB_OBJECT_LOCAL_ROOT")
	setStr(&c.ObjectStore.S3.Endpoint, "SKILLHUB_S3_ENDPOINT")
	setStr(&c.ObjectStore.S3.Region, "SKILLHUB_S3_REGION")
	setStr(&c.ObjectStore.S3.AccessKey, "SKILLHUB_S3_ACCESS_KEY")
	setStr(&c.ObjectStore.S3.SecretKey, "SKILLHUB_S3_SECRET_KEY")
	setStr(&c.ObjectStore.S3.Bucket, "SKILLHUB_S3_BUCKET")
	setBool(&c.ObjectStore.S3.UseSSL, "SKILLHUB_S3_USE_SSL")
	setBool(&c.Auth.Enabled, "SKILLHUB_AUTH_ENABLED")
	setStr(&c.Auth.Mode, "SKILLHUB_AUTH_MODE")
	setBool(&c.Auth.Local.Enabled, "SKILLHUB_LOCAL_AUTH_ENABLED")
	setStr(&c.Auth.Local.SigningSecret, "SKILLHUB_LOCAL_AUTH_SIGNING_SECRET")
	setStr(&c.Auth.Local.AccountFile, "SKILLHUB_LOCAL_AUTH_ACCOUNT_FILE")
	setBool(&c.Auth.Local.SetupEnabled, "SKILLHUB_LOCAL_AUTH_SETUP_ENABLED")
	setStr(&c.Auth.Local.SetupToken, "SKILLHUB_LOCAL_AUTH_SETUP_TOKEN")
	setBool(&c.Auth.AllowAnonymous, "SKILLHUB_AUTH_ALLOW_ANONYMOUS")
	setBool(&c.Auth.AllowPublicRead, "SKILLHUB_AUTH_ALLOW_PUBLIC_READ")
	setBool(&c.Lock.Enabled, "SKILLHUB_LOCK_ENABLED")
	setStr(&c.Lock.Provider, "SKILLHUB_LOCK_PROVIDER")
	setInt(&c.Lock.TTLSeconds, "SKILLHUB_LOCK_TTL_SECONDS")
	setInt(&c.Lock.WaitSeconds, "SKILLHUB_LOCK_WAIT_SECONDS")
	setBool(&c.Web.Enabled, "SKILLHUB_WEB_ENABLED")
	setStr(&c.Web.Dir, "SKILLHUB_WEB_DIR")
	setStr(&c.Web.RoutePrefix, "SKILLHUB_WEB_ROUTE_PREFIX")

	setBool(&c.Sandbox.Enabled, "AIHUB_SANDBOX_ENABLED")
	setStr(&c.Sandbox.Driver, "AIHUB_SANDBOX_DRIVER")
	setStr(&c.Sandbox.Kubernetes.Namespace, "AIHUB_SANDBOX_K8S_NAMESPACE")
	setBool(&c.Sandbox.Kubernetes.CreateNamespace, "AIHUB_SANDBOX_K8S_CREATE_NAMESPACE")
	setStr(&c.Sandbox.Kubernetes.APIServer, "AIHUB_SANDBOX_K8S_API_SERVER")
	setStr(&c.Sandbox.Kubernetes.Kubeconfig, "AIHUB_SANDBOX_KUBECONFIG")
	setStr(&c.Sandbox.Kubernetes.Token, "AIHUB_SANDBOX_K8S_TOKEN")
	setStr(&c.Sandbox.Kubernetes.TokenFile, "AIHUB_SANDBOX_K8S_TOKEN_FILE")
	setStr(&c.Sandbox.Kubernetes.CAFile, "AIHUB_SANDBOX_K8S_CA_FILE")
	setBool(&c.Sandbox.Kubernetes.Insecure, "AIHUB_SANDBOX_K8S_INSECURE")
	setStr(&c.Sandbox.Kubernetes.ServiceAccount, "AIHUB_SANDBOX_K8S_SERVICE_ACCOUNT")
	setStr(&c.Sandbox.Kubernetes.RuntimeClassName, "AIHUB_SANDBOX_K8S_RUNTIME_CLASS")
	setBool(&c.Sandbox.Kubernetes.NetworkPolicyEnabled, "AIHUB_SANDBOX_K8S_NETWORK_POLICY_ENABLED")
	setStr(&c.Sandbox.DefaultNetworkMode, "AIHUB_SANDBOX_DEFAULT_NETWORK_MODE")
	setCSV(&c.Sandbox.DefaultEgressCIDRs, "AIHUB_SANDBOX_DEFAULT_EGRESS_CIDRS")
	setStr(&c.Sandbox.Image, "AIHUB_SANDBOX_IMAGE")
	setStr(&c.Sandbox.ImagePullPolicy, "AIHUB_SANDBOX_IMAGE_PULL_POLICY")
	setStr(&c.Sandbox.WorkspaceMountPath, "AIHUB_SANDBOX_WORKSPACE_MOUNT_PATH")
	setStr(&c.Sandbox.StorageClass, "AIHUB_SANDBOX_STORAGE_CLASS")
	setStr(&c.Sandbox.WorkspaceSize, "AIHUB_SANDBOX_WORKSPACE_SIZE")
	setInt(&c.Sandbox.ToolPort, "AIHUB_SANDBOX_TOOL_PORT")
	setInt(&c.Sandbox.BrowserPort, "AIHUB_SANDBOX_BROWSER_PORT")
	setInt(&c.Sandbox.VNCOrWebPort, "AIHUB_SANDBOX_WEB_PORT")
	setStr(&c.Sandbox.DefaultCPU, "AIHUB_SANDBOX_DEFAULT_CPU")
	setStr(&c.Sandbox.DefaultMemory, "AIHUB_SANDBOX_DEFAULT_MEMORY")
	setStr(&c.Sandbox.MaxCPU, "AIHUB_SANDBOX_MAX_CPU")
	setStr(&c.Sandbox.MaxMemory, "AIHUB_SANDBOX_MAX_MEMORY")
	setInt(&c.Sandbox.IdleTTLSeconds, "AIHUB_SANDBOX_IDLE_TTL_SECONDS")

	// aisphere-auth integration env overrides. Naming mirrors the
	// aisphere-auth service itself (AISPHERE_*) so operators can source
	// the same secret file into both processes.
	setBool(&c.AISphere.Enabled, "AISPHERE_AUTH_ENABLED")
	setStr(&c.AISphere.Endpoint, "AISPHERE_AUTH_ENDPOINT")
	setStr(&c.AISphere.ServiceToken, "AISPHERE_SERVICE_TOKEN")
	setStr(&c.AISphere.ServiceTokenHeader, "AISPHERE_SERVICE_TOKEN_HEADER")
	setStr(&c.AISphere.CookieName, "AISPHERE_SESSION_COOKIE_NAME")
	setStr(&c.AISphere.App, "AISPHERE_AUTH_APP")
	setInt(&c.AISphere.HTTPTimeoutSeconds, "AISPHERE_AUTH_HTTP_TIMEOUT_SECONDS")
	setInt(&c.AISphere.CacheTTLSeconds, "AISPHERE_AUTH_CACHE_TTL_SECONDS")
	setBool(&c.AISphere.FailClosed, "AISPHERE_AUTH_FAIL_CLOSED")
}

func setStr(dst *string, key string) {
	if v := os.Getenv(key); v != "" {
		*dst = v
	}
}
func setInt(dst *int, key string) {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			*dst = n
		}
	}
}
func setBool(dst *bool, key string) {
	if v := os.Getenv(key); v != "" {
		*dst = strings.EqualFold(v, "true") || v == "1" || strings.EqualFold(v, "yes")
	}
}
func setCSV(dst *[]string, key string) {
	if v := os.Getenv(key); v != "" {
		*dst = splitCSV(v)
	}
}
func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := []string{}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
func firstNonEmpty(v, d string) string {
	if strings.TrimSpace(v) != "" {
		return v
	}
	return d
}
