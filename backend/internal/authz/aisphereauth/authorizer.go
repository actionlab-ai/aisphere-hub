// Package aisphereauth is the AIHub-side Authorizer that delegates
// allow/deny decisions to aisphere-auth's /authz/check endpoint. It
// complements (does not replace) the existing static / store / casbin /
// casdoor-remote authorizers.
//
// The authorizer maps AIHub's (action, ResourceRef) tuple to the
// platform (sub, obj, act) tuple by reusing the same objectFor helper
// convention the casdoor-remote authorizer uses, so policy writers can
// treat AIHub resources identically across both backends.
package aisphereauth

import (
	"context"
	"fmt"
	"strings"

	"github.com/actionlab-ai/aisphere-auth/pkg/aisphereauth"
	authclient "github.com/actionlab-ai/aisphere-auth/pkg/aisphereauth/client"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/aisphereclient"
	core "github.com/actionlab-ai/aisphere-hub/backend/internal/auth"
)

// Authorizer implements core.Authorizer.
type Authorizer struct {
	client *aisphereclient.Client
}

// New returns an Authorizer. If client is nil the integration is
// disabled and Authorize returns nil for every non-public action so
// AIHub still functions (when paired with another authorizer such as
// static or casdoor-remote).
func New(client *aisphereclient.Client) *Authorizer {
	if client == nil {
		return nil
	}
	return &Authorizer{client: client}
}

// Authorize maps AIHub's action+resource to the AI Sphere
// (subject, object, action) triple and asks aisphere-auth to decide.
func (a *Authorizer) Authorize(ctx context.Context, p *core.Principal, action string, res core.ResourceRef) error {
	if a == nil || a.client == nil {
		// Disabled integration: never block. Another authorizer in the
		// chain (static / store / casbin) is expected to make the call.
		return nil
	}
	// AIHub semantics: these actions are always public.
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
	sub := subjectFor(p)
	app := appFor(a.client)
	obj := objectFor(app, action, res)
	platformAction := actionFor(action, res)
	resourceType, resourceID := resourceFor(action, res)
	allowed, reason, _, err := a.client.CheckDetailed(authclient.CheckRequest{Subject: sub, Principal: convertPrincipal(p, app), Object: obj, Action: platformAction, App: app, ResourceType: resourceType, ResourceID: resourceID})
	if err != nil {
		if a.client.Config().FailClosed {
			return fmt.Errorf("%w: aisphere-auth enforce error: %v", core.ErrForbidden, err)
		}
		// Fail-open path: log via reason and let the next authorizer decide.
		return nil
	}
	if allowed {
		return nil
	}
	return fmt.Errorf("%w: subject=%s action=%s object=%s reason=%s", core.ErrForbidden, sub, platformAction, obj, reason)
}

// subjectFor mirrors the casdoor-remote authorizer's "casdoor" subject
// format: prefer CasdoorSubject (org/name) when available so existing
// Casdoor policies keep working unchanged.
func subjectFor(p *core.Principal) string {
	if p == nil {
		return ""
	}
	if p.ExternalSubject != "" && strings.Contains(p.ExternalSubject, "/") {
		return p.ExternalSubject
	}
	if p.Organization != "" && p.Username != "" {
		return p.Organization + "/" + p.Username
	}
	return firstNonEmpty(p.SubjectID, p.ExternalSubject, p.Username)
}

// objectFor reuses the casdoor-remote convention so AIHub-side
// policy objects stay portable across the two authorizers.
func objectFor(app string, action string, res core.ResourceRef) string {
	if strings.TrimSpace(app) == "" {
		app = "aihub"
	}
	path := strings.TrimSpace(res.Path)
	if strings.HasPrefix(action, "runtime:") || strings.Contains(path, "/v3/aihub/runtime") {
		return app + ":runtime:*"
	}
	if strings.HasPrefix(action, "agent:") || strings.Contains(path, "/v3/aihub/agents") {
		if res.AgentID != "" {
			return app + ":agent:" + res.AgentID
		}
		return app + ":agent:*"
	}
	if strings.HasPrefix(action, "workflow:") || strings.Contains(path, "/v3/aihub/workflows") {
		if res.WorkflowID != "" {
			return app + ":workflow:" + res.WorkflowID
		}
		return app + ":workflow:*"
	}
	if strings.HasPrefix(action, "run:") || strings.Contains(path, "/v3/aihub/runs") {
		if res.RunID != "" {
			return app + ":run:" + res.RunID
		}
		return app + ":run:*"
	}
	if strings.HasPrefix(action, "skill:group") || strings.HasPrefix(path, "/v3/aihub/skillset") || strings.HasPrefix(path, "/v3/client/ai/groups") || strings.Contains(path, "skill-groups") {
		if res.GroupName != "" {
			return app + ":skillset:" + res.GroupName
		}
		return app + ":skillset:*"
	}
	if strings.HasPrefix(action, "skill:proposal") || strings.Contains(path, "skill-proposals") {
		if res.ProposalID != "" {
			return app + ":proposal:" + res.ProposalID
		}
		return app + ":proposal:*"
	}
	if strings.Contains(action, "overlay") || strings.Contains(path, "skill-overlays") {
		return app + ":workflow:*"
	}
	if strings.HasPrefix(action, "audit") || strings.Contains(path, "audit") {
		return app + ":audit:*"
	}
	if strings.HasPrefix(action, "metrics") || strings.Contains(path, "metrics") {
		return app + ":admin:*"
	}
	if strings.HasPrefix(action, "access:") || strings.Contains(path, "/v3/admin/access") {
		return app + ":admin:*"
	}
	if strings.HasPrefix(action, "iam:") || strings.Contains(path, "/iam") || strings.Contains(path, "namespaces") {
		return app + ":skillset:*"
	}
	if strings.HasPrefix(action, "notification") {
		return app + ":admin:*"
	}
	if res.Name != "" {
		return app + ":skill:" + res.Name
	}
	if strings.HasPrefix(action, "skill:") {
		return app + ":skill:*"
	}
	return app + ":admin:*"
}

func actionFor(action string, res core.ResourceRef) string {
	path := strings.ToLower(strings.TrimSpace(res.Path))
	method := strings.ToUpper(strings.TrimSpace(res.HTTPMethod))
	switch {
	case strings.Contains(path, "/download") || strings.Contains(action, "download"):
		return "download"
	case strings.Contains(path, "approve"):
		return "approve"
	case strings.Contains(path, "reject"):
		return "reject"
	case strings.Contains(path, "publish") || strings.Contains(action, "publish"):
		return "publish"
	case strings.Contains(path, "rollback") || strings.Contains(action, "rollback"):
		return "rollback"
	case strings.Contains(path, "/run") || strings.Contains(action, ":run"):
		return "run"
	case strings.Contains(path, "/cancel") || strings.Contains(action, "cancel"):
		return "cancel"
	case strings.Contains(path, "/retry") || strings.Contains(action, "retry"):
		return "retry"
	case strings.HasPrefix(action, "access:admin:read") || strings.HasPrefix(action, "skill:admin:read") || strings.HasSuffix(action, ":read") || method == "GET":
		return "read"
	case method == "POST":
		return "create"
	case method == "PUT" || method == "PATCH":
		return "update"
	case method == "DELETE":
		return "delete"
	case strings.HasPrefix(action, "access:admin:write") || strings.HasPrefix(action, "iam:") || strings.HasSuffix(action, ":write"):
		return "admin:write"
	case action == "system:admin":
		return "admin:write"
	default:
		return action
	}
}

// Overview returns a small descriptor for the Access diagnostics page.
func (a *Authorizer) Overview(p *core.Principal) map[string]any {
	cfg := a.client.Config()
	subject := ""
	if p != nil {
		subject = subjectFor(p)
	}
	return map[string]any{
		"provider":        "aisphere-auth",
		"endpoint":        cfg.Endpoint,
		"app":             appFor(a.client),
		"cookieName":      cfg.CookieName,
		"cacheTTLSeconds": cfg.CacheTTLSeconds,
		"failClosed":      cfg.FailClosed,
		"resolvedSubject": subject,
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func appFor(client *aisphereclient.Client) string {
	if client == nil || strings.TrimSpace(client.Config().App) == "" {
		return "aihub"
	}
	return strings.TrimSpace(client.Config().App)
}

func resourceFor(action string, res core.ResourceRef) (string, string) {
	path := strings.TrimSpace(res.Path)
	if strings.HasPrefix(action, "runtime:") || strings.Contains(path, "/v3/aihub/runtime") {
		return "runtime", "*"
	}
	if strings.HasPrefix(action, "agent:") || strings.Contains(path, "/v3/aihub/agents") {
		if res.AgentID != "" {
			return "agent", res.AgentID
		}
		return "agent", "*"
	}
	if strings.HasPrefix(action, "workflow:") || strings.Contains(path, "/v3/aihub/workflows") {
		if res.WorkflowID != "" {
			return "workflow", res.WorkflowID
		}
		return "workflow", "*"
	}
	if strings.HasPrefix(action, "run:") || strings.Contains(path, "/v3/aihub/runs") {
		if res.RunID != "" {
			return "run", res.RunID
		}
		return "run", "*"
	}
	if strings.HasPrefix(action, "skill:group") || strings.Contains(path, "skillset") || strings.Contains(path, "skill-groups") || strings.Contains(path, "/group") {
		if res.GroupName != "" {
			return "skillset", res.GroupName
		}
		return "skillset", "*"
	}
	if strings.HasPrefix(action, "skill:proposal") || strings.Contains(path, "skill-proposals") {
		if res.ProposalID != "" {
			return "proposal", res.ProposalID
		}
		return "proposal", "*"
	}
	if res.Name != "" {
		return "skill", res.Name
	}
	return "skill", "*"
}

func convertPrincipal(p *core.Principal, app string) *aisphereauth.Principal {
	if p == nil {
		return nil
	}
	out := &aisphereauth.Principal{
		SubjectID:    p.SubjectID,
		Username:     p.Username,
		Email:        p.Email,
		Organization: p.Organization,
		Roles:        append([]string(nil), p.Roles...),
		Groups:       append([]string(nil), p.Groups...),
		App:          app,
		Claims:       map[string]any{},
	}
	if strings.Contains(p.ExternalSubject, "/") {
		out.CasdoorSubject = p.ExternalSubject
	}
	if p.Claims != nil {
		for k, v := range p.Claims {
			out.Claims[k] = v
		}
		if v, ok := p.Claims["orgId"].(string); ok {
			out.OrgID = v
		}
		if v, ok := p.Claims["projectId"].(string); ok && v != "" {
			out.ProjectIDs = append(out.ProjectIDs, v)
		}
		if vs, ok := p.Claims["projectIds"].([]string); ok {
			out.ProjectIDs = append(out.ProjectIDs, vs...)
		}
		if raw, ok := p.Claims["projectIds"].([]any); ok {
			for _, item := range raw {
				if v, ok := item.(string); ok && v != "" {
					out.ProjectIDs = append(out.ProjectIDs, v)
				}
			}
		}
	}
	return out
}
