package casdoorremote

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	core "github.com/actionlab-ai/aisphere-hub/backend/internal/auth"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/config"
)

type Authorizer struct {
	endpoint      string
	owner         string
	permissionID  string
	modelID       string
	resourceID    string
	enforcerID    string
	clientID      string
	clientSecret  string
	subjectFormat string
	cacheTTL      time.Duration
	failClosed    bool
	httpClient    *http.Client
	cache         sync.Map
}

type cacheEntry struct {
	Allowed  bool
	ExpireAt time.Time
}

type enforceResp struct {
	Status string `json:"status"`
	Msg    string `json:"msg"`
	Data   []any  `json:"data"`
	Data2  []any  `json:"data2"`
}

func NewAuthorizer(cfg config.Config) (*Authorizer, error) {
	c := cfg.Authz.Casdoor
	endpoint := strings.TrimRight(first(c.Endpoint, firstOIDCIssuer(cfg.Auth.Providers)), "/")
	if endpoint == "" {
		return nil, errors.New("authz.casdoor.endpoint is required when authz.provider=casdoor-remote")
	}
	clientID := first(c.ClientID, firstOIDCClientID(cfg.Auth.Providers))
	clientSecret := first(c.ClientSecret, firstOIDCClientSecret(cfg.Auth.Providers))
	if clientID == "" || clientSecret == "" {
		return nil, errors.New("authz.casdoor.clientId/clientSecret are required when authz.provider=casdoor-remote")
	}
	owner := first(c.Owner, "built-in")
	permissionID := strings.TrimSpace(c.PermissionID)
	if permissionID == "" && c.Permission != "" {
		permissionID = owner + "/" + c.Permission
	}
	modelID := strings.TrimSpace(c.ModelID)
	if modelID == "" && c.Model != "" {
		modelID = owner + "/" + c.Model
	}
	if permissionID == "" && modelID == "" && c.ResourceID == "" && c.EnforcerID == "" && owner == "" {
		return nil, errors.New("one of authz.casdoor.permission/permissionId/model/modelId/resourceId/enforcerId/owner is required")
	}
	ttl := time.Duration(c.CacheTTLSeconds) * time.Second
	if ttl < 0 {
		ttl = 0
	}
	return &Authorizer{
		endpoint: endpoint, owner: owner, permissionID: permissionID, modelID: modelID, resourceID: strings.TrimSpace(c.ResourceID), enforcerID: strings.TrimSpace(c.EnforcerID),
		clientID: clientID, clientSecret: clientSecret, subjectFormat: first(c.SubjectFormat, "casdoor"), cacheTTL: ttl, failClosed: c.FailClosed,
		httpClient: &http.Client{Timeout: 8 * time.Second},
	}, nil
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
	sub := a.SubjectForPrincipal(p)
	allowed, err := a.Enforce(ctx, sub, obj, action)
	if err != nil {
		if a.failClosed {
			return fmt.Errorf("%w: casdoor enforce error: %v", core.ErrForbidden, err)
		}
		return nil
	}
	if allowed {
		return nil
	}
	return fmt.Errorf("%w: subject=%s casdoorSubject=%s action=%s object=%s", core.ErrForbidden, p.SubjectID, sub, action, obj)
}

func (a *Authorizer) Enforce(ctx context.Context, sub, obj, act string) (bool, error) {
	sub = strings.TrimSpace(sub)
	obj = strings.TrimSpace(obj)
	act = strings.TrimSpace(act)
	if sub == "" || obj == "" || act == "" {
		return false, errors.New("sub, obj and act are required")
	}
	key := sub + "\x1f" + obj + "\x1f" + act
	if a.cacheTTL > 0 {
		if v, ok := a.cache.Load(key); ok {
			ce := v.(cacheEntry)
			if time.Now().Before(ce.ExpireAt) {
				return ce.Allowed, nil
			}
			a.cache.Delete(key)
		}
	}
	allowed, err := a.enforceNoCache(ctx, sub, obj, act)
	if err != nil {
		return false, err
	}
	if a.cacheTTL > 0 {
		a.cache.Store(key, cacheEntry{Allowed: allowed, ExpireAt: time.Now().Add(a.cacheTTL)})
	}
	return allowed, nil
}

func (a *Authorizer) enforceNoCache(ctx context.Context, sub, obj, act string) (bool, error) {
	q := url.Values{}
	switch {
	case a.permissionID != "":
		q.Set("permissionId", a.permissionID)
	case a.modelID != "":
		q.Set("modelId", a.modelID)
	case a.resourceID != "":
		q.Set("resourceId", a.resourceID)
	case a.enforcerID != "":
		q.Set("enforcerId", a.enforcerID)
	default:
		q.Set("owner", a.owner)
	}
	body, _ := json.Marshal([]string{sub, obj, act})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.endpoint+"/api/enforce?"+q.Encode(), bytes.NewReader(body))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(a.clientID, a.clientSecret)
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	var out enforceResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return false, fmt.Errorf("casdoor enforce decode: %w", err)
	}
	if resp.StatusCode/100 != 2 || (out.Status != "" && out.Status != "ok") {
		msg := out.Msg
		if msg == "" {
			msg = resp.Status
		}
		return false, fmt.Errorf("casdoor enforce http=%d status=%s msg=%s", resp.StatusCode, out.Status, msg)
	}
	return responseAllowed(out.Data), nil
}

func responseAllowed(data []any) bool {
	for _, v := range data {
		switch t := v.(type) {
		case bool:
			if t {
				return true
			}
		case []any:
			for _, x := range t {
				if b, ok := x.(bool); ok && b {
					return true
				}
			}
		}
	}
	return false
}

func (a *Authorizer) SubjectForPrincipal(p *core.Principal) string {
	if p == nil {
		return ""
	}
	switch strings.ToLower(a.subjectFormat) {
	case "principal", "aihub", "subjectid":
		return p.SubjectID
	case "external", "sub":
		return first(p.ExternalSubject, p.SubjectID)
	default:
		return casdoorUserID(p)
	}
}

func casdoorUserID(p *core.Principal) string {
	org := first(p.Organization, claimString(p.Claims, "owner"), claimString(p.Claims, "organization"))
	name := first(p.Username, claimString(p.Claims, "name"), claimString(p.Claims, "preferred_username"), claimString(p.Claims, "username"))
	if org != "" && name != "" {
		return org + "/" + name
	}
	if strings.Contains(p.ExternalSubject, "/") {
		return p.ExternalSubject
	}
	return first(p.SubjectID, p.ExternalSubject)
}

func (a *Authorizer) Overview(p *core.Principal) map[string]any {
	subject := ""
	if p != nil {
		subject = a.SubjectForPrincipal(p)
	}
	return map[string]any{
		"provider": "casdoor-remote", "endpoint": a.endpoint, "owner": a.owner, "permissionId": a.permissionID, "modelId": a.modelID, "resourceId": a.resourceID, "enforcerId": a.enforcerID,
		"subjectFormat": a.subjectFormat, "resolvedSubject": subject, "cacheTTLSeconds": int(a.cacheTTL.Seconds()), "failClosed": a.failClosed,
		"principal": p, "resources": ResourceTemplates(), "quickLinks": a.QuickLinks(),
	}
}

func (a *Authorizer) QuickLinks() []map[string]string {
	owner := url.QueryEscape(a.owner)
	return []map[string]string{
		{"title": "Casdoor Users", "url": a.endpoint + "/users"},
		{"title": "Casdoor Roles", "url": a.endpoint + "/roles"},
		{"title": "Casdoor Permissions", "url": a.endpoint + "/permissions"},
		{"title": "Casdoor Models", "url": a.endpoint + "/models"},
		{"title": "Casdoor Permission: " + a.permissionID, "url": a.endpoint + "/permissions/" + owner},
	}
}

func ResourceTemplates() []map[string]string {
	return []map[string]string{
		{"area": "Skill", "object": "skill:*", "action": "skill:admin:read", "description": "后台读取 Skill 列表/详情"},
		{"area": "Skill", "object": "skill:*", "action": "skill:admin:write", "description": "上传、编辑、上下线 Skill"},
		{"area": "Skill", "object": "skill:*", "action": "skill:publish", "description": "发布/变更 stable/latest 标签"},
		{"area": "Runtime", "object": "skill:*", "action": "skill:read", "description": "运行时读取/下载 Skill"},
		{"area": "Group", "object": "group:*", "action": "skill:group:read", "description": "读取 Group 能力包"},
		{"area": "Group", "object": "group:*", "action": "skill:group:write", "description": "创建/维护 Group 和成员"},
		{"area": "Proposal", "object": "proposal:*", "action": "skill:proposal:review", "description": "审核 Agent Proposal"},
		{"area": "Proposal", "object": "proposal:*", "action": "skill:proposal:create", "description": "Agent 提交 Proposal"},
		{"area": "Access", "object": "access:*", "action": "access:admin:read", "description": "查看 AIHub 权限诊断页"},
		{"area": "Ops", "object": "notification:*", "action": "notification:read", "description": "读取通知"},
		{"area": "System", "object": "system:*", "action": "system:admin", "description": "系统级管理能力"},
	}
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

func first(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
func claimString(c map[string]any, k string) string {
	if c == nil || k == "" {
		return ""
	}
	if v, ok := c[k].(string); ok {
		return v
	}
	return ""
}

func firstOIDCIssuer(ps []config.AuthProviderConfig) string {
	for _, p := range ps {
		if strings.EqualFold(p.Type, "oidc") && p.Issuer != "" {
			return p.Issuer
		}
	}
	return ""
}
func firstOIDCClientID(ps []config.AuthProviderConfig) string {
	for _, p := range ps {
		if strings.EqualFold(p.Type, "oidc") && p.ClientID != "" {
			return p.ClientID
		}
	}
	return ""
}
func firstOIDCClientSecret(ps []config.AuthProviderConfig) string {
	for _, p := range ps {
		if strings.EqualFold(p.Type, "oidc") && p.ClientSecret != "" {
			return p.ClientSecret
		}
	}
	return ""
}
