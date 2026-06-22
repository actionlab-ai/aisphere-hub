package introspection

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/actionlab-ai/aisphere-hub/backend/internal/auth"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/config"
)

type Provider struct {
	name string
	cfg  config.AuthProviderConfig
}

func New(name string, cfg config.AuthProviderConfig) *Provider {
	return &Provider{name: name, cfg: cfg}
}
func (p *Provider) Name() string { return p.name }
func (p *Provider) Authenticate(ctx context.Context, r *http.Request) (*auth.ExternalIdentity, bool, error) {
	token := bearer(r.Header.Get("Authorization"))
	if token == "" {
		return nil, false, nil
	}
	if p.cfg.IntrospectionURL == "" {
		return nil, false, fmt.Errorf("introspectionUrl is required")
	}
	form := url.Values{"token": []string{token}}
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.IntrospectionURL, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if p.cfg.ClientID != "" || p.cfg.ClientSecret != "" {
		req.SetBasicAuth(p.cfg.ClientID, p.cfg.ClientSecret)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return nil, false, fmt.Errorf("introspection http %d", resp.StatusCode)
	}
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, false, err
	}
	active, _ := out["active"].(bool)
	if !active {
		return nil, false, nil
	}
	m := p.cfg.ClaimMapping
	if m.Subject == "" {
		m.Subject = "sub"
	}
	if m.Username == "" {
		m.Username = "username"
	}
	if m.Groups == "" {
		m.Groups = "groups"
	}
	if m.Roles == "" {
		m.Roles = "roles"
	}
	if m.Organization == "" {
		m.Organization = "organization"
	}
	return &auth.ExternalIdentity{Provider: p.name, ProviderType: "introspection", Issuer: p.cfg.Issuer, Subject: claimString(out, m.Subject), SubjectType: claimString(out, m.SubjectType), Username: claimString(out, m.Username), Email: claimString(out, m.Email), Organization: claimString(out, m.Organization), Groups: claimSlice(out, m.Groups), Roles: claimSlice(out, m.Roles), Claims: out}, true, nil
}
func bearer(v string) string {
	if strings.HasPrefix(strings.ToLower(v), "bearer ") {
		return strings.TrimSpace(v[7:])
	}
	return ""
}
func claimString(c map[string]any, k string) string {
	if k == "" {
		return ""
	}
	if s, ok := c[k].(string); ok {
		return s
	}
	return ""
}
func claimSlice(c map[string]any, k string) []string {
	if k == "" {
		return nil
	}
	switch t := c[k].(type) {
	case []any:
		out := []string{}
		for _, x := range t {
			if s, ok := x.(string); ok {
				out = append(out, s)
			}
		}
		return out
	case []string:
		return t
	case string:
		return strings.Fields(t)
	default:
		return nil
	}
}
