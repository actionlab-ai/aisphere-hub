package authhttp

import (
	"net/http"

	core "github.com/actionlab-ai/aisphere-hub/backend/internal/auth"
	localprovider "github.com/actionlab-ai/aisphere-hub/backend/internal/auth/providers/local"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/config"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine, cfg config.AuthConfig, localMgr *localprovider.Manager) {
	h := &localHandler{cfg: cfg, local: localMgr}
	r.GET("/v3/auth/setup/status", h.setupStatus)
	r.POST("/v3/auth/setup", h.setup)
	r.POST("/v3/auth/login", h.login)
	r.POST("/v3/auth/refresh", h.refresh)
	r.GET("/v3/auth/me", h.me)
	r.GET("/v3/auth/oidc/login", h.oidcLogin)
	r.GET("/v3/auth/oidc/callback", h.oidcCallback)

	admin := r.Group("/v3/admin/iam/local-users")
	admin.GET("/list", h.listUsers)
	admin.POST("", h.saveUser)
	admin.PUT("/:username", h.saveUser)
	admin.DELETE("/:username", h.disableUser)
}

type localHandler struct {
	cfg   config.AuthConfig
	local *localprovider.Manager
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type setupRequest struct {
	Username     string `json:"username"`
	Password     string `json:"password"`
	DisplayName  string `json:"displayName"`
	Email        string `json:"email"`
	Organization string `json:"organization"`
	SetupToken   string `json:"setupToken"`
}

func (h *localHandler) setupRequired() bool {
	return h.local != nil && h.cfg.Local.SetupEnabled && !h.local.IsInitialized()
}

func (h *localHandler) setupStatus(c *gin.Context) {
	localEnabled := h.local != nil
	secretErr := ""
	secretReady := true
	if localEnabled {
		if err := localprovider.ValidateSigningSecret(h.cfg.Local.SigningSecret); err != nil {
			secretReady = false
			secretErr = err.Error()
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"localEnabled":       localEnabled,
		"setupEnabled":       h.cfg.Local.SetupEnabled,
		"setupRequired":      h.setupRequired(),
		"setupTokenRequired": h.cfg.Local.SetupToken != "",
		"signingSecretReady": secretReady,
		"signingSecretError": secretErr,
		"mode":               h.cfg.Mode,
	})
}

func (h *localHandler) setup(c *gin.Context) {
	if h.local == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "local auth is not enabled"})
		return
	}
	if !h.cfg.Local.SetupEnabled {
		c.JSON(http.StatusForbidden, gin.H{"error": "initial setup is disabled"})
		return
	}
	if h.local.IsInitialized() {
		c.JSON(http.StatusConflict, gin.H{"error": "local auth has already been initialized"})
		return
	}
	var req setupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if h.cfg.Local.SetupToken != "" && req.SetupToken != h.cfg.Local.SetupToken {
		c.JSON(http.StatusForbidden, gin.H{"error": "invalid setup token"})
		return
	}
	if err := localprovider.ValidateSigningSecret(h.cfg.Local.SigningSecret); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "local signingSecret is not ready: " + err.Error()})
		return
	}
	if req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "password is required"})
		return
	}
	acc, err := h.local.SetupInitialAdmin(config.LocalUserAuth{
		Username: req.Username, Password: req.Password, DisplayName: req.DisplayName, Email: req.Email, Organization: req.Organization,
		SubjectType: "human", Roles: []string{"admin"}, Permissions: []string{"*"}, Namespaces: []string{"*"},
	})
	if err != nil {
		status := http.StatusBadRequest
		if err == localprovider.ErrAlreadyInitialized {
			status = http.StatusConflict
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	tok, err := h.local.IssueTokens(acc)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"account": sanitizeAccount(acc), "tokens": tok}})
}

func sanitizeAccount(acc *localprovider.Account) *localprovider.Account {
	if acc == nil {
		return nil
	}
	cp := *acc
	cp.PasswordHash = ""
	return &cp
}

func (h *localHandler) login(c *gin.Context) {
	if h.local == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "local auth is not enabled"})
		return
	}
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Username == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username and password are required"})
		return
	}
	acc, err := h.local.Authenticate(req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid username or password"})
		return
	}
	tok, err := h.local.IssueTokens(acc)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, tok)
}

func (h *localHandler) refresh(c *gin.Context) {
	if h.local == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "local auth is not enabled"})
		return
	}
	var req struct {
		RefreshToken string `json:"refreshToken"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.RefreshToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "refreshToken is required"})
		return
	}
	claims, err := h.local.VerifyToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	if claims["token_type"] != "refresh" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not a refresh token"})
		return
	}
	username, _ := claims["preferred_username"].(string)
	for _, acc := range h.local.List() {
		if acc.Username == username && !acc.Disabled {
			tok, err := h.local.IssueTokens(acc)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, tok)
			return
		}
	}
	c.JSON(http.StatusUnauthorized, gin.H{"error": "account not found"})
}

func (h *localHandler) me(c *gin.Context) {
	v, ok := c.Get(core.PrincipalContextKey)
	if !ok {
		c.JSON(http.StatusOK, gin.H{"anonymous": true})
		return
	}
	c.JSON(http.StatusOK, gin.H{"principal": v})
}

func (h *localHandler) listUsers(c *gin.Context) {
	if h.local == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "local auth is not enabled"})
		return
	}
	users := h.local.List()
	for _, u := range users {
		u.PasswordHash = ""
	}
	c.JSON(http.StatusOK, gin.H{"data": users})
}

func (h *localHandler) saveUser(c *gin.Context) {
	if h.local == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "local auth is not enabled"})
		return
	}
	var req config.LocalUserAuth
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Username == "" {
		req.Username = c.Param("username")
	}
	acc, err := h.local.CreateOrUpdate(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	acc.PasswordHash = ""
	c.JSON(http.StatusOK, gin.H{"data": acc})
}

func (h *localHandler) disableUser(c *gin.Context) {
	if h.local == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "local auth is not enabled"})
		return
	}
	if err := h.local.Disable(c.Param("username")); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": "ok"})
}
