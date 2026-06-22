package casbin

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	core "github.com/actionlab-ai/aisphere-hub/backend/internal/auth"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/config"
	casbinv2 "github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/persist/file-adapter"
)

type Authorizer struct {
	enforcer           *casbinv2.Enforcer
	policyStore        string
	policyFile         string
	dsn                string
	configRoleMappings []config.RoleMappingConfig
}

type closer interface{ Close() error }

func NewAuthorizer(cfg config.Config) (*Authorizer, error) {
	az := cfg.Authz
	modelPath := first(az.Model, "./configs/casbin/model.conf")
	var e *casbinv2.Enforcer
	var err error
	store := strings.ToLower(first(az.PolicyStore, "file"))
	switch store {
	case "mysql":
		adp, err := NewMySQLAdapter(cfg.Database.DSN)
		if err != nil {
			return nil, err
		}
		e, err = casbinv2.NewEnforcer(modelPath, adp)
	case "file", "csv", "":
		e, err = casbinv2.NewEnforcer(modelPath, fileadapter.NewAdapter(first(az.PolicyFile, "./configs/casbin/policy.csv")))
	default:
		return nil, fmt.Errorf("unsupported authz.policyStore %q", az.PolicyStore)
	}
	if err != nil {
		return nil, err
	}
	e.EnableAutoSave(az.AutoSave)
	if store == "mysql" && az.PolicyFile != "" {
		policies, err := e.GetPolicy()
		if err != nil {
			return nil, err
		}
		groupingPolicies, err := e.GetGroupingPolicy()
		if err != nil {
			return nil, err
		}
		if len(policies) == 0 && len(groupingPolicies) == 0 {
			if err := seedFromFile(e, modelPath, az.PolicyFile); err != nil {
				return nil, err
			}
		}
	}
	return &Authorizer{enforcer: e, policyStore: store, policyFile: az.PolicyFile, dsn: cfg.Database.DSN, configRoleMappings: cfg.Auth.RoleMappings}, nil
}

func seedFromFile(target *casbinv2.Enforcer, modelPath, policyFile string) error {
	seed, err := casbinv2.NewEnforcer(modelPath, fileadapter.NewAdapter(policyFile))
	if err != nil {
		return err
	}
	policies, err := seed.GetPolicy()
	if err != nil {
		return err
	}
	for _, p := range policies {
		if ok, err := target.AddPolicy(toInterfaces(p)...); err != nil {
			return err
		} else if !ok {
			continue
		}
	}
	groupingPolicies, err := seed.GetGroupingPolicy()
	if err != nil {
		return err
	}
	for _, g := range groupingPolicies {
		if ok, err := target.AddGroupingPolicy(toInterfaces(g)...); err != nil {
			return err
		} else if !ok {
			continue
		}
	}
	return nil
}

func toInterfaces(values []string) []interface{} {
	args := make([]interface{}, 0, len(values))
	for _, v := range values {
		args = append(args, v)
	}
	return args
}

func (a *Authorizer) Authorize(ctx context.Context, p *core.Principal, action string, res core.ResourceRef) error {
	if action == "public:read" || action == "auth:login" || action == "auth:setup" {
		return nil
	}
	if action == "auth:me" {
		if p == nil {
			return fmt.Errorf("%w: missing principal", core.ErrForbidden)
		}
		return nil
	}
	if p == nil || strings.TrimSpace(p.SubjectID) == "" {
		return fmt.Errorf("%w: missing principal", core.ErrForbidden)
	}
	obj := objectFor(action, res)
	if ok, err := a.enforcer.Enforce(p.SubjectID, obj, action); err != nil {
		return err
	} else if ok {
		return nil
	}
	// Also honor roles delivered by Casdoor/OIDC claims and role mappings even when
	// g(subject, role) has not been synced into the local Casbin policy store yet.
	for _, r := range a.mappedRolesFor(p) {
		r = normalizeRole(r)
		if r == "" {
			continue
		}
		if ok, err := a.enforcer.Enforce(r, obj, action); err != nil {
			return err
		} else if ok {
			return nil
		}
	}
	return fmt.Errorf("%w: subject=%s action=%s object=%s", core.ErrForbidden, p.SubjectID, action, obj)
}

func (a *Authorizer) mappedRolesFor(p *core.Principal) []string {
	if p == nil {
		return nil
	}
	out := make([]string, 0, len(p.Roles)+len(p.ExternalRoles)+4)
	seen := map[string]bool{}
	add := func(v string) {
		v = normalizeRole(v)
		if v == "" || seen[v] {
			return
		}
		out = append(out, v)
		seen[v] = true
	}
	for _, r := range p.Roles {
		add(r)
	}
	for _, rm := range a.configRoleMappings {
		if rm.Provider != "" && rm.Provider != p.Provider {
			continue
		}
		if rm.SubjectType != "" && rm.SubjectType != p.SubjectType {
			continue
		}
		if rm.ExternalRole != "" && !containsString(p.ExternalRoles, rm.ExternalRole) && !containsString(p.Roles, rm.ExternalRole) {
			continue
		}
		if rm.ExternalGroup != "" && !containsString(p.Groups, rm.ExternalGroup) {
			continue
		}
		for _, r := range rm.InternalRoles {
			add(r)
		}
	}
	for _, r := range a.dbMappedRoles(p) {
		add(r)
	}
	return out
}

func (a *Authorizer) dbMappedRoles(p *core.Principal) []string {
	if a == nil || strings.TrimSpace(a.dsn) == "" || p == nil {
		return nil
	}
	db, err := sql.Open("mysql", a.dsn)
	if err != nil {
		return nil
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		return nil
	}
	extRoles := append([]string{}, p.ExternalRoles...)
	extRoles = append(extRoles, p.Roles...)
	if len(extRoles) == 0 {
		return nil
	}
	out := []string{}
	for _, ext := range extRoles {
		rows, err := db.QueryContext(context.Background(), `SELECT internal_role FROM aihub_role_mapping WHERE enabled=1 AND (provider='' OR provider=?) AND external_role=?`, p.Provider, ext)
		if err != nil {
			continue
		}
		for rows.Next() {
			var r string
			if err := rows.Scan(&r); err == nil {
				out = append(out, r)
			}
		}
		_ = rows.Close()
	}
	return out
}

func containsString(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

func objectFor(action string, res core.ResourceRef) string {
	path := strings.TrimSpace(res.Path)
	if strings.HasPrefix(action, "skill:group") || strings.HasPrefix(path, "/v3/aihub/skillset") || strings.HasPrefix(path, "/v3/client/ai/groups") {
		if res.GroupName != "" {
			return "group:" + res.GroupName
		}
		return "group:*"
	}
	if strings.HasPrefix(action, "skill:proposal") || strings.Contains(path, "skill-proposals") {
		if res.ProposalID != "" {
			return "proposal:" + res.ProposalID
		}
		return "proposal:*"
	}
	if strings.Contains(action, "overlay") || strings.Contains(path, "skill-overlays") {
		return "overlay:*"
	}
	if strings.HasPrefix(action, "audit") || strings.Contains(path, "audit") {
		return "audit:*"
	}
	if strings.HasPrefix(action, "metrics") || strings.Contains(path, "metrics") {
		return "metrics:*"
	}
	if strings.HasPrefix(action, "access:") || strings.Contains(path, "/v3/admin/access") {
		return "access:*"
	}
	if strings.HasPrefix(action, "iam:") || strings.Contains(path, "/iam") {
		return "iam:*"
	}
	if strings.HasPrefix(action, "notification") {
		return "notification:*"
	}
	if res.Name != "" {
		return "skill:" + res.Name
	}
	if strings.HasPrefix(action, "skill:") {
		return "skill:*"
	}
	return "system:*"
}

func normalizeRole(role string) string {
	role = strings.TrimSpace(role)
	if role == "" {
		return ""
	}
	if strings.HasPrefix(role, "role:") {
		return role
	}
	return "role:" + role
}

func first(v, d string) string {
	if strings.TrimSpace(v) != "" {
		return v
	}
	return d
}

// Enforcer exposes read-only access for lightweight policy APIs and tests.
func (a *Authorizer) Enforcer() *casbinv2.Enforcer { return a.enforcer }

func IsWrite(method string) bool {
	return method != http.MethodGet && method != http.MethodHead && method != http.MethodOptions
}
