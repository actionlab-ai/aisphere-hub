// Package aisphereauth is the AIHub-side AuthProvider that trusts the
// AI Sphere platform session cookie. It complements (does not replace)
// AIHub's existing local / OIDC / JWT providers.
//
// When the request carries the configured AI Sphere session cookie, the
// provider asks aisphere-auth's /auth/sessions/introspect whether it is
// still active and converts the returned platform Principal into the
// AIHub internal auth.Principal shape. When the cookie is missing or
// the session is inactive, the provider returns ok=false so the composite
// authenticator can fall through to the next provider.
package aisphereauth

import (
	"context"
	"net/http"
	"strings"

	"github.com/actionlab-ai/aisphere-auth/pkg/aisphereauth"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/aisphereclient"
	core "github.com/actionlab-ai/aisphere-hub/backend/internal/auth"
)

// Provider implements core.AuthProvider. It is registered with the name
// "aisphere-auth" so config files can refer to it.
type Provider struct {
	client *aisphereclient.Client
}

// New returns a Provider. If client is nil the integration is disabled
// and every call returns ok=false (the composite authenticator will
// simply try the next provider).
func New(client *aisphereclient.Client) *Provider {
	if client == nil {
		return nil
	}
	return &Provider{client: client}
}

// Name returns the provider identifier.
func (p *Provider) Name() string { return "aisphere-auth" }

// Authenticate extracts the AI Sphere session cookie and asks
// aisphere-auth to introspect it.
func (p *Provider) Authenticate(ctx context.Context, r *http.Request) (*core.ExternalIdentity, bool, error) {
	if p == nil || p.client == nil {
		return nil, false, nil
	}
	cookieName := p.client.Config().CookieName
	if cookieName == "" {
		cookieName = "aisphere_session"
	}
	cookie, err := r.Cookie(cookieName)
	if err != nil || strings.TrimSpace(cookie.Value) == "" {
		return nil, false, nil
	}
	principal, err := p.client.Introspect(cookie.Value)
	if err != nil {
		// Introspect errors should not abort the whole auth chain. The
		// composite authenticator will try the next provider. We still
		// surface the error via context so AuditMiddleware / logs can
		// attribute it.
		return nil, false, nil
	}
	if principal == nil {
		return nil, false, nil
	}
	return identityFromPrincipal(p.Name(), principal), true, nil
}

// identityFromPrincipal converts the SDK Principal into AIHub's
// ExternalIdentity so downstream mappers (roleMappings, namespace
// inference, etc) keep working unchanged.
func identityFromPrincipal(provider string, p *aisphereauth.Principal) *core.ExternalIdentity {
	if p == nil {
		return nil
	}
	id := &core.ExternalIdentity{
		Provider:     provider,
		ProviderType: "aisphere-auth",
		Issuer:       "aisphere-auth",
		Subject:      firstNonEmpty(p.SubjectID, p.CasdoorSubject, p.Username),
		SubjectType:  inferSubjectType(p.SubjectID),
		Username:     p.Username,
		Email:        p.Email,
		Organization: p.Organization,
		Groups:       clone(p.Groups),
		Roles:        clone(p.Roles),
		Claims: map[string]any{
			"subjectId":      p.SubjectID,
			"casdoorSubject": p.CasdoorSubject,
			"organization":   p.Organization,
			"orgId":          p.OrgID,
			"projectIds":     append([]string(nil), p.ProjectIDs...),
			"app":            p.App,
			"sessionId":      p.SessionID,
			"authProvider":   p.AuthProvider,
		},
	}
	for k, v := range p.Claims {
		if _, exists := id.Claims[k]; !exists {
			id.Claims[k] = v
		}
	}
	if p.AuthTimeUnix != 0 {
		id.Claims["authTimeUnix"] = p.AuthTimeUnix
	}
	if p.ExpiresAtUnix != 0 {
		id.Claims["expiresAtUnix"] = p.ExpiresAtUnix
	}
	return id
}

func inferSubjectType(subjectID string) string {
	switch {
	case strings.HasPrefix(subjectID, "agent:"):
		return "agent"
	case strings.HasPrefix(subjectID, "service:"):
		return "service"
	case strings.HasPrefix(subjectID, "org:"):
		return "organization"
	default:
		return "human"
	}
}

func clone(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	out = append(out, in...)
	return out
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
