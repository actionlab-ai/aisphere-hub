package auth

import (
	"context"
	"fmt"
	"strings"

	"github.com/actionlab-ai/aisphere-hub/backend/internal/config"
)

type StaticMapper struct{ mappings []config.RoleMappingConfig }

func NewStaticMapper(m []config.RoleMappingConfig) *StaticMapper { return &StaticMapper{mappings: m} }

func (m *StaticMapper) Map(ctx context.Context, id *ExternalIdentity) (*Principal, error) {
	if id == nil || strings.TrimSpace(id.Subject) == "" {
		return nil, fmt.Errorf("empty external identity")
	}
	subjectType := firstNonEmptyAuth(id.SubjectType, inferSubjectType(id))
	subjectID := id.Subject
	if !strings.Contains(subjectID, ":") {
		subjectID = subjectType + ":" + subjectID
	}
	p := &Principal{SubjectID: subjectID, SubjectType: subjectType, Organization: id.Organization, Provider: id.Provider, ExternalIssuer: id.Issuer, ExternalSubject: id.Subject, Username: id.Username, Email: id.Email, Groups: clone(id.Groups), ExternalRoles: clone(id.Roles), Roles: clone(id.Roles), Permissions: clone(id.Permissions), Namespaces: clone(id.Namespaces), Claims: id.Claims}
	for _, rm := range m.mappings {
		if rm.Provider != "" && rm.Provider != id.Provider {
			continue
		}
		if rm.SubjectType != "" && rm.SubjectType != subjectType {
			continue
		}
		match := false
		if rm.ExternalGroup != "" && contains(id.Groups, rm.ExternalGroup) {
			match = true
		}
		if rm.ExternalRole != "" && contains(id.Roles, rm.ExternalRole) {
			match = true
		}
		if rm.ExternalGroup == "" && rm.ExternalRole == "" {
			match = true
		}
		if !match {
			continue
		}
		p.Roles = appendUnique(p.Roles, rm.InternalRoles...)
		p.Permissions = appendUnique(p.Permissions, rm.Permissions...)
		p.Namespaces = appendUnique(p.Namespaces, rm.Namespaces...)
	}
	return p, nil
}

func inferSubjectType(id *ExternalIdentity) string {
	if strings.HasPrefix(id.Subject, "agent:") {
		return "agent"
	}
	if strings.HasPrefix(id.Subject, "service:") {
		return "service"
	}
	if strings.HasPrefix(id.Subject, "org:") {
		return "organization"
	}
	return "human"
}
func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}
func clone(xs []string) []string { out := make([]string, 0, len(xs)); return append(out, xs...) }
func appendUnique(xs []string, ys ...string) []string {
	seen := map[string]bool{}
	for _, x := range xs {
		seen[x] = true
	}
	for _, y := range ys {
		if y == "" || seen[y] {
			continue
		}
		xs = append(xs, y)
		seen[y] = true
	}
	return xs
}
func firstNonEmptyAuth(v, d string) string {
	if strings.TrimSpace(v) != "" {
		return v
	}
	return d
}
