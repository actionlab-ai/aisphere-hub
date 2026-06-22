package jwt

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/actionlab-ai/aisphere-hub/backend/internal/auth"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/config"
)

type Provider struct {
	name          string
	providerType  string
	issuer        string
	audience      string
	jwksURL       string
	publicKeyFile string
	mapping       config.ClaimMappingConfig
	mu            sync.Mutex
	keys          map[string]*rsa.PublicKey
	loadedAt      time.Time
}

func New(name, typ string, cfg config.AuthProviderConfig) *Provider {
	if typ == "" {
		typ = "jwt"
	}
	return &Provider{name: name, providerType: typ, issuer: cfg.Issuer, audience: cfg.Audience, jwksURL: cfg.JWKSURL, publicKeyFile: cfg.PublicKeyFile, mapping: normalizeMapping(cfg.ClaimMapping)}
}

func (p *Provider) Name() string { return p.name }

func (p *Provider) Authenticate(ctx context.Context, r *http.Request) (*auth.ExternalIdentity, bool, error) {
	tok := bearer(r.Header.Get("Authorization"))
	if tok == "" {
		return nil, false, nil
	}
	claims, err := p.Verify(ctx, tok)
	if err != nil {
		return nil, false, err
	}
	return identityFromClaims(p.name, p.providerType, p.issuer, p.mapping, claims), true, nil
}

func (p *Provider) Verify(ctx context.Context, token string) (map[string]any, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid jwt")
	}
	var header map[string]any
	if err := decodeJSON(parts[0], &header); err != nil {
		return nil, err
	}
	alg, _ := header["alg"].(string)
	if alg != "RS256" {
		return nil, fmt.Errorf("unsupported jwt alg %s", alg)
	}
	kid, _ := header["kid"].(string)
	var claims map[string]any
	if err := decodeJSON(parts[1], &claims); err != nil {
		return nil, err
	}
	key, err := p.key(ctx, kid)
	if err != nil {
		return nil, err
	}
	signed := []byte(parts[0] + "." + parts[1])
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, err
	}
	h := sha256.Sum256(signed)
	if err := rsa.VerifyPKCS1v15(key, crypto.SHA256, h[:], sig); err != nil {
		return nil, fmt.Errorf("jwt signature: %w", err)
	}
	if err := p.validateClaims(claims); err != nil {
		return nil, err
	}
	return claims, nil
}

func (p *Provider) validateClaims(c map[string]any) error {
	now := time.Now().Unix()
	if exp, ok := number(c["exp"]); ok && int64(exp) < now {
		return errors.New("jwt expired")
	}
	if nbf, ok := number(c["nbf"]); ok && int64(nbf) > now+60 {
		return errors.New("jwt not valid yet")
	}
	if p.issuer != "" {
		if iss, _ := c["iss"].(string); iss != p.issuer {
			return fmt.Errorf("invalid issuer %q", iss)
		}
	}
	if p.audience != "" && !audienceContains(c["aud"], p.audience) {
		return fmt.Errorf("invalid audience")
	}
	return nil
}

func (p *Provider) key(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.keys == nil || time.Since(p.loadedAt) > 10*time.Minute {
		keys, err := p.loadKeys(ctx)
		if err != nil {
			return nil, err
		}
		p.keys = keys
		p.loadedAt = time.Now()
	}
	if kid != "" {
		if k := p.keys[kid]; k != nil {
			return k, nil
		}
	}
	if len(p.keys) == 1 {
		for _, k := range p.keys {
			return k, nil
		}
	}
	return nil, fmt.Errorf("jwt key not found kid=%s", kid)
}

func (p *Provider) loadKeys(ctx context.Context) (map[string]*rsa.PublicKey, error) {
	if p.publicKeyFile != "" {
		b, err := os.ReadFile(p.publicKeyFile)
		if err != nil {
			return nil, err
		}
		block, _ := pem.Decode(b)
		if block == nil {
			return nil, errors.New("invalid pem public key")
		}
		pubAny, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		pub, ok := pubAny.(*rsa.PublicKey)
		if !ok {
			return nil, errors.New("public key is not rsa")
		}
		return map[string]*rsa.PublicKey{"default": pub}, nil
	}
	if p.jwksURL == "" {
		return nil, errors.New("jwksUrl or publicKeyFile is required")
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, p.jwksURL, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("jwks http %d", resp.StatusCode)
	}
	var jwks struct {
		Keys []struct {
			Kty string `json:"kty"`
			Kid string `json:"kid"`
			N   string `json:"n"`
			E   string `json:"e"`
			Alg string `json:"alg"`
			Use string `json:"use"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, err
	}
	out := map[string]*rsa.PublicKey{}
	for _, k := range jwks.Keys {
		if k.Kty != "RSA" {
			continue
		}
		nb, err := base64.RawURLEncoding.DecodeString(k.N)
		if err != nil {
			continue
		}
		eb, err := base64.RawURLEncoding.DecodeString(k.E)
		if err != nil {
			continue
		}
		e := 0
		for _, b := range eb {
			e = e<<8 + int(b)
		}
		out[k.Kid] = &rsa.PublicKey{N: new(big.Int).SetBytes(nb), E: e}
	}
	if len(out) == 0 {
		return nil, errors.New("empty jwks rsa keys")
	}
	return out, nil
}

func identityFromClaims(provider, typ, issuer string, m config.ClaimMappingConfig, c map[string]any) *auth.ExternalIdentity {
	sub := claimString(c, m.Subject)
	st := claimString(c, m.SubjectType)
	return &auth.ExternalIdentity{Provider: provider, ProviderType: typ, Issuer: first(claimString(c, "iss"), issuer), Subject: sub, SubjectType: st,
		Username: claimString(c, m.Username), Email: claimString(c, m.Email), Organization: claimString(c, m.Organization),
		Groups: claimStringSlice(c, m.Groups), Roles: claimStringSlice(c, m.Roles), Scopes: strings.Fields(claimString(c, "scope")), Claims: c}
}

func normalizeMapping(m config.ClaimMappingConfig) config.ClaimMappingConfig {
	if m.Subject == "" {
		m.Subject = "sub"
	}
	if m.Username == "" {
		m.Username = "preferred_username"
	}
	if m.Email == "" {
		m.Email = "email"
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
	return m
}
func decodeJSON(seg string, v any) error {
	b, err := base64.RawURLEncoding.DecodeString(seg)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
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
	v := c[k]
	switch t := v.(type) {
	case string:
		return t
	case json.Number:
		return t.String()
	case float64:
		return fmt.Sprintf("%.0f", t)
	default:
		return ""
	}
}
func claimStringSlice(c map[string]any, k string) []string {
	if k == "" {
		return nil
	}
	v := c[k]
	switch t := v.(type) {
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
		if t == "" {
			return nil
		}
		if strings.Contains(t, ",") {
			return splitCSV(t)
		}
		return strings.Fields(t)
	default:
		return nil
	}
}
func audienceContains(v any, want string) bool {
	switch t := v.(type) {
	case string:
		return t == want
	case []any:
		for _, x := range t {
			if s, ok := x.(string); ok && s == want {
				return true
			}
		}
		return false
	case []string:
		for _, s := range t {
			if s == want {
				return true
			}
		}
		return false
	default:
		return false
	}
}
func number(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case int64:
		return float64(t), true
	case json.Number:
		f, _ := t.Float64()
		return f, true
	default:
		return 0, false
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
func first(v, d string) string {
	if strings.TrimSpace(v) != "" {
		return v
	}
	return d
}
