package model

import "time"

const (
	DefaultNamespace = "_global"
	DefaultApp       = "aihub"

	ScopePublic  = "PUBLIC"
	ScopePrivate = "PRIVATE"

	MetaStatusEnable  = "enable"
	MetaStatusDisable = "disable"

	VersionStatusDraft     = "draft"
	VersionStatusReviewing = "reviewing"
	VersionStatusReviewed  = "reviewed"
	VersionStatusOnline    = "online"
	VersionStatusOffline   = "offline"

	LabelLatest = "latest"
)

type Result struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type Page struct {
	PageNumber     int         `json:"pageNumber"`
	PagesAvailable int         `json:"pagesAvailable"`
	TotalCount     int         `json:"totalCount"`
	PageItems      interface{} `json:"pageItems"`
}

type SkillBase struct {
	NamespaceID      string   `json:"-"`
	Name             string   `json:"name,omitempty"`
	Description      string   `json:"description,omitempty"`
	SkillSet         string   `json:"skillSet,omitempty"`
	Groups           []string `json:"groups,omitempty"`
	Keywords         []string `json:"keywords,omitempty"`
	ModelName        string   `json:"modelName,omitempty"`
	ModelDescription string   `json:"modelDescription,omitempty"`
	MatchHint        string   `json:"matchHint,omitempty"`
	Activation       string   `json:"activation,omitempty"`
	Priority         *int     `json:"priority,omitempty"`
}

type SkillResource struct {
	Name     string                 `json:"name"`
	Type     string                 `json:"type,omitempty"`
	Content  string                 `json:"content"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type Skill struct {
	SkillBase
	SkillMD  string                    `json:"skillMd,omitempty"`
	Resource map[string]*SkillResource `json:"resource,omitempty"`
}

type SkillBasicInfo struct {
	SkillBase
	UpdateTime *int64 `json:"updateTime,omitempty"`
}

type SkillSummary struct {
	SkillBasicInfo
	App              string            `json:"app,omitempty"`
	OrgID            string            `json:"orgId,omitempty"`
	ProjectID        string            `json:"projectId,omitempty"`
	OwnerSubject     string            `json:"ownerSubject,omitempty"`
	Owner            string            `json:"owner,omitempty"`
	Enable           bool              `json:"enable"`
	BizTags          string            `json:"bizTags,omitempty"`
	From             string            `json:"from,omitempty"`
	Scope            string            `json:"scope,omitempty"`
	Labels           map[string]string `json:"labels,omitempty"`
	EditingVersion   string            `json:"editingVersion,omitempty"`
	ReviewingVersion string            `json:"reviewingVersion,omitempty"`
	OnlineCnt        int               `json:"onlineCnt"`
	DownloadCount    int64             `json:"downloadCount"`
}

type SkillVersionSummary struct {
	Version             string `json:"version"`
	Status              string `json:"status"`
	Author              string `json:"author,omitempty"`
	CommitMsg           string `json:"commitMsg,omitempty"`
	CreateTime          int64  `json:"createTime"`
	UpdateTime          int64  `json:"updateTime"`
	PublishPipelineInfo string `json:"publishPipelineInfo,omitempty"`
	DownloadCount       int64  `json:"downloadCount"`
	SHA256              string `json:"sha256,omitempty"`
	Revision            string `json:"revision,omitempty"`
	SizeBytes           int64  `json:"sizeBytes,omitempty"`
}

type SkillMeta struct {
	SkillSummary
	Versions []SkillVersionSummary `json:"versions"`
}

type BatchUploadResult struct {
	Succeeded []string     `json:"succeeded"`
	Failed    []FailedItem `json:"failed"`
}

type FailedItem struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

type SkillIndexManifest struct {
	Labels   map[string]string   `json:"labels"`
	Versions map[string][]string `json:"versions"`
}

type StorageDescriptor struct {
	Provider          string `json:"provider"`
	Bucket            string `json:"bucket,omitempty"`
	ObjectKey         string `json:"objectKey,omitempty"`
	SkillMdObjectKey  string `json:"skillMdObjectKey,omitempty"`
	ManifestObjectKey string `json:"manifestObjectKey,omitempty"`
	ContentMD5        string `json:"contentMd5,omitempty"`
	SHA256            string `json:"sha256,omitempty"`
	SizeBytes         int64  `json:"sizeBytes,omitempty"`
}

type VersionRecord struct {
	Version             string             `json:"version"`
	Status              string             `json:"status"`
	Author              string             `json:"author,omitempty"`
	CommitMsg           string             `json:"commitMsg,omitempty"`
	CreateTime          int64              `json:"createTime"`
	UpdateTime          int64              `json:"updateTime"`
	PublishPipelineInfo string             `json:"publishPipelineInfo,omitempty"`
	DownloadCount       int64              `json:"downloadCount"`
	MD5                 string             `json:"md5"`
	SHA256              string             `json:"sha256,omitempty"`
	Revision            string             `json:"revision,omitempty"`
	SizeBytes           int64              `json:"sizeBytes,omitempty"`
	Files               []string           `json:"files"`
	Storage             *StorageDescriptor `json:"storage,omitempty"`
	Skill               Skill              `json:"skill"`
}

type SkillRecord struct {
	NamespaceID      string                    `json:"-"`
	App              string                    `json:"app,omitempty"`
	OrgID            string                    `json:"orgId,omitempty"`
	ProjectID        string                    `json:"projectId,omitempty"`
	OwnerSubject     string                    `json:"ownerSubject,omitempty"`
	Name             string                    `json:"name"`
	Description      string                    `json:"description"`
	SkillSet         string                    `json:"skillSet,omitempty"`
	Groups           []string                  `json:"groups,omitempty"`
	Keywords         []string                  `json:"keywords,omitempty"`
	ModelName        string                    `json:"modelName,omitempty"`
	ModelDescription string                    `json:"modelDescription,omitempty"`
	MatchHint        string                    `json:"matchHint,omitempty"`
	Activation       string                    `json:"activation,omitempty"`
	Priority         *int                      `json:"priority,omitempty"`
	Owner            string                    `json:"owner,omitempty"`
	Status           string                    `json:"status"`
	BizTags          string                    `json:"bizTags,omitempty"`
	From             string                    `json:"from,omitempty"`
	Scope            string                    `json:"scope"`
	Labels           map[string]string         `json:"labels"`
	EditingVersion   string                    `json:"editingVersion,omitempty"`
	ReviewingVersion string                    `json:"reviewingVersion,omitempty"`
	CreateTime       int64                     `json:"createTime"`
	UpdateTime       int64                     `json:"updateTime"`
	DownloadCount    int64                     `json:"downloadCount"`
	Versions         map[string]*VersionRecord `json:"versions"`
}

func NowMillis() int64 { return time.Now().UnixMilli() }

func NewSkillRecord(namespaceID string, skill Skill, owner, from string) *SkillRecord {
	now := NowMillis()
	return &SkillRecord{
		NamespaceID:      namespaceID,
		Name:             skill.Name,
		Description:      skill.Description,
		SkillSet:         skill.SkillSet,
		Groups:           append([]string(nil), skill.Groups...),
		Keywords:         append([]string(nil), skill.Keywords...),
		ModelName:        skill.ModelName,
		ModelDescription: skill.ModelDescription,
		MatchHint:        skill.MatchHint,
		Activation:       skill.Activation,
		Priority:         skill.Priority,
		App:              DefaultApp,
		OwnerSubject:     owner,
		Owner:            owner,
		Status:           MetaStatusEnable,
		BizTags:          "[]",
		From:             from,
		Scope:            ScopePublic,
		Labels:           map[string]string{},
		CreateTime:       now,
		UpdateTime:       now,
		Versions:         map[string]*VersionRecord{},
	}
}

func (r *SkillRecord) ApplySkillMetadata(skill Skill) {
	r.Description = skill.Description
	r.SkillSet = skill.SkillSet
	r.Groups = append([]string(nil), skill.Groups...)
	r.Keywords = append([]string(nil), skill.Keywords...)
	r.ModelName = skill.ModelName
	r.ModelDescription = skill.ModelDescription
	r.MatchHint = skill.MatchHint
	r.Activation = skill.Activation
	r.Priority = skill.Priority
	r.UpdateTime = NowMillis()
}

func (r *SkillRecord) OnlineCount() int {
	cnt := 0
	for _, v := range r.Versions {
		if v.Status == VersionStatusOnline {
			cnt++
		}
	}
	return cnt
}

func (r *SkillRecord) Summary() SkillSummary {
	ut := r.UpdateTime
	return SkillSummary{
		App:          firstNonEmpty(r.App, DefaultApp),
		OrgID:        r.OrgID,
		ProjectID:    r.ProjectID,
		OwnerSubject: firstNonEmpty(r.OwnerSubject, r.Owner),
		SkillBasicInfo: SkillBasicInfo{SkillBase: SkillBase{
			NamespaceID:      r.NamespaceID,
			Name:             r.Name,
			Description:      r.Description,
			SkillSet:         r.SkillSet,
			Groups:           r.Groups,
			Keywords:         r.Keywords,
			ModelName:        r.ModelName,
			ModelDescription: r.ModelDescription,
			MatchHint:        r.MatchHint,
			Activation:       r.Activation,
			Priority:         r.Priority,
		}, UpdateTime: &ut},
		Owner:            r.Owner,
		Enable:           r.Status == MetaStatusEnable,
		BizTags:          r.BizTags,
		From:             r.From,
		Scope:            r.Scope,
		Labels:           cloneStringMap(r.Labels),
		EditingVersion:   r.EditingVersion,
		ReviewingVersion: r.ReviewingVersion,
		OnlineCnt:        r.OnlineCount(),
		DownloadCount:    r.DownloadCount,
	}
}

func (r *SkillRecord) Meta() SkillMeta {
	versions := make([]SkillVersionSummary, 0, len(r.Versions))
	for _, v := range r.Versions {
		versions = append(versions, SkillVersionSummary{
			Version:             v.Version,
			Status:              v.Status,
			Author:              v.Author,
			CommitMsg:           v.CommitMsg,
			CreateTime:          v.CreateTime,
			UpdateTime:          v.UpdateTime,
			PublishPipelineInfo: v.PublishPipelineInfo,
			DownloadCount:       v.DownloadCount,
			SHA256:              v.SHA256,
			Revision:            v.Revision,
			SizeBytes:           v.SizeBytes,
		})
	}
	return SkillMeta{SkillSummary: r.Summary(), Versions: versions}
}

func cloneStringMap(in map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range in {
		out[k] = v
	}
	return out
}

const (
	ProposalStatusSubmitted  = "submitted"
	ProposalStatusValidating = "validating"
	ProposalStatusApproved   = "approved"
	ProposalStatusRejected   = "rejected"
	ProposalStatusPromoted   = "promoted"

	OverlayStatusActive  = "active"
	OverlayStatusExpired = "expired"
)

// SkillProposal is a low-trust Agent contribution. It must not mutate runtime
// routable skill versions until an admin/pipeline approves and promotes it.
type SkillProposal struct {
	ProposalID       string                 `json:"proposalId"`
	NamespaceID      string                 `json:"-"`
	SkillName        string                 `json:"skillName"`
	BaseVersion      string                 `json:"baseVersion"`
	CandidateVersion string                 `json:"candidateVersion,omitempty"`
	ProposalType     string                 `json:"proposalType"`
	Status           string                 `json:"status"`
	Source           ProposalSource         `json:"source"`
	Reason           string                 `json:"reason,omitempty"`
	Delta            map[string]interface{} `json:"delta,omitempty"`
	Evidence         map[string]interface{} `json:"evidence,omitempty"`
	OverlayRef       string                 `json:"overlayRef,omitempty"`
	CreatedBy        string                 `json:"createdBy,omitempty"`
	CreateTime       int64                  `json:"createTime"`
	UpdateTime       int64                  `json:"updateTime"`
}

type ProposalSource struct {
	AgentID   string `json:"agentId,omitempty"`
	SessionID string `json:"sessionId,omitempty"`
	RunID     string `json:"runId,omitempty"`
	TaskID    string `json:"taskId,omitempty"`
}

type SkillOverlay struct {
	OverlayRef  string                 `json:"overlayRef"`
	NamespaceID string                 `json:"-"`
	SkillName   string                 `json:"skillName"`
	BaseVersion string                 `json:"baseVersion"`
	ProposalID  string                 `json:"proposalId"`
	Overlay     map[string]interface{} `json:"overlay"`
	Status      string                 `json:"status"`
	ExpiresAt   int64                  `json:"expiresAt,omitempty"`
	CreateTime  int64                  `json:"createTime"`
}

type ProposalValidation struct {
	ProposalID       string                 `json:"proposalId"`
	ValidationStatus string                 `json:"validationStatus"`
	Score            float64                `json:"score,omitempty"`
	CheckResult      map[string]interface{} `json:"checkResult,omitempty"`
	TestResult       map[string]interface{} `json:"testResult,omitempty"`
	CreateTime       int64                  `json:"createTime"`
}

type ProposalQuery struct {
	NamespaceID string
	SkillName   string
	Status      string
	PageNo      int
	PageSize    int
}

type ProposalApproveOptions struct {
	TargetVersion string `json:"targetVersion,omitempty"`
	Label         string `json:"label,omitempty"` // latest/gray/stable/custom; empty means no label update
	Online        bool   `json:"online,omitempty"`
	Publish       bool   `json:"publish,omitempty"`
	Reviewer      string `json:"reviewer,omitempty"`
	Comment       string `json:"comment,omitempty"`
}

// SkillGroup is a first-class grouping resource for managing a related set of
// skills, for example a novel-writing skill suite or an ops-remediation pack.
// It is intentionally separated from SkillBase.Groups: SkillBase.Groups is a
// tag-like metadata field, while SkillGroup has lifecycle, labels and members.
type SkillGroup struct {
	NamespaceID   string                 `json:"-"`
	Name          string                 `json:"name"`
	DisplayName   string                 `json:"displayName,omitempty"`
	Description   string                 `json:"description,omitempty"`
	Owner         string                 `json:"owner,omitempty"`
	Scope         string                 `json:"scope,omitempty"`
	Labels        map[string]string      `json:"labels,omitempty"` // stable/latest/gray -> group manifest version
	Members       []SkillGroupMember     `json:"members"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	CreateTime    int64                  `json:"createTime"`
	UpdateTime    int64                  `json:"updateTime"`
	DownloadCount int64                  `json:"downloadCount"`
}

type SkillGroupMember struct {
	SkillName string `json:"skillName"`
	Version   string `json:"version,omitempty"`
	Label     string `json:"label,omitempty"`
	Required  bool   `json:"required"`
	Order     int    `json:"order"`
}

type SkillGroupManifest struct {
	NamespaceID string             `json:"-"`
	Name        string             `json:"name"`
	Version     string             `json:"version,omitempty"`
	Label       string             `json:"label,omitempty"`
	Members     []ResolvedSkillRef `json:"members"`
	UpdateTime  int64              `json:"updateTime"`
}

type ResolvedSkillRef struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Label       string `json:"label,omitempty"`
	MD5         string `json:"md5,omitempty"`
	DownloadURL string `json:"downloadUrl,omitempty"`
	Required    bool   `json:"required"`
	Order       int    `json:"order"`
}

func NewSkillGroup(namespaceID, name, displayName, description, owner, scope string) *SkillGroup {
	now := NowMillis()
	if scope == "" {
		scope = ScopePrivate
	}
	return &SkillGroup{NamespaceID: namespaceID, Name: name, DisplayName: displayName, Description: description, Owner: owner, Scope: scope, Labels: map[string]string{}, Members: []SkillGroupMember{}, Metadata: map[string]interface{}{}, CreateTime: now, UpdateTime: now}
}

// SkillSet is the canonical AIHub collection model. The storage table is still
// ai_resource_group in this transition branch, but API and UI should expose
// SkillSet instead of business group.
type SkillSet = SkillGroup
type SkillSetMember = SkillGroupMember
type SkillSetManifest = SkillGroupManifest

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

type SkillFileInfo struct {
	Path   string `json:"path"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	Size   int64  `json:"size"`
	Binary bool   `json:"binary"`
}

type SkillVersionFileList struct {
	NamespaceID string          `json:"-"`
	SkillName   string          `json:"skillName"`
	Version     string          `json:"version"`
	Files       []SkillFileInfo `json:"files"`
}

type SkillVersionFileContent struct {
	NamespaceID string `json:"-"`
	SkillName   string `json:"skillName"`
	Version     string `json:"version"`
	Path        string `json:"path"`
	Content     string `json:"content"`
	Binary      bool   `json:"binary"`
}

type SkillVersionCompare struct {
	NamespaceID   string          `json:"-"`
	SkillName     string          `json:"skillName"`
	BaseVersion   string          `json:"baseVersion"`
	TargetVersion string          `json:"targetVersion"`
	BaseSkillMD   string          `json:"baseSkillMd"`
	TargetSkillMD string          `json:"targetSkillMd"`
	BaseFiles     []SkillFileInfo `json:"baseFiles"`
	TargetFiles   []SkillFileInfo `json:"targetFiles"`
}
