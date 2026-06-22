package model

const (
	AccessRoleOwner         = "owner"
	AccessRoleAdmin         = "admin"
	AccessRoleDeveloper     = "developer"
	AccessRoleReviewer      = "reviewer"
	AccessRoleViewer        = "viewer"
	SubscriptionTargetSkill = "skill"
	SubscriptionTargetGroup = "group"
)

// NamespaceInfo is retained for IAM/access-space compatibility. It is not a skill grouping dimension.
type NamespaceInfo struct {
	NamespaceID string                 `json:"namespaceId"`
	DisplayName string                 `json:"displayName,omitempty"`
	Description string                 `json:"description,omitempty"`
	Owner       string                 `json:"owner,omitempty"`
	Visibility  string                 `json:"visibility,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreateTime  int64                  `json:"createTime"`
	UpdateTime  int64                  `json:"updateTime"`
}

// NamespaceMember is retained as Access Space membership for RBAC/ABAC only.
type NamespaceMember struct {
	NamespaceID string   `json:"namespaceId"`
	SubjectID   string   `json:"subjectId"`
	SubjectType string   `json:"subjectType"`
	DisplayName string   `json:"displayName,omitempty"`
	Roles       []string `json:"roles"`
	CreateTime  int64    `json:"createTime"`
	UpdateTime  int64    `json:"updateTime"`
}

type NamespaceMemberQuery struct {
	NamespaceID string
	SubjectID   string
	PageNo      int
	PageSize    int
}

type SkillSocialStats struct {
	NamespaceID    string  `json:"-"`
	SkillName      string  `json:"skillName"`
	Stars          int64   `json:"stars"`
	RatingAverage  float64 `json:"ratingAverage"`
	RatingCount    int64   `json:"ratingCount"`
	Subscribers    int64   `json:"subscribers"`
	MyStarred      bool    `json:"myStarred,omitempty"`
	MySubscribed   bool    `json:"mySubscribed,omitempty"`
	MyRating       int     `json:"myRating,omitempty"`
	DownloadCount  int64   `json:"downloadCount,omitempty"`
	ProposalCount  int64   `json:"proposalCount,omitempty"`
	GovernanceOpen int64   `json:"governanceOpen,omitempty"`
}

type RatingRecord struct {
	NamespaceID string `json:"-"`
	SkillName   string `json:"skillName"`
	SubjectID   string `json:"subjectId"`
	Rating      int    `json:"rating"`
	Comment     string `json:"comment,omitempty"`
	CreateTime  int64  `json:"createTime"`
	UpdateTime  int64  `json:"updateTime"`
}

type SubscriptionRecord struct {
	NamespaceID string `json:"-"`
	TargetType  string `json:"targetType"`
	TargetName  string `json:"targetName"`
	SubjectID   string `json:"subjectId"`
	CreateTime  int64  `json:"createTime"`
}

type AuditLog struct {
	ID           string                 `json:"id"`
	NamespaceID  string                 `json:"accessSpaceId,omitempty"`
	ResourceType string                 `json:"resourceType,omitempty"`
	ResourceName string                 `json:"resourceName,omitempty"`
	Version      string                 `json:"version,omitempty"`
	Action       string                 `json:"action"`
	Operator     string                 `json:"operator,omitempty"`
	Detail       map[string]interface{} `json:"detail,omitempty"`
	RequestID    string                 `json:"requestId,omitempty"`
	CreateTime   int64                  `json:"createTime"`
}

type AuditQuery struct {
	NamespaceID  string
	ResourceType string
	ResourceName string
	Action       string
	Operator     string
	PageNo       int
	PageSize     int
}

type TokenInfo struct {
	KeyID       string   `json:"keyId"`
	Name        string   `json:"name"`
	SubjectID   string   `json:"subjectId"`
	SubjectType string   `json:"subjectType"`
	Roles       []string `json:"roles,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
	Namespaces  []string `json:"namespaces,omitempty"`
	Status      string   `json:"status"`
	ExpiresAt   int64    `json:"expiresAt,omitempty"`
	LastUsedAt  int64    `json:"lastUsedAt,omitempty"`
	CreateTime  int64    `json:"createTime"`
	Token       string   `json:"token,omitempty"` // only returned when created
	TokenHash   string   `json:"-"`
}

type Notification struct {
	ID          string                 `json:"id"`
	NamespaceID string                 `json:"-"`
	SubjectID   string                 `json:"subjectId"`
	TargetType  string                 `json:"targetType"`
	TargetName  string                 `json:"targetName"`
	EventType   string                 `json:"eventType"`
	Title       string                 `json:"title"`
	Message     string                 `json:"message"`
	Payload     map[string]interface{} `json:"payload,omitempty"`
	Read        bool                   `json:"read"`
	CreateTime  int64                  `json:"createTime"`
}

type NotificationQuery struct {
	SubjectID  string
	UnreadOnly bool
	PageNo     int
	PageSize   int
}

type TokenCreateRequest struct {
	Name        string   `json:"name"`
	SubjectID   string   `json:"subjectId"`
	SubjectType string   `json:"subjectType"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
	Namespaces  []string `json:"namespaces"`
	ExpiresAt   int64    `json:"expiresAt,omitempty"`
}

type IdempotencyRecord struct {
	Key          string `json:"key"`
	Method       string `json:"method"`
	Path         string `json:"path"`
	RequestHash  string `json:"requestHash"`
	StatusCode   int    `json:"statusCode"`
	ResponseBody string `json:"responseBody"`
	CreateTime   int64  `json:"createTime"`
	ExpiresAt    int64  `json:"expiresAt"`
}

type MetricsSnapshot struct {
	UptimeSeconds int64            `json:"uptimeSeconds"`
	RequestsTotal int64            `json:"requestsTotal"`
	ErrorsTotal   int64            `json:"errorsTotal"`
	ByPath        map[string]int64 `json:"byPath"`
	ByStatus      map[string]int64 `json:"byStatus"`
	Skills        int64            `json:"skills"`
	Groups        int64            `json:"groups"`
	Proposals     int64            `json:"proposals"`
}
