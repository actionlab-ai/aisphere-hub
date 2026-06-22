package dbapikey

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/actionlab-ai/aisphere-hub/backend/internal/auth"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/store"
)

type Provider struct {
	name   string
	header string
	st     store.Backend
}

func New(name, header string, st store.Backend) *Provider {
	if name == "" {
		name = "db-api-key"
	}
	if header == "" {
		header = "X-API-Key"
	}
	return &Provider{name: name, header: header, st: st}
}
func (p *Provider) Name() string { return p.name }
func (p *Provider) Authenticate(ctx context.Context, r *http.Request) (*auth.ExternalIdentity, bool, error) {
	if p.st == nil {
		return nil, false, nil
	}
	token := strings.TrimSpace(r.Header.Get(p.header))
	if token == "" {
		token = bearer(r.Header.Get("Authorization"))
	}
	if token == "" {
		return nil, false, nil
	}
	h := sha256.Sum256([]byte(token))
	t, err := p.st.FindActiveTokenByHash(hex.EncodeToString(h[:]))
	if err != nil || t == nil {
		return nil, false, err
	}
	return &auth.ExternalIdentity{Provider: p.name, ProviderType: "api_key", Subject: t.SubjectID, SubjectType: first(t.SubjectType, "service"), Roles: t.Roles, Permissions: t.Permissions, Namespaces: t.Namespaces, Claims: map[string]any{"key_id": t.KeyID, "token_name": t.Name}}, true, nil
}
func bearer(v string) string {
	if strings.HasPrefix(strings.ToLower(v), "bearer ") {
		return strings.TrimSpace(v[7:])
	}
	return ""
}
func first(v, d string) string {
	if strings.TrimSpace(v) != "" {
		return v
	}
	return d
}
