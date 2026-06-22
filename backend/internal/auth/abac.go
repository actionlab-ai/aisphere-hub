package auth

import (
	"context"
	"fmt"
	"strings"

	"github.com/actionlab-ai/aisphere-hub/backend/internal/model"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/store"
)

type StoreAuthorizer struct{ st store.Backend }

func NewStoreAuthorizer(st store.Backend) *StoreAuthorizer { return &StoreAuthorizer{st: st} }
func (a *StoreAuthorizer) Authorize(ctx context.Context, p *Principal, action string, res ResourceRef) error {
	if err := NewStaticAuthorizer().Authorize(ctx, p, action, res); err == nil {
		return nil
	}
	if p == nil || a.st == nil || strings.TrimSpace(res.NamespaceID) == "" {
		return fmt.Errorf("%w: action %s", ErrForbidden, action)
	}
	ms, _, err := a.st.ListNamespaceMembers(model.NamespaceMemberQuery{NamespaceID: res.NamespaceID, SubjectID: p.SubjectID, PageNo: 1, PageSize: 10})
	if err != nil || len(ms) == 0 {
		return fmt.Errorf("%w: namespace %s", ErrForbidden, res.NamespaceID)
	}
	roles := map[string]bool{}
	for _, m := range ms {
		for _, r := range m.Roles {
			roles[strings.ToLower(strings.TrimSpace(r))] = true
		}
	}
	if roles["owner"] || roles["admin"] {
		return nil
	}
	if strings.Contains(action, ":read") || action == "skill:read" || action == "public:read" {
		if roles["viewer"] || roles["developer"] || roles["reviewer"] {
			return nil
		}
	}
	if strings.Contains(action, "proposal:review") {
		if roles["reviewer"] || roles["developer"] {
			return nil
		}
	}
	if strings.Contains(action, ":write") || strings.Contains(action, ":create") || strings.Contains(action, ":update") {
		if roles["developer"] {
			return nil
		}
	}
	return fmt.Errorf("%w: action %s", ErrForbidden, action)
}
