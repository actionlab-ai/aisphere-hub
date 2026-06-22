package store

import "github.com/actionlab-ai/aisphere-hub/backend/internal/model"

type Backend interface {
	WithWrite(fn func() error) error
	WithRead(fn func() error) error
	Load(namespaceID, name string) (*model.SkillRecord, error)
	Save(rec *model.SkillRecord) error
	Delete(namespaceID, name string) error
	List(namespaceID string) ([]*model.SkillRecord, error)
	LoadAgent(namespaceID, id string) (*model.AgentRecord, error)
	SaveAgent(rec *model.AgentRecord) error
	DeleteAgent(namespaceID, id string) error
	ListAgents(namespaceID string) ([]*model.AgentRecord, error)
	LoadTool(namespaceID, id string) (*model.ToolRecord, error)
	SaveTool(rec *model.ToolRecord) error
	DeleteTool(namespaceID, id string) error
	ListTools(namespaceID string) ([]*model.ToolRecord, error)
	AppendToolFailure(f *model.ToolFailureRecord) error
	ListToolFailures(q model.ToolFailureQuery) ([]*model.ToolFailureRecord, int64, error)
	LoadGroup(namespaceID, name string) (*model.SkillGroup, error)
	SaveGroup(g *model.SkillGroup) error
	DeleteGroup(namespaceID, name string) error
	ListGroups(namespaceID string) ([]*model.SkillGroup, error)
	SaveProposal(p *model.SkillProposal) error
	LoadProposal(proposalID string) (*model.SkillProposal, error)
	ListProposals(q model.ProposalQuery) ([]*model.SkillProposal, int64, error)
	SaveOverlay(o *model.SkillOverlay) error
	LoadOverlay(overlayRef string) (*model.SkillOverlay, error)
	SaveProposalValidation(v *model.ProposalValidation) error
	ListProposalValidations(proposalID string) ([]*model.ProposalValidation, error)
	SaveNamespace(ns *model.NamespaceInfo) error
	LoadNamespace(namespaceID string) (*model.NamespaceInfo, error)
	ListNamespaces() ([]*model.NamespaceInfo, error)
	SaveNamespaceMember(m *model.NamespaceMember) error
	DeleteNamespaceMember(namespaceID, subjectID string) error
	ListNamespaceMembers(q model.NamespaceMemberQuery) ([]*model.NamespaceMember, int64, error)
	SetStar(namespaceID, skillName, subjectID string, starred bool) error
	SetRating(r *model.RatingRecord) error
	SetSubscription(namespaceID, targetType, targetName, subjectID string, subscribed bool) error
	ListSubscribers(namespaceID, targetType, targetName string) ([]string, error)
	GetSkillSocialStats(namespaceID, skillName, subjectID string) (*model.SkillSocialStats, error)
	AppendAudit(log *model.AuditLog) error
	ListAuditLogs(q model.AuditQuery) ([]*model.AuditLog, int64, error)
	SaveToken(t *model.TokenInfo) error
	DeleteToken(keyID string) error
	ListTokens(subjectID string) ([]*model.TokenInfo, error)
	FindActiveTokenByHash(tokenHash string) (*model.TokenInfo, error)
	AppendNotification(n *model.Notification) error
	ListNotifications(q model.NotificationQuery) ([]*model.Notification, int64, error)
	MarkNotificationRead(subjectID, notificationID string) error
	SaveIdempotency(r *model.IdempotencyRecord) error
	LoadIdempotency(key string) (*model.IdempotencyRecord, error)
	AppendCatalogEvent(e *model.CatalogEvent) error

	LoadSandboxProfile(namespaceID, id string) (*model.SandboxProfile, error)
	SaveSandboxProfile(p *model.SandboxProfile) error
	DeleteSandboxProfile(namespaceID, id string) error
	ListSandboxProfiles(namespaceID string) ([]*model.SandboxProfile, error)
	LoadModelProfile(namespaceID, id string) (*model.ModelProfile, error)
	SaveModelProfile(p *model.ModelProfile) error
	DeleteModelProfile(namespaceID, id string) error
	ListModelProfiles(namespaceID string) ([]*model.ModelProfile, error)
	LoadSandboxPolicy(namespaceID, id string) (*model.SandboxPolicy, error)
	SaveSandboxPolicy(p *model.SandboxPolicy) error
	DeleteSandboxPolicy(namespaceID, id string) error
	ListSandboxPolicies(namespaceID string) ([]*model.SandboxPolicy, error)
	ListCatalogEvents(q model.CatalogEventQuery) ([]*model.CatalogEvent, int64, error)
}

var _ Backend = (*Store)(nil)
