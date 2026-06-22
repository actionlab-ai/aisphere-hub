package casbin

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	core "github.com/actionlab-ai/aisphere-hub/backend/internal/auth"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
)

type PolicyRule struct {
	ID      string   `json:"id"`
	PType   string   `json:"ptype"`
	Subject string   `json:"subject"`
	Object  string   `json:"object,omitempty"`
	Action  string   `json:"action,omitempty"`
	Effect  string   `json:"effect,omitempty"`
	Role    string   `json:"role,omitempty"`
	Raw     []string `json:"raw"`
}

type policyRequest struct {
	PType   string `json:"ptype"`
	Subject string `json:"subject"`
	Object  string `json:"object"`
	Action  string `json:"action"`
	Effect  string `json:"effect"`
	Role    string `json:"role"`
}

type evaluateRequest struct {
	Subject string `json:"subject"`
	Object  string `json:"object"`
	Action  string `json:"action"`
}

type RoleMapping struct {
	ID           int64  `json:"id,omitempty"`
	Provider     string `json:"provider"`
	ExternalRole string `json:"externalRole"`
	InternalRole string `json:"internalRole"`
	Source       string `json:"source"`
	Enabled      bool   `json:"enabled"`
	Description  string `json:"description,omitempty"`
	CreatedAt    string `json:"createdAt,omitempty"`
	UpdatedAt    string `json:"updatedAt,omitempty"`
}

// RegisterAccessRoutes exposes a platformized Casbin admin API.
// Routes are protected by the global middleware: GET/evaluate require access:admin:read; writes require access:admin:write.
func RegisterAccessRoutes(r *gin.Engine, az *Authorizer) {
	h := &accessHandler{az: az}
	g := r.Group("/v3/admin/access")
	g.GET("/overview", h.overview)
	g.GET("/policies", h.listPolicies)
	g.POST("/policies", h.addPolicy)
	g.POST("/policies/remove", h.removePolicy)
	g.GET("/role-bindings", h.listRoleBindings)
	g.POST("/role-bindings", h.addRoleBinding)
	g.DELETE("/role-bindings", h.removeRoleBinding)
	g.POST("/evaluate", h.evaluate)
	g.POST("/reload", h.reload)
	g.GET("/role-mappings", h.listRoleMappings)
	g.POST("/role-mappings", h.saveRoleMapping)
	g.DELETE("/role-mappings/:id", h.deleteRoleMapping)
}

type accessHandler struct{ az *Authorizer }

func (h *accessHandler) overview(c *gin.Context) {
	policies, _ := h.az.enforcer.GetPolicy()
	groups, _ := h.az.enforcer.GetGroupingPolicy()
	principal := currentPrincipal(c)
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"policyStore": h.az.policyStore,
		"policyFile":  h.az.policyFile,
		"policies":    len(policies),
		"bindings":    len(groups),
		"principal":   principal,
		"roles":       rolesForPrincipal(h.az, principal),
		"menus":       defaultMenus(),
	}})
}

func (h *accessHandler) listPolicies(c *gin.Context) {
	rules := []PolicyRule{}
	policies, err := h.az.enforcer.GetPolicy()
	if err != nil {
		fail(c, err)
		return
	}
	for _, p := range policies {
		rules = append(rules, toPolicyRule("p", p))
	}
	rules = filterRules(rules, c)
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"items": rules, "total": len(rules)}})
}

func (h *accessHandler) listRoleBindings(c *gin.Context) {
	rules := []PolicyRule{}
	groups, err := h.az.enforcer.GetGroupingPolicy()
	if err != nil {
		fail(c, err)
		return
	}
	for _, g := range groups {
		rules = append(rules, toPolicyRule("g", g))
	}
	rules = filterRules(rules, c)
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"items": rules, "total": len(rules)}})
}

func (h *accessHandler) addPolicy(c *gin.Context) {
	var req policyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		failBad(c, err.Error())
		return
	}
	ptype := strings.ToLower(strings.TrimSpace(req.PType))
	if ptype == "" {
		ptype = "p"
	}
	switch ptype {
	case "p":
		if req.Subject == "" || req.Object == "" || req.Action == "" {
			failBad(c, "subject, object and action are required")
			return
		}
		eff := firstNonEmpty(req.Effect, "allow")
		ok, err := h.az.enforcer.AddPolicy(req.Subject, req.Object, req.Action, eff)
		if err != nil {
			fail(c, err)
			return
		}
		if err := h.az.persist(); err != nil {
			fail(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"added": ok}})
	case "g":
		if req.Subject == "" || req.Role == "" {
			failBad(c, "subject and role are required")
			return
		}
		ok, err := h.az.enforcer.AddGroupingPolicy(req.Subject, normalizeRole(req.Role))
		if err != nil {
			fail(c, err)
			return
		}
		if err := h.az.persist(); err != nil {
			fail(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"added": ok}})
	default:
		failBad(c, "ptype must be p or g")
	}
}

func (h *accessHandler) addRoleBinding(c *gin.Context) {
	var req policyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		failBad(c, err.Error())
		return
	}
	if req.Subject == "" || req.Role == "" {
		failBad(c, "subject and role are required")
		return
	}
	ok, err := h.az.enforcer.AddGroupingPolicy(req.Subject, normalizeRole(req.Role))
	if err != nil {
		fail(c, err)
		return
	}
	if err := h.az.persist(); err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"added": ok}})
}

func (h *accessHandler) removePolicy(c *gin.Context) {
	var req policyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		failBad(c, err.Error())
		return
	}
	ptype := strings.ToLower(strings.TrimSpace(req.PType))
	if ptype == "" {
		ptype = "p"
	}
	var ok bool
	var err error
	switch ptype {
	case "p":
		eff := firstNonEmpty(req.Effect, "allow")
		ok, err = h.az.enforcer.RemovePolicy(req.Subject, req.Object, req.Action, eff)
	case "g":
		ok, err = h.az.enforcer.RemoveGroupingPolicy(req.Subject, normalizeRole(req.Role))
	default:
		failBad(c, "ptype must be p or g")
		return
	}
	if err != nil {
		fail(c, err)
		return
	}
	if err := h.az.persist(); err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"removed": ok}})
}

func (h *accessHandler) removeRoleBinding(c *gin.Context) {
	req := policyRequest{Subject: c.Query("subject"), Role: c.Query("role")}
	if req.Subject == "" || req.Role == "" {
		_ = c.ShouldBindJSON(&req)
	}
	if req.Subject == "" || req.Role == "" {
		failBad(c, "subject and role are required")
		return
	}
	ok, err := h.az.enforcer.RemoveGroupingPolicy(req.Subject, normalizeRole(req.Role))
	if err != nil {
		fail(c, err)
		return
	}
	if err := h.az.persist(); err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"removed": ok}})
}

func (h *accessHandler) evaluate(c *gin.Context) {
	var req evaluateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		failBad(c, err.Error())
		return
	}
	if req.Subject == "" || req.Object == "" || req.Action == "" {
		failBad(c, "subject, object and action are required")
		return
	}
	allowed, err := h.az.enforcer.Enforce(req.Subject, req.Object, req.Action)
	if err != nil {
		fail(c, err)
		return
	}
	roles, _ := h.az.enforcer.GetRolesForUser(req.Subject)
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"allowed": allowed, "subject": req.Subject, "object": req.Object, "action": req.Action, "roles": roles}})
}

func (h *accessHandler) reload(c *gin.Context) {
	if err := h.az.enforcer.LoadPolicy(); err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"reloaded": true}})
}

func (h *accessHandler) listRoleMappings(c *gin.Context) {
	items, err := h.az.ListRoleMappings(c.Request.Context())
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"items": items, "total": len(items)}})
}

func (h *accessHandler) saveRoleMapping(c *gin.Context) {
	var req RoleMapping
	if err := c.ShouldBindJSON(&req); err != nil {
		failBad(c, err.Error())
		return
	}
	if req.ExternalRole == "" || req.InternalRole == "" {
		failBad(c, "externalRole and internalRole are required")
		return
	}
	if req.Provider == "" {
		req.Provider = "casdoor"
	}
	if req.Source == "" {
		req.Source = "db"
	}
	req.Enabled = true
	if err := h.az.SaveRoleMapping(c.Request.Context(), req); err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": req})
}

func (h *accessHandler) deleteRoleMapping(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id <= 0 {
		failBad(c, "invalid id")
		return
	}
	if err := h.az.DeleteRoleMapping(c.Request.Context(), id); err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"deleted": true}})
}

func toPolicyRule(ptype string, vals []string) PolicyRule {
	r := PolicyRule{PType: ptype, Raw: vals, ID: stableRuleID(ptype, vals)}
	if len(vals) > 0 {
		r.Subject = vals[0]
	}
	if ptype == "g" {
		if len(vals) > 1 {
			r.Role = vals[1]
		}
		return r
	}
	if len(vals) > 1 {
		r.Object = vals[1]
	}
	if len(vals) > 2 {
		r.Action = vals[2]
	}
	if len(vals) > 3 {
		r.Effect = vals[3]
	}
	return r
}

func filterRules(rules []PolicyRule, c *gin.Context) []PolicyRule {
	ptype := strings.TrimSpace(c.Query("ptype"))
	subject := strings.TrimSpace(c.Query("subject"))
	object := strings.TrimSpace(c.Query("object"))
	action := strings.TrimSpace(c.Query("action"))
	out := make([]PolicyRule, 0, len(rules))
	for _, r := range rules {
		if ptype != "" && r.PType != ptype {
			continue
		}
		if subject != "" && !strings.Contains(r.Subject, subject) {
			continue
		}
		if object != "" && !strings.Contains(r.Object, object) {
			continue
		}
		if action != "" && !strings.Contains(r.Action, action) {
			continue
		}
		out = append(out, r)
	}
	return out
}

func stableRuleID(ptype string, vals []string) string {
	parts := append([]string{ptype}, vals...)
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x1f")))
	return hex.EncodeToString(sum[:])
}

func currentPrincipal(c *gin.Context) *core.Principal {
	v, ok := c.Get(core.PrincipalContextKey)
	if !ok {
		return nil
	}
	if p, ok := v.(core.Principal); ok {
		return &p
	}
	if p, ok := v.(*core.Principal); ok {
		return p
	}
	return nil
}

func rolesForPrincipal(az *Authorizer, p *core.Principal) []string {
	if p == nil {
		return nil
	}
	roles := []string{}
	seen := map[string]bool{}
	add := func(r string) {
		r = normalizeRole(r)
		if r != "" && !seen[r] {
			roles = append(roles, r)
			seen[r] = true
		}
	}
	for _, r := range p.Roles {
		add(r)
	}
	for _, r := range az.mappedRolesFor(p) {
		add(r)
	}
	dbRoles, _ := az.enforcer.GetRolesForUser(p.SubjectID)
	for _, r := range dbRoles {
		add(r)
	}
	return roles
}

func defaultMenus() []gin.H {
	return []gin.H{
		{"key": "skills", "title": "Skills", "object": "skill:*", "action": "skill:admin:read"},
		{"key": "groups", "title": "Groups", "object": "group:*", "action": "skill:group:read"},
		{"key": "proposals", "title": "Proposals", "object": "proposal:*", "action": "skill:proposal:review"},
		{"key": "access", "title": "Access", "object": "access:*", "action": "access:admin:read"},
		{"key": "ops", "title": "Ops", "object": "system:*", "action": "system:admin"},
	}
}

func fail(c *gin.Context, err error) {
	c.JSON(http.StatusInternalServerError, gin.H{"error": "access_error", "message": err.Error()})
}
func failBad(c *gin.Context, msg string) {
	c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": msg})
}
func firstNonEmpty(v, d string) string {
	if strings.TrimSpace(v) != "" {
		return v
	}
	return d
}

func (a *Authorizer) persist() error {
	if a == nil || a.enforcer == nil {
		return nil
	}
	if strings.EqualFold(a.policyStore, "file") || strings.EqualFold(a.policyStore, "csv") || strings.EqualFold(a.policyStore, "mysql") {
		return a.enforcer.SavePolicy()
	}
	return nil
}

func (a *Authorizer) openDB() (*sql.DB, error) {
	if a == nil || strings.TrimSpace(a.dsn) == "" {
		return nil, fmt.Errorf("database dsn is not configured")
	}
	db, err := sql.Open("mysql", a.dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func (a *Authorizer) ListRoleMappings(ctx context.Context) ([]RoleMapping, error) {
	items := []RoleMapping{}
	for _, rm := range a.configRoleMappings {
		for _, ir := range rm.InternalRoles {
			items = append(items, RoleMapping{Provider: firstNonEmpty(rm.Provider, "casdoor"), ExternalRole: rm.ExternalRole, InternalRole: normalizeRole(ir), Source: "config", Enabled: true, Description: "from auth.roleMappings"})
		}
	}
	db, err := a.openDB()
	if err != nil {
		return items, nil
	}
	defer db.Close()
	rows, err := db.QueryContext(ctx, `SELECT id,provider,external_role,internal_role,source,enabled,description,created_at,updated_at FROM aihub_role_mapping ORDER BY id DESC`)
	if err != nil {
		return items, nil
	}
	defer rows.Close()
	for rows.Next() {
		var it RoleMapping
		var enabled int
		var desc sql.NullString
		var ca, ua time.Time
		if err := rows.Scan(&it.ID, &it.Provider, &it.ExternalRole, &it.InternalRole, &it.Source, &enabled, &desc, &ca, &ua); err != nil {
			return items, err
		}
		it.Enabled = enabled == 1
		it.Description = desc.String
		it.CreatedAt = ca.Format(time.RFC3339)
		it.UpdatedAt = ua.Format(time.RFC3339)
		items = append(items, it)
	}
	return items, rows.Err()
}

func (a *Authorizer) SaveRoleMapping(ctx context.Context, m RoleMapping) error {
	db, err := a.openDB()
	if err != nil {
		return err
	}
	defer db.Close()
	if m.Provider == "" {
		m.Provider = "casdoor"
	}
	if m.Source == "" {
		m.Source = "db"
	}
	_, err = db.ExecContext(ctx, `INSERT INTO aihub_role_mapping(provider,external_role,internal_role,source,enabled,description) VALUES(?,?,?,?,?,?) ON DUPLICATE KEY UPDATE internal_role=VALUES(internal_role), enabled=VALUES(enabled), description=VALUES(description), updated_at=CURRENT_TIMESTAMP`, m.Provider, m.ExternalRole, normalizeRole(m.InternalRole), m.Source, boolInt(m.Enabled), m.Description)
	return err
}

func (a *Authorizer) DeleteRoleMapping(ctx context.Context, id int64) error {
	db, err := a.openDB()
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.ExecContext(ctx, `DELETE FROM aihub_role_mapping WHERE id=?`, id)
	return err
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
