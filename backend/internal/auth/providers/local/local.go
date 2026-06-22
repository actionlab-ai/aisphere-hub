package local

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/actionlab-ai/aisphere-hub/backend/internal/auth"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/config"
)

type Account struct {
	Username     string    `json:"username"`
	PasswordHash string    `json:"passwordHash"`
	SubjectID    string    `json:"subjectId"`
	SubjectType  string    `json:"subjectType"`
	DisplayName  string    `json:"displayName,omitempty"`
	Email        string    `json:"email,omitempty"`
	Organization string    `json:"organization,omitempty"`
	Roles        []string  `json:"roles,omitempty"`
	Permissions  []string  `json:"permissions,omitempty"`
	Namespaces   []string  `json:"namespaces,omitempty"`
	Disabled     bool      `json:"disabled,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type AccountStore interface {
	ListLocalAccounts(ctx context.Context) ([]*Account, error)
	SaveLocalAccount(ctx context.Context, acc *Account) error
}

type Manager struct {
	mu       sync.RWMutex
	cfg      config.LocalAuthConfig
	accounts map[string]*Account
	store    AccountStore
}

var ErrAlreadyInitialized = errors.New("local auth has already been initialized")

func (m *Manager) IsInitialized() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, a := range m.accounts {
		if a != nil && !a.Disabled {
			return true
		}
	}
	return false
}

func (m *Manager) SetupInitialAdmin(u config.LocalUserAuth) (*Account, error) {
	if strings.TrimSpace(u.Username) == "" {
		u.Username = "admin"
	}
	if strings.TrimSpace(u.Password) == "" && strings.TrimSpace(u.PasswordHash) == "" {
		return nil, fmt.Errorf("password is required")
	}
	if u.SubjectType == "" {
		u.SubjectType = "human"
	}
	if u.SubjectID == "" {
		u.SubjectID = "user:" + u.Username
	}
	if len(u.Roles) == 0 {
		u.Roles = []string{"admin"}
	}
	if len(u.Permissions) == 0 {
		u.Permissions = []string{"*"}
	}
	if len(u.Namespaces) == 0 {
		u.Namespaces = []string{"*"}
	}
	if u.Organization == "" {
		u.Organization = "default"
	}
	acc, err := accountFromConfig(u)
	if err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, a := range m.accounts {
		if a != nil && !a.Disabled {
			return nil, ErrAlreadyInitialized
		}
	}
	m.accounts[acc.Username] = acc
	if err := m.saveLocked(); err != nil {
		return nil, err
	}
	return cloneAccount(acc), nil
}

func NewManager(cfg config.LocalAuthConfig) (*Manager, error) {
	return NewManagerWithStore(cfg, nil)
}

func NewManagerWithStore(cfg config.LocalAuthConfig, st AccountStore) (*Manager, error) {
	m := &Manager{cfg: cfg, accounts: map[string]*Account{}, store: st}
	if err := m.load(); err != nil {
		return nil, err
	}
	if cfg.AutoCreateBootstrap {
		changed := false
		for _, u := range cfg.Users {
			if strings.TrimSpace(u.Username) == "" {
				continue
			}
			if _, ok := m.accounts[u.Username]; ok {
				continue
			}
			acc, err := accountFromConfig(u)
			if err != nil {
				return nil, err
			}
			m.accounts[acc.Username] = acc
			changed = true
		}
		if changed {
			_ = m.saveLocked()
		}
	}
	return m, nil
}

func (m *Manager) Authenticate(username, password string) (*Account, error) {
	m.mu.RLock()
	acc := cloneAccount(m.accounts[username])
	m.mu.RUnlock()
	if acc == nil || acc.Disabled {
		return nil, errors.New("invalid username or password")
	}
	if !VerifyPassword(acc.PasswordHash, password) {
		return nil, errors.New("invalid username or password")
	}
	return acc, nil
}

func (m *Manager) List() []*Account {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := []*Account{}
	for _, a := range m.accounts {
		out = append(out, cloneAccount(a))
	}
	return out
}

func (m *Manager) CreateOrUpdate(u config.LocalUserAuth) (*Account, error) {
	acc, err := accountFromConfig(u)
	if err != nil {
		return nil, err
	}
	m.mu.Lock()
	if old := m.accounts[acc.Username]; old != nil && acc.PasswordHash == "" {
		acc.PasswordHash = old.PasswordHash
	}
	acc.UpdatedAt = time.Now()
	m.accounts[acc.Username] = acc
	err = m.saveLocked()
	m.mu.Unlock()
	return cloneAccount(acc), err
}

func (m *Manager) Disable(username string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	acc := m.accounts[username]
	if acc == nil {
		return os.ErrNotExist
	}
	acc.Disabled = true
	acc.UpdatedAt = time.Now()
	return m.saveLocked()
}

func (m *Manager) IssueTokens(acc *Account) (map[string]any, error) {
	now := time.Now()
	accessTTL := time.Duration(m.cfg.AccessTokenTTLSeconds) * time.Second
	refreshTTL := time.Duration(m.cfg.RefreshTTLSeconds) * time.Second
	if accessTTL <= 0 {
		accessTTL = time.Hour
	}
	if refreshTTL <= 0 {
		refreshTTL = 7 * 24 * time.Hour
	}
	access, err := SignJWT(m.cfg.SigningSecret, map[string]any{
		"iss": m.cfg.Issuer, "sub": acc.SubjectID, "typ": acc.SubjectType, "preferred_username": acc.Username,
		"email": acc.Email, "organization": acc.Organization, "roles": acc.Roles, "permissions": acc.Permissions,
		"namespaces": acc.Namespaces, "iat": now.Unix(), "nbf": now.Unix(), "exp": now.Add(accessTTL).Unix(), "token_type": "access",
	})
	if err != nil {
		return nil, err
	}
	refresh, err := SignJWT(m.cfg.SigningSecret, map[string]any{
		"iss": m.cfg.Issuer, "sub": acc.SubjectID, "typ": acc.SubjectType, "preferred_username": acc.Username,
		"iat": now.Unix(), "nbf": now.Unix(), "exp": now.Add(refreshTTL).Unix(), "token_type": "refresh",
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{"accessToken": access, "refreshToken": refresh, "tokenType": "Bearer", "expiresIn": int(accessTTL.Seconds()), "subjectId": acc.SubjectID, "subjectType": acc.SubjectType}, nil
}

func (m *Manager) VerifyToken(token string) (map[string]any, error) {
	return VerifyJWT(m.cfg.SigningSecret, m.cfg.Issuer, token)
}

func (m *Manager) load() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.store != nil {
		accounts, err := m.store.ListLocalAccounts(context.Background())
		if err != nil {
			return err
		}
		for _, a := range accounts {
			if a != nil && a.Username != "" {
				m.accounts[a.Username] = a
			}
		}
		return nil
	}
	if m.cfg.AccountFile == "" {
		return nil
	}
	b, err := os.ReadFile(m.cfg.AccountFile)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var doc struct {
		Accounts []*Account `json:"accounts"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		return err
	}
	for _, a := range doc.Accounts {
		if a != nil && a.Username != "" {
			m.accounts[a.Username] = a
		}
	}
	return nil
}

func (m *Manager) saveLocked() error {
	if m.store != nil {
		for _, a := range m.accounts {
			if a == nil || a.Username == "" {
				continue
			}
			if err := m.store.SaveLocalAccount(context.Background(), a); err != nil {
				return err
			}
		}
		return nil
	}
	if m.cfg.AccountFile == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(m.cfg.AccountFile), 0700); err != nil {
		return err
	}
	out := struct {
		Accounts []*Account `json:"accounts"`
	}{Accounts: []*Account{}}
	for _, a := range m.accounts {
		out.Accounts = append(out.Accounts, a)
	}
	b, _ := json.MarshalIndent(out, "", "  ")
	return os.WriteFile(m.cfg.AccountFile, b, 0600)
}

func accountFromConfig(u config.LocalUserAuth) (*Account, error) {
	now := time.Now()
	subType := first(u.SubjectType, "human")
	subID := first(u.SubjectID, subType+":"+u.Username)
	h := strings.TrimSpace(u.PasswordHash)
	if h == "" && u.Password != "" {
		h = HashPassword(u.Password)
	}
	if h == "" {
		return nil, fmt.Errorf("password or passwordHash is required for local user %s", u.Username)
	}
	return &Account{Username: u.Username, PasswordHash: h, SubjectID: subID, SubjectType: subType, DisplayName: u.DisplayName, Email: u.Email, Organization: u.Organization, Roles: u.Roles, Permissions: u.Permissions, Namespaces: u.Namespaces, Disabled: u.Disabled, CreatedAt: now, UpdatedAt: now}, nil
}

type Provider struct {
	name    string
	manager *Manager
}

func NewProvider(name string, manager *Manager) *Provider {
	return &Provider{name: name, manager: manager}
}
func (p *Provider) Name() string { return p.name }
func (p *Provider) Authenticate(ctx context.Context, r *http.Request) (*auth.ExternalIdentity, bool, error) {
	tok := bearer(r.Header.Get("Authorization"))
	if tok == "" {
		return nil, false, nil
	}
	claims, err := p.manager.VerifyToken(tok)
	if err != nil {
		if unsafeIssuer(tok) != p.manager.cfg.Issuer {
			return nil, false, nil
		}
		return nil, false, err
	}
	return &auth.ExternalIdentity{Provider: p.name, ProviderType: "local", Issuer: stringClaim(claims, "iss"), Subject: stringClaim(claims, "sub"), SubjectType: stringClaim(claims, "typ"), Username: stringClaim(claims, "preferred_username"), Email: stringClaim(claims, "email"), Organization: stringClaim(claims, "organization"), Roles: sliceClaim(claims, "roles"), Permissions: sliceClaim(claims, "permissions"), Namespaces: sliceClaim(claims, "namespaces"), Claims: claims}, true, nil
}

func HashPassword(password string) string {
	salt := make([]byte, 16)
	_, _ = rand.Read(salt)
	dk := pbkdf2SHA256([]byte(password), salt, 200000, 32)
	return fmt.Sprintf("pbkdf2_sha256$200000$%s$%s", hex.EncodeToString(salt), hex.EncodeToString(dk))
}
func VerifyPassword(hash, password string) bool {
	parts := strings.Split(hash, "$")
	if len(parts) != 4 || parts[0] != "pbkdf2_sha256" {
		return false
	}
	iter := 200000
	fmt.Sscanf(parts[1], "%d", &iter)
	salt, err1 := hex.DecodeString(parts[2])
	want, err2 := hex.DecodeString(parts[3])
	if err1 != nil || err2 != nil {
		return false
	}
	got := pbkdf2SHA256([]byte(password), salt, iter, len(want))
	return subtle.ConstantTimeCompare(got, want) == 1
}
func pbkdf2SHA256(password, salt []byte, iter, keyLen int) []byte {
	hLen := 32
	numBlocks := (keyLen + hLen - 1) / hLen
	out := make([]byte, 0, numBlocks*hLen)
	for block := 1; block <= numBlocks; block++ {
		mac := hmac.New(sha256.New, password)
		mac.Write(salt)
		mac.Write([]byte{byte(block >> 24), byte(block >> 16), byte(block >> 8), byte(block)})
		u := mac.Sum(nil)
		t := append([]byte(nil), u...)
		for i := 1; i < iter; i++ {
			mac = hmac.New(sha256.New, password)
			mac.Write(u)
			u = mac.Sum(nil)
			for j := range t {
				t[j] ^= u[j]
			}
		}
		out = append(out, t...)
	}
	return out[:keyLen]
}
func ValidateSigningSecret(secret string) error {
	secret = strings.TrimSpace(secret)
	if secret == "" || strings.Contains(secret, "${") || strings.Contains(strings.ToUpper(secret), "CHANGE_ME") {
		return errors.New("local signingSecret must be configured to a strong random value")
	}
	if len(secret) < 32 {
		return errors.New("local signingSecret is too short; use at least 32 characters")
	}
	return nil
}

func SignJWT(secret string, claims map[string]any) (string, error) {
	if err := ValidateSigningSecret(secret); err != nil {
		return "", err
	}
	header := map[string]any{"alg": "HS256", "typ": "JWT"}
	hb, _ := json.Marshal(header)
	cb, _ := json.Marshal(claims)
	unsigned := base64.RawURLEncoding.EncodeToString(hb) + "." + base64.RawURLEncoding.EncodeToString(cb)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(unsigned))
	sig := mac.Sum(nil)
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}
func VerifyJWT(secret, issuer, token string) (map[string]any, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid local jwt")
	}
	unsigned := parts[0] + "." + parts[1]
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, err
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(unsigned))
	want := mac.Sum(nil)
	if subtle.ConstantTimeCompare(sig, want) != 1 {
		return nil, errors.New("invalid local jwt signature")
	}
	var c map[string]any
	b, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	now := time.Now().Unix()
	if exp, ok := number(c["exp"]); ok && int64(exp) < now {
		return nil, errors.New("local jwt expired")
	}
	if issuer != "" && stringClaim(c, "iss") != issuer {
		return nil, errors.New("invalid local jwt issuer")
	}
	return c, nil
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
func stringClaim(c map[string]any, k string) string {
	if s, ok := c[k].(string); ok {
		return s
	}
	return ""
}
func sliceClaim(c map[string]any, k string) []string {
	switch v := c[k].(type) {
	case []any:
		out := []string{}
		for _, x := range v {
			if s, ok := x.(string); ok {
				out = append(out, s)
			}
		}
		return out
	case []string:
		return v
	case string:
		return strings.Fields(v)
	default:
		return nil
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
func cloneAccount(a *Account) *Account {
	if a == nil {
		return nil
	}
	b := *a
	b.Roles = append([]string{}, a.Roles...)
	b.Permissions = append([]string{}, a.Permissions...)
	b.Namespaces = append([]string{}, a.Namespaces...)
	return &b
}

func unsafeIssuer(token string) string {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return ""
	}
	b, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}
	var c map[string]any
	if json.Unmarshal(b, &c) != nil {
		return ""
	}
	return stringClaim(c, "iss")
}
