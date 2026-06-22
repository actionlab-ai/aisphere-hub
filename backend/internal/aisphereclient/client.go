// Package aisphereclient wraps the aisphere-auth public SDK so AIHub
// internals don't depend on the SDK directly. It centralizes configuration
// (endpoint, service token, cookie name, timeouts) and exposes the same
// surface AIHub needs for both authentication (Introspect) and
// authorization (Check / BatchCheck).
//
// The wrapper is intentionally thin. The underlying SDK at
// github.com/actionlab-ai/aisphere-auth/pkg/aisphereauth/client already
// handles retries, header injection and JSON decoding.
package aisphereclient

import (
	"context"
	"time"

	"github.com/actionlab-ai/aisphere-auth/pkg/aisphereauth"
	authclient "github.com/actionlab-ai/aisphere-auth/pkg/aisphereauth/client"
)

// Config is the AIHub-side view of the aisphere-auth integration.
// It is populated from config.yaml under `aisphereAuth`.
type Config struct {
	// Enabled toggles the integration. When false the wrapper is a no-op.
	Enabled bool `yaml:"enabled"`

	// Endpoint is the aisphere-auth base URL, e.g. http://aisphere-auth:18080.
	Endpoint string `yaml:"endpoint"`

	// ServiceToken is the internal service credential issued by aisphere-auth.
	// AIHub must present this token when calling /authz/check and
	// /auth/sessions/introspect.
	ServiceToken string `yaml:"serviceToken"`

	// ServiceTokenHeader overrides the default `X-Aisphere-Service-Token`
	// header name. Leave empty to use the SDK default.
	ServiceTokenHeader string `yaml:"serviceTokenHeader"`

	// CookieName is the AI Sphere session cookie name. Default `aisphere_session`.
	CookieName string `yaml:"cookieName"`

	// App is the AIHub application identifier passed to aisphere-auth
	// during introspect so the platform can attribute the session.
	App string `yaml:"app"`

	// HTTPTimeoutSeconds controls the underlying HTTP client timeout.
	HTTPTimeoutSeconds int `yaml:"httpTimeoutSeconds"`

	// CacheTTLSeconds enables an in-process introspect cache when > 0.
	// Used by the auth provider to avoid hitting aisphere-auth on every
	// request when the same session cookie repeats.
	CacheTTLSeconds int `yaml:"cacheTTLSeconds"`

	// FailClosed mirrors authz.failClosed: when aisphere-auth is
	// unreachable, deny rather than allow. AuthN still fails closed.
	FailClosed bool `yaml:"failClosed"`
}

// Client is the AIHub-facing façade. Construct once at startup and
// share across providers.
type Client struct {
	cfg     Config
	wrapped *authclient.HTTPClient
	cache   *introspectCache
}

// New constructs a Client. Returns nil if cfg.Enabled is false so callers
// can treat a disabled integration uniformly (nil-safe).
func New(cfg Config) *Client {
	if !cfg.Enabled {
		return nil
	}
	if cfg.Endpoint == "" {
		cfg.Endpoint = "http://aisphere-auth:18080"
	}
	if cfg.CookieName == "" {
		cfg.CookieName = "aisphere_session"
	}
	if cfg.App == "" {
		cfg.App = "aihub"
	}
	if cfg.HTTPTimeoutSeconds <= 0 {
		cfg.HTTPTimeoutSeconds = 10
	}
	opts := []authclient.HTTPClientOption{
		authclient.WithServiceToken(cfg.ServiceToken),
		authclient.WithTimeout(time.Duration(cfg.HTTPTimeoutSeconds) * time.Second),
	}
	if cfg.ServiceTokenHeader != "" {
		opts = append(opts, authclient.WithServiceTokenHeader(cfg.ServiceTokenHeader))
	}
	wrapped := authclient.NewHTTPClient(cfg.Endpoint, opts...)
	c := &Client{cfg: cfg, wrapped: wrapped}
	if cfg.CacheTTLSeconds > 0 {
		c.cache = newIntrospectCache(time.Duration(cfg.CacheTTLSeconds) * time.Second)
	}
	return c
}

// Config returns the resolved configuration (read-only).
func (c *Client) Config() Config { return c.cfg }

// Introspect asks aisphere-auth whether the given sessionID is still
// active and returns the normalized Principal if so.
func (c *Client) Introspect(sessionID string) (*aisphereauth.Principal, error) {
	if c == nil {
		return nil, ErrDisabled
	}
	if c.cache != nil {
		if p, ok := c.cache.get(sessionID); ok {
			return p, nil
		}
	}
	p, err := c.wrapped.Introspect(context.Background(), sessionID, c.cfg.App)
	if err != nil {
		return nil, err
	}
	if c.cache != nil && p != nil {
		c.cache.put(sessionID, p)
	}
	return p, nil
}

// Check delegates an authorization decision to aisphere-auth.
func (c *Client) Check(subject, object, action string) (bool, string, error) {
	if c == nil {
		return false, "aisphere-auth disabled", ErrDisabled
	}
	dec, err := c.wrapped.Check(context.Background(), authclient.CheckRequest{
		Subject: subject,
		Object:  object,
		Action:  action,
		App:     c.cfg.App,
	})
	if err != nil {
		return false, "aisphere-auth error", err
	}
	if dec == nil {
		return false, "aisphere-auth empty decision", nil
	}
	if !dec.Allow && dec.Reason != "" {
		return false, dec.Reason, nil
	}
	return dec.Allow, dec.Source, nil
}

func (c *Client) CheckDetailed(req authclient.CheckRequest) (bool, string, *authclient.Decision, error) {
	if c == nil {
		return false, "aisphere-auth disabled", nil, ErrDisabled
	}
	if req.App == "" {
		req.App = c.cfg.App
	}
	dec, err := c.wrapped.Check(context.Background(), req)
	if err != nil {
		return false, "aisphere-auth error", dec, err
	}
	if dec == nil {
		return false, "aisphere-auth empty decision", nil, nil
	}
	if !dec.Allow && dec.Reason != "" {
		return false, dec.Reason, dec, nil
	}
	return dec.Allow, dec.Source, dec, nil
}

// BatchCheck delegates a batched authorization decision. AIHub uses it
// for the Access diagnostics page.
func (c *Client) BatchCheck(reqs []authclient.CheckRequest) ([]authclient.Decision, error) {
	if c == nil {
		return nil, ErrDisabled
	}
	for i := range reqs {
		if reqs[i].App == "" {
			reqs[i].App = c.cfg.App
		}
	}
	return c.wrapped.BatchCheck(context.Background(), reqs)
}

func (c *Client) CreateResourceGrant(ctx context.Context, grant aisphereauth.ResourceGrant) (*aisphereauth.ResourceGrant, error) {
	if c == nil {
		return nil, ErrDisabled
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if grant.App == "" {
		grant.App = c.cfg.App
	}
	return c.wrapped.CreateResourceGrant(ctx, grant)
}

func (c *Client) ListResourceGrants(ctx context.Context, req aisphereauth.ResourceGrantQuery) (*aisphereauth.ResourceGrantListResponse, error) {
	if c == nil {
		return nil, ErrDisabled
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if req.App == "" {
		req.App = c.cfg.App
	}
	return c.wrapped.ListResourceGrants(ctx, req)
}

func (c *Client) DeleteResourceGrant(ctx context.Context, id string) error {
	if c == nil {
		return ErrDisabled
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return c.wrapped.DeleteResourceGrant(ctx, id)
}

// WriteAudit forwards an audit event to aisphere-auth. It is intentionally best-effort for callers: callers decide whether errors should block the business action.
func (c *Client) WriteAudit(ctx context.Context, event aisphereauth.AuditEvent) (*aisphereauth.AuditEvent, error) {
	if c == nil {
		return nil, ErrDisabled
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if event.App == "" {
		event.App = c.cfg.App
	}
	return c.wrapped.WriteAudit(ctx, event)
}

// ListAudit queries audit events from aisphere-auth.
func (c *Client) ListAudit(ctx context.Context, req aisphereauth.AuditListRequest) (*aisphereauth.AuditListResponse, error) {
	if c == nil {
		return nil, ErrDisabled
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if req.App == "" {
		req.App = c.cfg.App
	}
	return c.wrapped.ListAudit(ctx, req)
}
