package authhttp

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/actionlab-ai/aisphere-hub/backend/internal/aisphereclient"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/config"
	"github.com/gin-gonic/gin"
)

type oidcDiscovery struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	JWKSURI               string `json:"jwks_uri"`
}

func (h *localHandler) oidcLogin(c *gin.Context) {
	p, ok := firstOIDCProvider(h.cfg)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "oidc provider is not configured"})
		return
	}
	d, err := discoverOIDC(c.Request.Context(), p.Issuer)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	redirect := normalizeOIDCRedirect(c.Query("redirect"))
	if redirect == "" {
		redirect = "/ui/"
	}
	state := base64.RawURLEncoding.EncodeToString([]byte(redirect))
	q := url.Values{}
	q.Set("client_id", p.ClientID)
	q.Set("redirect_uri", p.RedirectURL)
	q.Set("response_type", "code")
	q.Set("scope", strings.Join(append([]string{"openid", "profile", "email"}, p.Scopes...), " "))
	q.Set("state", state)
	c.Redirect(http.StatusFound, d.AuthorizationEndpoint+"?"+q.Encode())
}

func (h *localHandler) oidcCallback(c *gin.Context) {
	p, ok := firstOIDCProvider(h.cfg)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "oidc provider is not configured"})
		return
	}
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing code"})
		return
	}
	d, err := discoverOIDC(c.Request.Context(), p.Issuer)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", p.RedirectURL)
	form.Set("client_id", p.ClientID)
	if p.ClientSecret != "" {
		form.Set("client_secret", p.ClientSecret)
	}
	req, _ := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, d.TokenEndpoint, bytes.NewBufferString(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()
	var tokenResp map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "decode token response: " + err.Error()})
		return
	}
	if resp.StatusCode/100 != 2 {
		c.JSON(http.StatusBadGateway, gin.H{"error": "token endpoint error", "detail": tokenResp})
		return
	}
	accessToken, _ := tokenResp["access_token"].(string)
	refreshToken, _ := tokenResp["refresh_token"].(string)
	if accessToken == "" {
		c.JSON(http.StatusBadGateway, gin.H{"error": "token response has no access_token"})
		return
	}
	redirect := "/ui/"
	if st := c.Query("state"); st != "" {
		if b, err := base64.RawURLEncoding.DecodeString(st); err == nil {
			if r := normalizeOIDCRedirect(string(b)); r != "" {
				redirect = r
			}
		}
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	_ = oidcCallbackHTML.Execute(c.Writer, gin.H{
		"AccessTokenJS":    jsStringLiteral(accessToken),
		"RefreshTokenJS":   jsStringLiteral(refreshToken),
		"HasRefreshToken":  refreshToken != "",
		"RedirectJS":       jsStringLiteral(redirect),
		"UseFragmentRelay": strings.HasPrefix(redirect, "http://") || strings.HasPrefix(redirect, "https://"),
	})
}

func jsStringLiteral(v string) template.JS {
	b, _ := json.Marshal(v)
	return template.JS(b)
}

func normalizeOIDCRedirect(raw string) string {
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "/") && !strings.HasPrefix(raw, "//") {
		return raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	// Development-friendly allowlist. Production should usually use a relative /ui/ redirect
	// or put AIHub UI and API behind the same domain/reverse proxy.
	host := strings.ToLower(u.Hostname())
	if host == "localhost" || host == "127.0.0.1" || strings.HasPrefix(host, "10.") || strings.HasPrefix(host, "192.168.") || strings.HasPrefix(host, "172.") {
		return raw
	}
	return ""
}

func firstOIDCProvider(cfg config.AuthConfig) (config.AuthProviderConfig, bool) {
	for _, p := range cfg.Providers {
		if strings.EqualFold(p.Type, "oidc") && p.Issuer != "" && p.ClientID != "" && p.RedirectURL != "" {
			return p, true
		}
	}
	return config.AuthProviderConfig{}, false
}

func discoverOIDC(ctx context.Context, issuer string) (*oidcDiscovery, error) {
	if issuer == "" {
		return nil, fmt.Errorf("issuer is required")
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(issuer, "/")+"/.well-known/openid-configuration", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("oidc discovery http %d", resp.StatusCode)
	}
	var d oidcDiscovery
	if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
		return nil, err
	}
	if d.AuthorizationEndpoint == "" || d.TokenEndpoint == "" {
		return nil, fmt.Errorf("oidc discovery missing authorization_endpoint or token_endpoint")
	}
	return &d, nil
}

var oidcCallbackHTML = template.Must(template.New("oidc-callback").Parse(`<!doctype html>
<html><head><meta charset="utf-8"><title>AIHub 登录中</title></head>
<body><p>登录成功，正在进入 AIHub...</p>
<script>
const redirect = {{.RedirectJS}};
{{if .UseFragmentRelay}}
const fragment = new URLSearchParams();
fragment.set('access_token', {{.AccessTokenJS}});
{{if .HasRefreshToken}}fragment.set('refresh_token', {{.RefreshTokenJS}});{{end}}
window.location.replace(redirect + (redirect.includes('#') ? '&' : '#') + fragment.toString());
{{else}}
localStorage.setItem('aihub_console_token', {{.AccessTokenJS}});
{{if .HasRefreshToken}}localStorage.setItem('aihub_console_refresh', {{.RefreshTokenJS}});{{end}}
window.location.replace(redirect);
{{end}}
</script>
</body></html>`))

// RegisterAISphereRoutes mounts the redirect endpoints that bridge
// AIHub with the platform aisphere-auth service. The AIHub frontend
// can either link directly to aisphere-auth's /auth/login, or use these
// routes so the redirect URL stays under AIHub's domain (useful when
// aisphere-auth is on an internal-only hostname).
//
// GET /v3/auth/aisphere/login?redirect=<path>
//
//	302 to <aisphere-auth endpoint>/auth/login?app=aihub&redirect=<redirect>
//
// GET /v3/auth/aisphere/callback
//
//	Reserved for future server-side callback wiring. Today aisphere-auth
//	sets the aisphere_session cookie directly on its own domain and
//	redirects back to AIHub, so this endpoint is a thin landing page
//	that closes the popup / refreshes the opener.
func RegisterAISphereRoutes(r *gin.Engine, client *aisphereclient.Client) {
	if client == nil {
		return
	}
	cfg := client.Config()
	r.GET("/v3/auth/aisphere/login", func(c *gin.Context) {
		redirect := normalizeOIDCRedirect(c.Query("redirect"))
		if redirect == "" {
			redirect = "/"
		}
		q := url.Values{}
		q.Set("app", cfg.App)
		q.Set("redirect", redirect)
		c.Redirect(http.StatusFound, strings.TrimRight(cfg.Endpoint, "/")+"/auth/login?"+q.Encode())
	})
	r.GET("/v3/auth/aisphere/callback", func(c *gin.Context) {
		next := normalizeOIDCRedirect(c.Query("redirect"))
		if next == "" {
			next = "/"
		}
		c.Header("Content-Type", "text/html; charset=utf-8")
		nextJS := string(jsStringLiteral(next))
		_, _ = c.Writer.WriteString(`<!doctype html><html><head><meta charset="utf-8"><title>AIHub</title></head>` +
			`<body><p>AI Sphere login complete, returning to AIHub...</p>` +
			`<script>window.location.replace(` + nextJS + `);</script>` +
			`</body></html>`)
	})
}
