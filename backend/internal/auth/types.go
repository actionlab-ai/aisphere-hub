package auth

import (
	"context"
	"net/http"
)

type Principal struct {
	SubjectID       string
	SubjectType     string // human / organization / agent / service / anonymous
	Organization    string
	Provider        string
	ExternalIssuer  string
	ExternalSubject string
	Username        string
	Email           string
	Groups          []string
	ExternalRoles   []string
	Roles           []string
	Permissions     []string
	Namespaces      []string
	Claims          map[string]any
}

type ExternalIdentity struct {
	Provider     string
	ProviderType string
	Issuer       string
	Subject      string
	SubjectType  string
	Username     string
	Email        string
	Organization string
	Groups       []string
	Roles        []string
	Permissions  []string
	Namespaces   []string
	Scopes       []string
	Claims       map[string]any
}

type AuthProvider interface {
	Name() string
	Authenticate(ctx context.Context, r *http.Request) (*ExternalIdentity, bool, error)
}

type SubjectMapper interface {
	Map(ctx context.Context, id *ExternalIdentity) (*Principal, error)
}

type Authorizer interface {
	Authorize(ctx context.Context, p *Principal, action string, res ResourceRef) error
}

type ResourceRef struct {
	NamespaceID string
	Type        string
	Name        string
	GroupName   string
	AgentID     string
	ToolID      string
	WorkflowID  string
	RunID       string
	ProposalID  string
	Path        string
	HTTPMethod  string
}

const PrincipalContextKey = "principal"
