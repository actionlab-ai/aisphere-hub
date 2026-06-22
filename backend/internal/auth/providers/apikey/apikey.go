package apikey

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/actionlab-ai/aisphere-hub/backend/internal/auth"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/config"
)

type Provider struct {
	name   string
	header string
	keys   map[string]config.APIKeyAuth
}

func New(name, header string, keys []config.APIKeyAuth) *Provider {
	if header == "" {
		header = "X-API-Key"
	}
	m := map[string]config.APIKeyAuth{}
	for _, k := range keys {
		t := strings.TrimSpace(k.Token)
		if t == "" {
			continue
		}
		m[t] = k
		m[sha256Hex(t)] = k // allow configs to store token hash instead of plain text
	}
	return &Provider{name: name, header: header, keys: m}
}

func (p *Provider) Name() string { return p.name }

func (p *Provider) Authenticate(ctx context.Context, r *http.Request) (*auth.ExternalIdentity, bool, error) {
	token := strings.TrimSpace(r.Header.Get(p.header))
	if token == "" && p.header != "Authorization" {
		token = bearer(r.Header.Get("Authorization"))
	}
	if token == "" {
		return nil, false, nil
	}
	k, ok := p.keys[token]
	if !ok {
		k, ok = p.keys[sha256Hex(token)]
	}
	if !ok {
		return nil, false, nil
	}
	sub := first(k.SubjectID, k.Name)
	return &auth.ExternalIdentity{
		Provider: p.name, ProviderType: "api_key", Subject: sub, SubjectType: first(k.SubjectType, "service"),
		Organization: k.Organization, Roles: k.Roles, Permissions: k.Permissions, Namespaces: k.Namespaces, Claims: map[string]any{"api_key_name": k.Name},
	}, true, nil
}

func bearer(v string) string {
	if strings.HasPrefix(strings.ToLower(v), "bearer ") {
		return strings.TrimSpace(v[7:])
	}
	return ""
}
func sha256Hex(s string) string { sum := sha256.Sum256([]byte(s)); return hex.EncodeToString(sum[:]) }
func first(v, d string) string {
	if strings.TrimSpace(v) != "" {
		return v
	}
	return d
}
