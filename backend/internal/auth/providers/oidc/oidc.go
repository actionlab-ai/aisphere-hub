package oidc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/actionlab-ai/aisphere-hub/backend/internal/auth"
	jwtprovider "github.com/actionlab-ai/aisphere-hub/backend/internal/auth/providers/jwt"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/config"
)

type Provider struct {
	name     string
	cfg      config.AuthProviderConfig
	mu       sync.Mutex
	jwt      *jwtprovider.Provider
	loadedAt time.Time
}

func New(name string, cfg config.AuthProviderConfig) *Provider {
	return &Provider{name: name, cfg: cfg}
}
func (p *Provider) Name() string { return p.name }

func (p *Provider) Authenticate(ctx context.Context, r *http.Request) (*auth.ExternalIdentity, bool, error) {
	jp, err := p.jwtProvider(ctx)
	if err != nil {
		return nil, false, err
	}
	id, ok, err := jp.Authenticate(ctx, r)
	if id != nil {
		id.Provider = p.name
		id.ProviderType = "oidc"
	}
	return id, ok, err
}

func (p *Provider) jwtProvider(ctx context.Context) (*jwtprovider.Provider, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.jwt != nil && time.Since(p.loadedAt) < time.Hour {
		return p.jwt, nil
	}
	cfg := p.cfg
	if cfg.JWKSURL == "" {
		if cfg.Issuer == "" {
			return nil, fmt.Errorf("oidc issuer is required")
		}
		u := strings.TrimRight(cfg.Issuer, "/") + "/.well-known/openid-configuration"
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode/100 != 2 {
			return nil, fmt.Errorf("oidc discovery http %d", resp.StatusCode)
		}
		var d struct {
			JWKSURI string `json:"jwks_uri"`
			Issuer  string `json:"issuer"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
			return nil, err
		}
		cfg.JWKSURL = d.JWKSURI
		if cfg.Issuer == "" {
			cfg.Issuer = d.Issuer
		}
	}
	p.jwt = jwtprovider.New(p.name, "oidc", cfg)
	p.loadedAt = time.Now()
	return p.jwt, nil
}
