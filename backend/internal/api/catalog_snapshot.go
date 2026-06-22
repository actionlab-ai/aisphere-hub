package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/actionlab-ai/aisphere-hub/backend/internal/httputil"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/model"
	"github.com/gin-gonic/gin"
)

type catalogSkillSnapshotItem struct {
	Name        string        `json:"name"`
	Version     string        `json:"version"`
	Revision    string        `json:"revision"`
	Object      string        `json:"object"`
	SHA256      string        `json:"sha256,omitempty"`
	MD5         string        `json:"md5,omitempty"`
	Size        int64         `json:"size,omitempty"`
	DownloadURL string        `json:"downloadUrl"`
	Access      catalogAccess `json:"access"`
}

type catalogSkillSetSnapshot struct {
	Name        string                     `json:"name"`
	Object      string                     `json:"object"`
	Revision    string                     `json:"revision"`
	ETag        string                     `json:"etag"`
	GeneratedAt string                     `json:"generatedAt"`
	Members     []catalogSkillSnapshotItem `json:"members"`
}

type resolveSessionRequest struct {
	RuntimeID       string   `json:"runtimeId"`
	SessionID       string   `json:"sessionId"`
	SkillSet        string   `json:"skillset"`
	Policy          string   `json:"policy"`
	RequestedSkills []string `json:"requestedSkills"`
}

type resolveSessionResponse struct {
	RuntimeID   string                     `json:"runtimeId"`
	SessionID   string                     `json:"sessionId"`
	SkillSet    string                     `json:"skillset"`
	SnapshotID  string                     `json:"snapshotId"`
	Revision    string                     `json:"revision"`
	GeneratedAt string                     `json:"generatedAt"`
	Policy      string                     `json:"policy"`
	Skills      []catalogSkillSnapshotItem `json:"skills"`
}

func (h *Handler) catalogResolveSession(c *gin.Context) {
	var req resolveSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.Fail(c, httputil.BadRequest(err.Error()))
		return
	}
	if strings.TrimSpace(req.RuntimeID) == "" {
		req.RuntimeID = principalID(c)
	}
	if strings.TrimSpace(req.SessionID) == "" {
		req.SessionID = "sess_" + time.Now().UTC().Format("20060102150405.000000000")
	}
	if strings.TrimSpace(req.SkillSet) == "" {
		req.SkillSet = c.Query("skillset")
	}
	if strings.TrimSpace(req.Policy) == "" {
		req.Policy = "latest_authorized"
	}
	snap, err := h.buildCatalogSkillSetSnapshot(c, req.SkillSet, req.RequestedSkills)
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	resp := resolveSessionResponse{
		RuntimeID:   req.RuntimeID,
		SessionID:   req.SessionID,
		SkillSet:    snap.Name,
		SnapshotID:  "snap_" + shortHash(req.RuntimeID+"|"+req.SessionID+"|"+snap.Revision),
		Revision:    snap.Revision,
		GeneratedAt: snap.GeneratedAt,
		Policy:      req.Policy,
		Skills:      snap.Members,
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) catalogChanges(c *gin.Context) {
	skillset := strings.TrimSpace(c.Query("skillset"))
	if skillset != "" {
		if ok, _ := h.catalogCan(c, "skillset", skillset, "read"); !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "object": h.catalogObject("skillset", skillset), "action": "read"})
			return
		}
	}
	sinceID := parseCursor(c.Query("since"))
	limit := atoiDefault(c.Query("limit"), 100)
	events, _, err := h.svc.ListCatalogEvents(model.CatalogEventQuery{App: h.appCode(), SkillSetName: skillset, SinceID: sinceID, Limit: limit})
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	items := make([]gin.H, 0, len(events))
	cursor := sinceID
	for _, e := range events {
		if e == nil {
			continue
		}
		if e.ID > cursor {
			cursor = e.ID
		}
		items = append(items, gin.H{"id": strconv.FormatInt(e.ID, 10), "type": e.EventType, "object": e.Object, "resourceType": e.ResourceType, "resourceId": e.ResourceID, "skillset": e.SkillSetName, "version": e.Version, "revision": e.Revision, "payload": e.Payload, "createdAt": e.CreatedAt})
	}
	c.JSON(http.StatusOK, gin.H{"cursor": strconv.FormatInt(cursor, 10), "events": items})
}

func (h *Handler) catalogEvents(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	skillset := strings.TrimSpace(c.Query("skillset"))
	if skillset != "" {
		if ok, _ := h.catalogCan(c, "skillset", skillset, "read"); !ok {
			c.SSEvent("error", gin.H{"error": "forbidden", "object": h.catalogObject("skillset", skillset), "action": "read"})
			return
		}
	}
	sinceID := parseCursor(c.Query("cursor"))
	if sinceID == 0 {
		sinceID = parseCursor(c.Query("since"))
	}
	events, _, err := h.svc.ListCatalogEvents(model.CatalogEventQuery{App: h.appCode(), SkillSetName: skillset, SinceID: sinceID, Limit: 100})
	if err != nil {
		c.SSEvent("error", gin.H{"error": err.Error()})
		return
	}
	cursor := sinceID
	for _, e := range events {
		if e == nil {
			continue
		}
		if e.ID > cursor {
			cursor = e.ID
		}
		c.SSEvent(e.EventType, gin.H{"id": strconv.FormatInt(e.ID, 10), "object": e.Object, "resourceType": e.ResourceType, "resourceId": e.ResourceID, "skillset": e.SkillSetName, "version": e.Version, "revision": e.Revision, "payload": e.Payload, "createdAt": e.CreatedAt})
	}
	c.Writer.Header().Set("X-AIHub-Cursor", strconv.FormatInt(cursor, 10))
	c.Writer.Flush()
}

func parseCursor(s string) int64 {
	s = strings.TrimSpace(strings.TrimPrefix(s, "cursor_"))
	if s == "" {
		return 0
	}
	v, _ := strconv.ParseInt(s, 10, 64)
	return v
}

func (h *Handler) buildCatalogSkillSetSnapshot(c *gin.Context, skillset string, requested []string) (*catalogSkillSetSnapshot, error) {
	skillset = strings.TrimSpace(skillset)
	allowedSkills := map[string]bool{}
	name := skillset
	if skillset != "" {
		ok, _ := h.catalogCan(c, "skillset", skillset, "read")
		if !ok {
			return nil, &httputil.AppError{Status: http.StatusForbidden, Code: 403, Message: "forbidden: object=" + h.catalogObject("skillset", skillset) + " action=read"}
		}
		manifest, err := h.svc.ResolveGroupManifest(aihubNS(), skillset, c.Query("label"))
		if err != nil {
			return nil, err
		}
		name = manifest.Name
		for _, m := range manifest.Members {
			allowedSkills[m.Name] = true
		}
	}
	requestedSet := map[string]bool{}
	for _, s := range requested {
		s = strings.TrimSpace(s)
		if s != "" {
			requestedSet[s] = true
		}
	}
	page, err := h.svc.ListSkills(aihubNS(), "", "", "", "", "", "", skillset, 1, 500)
	if err != nil {
		return nil, err
	}
	summaries, _ := page.PageItems.([]model.SkillSummary)
	items := make([]catalogSkillSnapshotItem, 0, len(summaries))
	for _, s := range summaries {
		if skillset != "" && !allowedSkills[s.Name] {
			continue
		}
		if len(requestedSet) > 0 && !requestedSet[s.Name] {
			continue
		}
		ok, access := h.catalogCan(c, "skill", s.Name, "read")
		if !ok {
			continue
		}
		if ok, _ := h.catalogCan(c, "skill", s.Name, "download"); !ok {
			continue
		}
		meta, err := h.svc.GetSkillDetail(aihubNS(), s.Name)
		if err != nil {
			continue
		}
		version := latestCatalogVersion(meta)
		if version == "" {
			continue
		}
		vr, _ := h.svc.GetSkillVersionRecord(aihubNS(), s.Name, version)
		if vr == nil {
			continue
		}
		items = append(items, catalogSkillSnapshotItem{
			Name:        s.Name,
			Version:     version,
			Revision:    skillVersionRevision(s.Name, vr),
			Object:      h.catalogObject("skill", s.Name),
			SHA256:      versionSHA256(vr),
			MD5:         vr.MD5,
			Size:        versionSize(vr),
			DownloadURL: fmt.Sprintf("/v3/aihub/catalog/skills/%s/versions/%s/download", s.Name, version),
			Access:      access,
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	if name == "" {
		name = "_all"
	}
	snap := &catalogSkillSetSnapshot{Name: name, Object: h.catalogObject("skillset", name), GeneratedAt: time.Now().UTC().Format(time.RFC3339), Members: items}
	snap.Revision = snapshotRevision(name, items)
	snap.ETag = "sha256:" + snap.Revision
	return snap, nil
}

func latestCatalogVersion(meta model.SkillMeta) string {
	if meta.Labels != nil && meta.Labels[model.LabelLatest] != "" {
		return meta.Labels[model.LabelLatest]
	}
	versions := make([]string, 0, len(meta.Versions))
	for _, v := range meta.Versions {
		if v.Status == model.VersionStatusOnline || v.Status == model.VersionStatusReviewed || v.Status == model.VersionStatusDraft || v.Status == model.VersionStatusReviewing {
			versions = append(versions, v.Version)
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(versions)))
	if len(versions) > 0 {
		return versions[0]
	}
	if len(meta.Versions) > 0 {
		return meta.Versions[0].Version
	}
	return ""
}

func versionSHA256(v *model.VersionRecord) string {
	if v == nil {
		return ""
	}
	if v.SHA256 != "" {
		return v.SHA256
	}
	if v.Storage != nil && v.Storage.SHA256 != "" {
		return v.Storage.SHA256
	}
	return ""
}

func versionSize(v *model.VersionRecord) int64 {
	if v == nil {
		return 0
	}
	if v.SizeBytes > 0 {
		return v.SizeBytes
	}
	if v.Storage != nil {
		return v.Storage.SizeBytes
	}
	return 0
}

func snapshotRevision(name string, items []catalogSkillSnapshotItem) string {
	b, _ := json.Marshal(struct {
		Name  string                     `json:"name"`
		Items []catalogSkillSnapshotItem `json:"items"`
	}{Name: name, Items: items})
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func skillVersionRevision(name string, v *model.VersionRecord) string {
	if v == nil {
		return shortHash(name)
	}
	return shortHash(fmt.Sprintf("%s|%s|%s|%d|%s", name, v.Version, v.MD5, v.UpdateTime, v.CommitMsg))
}

func shortHash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])[:16]
}
