package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/actionlab-ai/aisphere-auth/pkg/aisphereauth"
	authclient "github.com/actionlab-ai/aisphere-auth/pkg/aisphereauth/client"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/auth"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/httputil"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/model"
	"github.com/gin-gonic/gin"
)

type catalogAccess struct {
	Role    string   `json:"role,omitempty"`
	Actions []string `json:"actions,omitempty"`
	Source  string   `json:"source,omitempty"`
	GrantID string   `json:"grantId,omitempty"`
	Reason  string   `json:"reason,omitempty"`
}

type catalogDownload struct {
	URL     string `json:"url"`
	Version string `json:"version,omitempty"`
	MD5     string `json:"md5,omitempty"`
	Size    int64  `json:"size,omitempty"`
}

type catalogSkillItem struct {
	Name          string            `json:"name"`
	DisplayName   string            `json:"displayName,omitempty"`
	Description   string            `json:"description,omitempty"`
	LatestVersion string            `json:"latestVersion,omitempty"`
	Object        string            `json:"object"`
	Status        string            `json:"status,omitempty"`
	OwnerSubject  string            `json:"ownerSubject,omitempty"`
	SkillSet      string            `json:"skillSet,omitempty"`
	Labels        map[string]string `json:"labels,omitempty"`
	Access        catalogAccess     `json:"access"`
	Download      *catalogDownload  `json:"download,omitempty"`
	UpdateTime    int64             `json:"updateTime,omitempty"`
}

type catalogSkillSetItem struct {
	Name         string            `json:"name"`
	DisplayName  string            `json:"displayName,omitempty"`
	Description  string            `json:"description,omitempty"`
	Object       string            `json:"object"`
	Owner        string            `json:"owner,omitempty"`
	Labels       map[string]string `json:"labels,omitempty"`
	MemberCount  int               `json:"memberCount"`
	Access       catalogAccess     `json:"access"`
	DownloadBase string            `json:"downloadBase,omitempty"`
	UpdateTime   int64             `json:"updateTime,omitempty"`
}

func (h *Handler) catalogSkills(c *gin.Context) {
	pageNo := atoiDefault(c.Query("pageNo"), 1)
	pageSize := atoiDefault(c.Query("pageSize"), 100)
	page, err := h.svc.ListSkills(aihubNS(), firstQuery(c, "skillName", "name"), c.Query("search"), c.Query("orderBy"), c.Query("owner"), "", c.Query("bizTag"), c.Query("skillSet"), pageNo, pageSize)
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	summaries, _ := page.PageItems.([]model.SkillSummary)
	items := make([]catalogSkillItem, 0, len(summaries))
	for _, s := range summaries {
		allowed, access := h.catalogCan(c, "skill", s.Name, "read")
		if !allowed {
			continue
		}
		latest, md5 := h.catalogLatestVersion(s.Name)
		item := catalogSkillItem{
			Name:          s.Name,
			DisplayName:   s.ModelName,
			Description:   s.Description,
			LatestVersion: latest,
			Object:        h.catalogObject("skill", s.Name),
			Status:        boolStatus(s.Enable),
			OwnerSubject:  firstNonEmptyString(s.OwnerSubject, s.Owner),
			SkillSet:      s.SkillSet,
			Labels:        s.Labels,
			Access:        access,
			UpdateTime:    valueOrZero(s.UpdateTime),
		}
		if latest != "" {
			item.Download = &catalogDownload{URL: fmt.Sprintf("/v3/aihub/catalog/skills/%s/versions/%s/download", s.Name, latest), Version: latest, MD5: md5}
		}
		items = append(items, item)
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": len(items), "pageNumber": pageNo, "pageSize": pageSize})
}

func (h *Handler) catalogSkillManifest(c *gin.Context) {
	name := c.Param("skillName")
	allowed, access := h.catalogCan(c, "skill", name, "read")
	if !allowed {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "object": h.catalogObject("skill", name), "action": "read", "reason": access.Reason})
		return
	}
	meta, err := h.svc.GetSkillDetail(aihubNS(), name)
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	latest, md5 := h.catalogLatestVersion(name)
	vr, _ := h.svc.GetSkillVersionRecord(aihubNS(), name, latest)
	dl := catalogDownload{URL: fmt.Sprintf("/v3/aihub/catalog/skills/%s/versions/%s/download", name, latest), Version: latest, MD5: md5}
	if vr != nil {
		dl.Size = versionSize(vr)
	}
	c.JSON(http.StatusOK, gin.H{
		"name":          meta.Name,
		"description":   meta.Description,
		"latestVersion": latest,
		"revision": firstNonEmptyString(func() string {
			if vr != nil {
				return vr.Revision
			}
			return ""
		}(), shortHash(name+"|"+latest+"|"+md5)),
		"sha256": func() string {
			if vr != nil {
				return versionSHA256(vr)
			}
			return ""
		}(),
		"object":       h.catalogObject("skill", name),
		"access":       access,
		"download":     dl,
		"versions":     meta.Versions,
		"labels":       meta.Labels,
		"skillSet":     meta.SkillSet,
		"ownerSubject": meta.OwnerSubject,
	})
}

func (h *Handler) catalogDownloadSkillVersion(c *gin.Context) {
	name := c.Param("skillName")
	version := c.Param("version")
	allowed, access := h.catalogCan(c, "skill", name, "download")
	if !allowed {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "object": h.catalogObject("skill", name), "action": "download", "reason": access.Reason})
		return
	}
	skill, b, md5, err := h.svc.DownloadSkillVersion(aihubNS(), name, version)
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	if vr, _ := h.svc.GetSkillVersionRecord(aihubNS(), name, version); vr != nil {
		c.Header("X-AIHub-Sha256", versionSHA256(vr))
		c.Header("X-AIHub-Revision", firstNonEmptyString(vr.Revision, skillVersionRevision(name, vr)))
	}
	writeZip(c, skill.Name, b, md5, version)
}

func (h *Handler) catalogSkillSets(c *gin.Context) {
	page, err := h.svc.ListGroups(aihubNS(), c.Query("q"), atoiDefault(c.Query("pageNo"), 1), atoiDefault(c.Query("pageSize"), 100))
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	groups, _ := page.PageItems.([]*model.SkillGroup)
	items := make([]catalogSkillSetItem, 0, len(groups))
	for _, g := range groups {
		if g == nil {
			continue
		}
		allowed, access := h.catalogCan(c, "skillset", g.Name, "read")
		if !allowed {
			continue
		}
		items = append(items, catalogSkillSetItem{
			Name:         g.Name,
			DisplayName:  g.DisplayName,
			Description:  g.Description,
			Object:       h.catalogObject("skillset", g.Name),
			Owner:        g.Owner,
			Labels:       g.Labels,
			MemberCount:  len(g.Members),
			Access:       access,
			DownloadBase: "/v3/aihub/catalog/skills",
			UpdateTime:   g.UpdateTime,
		})
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": len(items)})
}

func (h *Handler) catalogSkillSetManifest(c *gin.Context) {
	name := groupParam(c)
	snap, err := h.buildCatalogSkillSetSnapshot(c, name, nil)
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	if match := strings.TrimSpace(c.GetHeader("If-None-Match")); match != "" && (match == snap.ETag || strings.Trim(match, "\"") == snap.ETag) {
		c.Header("ETag", snap.ETag)
		c.Status(http.StatusNotModified)
		return
	}
	c.Header("ETag", snap.ETag)
	c.Header("X-AIHub-Revision", snap.Revision)
	c.JSON(http.StatusOK, gin.H{"skillset": snap, "object": snap.Object, "revision": snap.Revision, "etag": snap.ETag, "total": len(snap.Members)})
}

func (h *Handler) reportRuntimeInstalledSkills(c *gin.Context) {
	var body struct {
		RuntimeID string                 `json:"runtimeId"`
		Hostname  string                 `json:"hostname"`
		SkillSet  string                 `json:"skillSet,omitempty"`
		Skills    []map[string]any       `json:"skills"`
		Metadata  map[string]interface{} `json:"metadata,omitempty"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		httputil.Fail(c, httputil.BadRequest(err.Error()))
		return
	}
	if strings.TrimSpace(body.RuntimeID) == "" {
		body.RuntimeID = principalID(c)
	}
	reportedAt := time.Now().UnixMilli()
	_ = h.svc.AppendCatalogEvent(&model.CatalogEvent{
		App:          h.appCode(),
		EventType:    model.CatalogEventRuntimeReported,
		Object:       h.catalogObject("runtime", body.RuntimeID),
		ResourceType: "runtime",
		ResourceID:   body.RuntimeID,
		SkillSetName: body.SkillSet,
		Payload:      map[string]interface{}{"hostname": body.Hostname, "skills": body.Skills, "metadata": body.Metadata},
		CreatedAt:    reportedAt,
	})
	c.JSON(http.StatusOK, gin.H{"ok": true, "runtimeId": body.RuntimeID, "received": len(body.Skills), "reportedAt": reportedAt})
}

func (h *Handler) catalogCan(c *gin.Context, resourceType, resourceID, action string) (bool, catalogAccess) {
	access := catalogAccess{Actions: []string{action}}
	if h == nil || h.iam == nil {
		access.Role = "local"
		access.Source = "iam_disabled"
		return true, access
	}
	p := currentPrincipal(c)
	sub := catalogSubject(p)
	if sub == "" {
		sub = principalID(c)
	}
	obj := h.catalogObject(resourceType, resourceID)
	ok, reason, dec, err := h.iam.CheckDetailed(authclient.CheckRequest{
		Subject:      sub,
		Principal:    catalogPrincipal(p, h.appCode()),
		App:          h.appCode(),
		Object:       obj,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
	})
	if err != nil {
		access.Reason = err.Error()
		access.Source = "aisphere-auth_error"
		return false, access
	}
	if dec != nil {
		access.Source = dec.Source
		access.Reason = dec.Reason
		access.GrantID = dec.MatchedGrantID
		if dec.Source == "resource_grant" && dec.MatchedGrantID != "" {
			access.Role = "grant"
		}
	} else if reason != "" {
		access.Reason = reason
	}

	if ok && access.Role == "" {
		access.Role = "allowed"
	}
	return ok, access
}

func (h *Handler) catalogObject(resourceType, resourceID string) string {
	return h.appCode() + ":" + resourceType + ":" + strings.TrimSpace(resourceID)
}

func (h *Handler) catalogLatestVersion(skillName string) (string, string) {
	meta, err := h.svc.GetSkillDetail(aihubNS(), skillName)
	if err != nil {
		return "", ""
	}
	if v := meta.Labels[model.LabelLatest]; v != "" {
		return v, ""
	}
	for _, item := range meta.Versions {
		if item.Status == model.VersionStatusOnline || item.Status == model.VersionStatusReviewed || item.Status == model.VersionStatusDraft || item.Status == model.VersionStatusReviewing {
			return item.Version, ""
		}
	}
	if len(meta.Versions) > 0 {
		return meta.Versions[0].Version, ""
	}
	return "", ""
}

func currentPrincipal(c *gin.Context) *auth.Principal {
	if v, ok := c.Get(auth.PrincipalContextKey); ok {
		if p, ok := v.(*auth.Principal); ok {
			return p
		}
		if p, ok := v.(auth.Principal); ok {
			return &p
		}
	}
	return nil
}

func catalogSubject(p *auth.Principal) string {
	if p == nil {
		return ""
	}
	if p.ExternalSubject != "" && strings.Contains(p.ExternalSubject, "/") {
		return p.ExternalSubject
	}
	if p.Organization != "" && p.Username != "" {
		return p.Organization + "/" + p.Username
	}
	return firstNonEmptyString(p.SubjectID, p.ExternalSubject, p.Username)
}

func catalogPrincipal(p *auth.Principal, app string) *aisphereauth.Principal {
	if p == nil {
		return nil
	}
	out := &aisphereauth.Principal{SubjectID: p.SubjectID, Username: p.Username, Email: p.Email, Organization: p.Organization, Roles: append([]string(nil), p.Roles...), Groups: append([]string(nil), p.Groups...), App: app, Claims: map[string]any{}}
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

func firstNonEmptyString(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func boolStatus(enable bool) string {
	if enable {
		return model.MetaStatusEnable
	}
	return model.MetaStatusDisable
}

func valueOrZero(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}

// context import guard for older Go linters when build tags trim catalogCan in tests.
var _ = context.Background
