package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/actionlab-ai/aisphere-hub/backend/internal/aisphereclient"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/api"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/auth"
	localprovider "github.com/actionlab-ai/aisphere-hub/backend/internal/auth/providers/local"
	authhttp "github.com/actionlab-ai/aisphere-hub/backend/internal/authhttp"
	aisphereauthz "github.com/actionlab-ai/aisphere-hub/backend/internal/authz/aisphereauth"
	casbinauthz "github.com/actionlab-ai/aisphere-hub/backend/internal/authz/casbin"
	casdoorremote "github.com/actionlab-ai/aisphere-hub/backend/internal/authz/casdoorremote"
	basecache "github.com/actionlab-ai/aisphere-hub/backend/internal/cache"
	rediscache "github.com/actionlab-ai/aisphere-hub/backend/internal/cache/redis"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/config"
	memorylock "github.com/actionlab-ai/aisphere-hub/backend/internal/lock/memory"
	redislock "github.com/actionlab-ai/aisphere-hub/backend/internal/lock/redis"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/objectstore"
	s3store "github.com/actionlab-ai/aisphere-hub/backend/internal/objectstore/s3"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/ops"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/ports"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/sandbox"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/service"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/store"
	postgresstore "github.com/actionlab-ai/aisphere-hub/backend/internal/store/postgres"
	"github.com/gin-gonic/gin"
)

var newPostgresStore = func(dsn string, autoCreate bool) (store.Backend, error) {
	return postgresstore.NewWithAutoCreate(dsn, autoCreate)
}

func main() {
	cfg, cfgPath, err := config.LoadFromFlags()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	backend, err := initStore(cfg)
	if err != nil {
		log.Fatalf("init store: %v", err)
	}
	rawCache, err := initCache(cfg)
	if err != nil {
		log.Fatalf("init cache: %v", err)
	}
	obj, err := initObjectStore(cfg)
	if err != nil {
		log.Fatalf("init object store: %v", err)
	}
	localMgr, err := initLocalAuth(cfg, backend)
	if err != nil {
		log.Fatalf("init local auth: %v", err)
	}

	svc := service.New(backend, cfg.Server.Author).WithRuntimeCache(basecache.NewRuntime(rawCache)).WithCacheTTL(service.CacheTTL{
		Route:   seconds(cfg.Runtime.RouteCacheTTLSeconds),
		Version: seconds(cfg.Runtime.VersionCacheTTLSeconds),
		Group:   seconds(cfg.Runtime.GroupCacheTTLSeconds),
	})
	lockMgr, err := initLock(cfg)
	if err != nil {
		log.Fatalf("init lock: %v", err)
	}
	if lockMgr != nil {
		svc = svc.WithLockManager(lockMgr, seconds(cfg.Lock.TTLSeconds), seconds(cfg.Lock.WaitSeconds))
	}
	if obj != nil {
		svc = svc.WithObjectStore(obj, cfg.ObjectStore.Prefix)
	}
	sandboxMgr, err := initSandbox(cfg)
	if err != nil {
		log.Fatalf("init sandbox: %v", err)
	}
	if sandboxMgr != nil {
		svc = svc.WithSandboxManager(sandboxMgr)
	}
	middlewares := []gin.HandlerFunc{}
	metrics := ops.NewMetrics()
	if cfg.Ops.MetricsEnabled {
		middlewares = append(middlewares, metrics.Middleware())
	}
	if cfg.Ops.RateLimit.Enabled {
		if cfg.Cache.Provider == "redis" {
			middlewares = append(middlewares, ops.NewDistributedRateLimiter(rawCache, cfg.Ops.RateLimit.WriteLimitPerMinute, time.Minute).Middleware())
		} else {
			middlewares = append(middlewares, ops.NewRateLimiter(cfg.Ops.RateLimit.WriteLimitPerMinute, time.Minute).Middleware())
		}
	}
	if cfg.Ops.Idempotency.Enabled {
		middlewares = append(middlewares, ops.IdempotencyMiddleware(svc, seconds(cfg.Ops.Idempotency.TTLSeconds)))
	}
	aisphereClient := aisphereclient.New(aisphereclient.Config{
		Enabled:            cfg.AISphere.Enabled,
		Endpoint:           cfg.AISphere.Endpoint,
		ServiceToken:       cfg.AISphere.ServiceToken,
		ServiceTokenHeader: cfg.AISphere.ServiceTokenHeader,
		CookieName:         cfg.AISphere.CookieName,
		App:                cfg.AISphere.App,
		HTTPTimeoutSeconds: cfg.AISphere.HTTPTimeoutSeconds,
		CacheTTLSeconds:    cfg.AISphere.CacheTTLSeconds,
		FailClosed:         cfg.AISphere.FailClosed,
	})
	if aisphereClient != nil {
		log.Printf("aisphere-auth integration enabled: endpoint=%s app=%s cookie=%s", aisphereClient.Config().Endpoint, aisphereClient.Config().App, aisphereClient.Config().CookieName)
	}
	authz, err := initAuthorizer(cfg)
	if err != nil {
		log.Fatalf("init authorizer: %v", err)
	}
	// When the operator selected authz.provider=aisphere-auth, swap in
	// the aisphere-auth-backed authorizer. We keep the original
	// initAuthorizer() return around so existing setups that select
	// casdoor-remote / casbin / static keep working unchanged.
	if strings.EqualFold(strings.TrimSpace(cfg.Authz.Provider), "aisphere-auth") {
		if aisp := initAISphereAuthorizer(cfg, aisphereClient); aisp != nil {
			authz = aisp
		} else {
			log.Printf("warn: authz.provider=aisphere-auth but aisphereAuth.enabled=false; falling back to %s", cfg.Authz.Provider)
		}
	}
	middlewares = append(middlewares, authhttp.MiddlewareWithAISphere(cfg.Auth, localMgr, backend, authz, aisphereClient))
	if cfg.Ops.Audit.Enabled {
		middlewares = append(middlewares, ops.AuditMiddlewareWithAISphere(svc, aisphereClient))
	}
	r := api.NewRouterWithAISphere(svc, aisphereClient, middlewares...)
	if caz, ok := authz.(*casbinauthz.Authorizer); ok {
		casbinauthz.RegisterAccessRoutes(r, caz)
	}
	if caz, ok := authz.(*casdoorremote.Authorizer); ok {
		casdoorremote.RegisterAccessRoutes(r, caz)
	}
	if caz, ok := authz.(*aisphereauthz.Authorizer); ok {
		aisphereauthz.RegisterAccessRoutes(r, caz)
	}
	authhttp.RegisterRoutes(r, cfg.Auth, localMgr)
	// Register aisphere-auth redirect routes (login / callback) only
	// when the integration is enabled. They give the AIHub frontend
	// a stable in-domain URL to point "Sign in with AI Sphere" at.
	if aisphereClient != nil {
		authhttp.RegisterAISphereRoutes(r, aisphereClient)
	}
	if cfg.Ops.MetricsEnabled {
		metrics.Register(r)
	}
	if cfg.Web.Enabled {
		api.RegisterWeb(r, cfg.Web.RoutePrefix, cfg.Web.Dir, cfg.Web.IndexFallback)
	}
	log.Printf("gin aihub listening on %s config=%s database=%s cache=%s redisMode=%s objectStore=%s migration=%v aisphereAuth=%v", cfg.Server.Addr, cfgPath, cfg.Database.Provider, cfg.Cache.Provider, cfg.Cache.Redis.Mode, cfg.ObjectStore.Provider, cfg.Migration.Enabled, cfg.AISphere.Enabled)
	if err := r.Run(cfg.Server.Addr); err != nil {
		log.Fatal(err)
	}
}

func initAuthorizer(cfg config.Config) (auth.Authorizer, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.Authz.Provider)) {
	case "", "static":
		return nil, nil
	case "casbin", "casbin-local", "local-casbin":
		return casbinauthz.NewAuthorizer(cfg)
	case "casdoor-remote", "casdoor_remote", "casdoor":
		return casdoorremote.NewAuthorizer(cfg)
	case "aisphere-auth", "aisphere_auth", "aisphereauth":
		// Built later in main() once the aisphere-auth client is
		// constructed. Returning nil here lets the static
		// fallback kick in temporarily.
		return nil, nil
	case "store":
		return nil, nil
	default:
		return nil, nil
	}
}

// initAISphereAuthorizer builds the aisphere-auth-backed authorizer. It
// is only invoked when cfg.Authz.Provider == "aisphere-auth". The client
// is shared with the auth provider so introspect and enforce reuse the
// same HTTP connection pool and cache.
func initAISphereAuthorizer(cfg config.Config, client *aisphereclient.Client) auth.Authorizer {
	if client == nil {
		return nil
	}
	return aisphereauthz.New(client)
}

func initLocalAuth(cfg config.Config, backend store.Backend) (*localprovider.Manager, error) {
	if !cfg.Auth.Enabled || !cfg.Auth.Local.Enabled {
		return nil, nil
	}
	mode := strings.ToLower(cfg.Auth.Mode)
	if mode == "external" {
		return nil, nil
	}
	if strings.EqualFold(cfg.Database.Provider, "postgres") || strings.EqualFold(cfg.Database.Provider, "pg") || strings.EqualFold(cfg.Database.Provider, "mysql") {
		if accountStore, ok := backend.(localprovider.AccountStore); ok {
			return localprovider.NewManagerWithStore(cfg.Auth.Local, accountStore)
		}
	}
	return localprovider.NewManager(cfg.Auth.Local)
}

func initStore(cfg config.Config) (store.Backend, error) {
	switch strings.ToLower(cfg.Database.Provider) {
	case "postgres", "pg":
		return newPostgresStore(cfg.Database.DSN, cfg.Database.AutoCreate)
	case "mysql":
		return nil, fmt.Errorf("mysql store is deprecated and disabled in the unified PostgreSQL stack; set database.provider=postgres")
	case "local", "json", "file":
		return store.New(cfg.Database.Local.Root)
	default:
		return store.New(cfg.Database.Local.Root)
	}
}

func initCache(cfg config.Config) (ports.Cache, error) {
	switch strings.ToLower(cfg.Cache.Provider) {
	case "redis":
		return rediscache.New(rediscache.Config{
			Mode:     cfg.Cache.Redis.Mode,
			Addrs:    cfg.Cache.Redis.Addrs,
			Username: cfg.Cache.Redis.Username,
			Password: cfg.Cache.Redis.Password,
			DB:       cfg.Cache.Redis.DB,
			Prefix:   cfg.Cache.Redis.Prefix,
		})
	case "noop":
		return basecache.NoopCache{}, nil
	default:
		return basecache.NewMemoryCache(), nil
	}
}

func initLock(cfg config.Config) (ports.LockManager, error) {
	if !cfg.Lock.Enabled || strings.EqualFold(cfg.Lock.Provider, "noop") {
		return nil, nil
	}
	provider := strings.ToLower(cfg.Lock.Provider)
	if provider == "redis" || (provider == "" && cfg.Cache.Provider == "redis") {
		return redislock.New(redislock.Config{
			Mode:     cfg.Cache.Redis.Mode,
			Addrs:    cfg.Cache.Redis.Addrs,
			Username: cfg.Cache.Redis.Username,
			Password: cfg.Cache.Redis.Password,
			DB:       cfg.Cache.Redis.DB,
			Prefix:   cfg.Cache.Redis.Prefix,
		})
	}
	return memorylock.New(), nil
}

func initSandbox(cfg config.Config) (sandbox.Manager, error) {
	if !cfg.Sandbox.Enabled || strings.EqualFold(strings.TrimSpace(cfg.Sandbox.Driver), "noop") {
		return nil, nil
	}
	sc := sandbox.Config{
		Enabled:              cfg.Sandbox.Enabled,
		Driver:               cfg.Sandbox.Driver,
		Namespace:            cfg.Sandbox.Kubernetes.Namespace,
		CreateNamespace:      cfg.Sandbox.Kubernetes.CreateNamespace,
		APIServer:            cfg.Sandbox.Kubernetes.APIServer,
		Kubeconfig:           cfg.Sandbox.Kubernetes.Kubeconfig,
		Token:                cfg.Sandbox.Kubernetes.Token,
		TokenFile:            cfg.Sandbox.Kubernetes.TokenFile,
		CAFile:               cfg.Sandbox.Kubernetes.CAFile,
		Insecure:             cfg.Sandbox.Kubernetes.Insecure,
		ServiceAccount:       cfg.Sandbox.Kubernetes.ServiceAccount,
		RuntimeClassName:     cfg.Sandbox.Kubernetes.RuntimeClassName,
		NetworkPolicyEnabled: cfg.Sandbox.Kubernetes.NetworkPolicyEnabled,
		DefaultNetworkMode:   cfg.Sandbox.DefaultNetworkMode,
		DefaultEgressCIDRs:   cfg.Sandbox.DefaultEgressCIDRs,
		Image:                cfg.Sandbox.Image,
		ImagePullPolicy:      cfg.Sandbox.ImagePullPolicy,
		WorkspaceMountPath:   cfg.Sandbox.WorkspaceMountPath,
		StorageClass:         cfg.Sandbox.StorageClass,
		WorkspaceSize:        cfg.Sandbox.WorkspaceSize,
		ToolPort:             cfg.Sandbox.ToolPort,
		BrowserPort:          cfg.Sandbox.BrowserPort,
		VNCOrWebPort:         cfg.Sandbox.VNCOrWebPort,
		DefaultCPU:           cfg.Sandbox.DefaultCPU,
		DefaultMemory:        cfg.Sandbox.DefaultMemory,
		MaxCPU:               cfg.Sandbox.MaxCPU,
		MaxMemory:            cfg.Sandbox.MaxMemory,
		IdleTTLSeconds:       cfg.Sandbox.IdleTTLSeconds,
	}
	switch strings.ToLower(strings.TrimSpace(cfg.Sandbox.Driver)) {
	case "", "kubernetes", "k8s":
		return sandbox.NewKubernetesManager(sc)
	default:
		return nil, nil
	}
}

func initObjectStore(cfg config.Config) (ports.ObjectStore, error) {
	switch strings.ToLower(cfg.ObjectStore.Provider) {
	case "s3", "minio":
		prefix := cfg.ObjectStore.S3.Prefix
		if prefix == "" {
			prefix = cfg.ObjectStore.Prefix
		}
		return s3store.New(context.Background(), s3store.Config{
			Endpoint:  cfg.ObjectStore.S3.Endpoint,
			Region:    cfg.ObjectStore.S3.Region,
			AccessKey: cfg.ObjectStore.S3.AccessKey,
			SecretKey: cfg.ObjectStore.S3.SecretKey,
			Bucket:    cfg.ObjectStore.S3.Bucket,
			UseSSL:    cfg.ObjectStore.S3.UseSSL,
			Prefix:    prefix,
		})
	case "none":
		return nil, nil
	default:
		return objectstore.NewLocal(cfg.ObjectStore.Local.Root), nil
	}
}

func seconds(n int) time.Duration {
	if n <= 0 {
		return 0
	}
	return time.Duration(n) * time.Second
}
