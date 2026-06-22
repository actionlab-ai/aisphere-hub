package authhttp

import (
	"errors"
	"net/http"
	"strings"

	"github.com/actionlab-ai/aisphere-hub/backend/internal/aisphereclient"
	core "github.com/actionlab-ai/aisphere-hub/backend/internal/auth"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/auth/providers/aisphereauth"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/auth/providers/apikey"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/auth/providers/dbapikey"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/auth/providers/introspection"
	jwtprovider "github.com/actionlab-ai/aisphere-hub/backend/internal/auth/providers/jwt"
	localprovider "github.com/actionlab-ai/aisphere-hub/backend/internal/auth/providers/local"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/auth/providers/oidc"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/config"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/store"
	"github.com/gin-gonic/gin"
)

func Middleware(cfg config.AuthConfig) gin.HandlerFunc {
	var localMgr *localprovider.Manager
	if localEnabled(cfg) {
		localMgr, _ = localprovider.NewManager(cfg.Local)
	}
	return MiddlewareWithLocal(cfg, localMgr)
}

func MiddlewareWithLocal(cfg config.AuthConfig, localMgr *localprovider.Manager) gin.HandlerFunc {
	return MiddlewareWithLocalAndStore(cfg, localMgr, nil)
}

func MiddlewareWithLocalAndStore(cfg config.AuthConfig, localMgr *localprovider.Manager, st store.Backend) gin.HandlerFunc {
	return MiddlewareWithLocalStoreAuthorizer(cfg, localMgr, st, nil)
}

func MiddlewareWithLocalStoreAuthorizer(cfg config.AuthConfig, localMgr *localprovider.Manager, st store.Backend, injectedAuthz core.Authorizer) gin.HandlerFunc {
	return MiddlewareWithAISphere(cfg, localMgr, st, injectedAuthz, nil)
}

// MiddlewareWithAISphere is the same as MiddlewareWithLocalStoreAuthorizer
// but additionally accepts an aisphere-auth client. When client is non-nil
// and cfg is paired with the global aisphereAuth.enabled flag, an extra
// aisphereauth.Provider is prepended to the auth chain. Existing providers
// (local / oidc / jwt / api_key / introspection / dbapikey) are preserved
// exactly as before so operators can mix AI Sphere sessions with legacy
// tokens during migration.
func MiddlewareWithAISphere(cfg config.AuthConfig, localMgr *localprovider.Manager, st store.Backend, injectedAuthz core.Authorizer, aisphereClient *aisphereclient.Client) gin.HandlerFunc {
	providers := BuildProviders(cfg, localMgr)
	mode := strings.ToLower(first(cfg.Mode, "mixed"))
	if st != nil && mode != "local" {
		providers = append(providers, dbapikey.New("db-token", "X-API-Key", st))
	}
	// Prepend the AI Sphere provider so the cookie session takes priority
	// over fallback providers. If it returns ok=false the composite
	// authenticator transparently falls through to local / oidc / etc.
	if aisp := aisphereauth.New(aisphereClient); aisp != nil {
		providers = append([]core.AuthProvider{aisp}, providers...)
	}

	authn := core.NewComposite(providers, core.NewStaticMapper(cfg.RoleMappings))
	var authz core.Authorizer = core.NewStaticAuthorizer()
	if st != nil {
		authz = core.NewStoreAuthorizer(st)
	}
	if injectedAuthz != nil {
		authz = injectedAuthz
	}
	return func(c *gin.Context) {
		if !cfg.Enabled {
			c.Next()
			return
		}
		action := RequiredPermission(c.Request.Method, c.Request.URL.Path)
		res := resourceRef(c, action)
		if action == "auth:login" || action == "auth:setup" || (action == "public:read" && cfg.AllowPublicRead) {
			c.Next()
			return
		}
		p, ok, err := authn.Authenticate(c.Request.Context(), c.Request)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized", "message": err.Error()})
			return
		}
		if !ok {
			if aisphereClient != nil && cfg.AllowPublicRead && isAnonymousResourceRead(action, c.Request.URL.Path) {
				anonymous := &core.Principal{SubjectID: "anonymous", SubjectType: "anonymous", Username: "anonymous", Provider: "anonymous"}
				if err := authz.Authorize(c.Request.Context(), anonymous, action, res); err == nil {
					c.Set(core.PrincipalContextKey, *anonymous)
					c.Next()
					return
				}
			}
			if cfg.AllowAnonymous && action == "public:read" {
				c.Next()
				return
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		if err := authz.Authorize(c.Request.Context(), p, action, res); err != nil {
			status := http.StatusForbidden
			if errors.Is(err, core.ErrForbidden) {
				status = http.StatusForbidden
			}
			c.AbortWithStatusJSON(status, gin.H{"error": "forbidden", "message": err.Error(), "required": action})
			return
		}
		c.Set(core.PrincipalContextKey, *p)
		c.Next()
	}
}

func BuildProviders(cfg config.AuthConfig, localMgr *localprovider.Manager) []core.AuthProvider {
	out := []core.AuthProvider{}
	mode := strings.ToLower(first(cfg.Mode, "mixed"))
	if localMgr != nil && (mode == "local" || mode == "mixed") {
		out = append(out, localprovider.NewProvider("local", localMgr))
	}
	if mode == "local" {
		return out
	}
	for _, p := range cfg.Providers {
		name := first(p.Name, p.Type)
		switch strings.ToLower(p.Type) {
		case "api_key", "apikey":
			out = append(out, apikey.New(name, p.Header, p.Keys))
		case "jwt", "external_jwt", "oauth2_jwt":
			out = append(out, jwtprovider.New(name, "jwt", p))
		case "oidc":
			out = append(out, oidc.New(name, p))
		case "introspection", "oauth2_introspection", "opaque":
			out = append(out, introspection.New(name, p))
		case "aisphere-auth", "aisphere_auth", "aisphereauth":
			// Type-level registration is a no-op here: the AI Sphere
			// provider is constructed from the dedicated aisphereAuth
			// config block and injected via MiddlewareWithAISphere.
			// We deliberately do NOT support configuring it as a
			// generic provider entry, to keep the service token /
			// endpoint / cookie configuration centralized.
			continue
		}
	}
	if len(cfg.APIKeys) > 0 {
		out = append(out, apikey.New("bootstrap", "X-API-Key", cfg.APIKeys))
	}
	return out
}

func RequiredPermission(method, path string) string {
	switch {
	case strings.HasPrefix(path, "/v3/auth/setup"):
		return "auth:setup"
	case strings.HasPrefix(path, "/v3/auth/aisphere/login") || strings.HasPrefix(path, "/v3/auth/aisphere/callback"):
		// AI Sphere platform login is a redirect to aisphere-auth.
		// It must stay public so unauthenticated users can kick
		// off the OAuth flow.
		return "auth:login"
	case strings.HasPrefix(path, "/v3/auth/oidc/login") || strings.HasPrefix(path, "/v3/auth/oidc/callback"):
		return "auth:login"
	case strings.HasPrefix(path, "/v3/auth/login") || strings.HasPrefix(path, "/v3/auth/refresh"):
		return "auth:login"
	case strings.HasPrefix(path, "/v3/auth/me"):
		return "auth:me"
	case strings.HasPrefix(path, "/v3/admin/access"):
		if method == http.MethodGet || strings.HasSuffix(path, "/evaluate") {
			return "access:admin:read"
		}
		return "access:admin:write"
	case strings.HasPrefix(path, "/ui") || path == "/":
		return "public:read"
	case strings.HasPrefix(path, "/healthz"), strings.HasPrefix(path, "/livez"), strings.HasPrefix(path, "/readyz"):
		return "public:read"
	case strings.HasPrefix(path, "/v3/client/ai/skills") || strings.HasPrefix(path, "/v3/client/ai/skill-groups"):
		return "skill:read"
	case strings.HasPrefix(path, "/registry/"):
		return "public:read"
	case strings.HasPrefix(path, "/v3/agent/ai/skill-proposals") && method == http.MethodPost:
		return "skill:proposal:create"
	case strings.HasPrefix(path, "/v3/agent/ai/skill-proposals"):
		return "skill:proposal:read"
	case strings.HasPrefix(path, "/v3/agent/ai/skill-overlays"):
		return "skill:overlay:read"
	case strings.HasPrefix(path, "/v3/admin/ai/skill-proposals"):
		return "skill:proposal:review"
	case strings.HasPrefix(path, "/v3/admin/notifications"):
		return "notification:read"
	case strings.HasPrefix(path, "/v3/aihub/catalog"):
		// Catalog endpoints do per-resource checks in handlers so a user with
		// only one shared skill can still list and download that skill without
		// needing global aihub:skill:* permission.
		return "auth:me"
	case strings.HasPrefix(path, "/v3/aihub/runtime/"):
		// Runtime endpoints authenticate the principal first, then perform
		// object-level checks in the concrete handlers. This keeps Agent
		// resolution usable for users who only have a ResourceGrant on one
		// Agent instead of a global system:admin permission.
		return "auth:me"
	case strings.HasPrefix(path, "/v3/aihub/tools"):
		if method == http.MethodGet {
			return "tool:read"
		}
		return "tool:write"
	case strings.HasPrefix(path, "/v3/aihub/tool-failures"):
		if method == http.MethodGet {
			return "tool:read"
		}
		return "tool:write"
	case strings.HasPrefix(path, "/v3/aihub/agents"):
		if method == http.MethodGet {
			return "agent:read"
		}
		return "agent:write"
	case strings.HasPrefix(path, "/v3/aihub/workflows"):
		if strings.Contains(path, "/run") {
			return "workflow:run"
		}
		if method == http.MethodGet {
			return "workflow:read"
		}
		return "workflow:write"
	case strings.HasPrefix(path, "/v3/aihub/runs"):
		if method == http.MethodGet {
			return "run:read"
		}
		return "run:write"
	case strings.HasPrefix(path, "/v3/aihub/skillsets") || strings.HasPrefix(path, "/v3/aihub/skillset"):
		if method == http.MethodGet {
			return "skill:group:read"
		}
		return "skill:group:write"
	case strings.HasPrefix(path, "/v3/aihub/skills/upload"):
		return "skill:admin:write"
	case strings.HasPrefix(path, "/v3/aihub/skills"):
		if method == http.MethodGet {
			return "skill:admin:read"
		}
		return "skill:admin:write"
	case strings.HasPrefix(path, "/v3/aihub/skill"):
		if method == http.MethodGet {
			return "skill:admin:read"
		}
		if strings.Contains(path, "/publish") || strings.Contains(path, "/force-publish") {
			return "skill:publish"
		}
		return "skill:admin:write"
	case strings.HasPrefix(path, "/v3/admin/iam"):
		return "iam:admin"
	case strings.HasPrefix(path, "/v3/admin/namespaces"):
		if method == http.MethodGet {
			return "namespace:read"
		}
		return "namespace:write"
	case strings.Contains(path, "/skill-groups"):
		if method == http.MethodGet {
			return "skill:group:read"
		}
		return "skill:group:write"
	case strings.HasPrefix(path, "/v3/admin/ai/skills"):
		if method == http.MethodGet {
			return "skill:admin:read"
		}
		return "skill:admin:write"
	default:
		return "system:admin"
	}
}

func isAnonymousResourceRead(action, path string) bool {
	if action != "skill:read" && action != "skill:group:read" {
		return false
	}
	return strings.HasPrefix(path, "/v3/client/ai/skills/") ||
		strings.HasPrefix(path, "/v3/client/ai/groups/") ||
		strings.HasPrefix(path, "/v3/client/ai/skill-groups/")
}

func resourceRef(c *gin.Context, action string) core.ResourceRef {
	ns := c.Query("namespaceId")
	if ns == "" {
		ns = c.PostForm("namespaceId")
	}
	if ns == "" {
		ns = c.Param("namespaceId")
	}
	return core.ResourceRef{
		NamespaceID: ns,
		Type:        "skill",
		Name:        firstParamOrQuery(c, "skillName", "name"),
		GroupName:   firstParamOrQuery(c, "skillSetName", "groupName"),
		AgentID:     firstParamOrQuery(c, "agentId"),
		ToolID:      firstParamOrQuery(c, "toolId"),
		WorkflowID:  firstParamOrQuery(c, "workflowId"),
		RunID:       firstParamOrQuery(c, "runId"),
		ProposalID:  c.Param("proposalId"),
		Path:        c.Request.URL.Path,
		HTTPMethod:  c.Request.Method,
	}
}
func firstParamOrQuery(c *gin.Context, keys ...string) string {
	for _, k := range keys {
		if v := c.Param(k); v != "" {
			return v
		}
	}
	return firstQuery(c, keys...)
}

func firstQuery(c *gin.Context, keys ...string) string {
	for _, k := range keys {
		if v := c.Query(k); v != "" {
			return v
		}
		if v := c.PostForm(k); v != "" {
			return v
		}
	}
	return ""
}
func first(v, d string) string {
	if strings.TrimSpace(v) != "" {
		return v
	}
	return d
}

func localEnabled(cfg config.AuthConfig) bool {
	mode := strings.ToLower(first(cfg.Mode, "mixed"))
	return cfg.Local.Enabled && (mode == "local" || mode == "mixed")
}
