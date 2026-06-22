package auth

import (
	"context"
	"fmt"
	"strings"
)

type StaticAuthorizer struct{}

func NewStaticAuthorizer() *StaticAuthorizer { return &StaticAuthorizer{} }

func (a *StaticAuthorizer) Authorize(ctx context.Context, p *Principal, action string, res ResourceRef) error {
	if action == "public:read" {
		return nil
	}
	if action == "auth:me" {
		if p == nil {
			return fmt.Errorf("%w: missing principal", ErrForbidden)
		}
		return nil
	}
	if p == nil {
		return fmt.Errorf("%w: missing principal", ErrForbidden)
	}
	if !allowedNamespace(p.Namespaces, res.NamespaceID) {
		return fmt.Errorf("%w: namespace %s", ErrForbidden, res.NamespaceID)
	}
	if hasPermission(*p, action) {
		return nil
	}
	return fmt.Errorf("%w: action %s", ErrForbidden, action)
}

func hasPermission(p Principal, required string) bool {
	for _, r := range p.Roles {
		r = strings.ToLower(strings.TrimSpace(r))
		if r == "admin" || r == "owner" || r == "aihub-admin" || r == "system-admin" {
			return true
		}
	}
	for _, perm := range p.Permissions {
		perm = strings.TrimSpace(perm)
		if perm == "*" || perm == required {
			return true
		}
		if strings.HasSuffix(perm, ":*") && strings.HasPrefix(required, strings.TrimSuffix(perm, "*")) {
			return true
		}
	}
	return false
}

func allowedNamespace(allowed []string, ns string) bool {
	if ns == "" || len(allowed) == 0 {
		return true
	}
	for _, a := range allowed {
		if a == "*" || a == ns {
			return true
		}
	}
	return false
}
