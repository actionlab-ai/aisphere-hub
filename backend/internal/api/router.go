package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/actionlab-ai/aisphere-auth/pkg/aisphereauth"
	platformhealth "github.com/actionlab-ai/aisphere-go/health"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/aisphereclient"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/auth"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/httputil"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/model"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/service"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *service.Service
	iam *aisphereclient.Client
}

func NewRouter(svc *service.Service, middlewares ...gin.HandlerFunc) *gin.Engine {
	return newRouter(svc, nil, middlewares...)
}

func NewRouterWithAISphere(svc *service.Service, iam *aisphereclient.Client, middlewares ...gin.HandlerFunc) *gin.Engine {
	return newRouter(svc, iam, middlewares...)
}

func newRouter(svc *service.Service, iam *aisphereclient.Client, middlewares ...gin.HandlerFunc) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	if len(middlewares) > 0 {
		r.Use(middlewares...)
	}
	h := &Handler{svc: svc, iam: iam}
	health := platformhealth.New(time.Second)
	r.GET("/healthz", gin.WrapH(health.LiveHandler()))
	r.GET("/livez", gin.WrapH(health.LiveHandler()))
	r.GET("/readyz", gin.WrapH(health.ReadyHandler()))

	admin := r.Group("/v3/admin/ai/skills")
	{
		admin.GET("/list", h.listSkills)
		admin.GET("", h.getSkill)
		admin.GET("/version", h.getSkillVersion)
		admin.GET("/version/files", h.listSkillVersionFiles)
		admin.GET("/version/file", h.getSkillVersionFile)
		admin.GET("/version/compare", h.compareSkillVersions)
		admin.GET("/version/download", h.downloadSkillVersion)
		admin.DELETE("", h.deleteSkill)
		admin.POST("/upload", h.uploadSkill)
		admin.POST("/upload/batch", h.batchUpload)
		admin.POST("/draft", h.createDraft)
		admin.PUT("/draft", h.updateDraft)
		admin.DELETE("/draft", h.deleteDraft)
		admin.POST("/submit", h.submit)
		admin.POST("/publish", h.publish)
		admin.POST("/force-publish", h.forcePublish)
		admin.POST("/redraft", h.redraft)
		admin.PUT("/labels", h.updateLabels)
		admin.PUT("/biz-tags", h.updateBizTags)
		admin.PUT("/metadata", h.updateMetadata)
		admin.POST("/online", h.online)
		admin.POST("/offline", h.offline)
		admin.PUT("/scope", h.updateScope)
	}

	groups := r.Group("/v3/admin/ai/skill-groups")
	{
		groups.GET("/list", h.listGroups)
		groups.GET("", h.getGroup)
		groups.POST("", h.saveGroup)
		groups.PUT("", h.saveGroup)
		groups.DELETE("", h.deleteGroup)
		groups.POST("/bind", h.bindGroupMember)
		groups.DELETE("/bind", h.unbindGroupMember)
	}
	agent := r.Group("/v3/agent/ai")
	{
		agent.POST("/skill-proposals", h.agentSubmitProposal)
		agent.GET("/skill-proposals/:proposalId", h.getProposal)
		agent.GET("/skill-overlays/:proposalId", h.getOverlayByProposal)
		agent.GET("/skill-overlays", h.getOverlay)
	}

	ns := r.Group("/v3/admin/namespaces")
	{
		ns.GET("", h.listNamespaces)
		ns.GET("/:namespaceId", h.getNamespace)
		ns.POST("", h.saveNamespace)
		ns.PUT("/:namespaceId", h.saveNamespace)
		ns.GET("/:namespaceId/members", h.listNamespaceMembers)
		ns.POST("/:namespaceId/members", h.saveNamespaceMember)
		ns.PUT("/:namespaceId/members/:subjectId", h.saveNamespaceMember)
		ns.DELETE("/:namespaceId/members/:subjectId", h.deleteNamespaceMember)
	}

	social := r.Group("/v3/admin/ai/skills/social")
	{
		social.GET("", h.getSkillSocial)
		social.POST("/star", h.setSkillStar)
		social.POST("/rating", h.rateSkill)
		social.POST("/subscribe", h.subscribeSkill)
	}

	audit := r.Group("/v3/admin/audit")
	{
		audit.GET("/logs", h.listAuditLogs)
	}
	tokens := r.Group("/v3/admin/iam/tokens")
	{
		tokens.GET("", h.listTokens)
		tokens.POST("", h.createToken)
		tokens.DELETE("/:keyId", h.deleteToken)
	}

	iamAdmin := r.Group("/v3/admin/iam")
	{
		iamAdmin.GET("/whoami", h.whoami)
	}

	notify := r.Group("/v3/admin/notifications")
	{
		notify.GET("", h.listNotifications)
		notify.GET("/stream", h.notificationsSSE)
		notify.POST("/:notificationId/read", h.markNotificationRead)
	}

	adminProposal := r.Group("/v3/admin/ai/skill-proposals")
	{
		adminProposal.GET("/list", h.listProposals)
		adminProposal.GET("/:proposalId", h.getProposal)
		adminProposal.POST("/:proposalId/validate", h.validateProposal)
		adminProposal.POST("/:proposalId/approve", h.approveProposal)
		adminProposal.POST("/:proposalId/reject", h.rejectProposal)
	}

	// Canonical namespace-free AIHub API.
	aihub := r.Group("/v3/aihub")
	{
		aihub.GET("/skills", h.listSkillsCanonical)
		aihub.POST("/skills/upload", h.uploadSkillCanonical)
		aihub.POST("/skills/upload/batch", h.batchUploadCanonical)
		aihub.POST("/skills/draft", h.createDraftCanonical)
		aihub.PUT("/skills/draft", h.updateDraftCanonical)
		aihub.DELETE("/skills/draft", h.deleteDraftCanonical)
		aihub.GET("/skill/:skillName", h.getSkillCanonical)
		aihub.DELETE("/skill/:skillName", h.deleteSkillCanonical)
		aihub.GET("/skill/:skillName/versions/:version", h.getSkillVersionCanonical)
		aihub.GET("/skill/:skillName/versions/:version/download", h.downloadSkillVersionCanonical)
		aihub.GET("/skill/:skillName/versions/:version/files", h.listSkillVersionFilesCanonical)
		aihub.GET("/skill/:skillName/versions/:version/file", h.getSkillVersionFileCanonical)
		aihub.GET("/skill/:skillName/compare", h.compareSkillVersionsCanonical)
		aihub.POST("/skill/:skillName/submit", h.submitCanonical)
		aihub.POST("/skill/:skillName/publish", h.publishCanonical)
		aihub.POST("/skill/:skillName/force-publish", h.forcePublishCanonical)
		aihub.POST("/skill/:skillName/redraft", h.redraftCanonical)
		aihub.POST("/skill/:skillName/online", h.onlineCanonical)
		aihub.POST("/skill/:skillName/offline", h.offlineCanonical)
		aihub.PUT("/skill/:skillName/labels", h.updateLabelsCanonical)
		aihub.PUT("/skill/:skillName/biz-tags", h.updateBizTagsCanonical)
		aihub.PUT("/skill/:skillName/metadata", h.updateMetadataCanonical)
		aihub.PUT("/skill/:skillName/scope", h.updateScopeCanonical)
		aihub.GET("/skill/:skillName/shares", h.listSkillShares)
		aihub.POST("/skill/:skillName/shares", h.createSkillShare)
		aihub.DELETE("/skill/:skillName/shares/:grantId", h.deleteShare)

		aihub.GET("/skillsets", h.listGroupsCanonical)
		aihub.POST("/skillsets", h.saveGroupCanonical)
		aihub.GET("/skillset/:skillSetName", h.getGroupCanonical)
		aihub.PUT("/skillset/:skillSetName", h.saveGroupCanonical)
		aihub.DELETE("/skillset/:skillSetName", h.deleteGroupCanonical)
		aihub.GET("/skillset/:skillSetName/skills", h.groupSkillsCanonical)
		aihub.POST("/skillset/:skillSetName/skills", h.bindGroupMemberCanonical)
		aihub.DELETE("/skillset/:skillSetName/skills/:skillName", h.unbindGroupMemberCanonical)
		aihub.GET("/skillset/:skillSetName/shares", h.listSkillSetShares)
		aihub.POST("/skillset/:skillSetName/shares", h.createSkillSetShare)
		aihub.DELETE("/skillset/:skillSetName/shares/:grantId", h.deleteShare)

		// SandboxProfile / SandboxPolicy are product-level sandbox declarations.
		// Hub stores profile/policy metadata; aisphere-sandbox adapter converts profile to agent-sandbox CRD.
		aihub.GET("/sandbox-profiles", h.listSandboxProfiles)
		aihub.POST("/sandbox-profiles", h.saveSandboxProfile)
		aihub.GET("/sandbox-profiles/:profileId", h.getSandboxProfile)
		aihub.PUT("/sandbox-profiles/:profileId", h.saveSandboxProfile)
		aihub.DELETE("/sandbox-profiles/:profileId", h.deleteSandboxProfile)
		aihub.GET("/sandbox-policies", h.listSandboxPolicies)
		aihub.POST("/sandbox-policies", h.saveSandboxPolicy)
		aihub.GET("/sandbox-policies/:policyId", h.getSandboxPolicy)
		aihub.PUT("/sandbox-policies/:policyId", h.saveSandboxPolicy)
		aihub.DELETE("/sandbox-policies/:policyId", h.deleteSandboxPolicy)

		// ModelProfile declares logical model runtime profiles. aisphere-gateway resolves these profiles and hides real provider credentials.
		aihub.GET("/model-profiles", h.listModelProfiles)
		aihub.POST("/model-profiles", h.saveModelProfile)
		aihub.GET("/model-profiles/:profileId", h.getModelProfile)
		aihub.PUT("/model-profiles/:profileId", h.saveModelProfile)
		aihub.DELETE("/model-profiles/:profileId", h.deleteModelProfile)

		// Runtime Catalog API. This is the permission-aware discovery surface
		// used by Agent Runtime. Keep .well-known for anonymous public discovery
		// only; authenticated runtimes should use these catalog endpoints.
		aihub.GET("/catalog/skills", h.catalogSkills)
		aihub.GET("/catalog/skills/:skillName/manifest", h.catalogSkillManifest)
		aihub.GET("/catalog/skills/:skillName/versions/:version/download", h.catalogDownloadSkillVersion)
		aihub.GET("/catalog/skillsets", h.catalogSkillSets)
		aihub.GET("/catalog/skillsets/:skillSetName/manifest", h.catalogSkillSetManifest)
		aihub.GET("/catalog/changes", h.catalogChanges)
		aihub.GET("/catalog/events", h.catalogEvents)
		aihub.POST("/runtime/sessions/resolve", h.catalogResolveSession)
		aihub.POST("/runtime/services/resolve", h.resolveRuntimeServices)
		aihub.POST("/runtime/agents/:agentId/resolve", h.resolveAgentRuntime)
		aihub.POST("/runtime/tools/:toolId/resolve", h.resolveToolRuntime)
		aihub.POST("/runtime/tool-failures", h.reportToolFailure)
		aihub.POST("/runtime/tools/:toolId/failures", h.reportToolFailure)
		aihub.POST("/runtime/installed-skills", h.reportRuntimeInstalledSkills)

		// Runtime Sandbox API. Sandboxes are scheduled as Kubernetes Pods with
		// /workspace persisted on PVC. AgentKit runtime uses these endpoints to
		// ensure/restart/delete execution environments and to fetch basic logs.
		aihub.GET("/runtime/sandboxes", h.listSandboxes)
		aihub.POST("/runtime/sandboxes", h.ensureSandbox)
		aihub.GET("/runtime/sandboxes/:sandboxId", h.getSandbox)
		aihub.POST("/runtime/sandboxes/:sandboxId/restart", h.restartSandbox)
		aihub.DELETE("/runtime/sandboxes/:sandboxId", h.deleteSandbox)
		aihub.GET("/runtime/sandboxes/:sandboxId/logs", h.sandboxLogs)
		aihub.GET("/runtime/sandboxes/:sandboxId/tools", h.listSandboxTools)
		aihub.POST("/runtime/sandboxes/:sandboxId/tools/call", h.callSandboxTool)

		// Tool resources are Hub-managed runtime capabilities. Agents reference
		// these by id; runtime resolve returns an immutable, permission-checked
		// manifest, and execution failures are reported back into Hub.
		aihub.GET("/tools", h.listTools)
		aihub.POST("/tools", h.createTool)
		aihub.GET("/tools/:toolId", h.getTool)
		aihub.PUT("/tools/:toolId", h.updateTool)
		aihub.DELETE("/tools/:toolId", h.deleteTool)
		aihub.GET("/tools/:toolId/shares", h.listToolShares)
		aihub.POST("/tools/:toolId/shares", h.createToolShare)
		aihub.DELETE("/tools/:toolId/shares/:grantId", h.deleteShare)
		aihub.GET("/tool-failures", h.listToolFailures)

		// Agent resources now live under AIHub. This cut exposes the canonical
		// routes and object naming; persistence/service implementation can be
		// filled in by the Agent platform module without changing IAM contracts.
		aihub.GET("/agents", h.listAgents)
		aihub.POST("/agents", h.createAgent)
		aihub.GET("/agents/:agentId", h.getAgent)
		aihub.PUT("/agents/:agentId", h.updateAgent)
		aihub.DELETE("/agents/:agentId", h.deleteAgent)
		aihub.GET("/agents/:agentId/shares", h.listAgentShares)
		aihub.POST("/agents/:agentId/shares", h.createAgentShare)
		aihub.DELETE("/agents/:agentId/shares/:grantId", h.deleteShare)

		aihub.GET("/workflows", h.listWorkflows)
		aihub.POST("/workflows", h.createWorkflow)
		aihub.GET("/workflows/:workflowId", h.getWorkflow)
		aihub.PUT("/workflows/:workflowId", h.updateWorkflow)
		aihub.DELETE("/workflows/:workflowId", h.deleteWorkflow)
		aihub.POST("/workflows/:workflowId/run", h.runWorkflow)
		aihub.GET("/workflows/:workflowId/shares", h.listWorkflowShares)
		aihub.POST("/workflows/:workflowId/shares", h.createWorkflowShare)
		aihub.DELETE("/workflows/:workflowId/shares/:grantId", h.deleteShare)

		aihub.GET("/runs", h.listRuns)
		aihub.GET("/runs/:runId", h.getRun)
		aihub.POST("/runs/:runId/cancel", h.cancelRun)
		aihub.POST("/runs/:runId/retry", h.retryRun)
	}
	r.GET("/v3/client/ai/skills/:skillName", h.clientSkillCanonical)
	r.GET("/v3/client/ai/groups/:groupName", h.clientGroupCanonical)
	r.GET("/registry-global/api/search", h.registrySearchCanonical)
	r.GET("/registry-global/.well-known/agent-skills/*path", h.registryWellKnownCanonical)

	r.GET("/v3/client/ai/skill-groups", h.clientGroup)
	r.GET("/v3/client/ai/skills", h.clientSkill)

	r.GET("/registry/:namespaceId/api/search", h.registrySearch)
	r.GET("/registry/:namespaceId/.well-known/agent-skills/*path", h.registryWellKnown)
	r.GET("/registry/:namespaceId/.well-known/skills/*path", h.registryWellKnown)
	return r
}

func (h *Handler) whoami(c *gin.Context) {
	v, ok := c.Get(auth.PrincipalContextKey)
	if !ok {
		c.JSON(200, gin.H{"anonymous": true})
		return
	}
	c.JSON(200, gin.H{"principal": v})
}

func (h *Handler) getSkill(c *gin.Context) {
	ns := httputil.Namespace(c.Query("namespaceId"))
	name := firstQuery(c, "skillName", "name")
	out, err := h.svc.GetSkillDetail(ns, name)
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, out)
}

func (h *Handler) listSkillVersionFiles(c *gin.Context) {
	ns := httputil.Namespace(c.Query("namespaceId"))
	name := firstQuery(c, "skillName", "name")
	version := c.Query("version")
	out, err := h.svc.ListSkillVersionFiles(ns, name, version)
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, out)
}
func (h *Handler) getSkillVersionFile(c *gin.Context) {
	ns := httputil.Namespace(c.Query("namespaceId"))
	name := firstQuery(c, "skillName", "name")
	version := c.Query("version")
	path := c.Query("path")
	out, err := h.svc.GetSkillVersionFile(ns, name, version, path)
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, out)
}
func (h *Handler) compareSkillVersions(c *gin.Context) {
	ns := httputil.Namespace(c.Query("namespaceId"))
	name := firstQuery(c, "skillName", "name")
	out, err := h.svc.CompareSkillVersions(ns, name, c.Query("baseVersion"), c.Query("targetVersion"))
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, out)
}

func (h *Handler) getSkillVersion(c *gin.Context) {
	ns := httputil.Namespace(c.Query("namespaceId"))
	name := firstQuery(c, "skillName", "name")
	version := c.Query("version")
	out, err := h.svc.GetSkillVersion(ns, name, version)
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, out)
}
func (h *Handler) downloadSkillVersion(c *gin.Context) {
	ns := httputil.Namespace(c.Query("namespaceId"))
	name := firstQuery(c, "skillName", "name")
	version := c.Query("version")
	skill, b, md5, err := h.svc.DownloadSkillVersion(ns, name, version)
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	writeZip(c, skill.Name, b, md5, version)
}
func (h *Handler) deleteSkill(c *gin.Context) {
	ns := httputil.Namespace(firstFormOrQuery(c, "namespaceId"))
	name := firstFormOrQuery(c, "skillName")
	if err := h.svc.DeleteSkill(ns, name); err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, "ok")
}
func (h *Handler) listSkills(c *gin.Context) {
	ns := httputil.Namespace(c.Query("namespaceId"))
	pageNo := atoiDefault(c.Query("pageNo"), 1)
	pageSize := atoiDefault(c.Query("pageSize"), 20)
	page, err := h.svc.ListSkills(ns, c.Query("skillName"), c.Query("search"), c.Query("orderBy"), c.Query("owner"), c.Query("scope"), c.Query("bizTag"), c.Query("groupName"), pageNo, pageSize)
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, page)
}
func (h *Handler) uploadSkill(c *gin.Context) {
	ns := httputil.Namespace(c.PostForm("namespaceId"))
	overwrite := parseBool(c.PostForm("overwrite"))
	targetVersion := c.PostForm("targetVersion")
	commitMsg := c.PostForm("commitMsg")
	file, err := c.FormFile("file")
	if err != nil {
		httputil.Fail(c, httputil.BadRequest("file is required"))
		return
	}
	f, err := file.Open()
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	defer f.Close()
	b, err := io.ReadAll(f)
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	name, err := h.svc.UploadSkillFromZip(ns, b, overwrite, targetVersion, commitMsg)
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, name)
}
func (h *Handler) batchUpload(c *gin.Context) {
	ns := httputil.Namespace(c.PostForm("namespaceId"))
	overwrite := parseBool(c.PostForm("overwrite"))
	file, err := c.FormFile("file")
	if err != nil {
		httputil.Fail(c, httputil.BadRequest("file is required"))
		return
	}
	f, err := file.Open()
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	defer f.Close()
	b, err := io.ReadAll(f)
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	out, err := h.svc.BatchUploadSkillsFromZip(ns, b, overwrite)
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, out)
}
func (h *Handler) createDraft(c *gin.Context) {
	vals := readParams(c)
	ns := httputil.Namespace(vals["namespaceId"])
	name := firstVal(vals, "skillName", "name")
	skill, err := parseDraftSkill(vals, ns, name)
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	version, err := h.svc.CreateDraft(ns, name, vals["basedOnVersion"], draftTargetVersion(vals), skill, vals["commitMsg"])
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, version)
}
func (h *Handler) updateDraft(c *gin.Context) {
	vals := readParams(c)
	ns := httputil.Namespace(vals["namespaceId"])
	name := firstVal(vals, "skillName", "name")
	skill, err := parseDraftSkill(vals, ns, name)
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	if skill == nil {
		httputil.Fail(c, httputil.BadRequest("skillCard is required"))
		return
	}
	if err := h.svc.UpdateDraft(ns, *skill, vals["commitMsg"]); err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, "ok")
}
func (h *Handler) deleteDraft(c *gin.Context) {
	ns := httputil.Namespace(firstFormOrQuery(c, "namespaceId"))
	name := firstFormOrQuery(c, "skillName")
	if err := h.svc.DeleteDraft(ns, name); err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, "ok")
}
func (h *Handler) submit(c *gin.Context) {
	h.versionAction(c, func(ns, name, version string) (interface{}, error) { return h.svc.Submit(ns, name, version) })
}
func (h *Handler) publish(c *gin.Context) {
	h.versionAction(c, func(ns, name, version string) (interface{}, error) {
		return "ok", h.svc.Publish(ns, name, version, true, false)
	})
}
func (h *Handler) forcePublish(c *gin.Context) {
	h.versionAction(c, func(ns, name, version string) (interface{}, error) {
		return "ok", h.svc.Publish(ns, name, version, true, true)
	})
}
func (h *Handler) redraft(c *gin.Context) {
	h.versionAction(c, func(ns, name, version string) (interface{}, error) { return "ok", h.svc.Redraft(ns, name, version) })
}
func (h *Handler) versionAction(c *gin.Context, fn func(ns, name, version string) (interface{}, error)) {
	vals := readParams(c)
	ns := httputil.Namespace(vals["namespaceId"])
	name := firstVal(vals, "skillName", "name")
	version := vals["version"]
	out, err := fn(ns, name, version)
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, out)
}
func (h *Handler) updateLabels(c *gin.Context) {
	vals := readParams(c)
	ns := httputil.Namespace(vals["namespaceId"])
	name := firstVal(vals, "skillName", "name")
	labels := map[string]string{}
	if vals["labels"] != "" {
		if err := json.Unmarshal([]byte(vals["labels"]), &labels); err != nil {
			httputil.Fail(c, httputil.BadRequest("invalid labels json: "+err.Error()))
			return
		}
	}
	if err := h.svc.UpdateLabels(ns, name, labels); err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, "ok")
}
func (h *Handler) updateBizTags(c *gin.Context) {
	vals := readParams(c)
	ns := httputil.Namespace(vals["namespaceId"])
	name := firstVal(vals, "skillName", "name")
	if err := h.svc.UpdateBizTags(ns, name, vals["bizTags"]); err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, "ok")
}
func (h *Handler) updateMetadata(c *gin.Context) {
	vals := readParams(c)
	ns := httputil.Namespace(vals["namespaceId"])
	name := firstVal(vals, "skillName", "name")
	metadata := model.Skill{SkillBase: model.SkillBase{SkillSet: vals["skillSet"], Groups: parseCSVorJSON(vals["groups"]), Keywords: parseCSVorJSON(vals["keywords"]), ModelName: vals["modelName"], ModelDescription: vals["modelDescription"], MatchHint: vals["matchHint"], Activation: vals["activation"]}}
	if vals["priority"] != "" {
		p, err := strconv.Atoi(vals["priority"])
		if err != nil {
			httputil.Fail(c, httputil.BadRequest("priority must be integer"))
			return
		}
		metadata.Priority = &p
	}
	if err := h.svc.UpdateMetadata(ns, name, metadata); err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, "ok")
}
func (h *Handler) updateScope(c *gin.Context) {
	vals := readParams(c)
	ns := httputil.Namespace(vals["namespaceId"])
	name := firstVal(vals, "skillName", "name")
	if err := h.svc.UpdateScope(ns, name, vals["scope"]); err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, "ok")
}
func (h *Handler) online(c *gin.Context)  { h.onlineAction(c, true) }
func (h *Handler) offline(c *gin.Context) { h.onlineAction(c, false) }
func (h *Handler) onlineAction(c *gin.Context, online bool) {
	vals := readParams(c)
	ns := httputil.Namespace(vals["namespaceId"])
	name := firstVal(vals, "skillName", "name")
	if err := h.svc.ChangeOnlineStatus(ns, name, vals["scope"], vals["version"], online); err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, "ok")
}
func (h *Handler) clientSkill(c *gin.Context) {
	if c.Query("namespaceId") != "" {
		c.Header(httputil.DeprecatedNamespaceWarningHeader, "namespaceId is deprecated and ignored by AIHub registry")
	}
	ns := httputil.Namespace(c.Query("namespaceId"))
	name := c.Query("name")
	skill, b, md5, resolved, notModified, err := h.svc.QuerySkill(ns, name, c.Query("version"), c.Query("label"), c.Query("md5"))
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	if notModified {
		c.Header("ETag", md5)
		c.Header("X-Nacos-Skill-Md5", md5)
		c.Header("X-Nacos-Skill-Resolved-Version", resolved)
		c.Status(http.StatusNotModified)
		return
	}
	writeZip(c, skill.Name, b, md5, resolved)
}
func (h *Handler) registrySearch(c *gin.Context) {
	c.Header(httputil.DeprecatedNamespaceWarningHeader, "registry namespace path is deprecated; use /registry-global")
	ns := httputil.Namespace(c.Param("namespaceId"))
	sourceBase := baseURL(c, "/registry/"+ns)
	items, err := h.svc.SearchPublic(ns, c.Query("q"), atoiDefault(c.Query("limit"), 20), sourceBase)
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"skills": items})
}
func (h *Handler) registryWellKnown(c *gin.Context) {
	c.Header(httputil.DeprecatedNamespaceWarningHeader, "registry namespace path is deprecated; use /registry-global")
	ns := httputil.Namespace(c.Param("namespaceId"))
	p := strings.TrimPrefix(c.Param("path"), "/")
	base := baseURL(c, strings.TrimSuffix(c.FullPath(), "/*path"))
	if p == "index.json" {
		idx, err := h.svc.WellKnownIndex(ns, base)
		if err != nil {
			httputil.Fail(c, err)
			return
		}
		c.JSON(http.StatusOK, idx)
		return
	}
	if strings.HasSuffix(p, ".zip") && !strings.Contains(p, "/") {
		name := strings.TrimSuffix(p, ".zip")
		_, b, md5, err := h.svc.RegistrySkill(ns, name)
		if err != nil {
			httputil.Fail(c, err)
			return
		}
		writeZip(c, name, b, md5, "")
		return
	}
	parts := strings.SplitN(p, "/", 2)
	if len(parts) != 2 {
		httputil.Fail(c, httputil.NotFound("registry path not found"))
		return
	}
	content, err := h.svc.RegistryFile(ns, parts[0], parts[1])
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(content))
}

func writeZip(c *gin.Context, name string, b []byte, md5, version string) {
	if name == "" {
		name = "skill"
	}
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", fmt.Sprintf("attachment;filename=%s.zip", name))
	if md5 != "" {
		c.Header("ETag", md5)
		c.Header("X-Nacos-Skill-Md5", md5)
	}
	if version != "" {
		c.Header("X-Nacos-Skill-Resolved-Version", version)
	}
	c.Data(http.StatusOK, "application/zip", b)
}

func readParams(c *gin.Context) map[string]string {
	vals := map[string]string{}
	_ = c.Request.ParseForm()
	for k, v := range c.Request.Form {
		if len(v) > 0 {
			vals[k] = v[0]
		}
	}
	ct := c.ContentType()
	if strings.Contains(ct, "json") {
		bodyBytes, _ := c.GetRawData()
		var m map[string]interface{}
		if err := json.Unmarshal(bodyBytes, &m); err == nil {
			for k, v := range m {
				switch x := v.(type) {
				case string:
					vals[k] = x
				default:
					b, _ := json.Marshal(x)
					vals[k] = string(b)
				}
			}
		}
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}
	return vals
}

func parseDraftSkill(vals map[string]string, namespaceID, fallbackName string) (*model.Skill, error) {
	if strings.TrimSpace(vals["skillCard"]) != "" {
		return service.ParseSkillCard(vals["skillCard"], namespaceID, fallbackName)
	}
	if !hasDirectDraftSkill(vals) {
		return nil, nil
	}
	skill := model.Skill{
		SkillBase: model.SkillBase{
			NamespaceID:      namespaceID,
			Name:             firstVal(vals, "skillName", "name"),
			Description:      vals["description"],
			SkillSet:         vals["skillSet"],
			Groups:           parseCSVorJSON(vals["groups"]),
			Keywords:         parseCSVorJSON(vals["keywords"]),
			ModelName:        vals["modelName"],
			ModelDescription: vals["modelDescription"],
			MatchHint:        vals["matchHint"],
			Activation:       vals["activation"],
		},
		SkillMD: firstVal(vals, "skillMd", "skillMD", "skillMarkdown"),
	}
	if skill.Name == "" {
		skill.Name = fallbackName
	}
	if skill.SkillMD == "" && strings.TrimSpace(skill.Description) != "" {
		skill.SkillMD = buildDraftSkillMarkdown(skill, draftTargetVersion(vals), vals["displayName"])
	}
	return &skill, nil
}

func hasDirectDraftSkill(vals map[string]string) bool {
	for _, key := range []string{
		"description",
		"displayName",
		"skillMd",
		"skillMD",
		"skillMarkdown",
		"skillSet",
		"groups",
		"keywords",
		"modelName",
		"modelDescription",
		"matchHint",
		"activation",
		"priority",
	} {
		if strings.TrimSpace(vals[key]) != "" {
			return true
		}
	}
	return false
}

func draftTargetVersion(vals map[string]string) string {
	return firstVal(vals, "targetVersion", "version")
}

func buildDraftSkillMarkdown(skill model.Skill, version, displayName string) string {
	var b strings.Builder
	b.WriteString("---\n")
	writeYAMLScalar(&b, "name", skill.Name)
	writeYAMLScalar(&b, "description", skill.Description)
	writeYAMLScalar(&b, "version", version)
	writeYAMLScalar(&b, "displayName", displayName)
	writeYAMLScalar(&b, "skillSet", skill.SkillSet)
	writeYAMLList(&b, "groups", skill.Groups)
	writeYAMLList(&b, "keywords", skill.Keywords)
	writeYAMLScalar(&b, "modelName", skill.ModelName)
	writeYAMLScalar(&b, "modelDescription", skill.ModelDescription)
	writeYAMLScalar(&b, "matchHint", skill.MatchHint)
	writeYAMLScalar(&b, "activation", skill.Activation)
	b.WriteString("---\n\n")
	title := strings.TrimSpace(displayName)
	if title == "" {
		title = strings.TrimSpace(skill.Name)
	}
	if title == "" {
		title = "Untitled Skill"
	}
	b.WriteString("# ")
	b.WriteString(title)
	b.WriteString("\n\n")
	b.WriteString(strings.TrimSpace(skill.Description))
	b.WriteString("\n")
	return b.String()
}

func writeYAMLScalar(b *strings.Builder, key, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return
	}
	b.WriteString(key)
	b.WriteString(": ")
	b.Write(encoded)
	b.WriteString("\n")
}

func writeYAMLList(b *strings.Builder, key string, values []string) {
	cleaned := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}
	if len(cleaned) == 0 {
		return
	}
	encoded, err := json.Marshal(cleaned)
	if err != nil {
		return
	}
	b.WriteString(key)
	b.WriteString(": ")
	b.Write(encoded)
	b.WriteString("\n")
}

func firstVal(m map[string]string, keys ...string) string {
	for _, k := range keys {
		if m[k] != "" {
			return m[k]
		}
	}
	return ""
}
func firstQuery(c *gin.Context, keys ...string) string {
	for _, k := range keys {
		if c.Query(k) != "" {
			return c.Query(k)
		}
	}
	return ""
}
func firstFormOrQuery(c *gin.Context, key string) string {
	if v := c.PostForm(key); v != "" {
		return v
	}
	return c.Query(key)
}

func groupParam(c *gin.Context) string {
	if v := c.Param("skillSetName"); v != "" {
		return v
	}
	return c.Param("groupName")
}
func atoiDefault(s string, d int) int {
	if i, err := strconv.Atoi(s); err == nil {
		return i
	}
	return d
}
func parseBool(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "true" || s == "1" || s == "yes" || s == "on"
}
func parseCSVorJSON(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	var arr []string
	if strings.HasPrefix(s, "[") && json.Unmarshal([]byte(s), &arr) == nil {
		return arr
	}
	parts := strings.Split(s, ",")
	out := []string{}
	for _, p := range parts {
		p = strings.Trim(strings.TrimSpace(p), `"'`)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
func baseURL(c *gin.Context, path string) string {
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	if xf := c.GetHeader("X-Forwarded-Proto"); xf != "" {
		scheme = xf
	}
	return scheme + "://" + c.Request.Host + path
}

func (h *Handler) listGroups(c *gin.Context) {
	ns := httputil.Namespace(c.Query("namespaceId"))
	page, err := h.svc.ListGroups(ns, c.Query("q"), atoiDefault(c.Query("pageNo"), 1), atoiDefault(c.Query("pageSize"), 20))
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, page)
}
func (h *Handler) getGroup(c *gin.Context) {
	ns := httputil.Namespace(c.Query("namespaceId"))
	name := firstQuery(c, "groupName", "name")
	out, err := h.svc.GetGroup(ns, name)
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, out)
}
func (h *Handler) saveGroup(c *gin.Context) {
	vals := readParams(c)
	ns := httputil.Namespace(vals["namespaceId"])
	var g model.SkillGroup
	if vals["group"] != "" {
		if err := json.Unmarshal([]byte(vals["group"]), &g); err != nil {
			httputil.Fail(c, httputil.BadRequest("invalid group json: "+err.Error()))
			return
		}
	} else {
		g.Name = firstVal(vals, "groupName", "name")
		g.DisplayName = vals["displayName"]
		g.Description = vals["description"]
		g.Owner = vals["owner"]
		g.Scope = vals["scope"]
		if vals["labels"] != "" {
			_ = json.Unmarshal([]byte(vals["labels"]), &g.Labels)
		}
		if vals["members"] != "" {
			_ = json.Unmarshal([]byte(vals["members"]), &g.Members)
		}
	}
	out, err := h.svc.SaveGroup(ns, g)
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, out)
}
func (h *Handler) deleteGroup(c *gin.Context) {
	ns := httputil.Namespace(firstFormOrQuery(c, "namespaceId"))
	name := firstFormOrQuery(c, "groupName")
	if name == "" {
		name = firstFormOrQuery(c, "name")
	}
	if err := h.svc.DeleteGroup(ns, name); err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, "ok")
}
func (h *Handler) bindGroupMember(c *gin.Context) {
	vals := readParams(c)
	ns := httputil.Namespace(vals["namespaceId"])
	var m model.SkillGroupMember
	if vals["member"] != "" {
		_ = json.Unmarshal([]byte(vals["member"]), &m)
	} else {
		m.SkillName = vals["skillName"]
		m.Version = vals["version"]
		m.Label = vals["label"]
		m.Required = parseBool(vals["required"])
		m.Order = atoiDefault(vals["order"], 0)
	}
	out, err := h.svc.BindGroupMember(ns, firstVal(vals, "groupName", "name"), m)
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, out)
}
func (h *Handler) unbindGroupMember(c *gin.Context) {
	vals := readParams(c)
	ns := httputil.Namespace(vals["namespaceId"])
	out, err := h.svc.UnbindGroupMember(ns, firstVal(vals, "groupName", "name"), vals["skillName"])
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, out)
}
func (h *Handler) clientGroup(c *gin.Context) {
	ns := httputil.Namespace(c.Query("namespaceId"))
	name := firstQuery(c, "groupName", "name")
	manifest, err := h.svc.ResolveGroupManifest(ns, name, c.Query("label"))
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	base := baseURL(c, "/v3/client/ai/skills")
	for i := range manifest.Members {
		manifest.Members[i].DownloadURL = fmt.Sprintf("%s/%s?version=%s", base, manifest.Members[i].Name, manifest.Members[i].Version)
	}
	c.JSON(http.StatusOK, manifest)
}

func (h *Handler) agentSubmitProposal(c *gin.Context) {
	vals := readParams(c)
	var p model.SkillProposal
	if vals["proposal"] != "" {
		if err := json.Unmarshal([]byte(vals["proposal"]), &p); err != nil {
			httputil.Fail(c, httputil.BadRequest("invalid proposal json: "+err.Error()))
			return
		}
	} else {
		p.NamespaceID = httputil.Namespace(vals["namespaceId"])
		p.SkillName = firstVal(vals, "skillName", "name")
		p.BaseVersion = vals["baseVersion"]
		p.ProposalType = vals["proposalType"]
		p.Reason = vals["reason"]
		p.Source.AgentID = vals["agentId"]
		p.Source.SessionID = vals["sessionId"]
		p.Source.RunID = vals["runId"]
		p.Source.TaskID = vals["taskId"]
		if vals["delta"] != "" {
			_ = json.Unmarshal([]byte(vals["delta"]), &p.Delta)
		}
		if vals["evidence"] != "" {
			_ = json.Unmarshal([]byte(vals["evidence"]), &p.Evidence)
		}
	}
	out, err := h.svc.SubmitSkillProposal(p)
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, out)
}

func (h *Handler) listProposals(c *gin.Context) {
	page, err := h.svc.ListSkillProposals(model.ProposalQuery{NamespaceID: httputil.Namespace(c.Query("namespaceId")), SkillName: c.Query("skillName"), Status: c.Query("status"), PageNo: atoiDefault(c.Query("pageNo"), 1), PageSize: atoiDefault(c.Query("pageSize"), 20)})
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, page)
}

func (h *Handler) getProposal(c *gin.Context) {
	id := c.Param("proposalId")
	if id == "" {
		id = c.Query("proposalId")
	}
	out, err := h.svc.GetSkillProposal(id)
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, out)
}

func (h *Handler) getOverlayByProposal(c *gin.Context) {
	p, err := h.svc.GetSkillProposal(c.Param("proposalId"))
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	out, err := h.svc.GetSkillOverlay(p.OverlayRef)
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, out)
}

func (h *Handler) getOverlay(c *gin.Context) {
	out, err := h.svc.GetSkillOverlay(c.Query("overlayRef"))
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, out)
}

func (h *Handler) validateProposal(c *gin.Context) {
	out, err := h.svc.ValidateSkillProposal(c.Param("proposalId"))
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, out)
}

func (h *Handler) approveProposal(c *gin.Context) {
	vals := readParams(c)
	var opts model.ProposalApproveOptions
	if vals["options"] != "" {
		_ = json.Unmarshal([]byte(vals["options"]), &opts)
	}
	if opts.TargetVersion == "" {
		opts.TargetVersion = vals["targetVersion"]
	}
	if opts.Label == "" {
		opts.Label = vals["label"]
	}
	if opts.Reviewer == "" {
		opts.Reviewer = vals["reviewer"]
	}
	if opts.Comment == "" {
		opts.Comment = vals["comment"]
	}
	if vals["publish"] != "" {
		opts.Publish = parseBool(vals["publish"])
	}
	if vals["online"] != "" {
		opts.Online = parseBool(vals["online"])
	}
	out, err := h.svc.ApproveSkillProposal(c.Param("proposalId"), opts)
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, out)
}

func (h *Handler) rejectProposal(c *gin.Context) {
	vals := readParams(c)
	out, err := h.svc.RejectSkillProposal(c.Param("proposalId"), vals["reviewer"], vals["reason"])
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, out)
}

func principalID(c *gin.Context) string {
	if v, ok := c.Get(auth.PrincipalContextKey); ok {
		if p, ok := v.(*auth.Principal); ok {
			return p.SubjectID
		}
		if p, ok := v.(auth.Principal); ok {
			return p.SubjectID
		}
		b, _ := json.Marshal(v)
		var m map[string]interface{}
		_ = json.Unmarshal(b, &m)
		if s, _ := m["subjectId"].(string); s != "" {
			return s
		}
	}
	return "anonymous"
}
func readJSON(c *gin.Context, dst interface{}) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		httputil.Fail(c, httputil.BadRequest(err.Error()))
		return false
	}
	return true
}

func (h *Handler) listNamespaces(c *gin.Context) {
	out, err := h.svc.ListNamespaces()
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, out)
}
func (h *Handler) getNamespace(c *gin.Context) {
	out, err := h.svc.GetNamespace(c.Param("namespaceId"))
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, out)
}
func (h *Handler) saveNamespace(c *gin.Context) {
	var ns model.NamespaceInfo
	if !readJSON(c, &ns) {
		return
	}
	if ns.NamespaceID == "" {
		ns.NamespaceID = c.Param("namespaceId")
	}
	out, err := h.svc.SaveNamespace(ns, principalID(c))
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, out)
}
func (h *Handler) listNamespaceMembers(c *gin.Context) {
	out, err := h.svc.ListNamespaceMembers(c.Param("namespaceId"), atoiDefault(c.Query("pageNo"), 1), atoiDefault(c.Query("pageSize"), 50))
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, out)
}
func (h *Handler) saveNamespaceMember(c *gin.Context) {
	var m model.NamespaceMember
	if !readJSON(c, &m) {
		return
	}
	m.NamespaceID = c.Param("namespaceId")
	if m.SubjectID == "" {
		m.SubjectID = c.Param("subjectId")
	}
	out, err := h.svc.SaveNamespaceMember(m, principalID(c))
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, out)
}
func (h *Handler) deleteNamespaceMember(c *gin.Context) {
	if err := h.svc.DeleteNamespaceMember(c.Param("namespaceId"), c.Param("subjectId"), principalID(c)); err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, "ok")
}

func (h *Handler) getSkillSocial(c *gin.Context) {
	ns := httputil.Namespace(c.Query("namespaceId"))
	name := firstQuery(c, "skillName", "name")
	out, err := h.svc.GetSkillSocialStats(ns, name, principalID(c))
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, out)
}
func (h *Handler) setSkillStar(c *gin.Context) {
	var req struct {
		NamespaceID string `json:"namespaceId"`
		SkillName   string `json:"skillName"`
		Starred     bool   `json:"starred"`
	}
	if !readJSON(c, &req) {
		return
	}
	out, err := h.svc.SetSkillStar(httputil.Namespace(req.NamespaceID), req.SkillName, principalID(c), req.Starred)
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, out)
}
func (h *Handler) rateSkill(c *gin.Context) {
	var req struct {
		NamespaceID string `json:"namespaceId"`
		SkillName   string `json:"skillName"`
		Rating      int    `json:"rating"`
		Comment     string `json:"comment"`
	}
	if !readJSON(c, &req) {
		return
	}
	out, err := h.svc.RateSkill(httputil.Namespace(req.NamespaceID), req.SkillName, principalID(c), req.Rating, req.Comment)
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, out)
}
func (h *Handler) subscribeSkill(c *gin.Context) {
	var req struct {
		NamespaceID string `json:"namespaceId"`
		SkillName   string `json:"skillName"`
		Subscribed  bool   `json:"subscribed"`
	}
	if !readJSON(c, &req) {
		return
	}
	if err := h.svc.Subscribe(httputil.Namespace(req.NamespaceID), model.SubscriptionTargetSkill, req.SkillName, principalID(c), req.Subscribed); err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, "ok")
}

func (h *Handler) listAuditLogs(c *gin.Context) {
	out, err := h.svc.ListAuditLogs(model.AuditQuery{NamespaceID: c.Query("namespaceId"), ResourceType: c.Query("resourceType"), ResourceName: c.Query("resourceName"), Action: c.Query("action"), Operator: c.Query("operator"), PageNo: atoiDefault(c.Query("pageNo"), 1), PageSize: atoiDefault(c.Query("pageSize"), 50)})
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, out)
}
func (h *Handler) listTokens(c *gin.Context) {
	out, err := h.svc.ListTokens(c.Query("subjectId"))
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, out)
}
func (h *Handler) createToken(c *gin.Context) {
	var req model.TokenCreateRequest
	if !readJSON(c, &req) {
		return
	}
	out, err := h.svc.CreateToken(req, principalID(c))
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, out)
}
func (h *Handler) deleteToken(c *gin.Context) {
	if err := h.svc.DeleteToken(c.Param("keyId"), principalID(c)); err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, "ok")
}

func (h *Handler) listNotifications(c *gin.Context) {
	p := principalID(c)
	if p == "" {
		p = c.Query("subjectId")
	}
	pageNo := atoiDefault(c.Query("pageNo"), 1)
	pageSize := atoiDefault(c.Query("pageSize"), 50)
	out, err := h.svc.ListNotifications(model.NotificationQuery{SubjectID: p, UnreadOnly: parseBool(c.Query("unreadOnly")), PageNo: pageNo, PageSize: pageSize})
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, out)
}
func (h *Handler) markNotificationRead(c *gin.Context) {
	if err := h.svc.MarkNotificationRead(principalID(c), c.Param("notificationId")); err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, "ok")
}
func (h *Handler) notificationsSSE(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	p := principalID(c)
	if p == "" {
		p = c.Query("subjectId")
	}
	flusher, _ := c.Writer.(http.Flusher)
	for i := 0; i < 30; i++ { // simple polling SSE; production can replace with pub/sub fanout
		page, err := h.svc.ListNotifications(model.NotificationQuery{SubjectID: p, UnreadOnly: true, PageNo: 1, PageSize: 20})
		if err == nil {
			b, _ := json.Marshal(page.PageItems)
			_, _ = c.Writer.Write([]byte("event: notifications\ndata: " + string(b) + "\n\n"))
		}
		if flusher != nil {
			flusher.Flush()
		}
		select {
		case <-c.Request.Context().Done():
			return
		case <-time.After(10 * time.Second):
		}
	}
}

// ---- Canonical group-first / namespace-free API handlers ----
func aihubNS() string { return model.DefaultNamespace }

func (h *Handler) listSkillsCanonical(c *gin.Context) {
	page, err := h.svc.ListSkills(aihubNS(), firstQuery(c, "skillName", "name"), c.Query("search"), c.Query("orderBy"), c.Query("owner"), c.Query("scope"), c.Query("bizTag"), c.Query("groupName"), atoiDefault(c.Query("pageNo"), 1), atoiDefault(c.Query("pageSize"), 20))
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, page)
}
func (h *Handler) uploadSkillCanonical(c *gin.Context) {
	f, err := c.FormFile("file")
	if err != nil {
		httputil.Fail(c, httputil.BadRequest("file is required"))
		return
	}
	rc, err := f.Open()
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	defer rc.Close()
	b, err := io.ReadAll(rc)
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	name, err := h.svc.UploadSkillFromZip(aihubNS(), b, parseBool(firstFormOrQuery(c, "overwrite")), firstFormOrQuery(c, "targetVersion"), firstFormOrQuery(c, "commitMsg"))
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, name)
}
func (h *Handler) getSkillCanonical(c *gin.Context) {
	out, err := h.svc.GetSkillDetail(aihubNS(), c.Param("skillName"))
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, out)
}
func (h *Handler) deleteSkillCanonical(c *gin.Context) {
	if err := h.svc.DeleteSkill(aihubNS(), c.Param("skillName")); err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, "ok")
}
func (h *Handler) getSkillVersionCanonical(c *gin.Context) {
	out, err := h.svc.GetSkillVersion(aihubNS(), c.Param("skillName"), c.Param("version"))
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, out)
}
func (h *Handler) downloadSkillVersionCanonical(c *gin.Context) {
	skill, b, md5, err := h.svc.DownloadSkillVersion(aihubNS(), c.Param("skillName"), c.Param("version"))
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	writeZip(c, skill.Name, b, md5, c.Param("version"))
}
func (h *Handler) listSkillVersionFilesCanonical(c *gin.Context) {
	out, err := h.svc.ListSkillVersionFiles(aihubNS(), c.Param("skillName"), c.Param("version"))
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, out)
}
func (h *Handler) getSkillVersionFileCanonical(c *gin.Context) {
	out, err := h.svc.GetSkillVersionFile(aihubNS(), c.Param("skillName"), c.Param("version"), c.Query("path"))
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, out)
}
func (h *Handler) compareSkillVersionsCanonical(c *gin.Context) {
	out, err := h.svc.CompareSkillVersions(aihubNS(), c.Param("skillName"), c.Query("baseVersion"), c.Query("targetVersion"))
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, out)
}
func (h *Handler) submitCanonical(c *gin.Context) {
	vals := readParams(c)
	version := firstVal(vals, "version")
	if version == "" {
		version = c.Query("version")
	}
	out, err := h.svc.Submit(aihubNS(), c.Param("skillName"), version)
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, out)
}
func (h *Handler) publishCanonical(c *gin.Context) {
	vals := readParams(c)
	version := firstVal(vals, "version")
	err := h.svc.Publish(aihubNS(), c.Param("skillName"), version, true, false)
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, "ok")
}
func (h *Handler) onlineCanonical(c *gin.Context) {
	vals := readParams(c)
	err := h.svc.ChangeOnlineStatus(aihubNS(), c.Param("skillName"), vals["scope"], firstVal(vals, "version"), true)
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, "ok")
}
func (h *Handler) offlineCanonical(c *gin.Context) {
	vals := readParams(c)
	err := h.svc.ChangeOnlineStatus(aihubNS(), c.Param("skillName"), vals["scope"], firstVal(vals, "version"), false)
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, "ok")
}
func (h *Handler) updateLabelsCanonical(c *gin.Context) {
	vals := readParams(c)
	labels := map[string]string{}
	_ = json.Unmarshal([]byte(firstVal(vals, "labels")), &labels)
	if len(labels) == 0 {
		var body struct {
			Labels map[string]string `json:"labels"`
		}
		_ = c.ShouldBindJSON(&body)
		labels = body.Labels
	}
	if err := h.svc.UpdateLabels(aihubNS(), c.Param("skillName"), labels); err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, "ok")
}

func (h *Handler) listGroupsCanonical(c *gin.Context) {
	out, err := h.svc.ListGroups(aihubNS(), c.Query("q"), atoiDefault(c.Query("pageNo"), 1), atoiDefault(c.Query("pageSize"), 20))
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, out)
}
func (h *Handler) getGroupCanonical(c *gin.Context) {
	out, err := h.svc.GetGroup(aihubNS(), groupParam(c))
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, out)
}
func (h *Handler) saveGroupCanonical(c *gin.Context) {
	vals := readParams(c)
	var g model.SkillGroup
	if vals["group"] != "" {
		_ = json.Unmarshal([]byte(vals["group"]), &g)
	} else {
		_ = c.ShouldBindJSON(&g)
		if g.Name == "" {
			g.Name = groupParam(c)
		}
	}
	out, err := h.svc.SaveGroup(aihubNS(), g)
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, out)
}
func (h *Handler) deleteGroupCanonical(c *gin.Context) {
	if err := h.svc.DeleteGroup(aihubNS(), groupParam(c)); err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, "ok")
}
func (h *Handler) groupSkillsCanonical(c *gin.Context) {
	g, err := h.svc.GetGroup(aihubNS(), groupParam(c))
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, g.Members)
}
func (h *Handler) bindGroupMemberCanonical(c *gin.Context) {
	var m model.SkillGroupMember
	if !readJSON(c, &m) {
		return
	}
	out, err := h.svc.BindGroupMember(aihubNS(), groupParam(c), m)
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, out)
}
func (h *Handler) unbindGroupMemberCanonical(c *gin.Context) {
	out, err := h.svc.UnbindGroupMember(aihubNS(), groupParam(c), c.Param("skillName"))
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, out)
}

func (h *Handler) listSkillShares(c *gin.Context) {
	h.listResourceShares(c, "skill", c.Param("skillName"))
}

func (h *Handler) createSkillShare(c *gin.Context) {
	h.createResourceShare(c, "skill", c.Param("skillName"))
}

func (h *Handler) listSkillSetShares(c *gin.Context) {
	h.listResourceShares(c, "skillset", groupParam(c))
}

func (h *Handler) createSkillSetShare(c *gin.Context) {
	h.createResourceShare(c, "skillset", groupParam(c))
}

func (h *Handler) deleteShare(c *gin.Context) {
	if h.iam == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "iam_resource_grants_disabled", "message": "aisphere-auth integration is not enabled"})
		return
	}
	grantID := c.Param("grantId")
	if err := h.iam.DeleteResourceGrant(c.Request.Context(), grantID); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "iam_resource_grant_delete_failed", "message": err.Error()})
		return
	}
	_ = h.svc.AppendCatalogEvent(&model.CatalogEvent{App: h.appCode(), EventType: model.CatalogEventGrantChanged, Object: h.appCode() + ":grant:" + grantID, ResourceType: "grant", ResourceID: grantID, Payload: map[string]interface{}{"op": "delete", "grantId": grantID}, CreatedAt: time.Now().UnixMilli()})
	c.JSON(http.StatusOK, gin.H{"deleted": true, "id": grantID})
}

func (h *Handler) listResourceShares(c *gin.Context, resourceType, resourceID string) {
	if h.iam == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "iam_resource_grants_disabled", "message": "aisphere-auth integration is not enabled"})
		return
	}
	out, err := h.iam.ListResourceGrants(c.Request.Context(), aisphereauth.ResourceGrantQuery{
		App:          h.appCode(),
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Limit:        atoiDefault(c.Query("limit"), 100),
		Offset:       atoiDefault(c.Query("offset"), 0),
	})
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "iam_resource_grant_list_failed", "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) createResourceShare(c *gin.Context, resourceType, resourceID string) {
	if h.iam == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "iam_resource_grants_disabled", "message": "aisphere-auth integration is not enabled"})
		return
	}
	var body struct {
		OrgID       string            `json:"orgId"`
		ProjectID   string            `json:"projectId"`
		SubjectType string            `json:"subjectType"`
		SubjectID   string            `json:"subjectId"`
		Role        string            `json:"role"`
		Actions     []string          `json:"actions"`
		Effect      string            `json:"effect"`
		ExpiresAt   int64             `json:"expiresAt"`
		Metadata    map[string]string `json:"metadata"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		httputil.Fail(c, httputil.BadRequest(err.Error()))
		return
	}
	app := h.appCode()
	grant := aisphereauth.ResourceGrant{
		App:          app,
		OrgID:        body.OrgID,
		ProjectID:    body.ProjectID,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Object:       app + ":" + resourceType + ":" + resourceID,
		SubjectType:  body.SubjectType,
		SubjectID:    body.SubjectID,
		Role:         body.Role,
		Actions:      body.Actions,
		Effect:       body.Effect,
		ExpiresAt:    body.ExpiresAt,
		CreatedBy:    principalID(c),
		Metadata:     body.Metadata,
	}
	out, err := h.iam.CreateResourceGrant(c.Request.Context(), grant)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "iam_resource_grant_create_failed", "message": err.Error()})
		return
	}
	_ = h.svc.AppendCatalogEvent(&model.CatalogEvent{App: app, EventType: model.CatalogEventGrantChanged, Object: grant.Object, ResourceType: resourceType, ResourceID: resourceID, SkillSetName: func() string {
		if resourceType == "skillset" {
			return resourceID
		}
		return ""
	}(), Payload: map[string]interface{}{"op": "create", "subjectType": body.SubjectType, "subjectId": body.SubjectID, "role": body.Role}, CreatedAt: time.Now().UnixMilli()})
	c.JSON(http.StatusOK, out)
}

func (h *Handler) appCode() string {
	if h != nil && h.iam != nil && strings.TrimSpace(h.iam.Config().App) != "" {
		return strings.TrimSpace(h.iam.Config().App)
	}
	return "aihub"
}

func (h *Handler) clientSkillCanonical(c *gin.Context) {
	name := c.Param("skillName")
	skill, b, md5, resolved, notModified, err := h.svc.QuerySkill(aihubNS(), name, c.Query("version"), c.Query("label"), c.Query("md5"))
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	if notModified {
		c.Header("ETag", md5)
		c.Header("X-AIHub-Md5", md5)
		c.Header("X-AIHub-Resolved-Version", resolved)
		c.Status(http.StatusNotModified)
		return
	}
	writeZip(c, skill.Name, b, md5, resolved)
}
func (h *Handler) clientGroupCanonical(c *gin.Context) {
	manifest, err := h.svc.ResolveGroupManifest(aihubNS(), groupParam(c), c.Query("label"))
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	base := baseURL(c, "/v3/client/ai/skills")
	for i := range manifest.Members {
		manifest.Members[i].DownloadURL = fmt.Sprintf("%s/%s?version=%s", base, manifest.Members[i].Name, manifest.Members[i].Version)
	}
	c.JSON(http.StatusOK, manifest)
}
func (h *Handler) registrySearchCanonical(c *gin.Context) {
	items, err := h.svc.SearchPublic(aihubNS(), c.Query("q"), atoiDefault(c.Query("limit"), 20), baseURL(c, "/registry-global"))
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"skills": items})
}
func (h *Handler) registryWellKnownCanonical(c *gin.Context) {
	p := strings.TrimPrefix(c.Param("path"), "/")
	base := baseURL(c, strings.TrimSuffix(c.FullPath(), "/*path"))
	if p == "index.json" {
		idx, err := h.svc.WellKnownIndex(aihubNS(), base)
		if err != nil {
			httputil.Fail(c, err)
			return
		}
		c.JSON(http.StatusOK, idx)
		return
	}
	if strings.HasSuffix(p, ".zip") && !strings.Contains(p, "/") {
		name := strings.TrimSuffix(p, ".zip")
		_, b, md5, err := h.svc.RegistrySkill(aihubNS(), name)
		if err != nil {
			httputil.Fail(c, err)
			return
		}
		writeZip(c, name, b, md5, "")
		return
	}
	parts := strings.SplitN(p, "/", 2)
	if len(parts) != 2 {
		httputil.Fail(c, httputil.NotFound("registry path not found"))
		return
	}
	content, err := h.svc.RegistryFile(aihubNS(), parts[0], parts[1])
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(content))
}

func (h *Handler) listAgentShares(c *gin.Context) {
	h.listResourceShares(c, "agent", c.Param("agentId"))
}
func (h *Handler) createAgentShare(c *gin.Context) {
	h.createResourceShare(c, "agent", c.Param("agentId"))
}

func (h *Handler) listWorkflows(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"items": []interface{}{}, "total": 0, "objectPrefix": "aihub:workflow", "status": "reserved"})
}
func (h *Handler) createWorkflow(c *gin.Context) {
	c.JSON(http.StatusAccepted, gin.H{"status": "reserved", "objectPrefix": "aihub:workflow"})
}
func (h *Handler) getWorkflow(c *gin.Context) {
	id := c.Param("workflowId")
	c.JSON(http.StatusOK, gin.H{"id": id, "object": "aihub:workflow:" + id, "status": "reserved"})
}
func (h *Handler) updateWorkflow(c *gin.Context) {
	c.JSON(http.StatusAccepted, gin.H{"status": "reserved", "object": "aihub:workflow:" + c.Param("workflowId")})
}
func (h *Handler) deleteWorkflow(c *gin.Context) {
	c.JSON(http.StatusAccepted, gin.H{"status": "reserved", "object": "aihub:workflow:" + c.Param("workflowId")})
}
func (h *Handler) runWorkflow(c *gin.Context) {
	c.JSON(http.StatusAccepted, gin.H{"status": "reserved", "object": "aihub:workflow:" + c.Param("workflowId"), "action": "run"})
}
func (h *Handler) listWorkflowShares(c *gin.Context) {
	h.listResourceShares(c, "workflow", c.Param("workflowId"))
}
func (h *Handler) createWorkflowShare(c *gin.Context) {
	h.createResourceShare(c, "workflow", c.Param("workflowId"))
}

func (h *Handler) listRuns(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"items": []interface{}{}, "total": 0, "objectPrefix": "aihub:run", "status": "reserved"})
}
func (h *Handler) getRun(c *gin.Context) {
	id := c.Param("runId")
	c.JSON(http.StatusOK, gin.H{"id": id, "object": "aihub:run:" + id, "status": "reserved"})
}
func (h *Handler) cancelRun(c *gin.Context) {
	c.JSON(http.StatusAccepted, gin.H{"status": "reserved", "object": "aihub:run:" + c.Param("runId"), "action": "cancel"})
}
func (h *Handler) retryRun(c *gin.Context) {
	c.JSON(http.StatusAccepted, gin.H{"status": "reserved", "object": "aihub:run:" + c.Param("runId"), "action": "retry"})
}

func (h *Handler) batchUploadCanonical(c *gin.Context) {
	// Support both repeated files and a single archive named "file" for old clients.
	form, err := c.MultipartForm()
	if err != nil {
		// Fall back to the single-file handler.
		h.uploadSkillCanonical(c)
		return
	}
	files := form.File["files"]
	if len(files) == 0 {
		files = form.File["file"]
	}
	if len(files) == 0 {
		httputil.Fail(c, httputil.BadRequest("files is required"))
		return
	}
	overwrite := parseBool(firstFormOrQuery(c, "overwrite"))
	var merged []byte
	if len(files) == 1 {
		rc, err := files[0].Open()
		if err != nil {
			httputil.Fail(c, err)
			return
		}
		defer rc.Close()
		merged, err = io.ReadAll(rc)
		if err != nil {
			httputil.Fail(c, err)
			return
		}
	} else {
		httputil.Fail(c, httputil.BadRequest("multi-file batch upload expects a pre-packed zip archive in this backend cut"))
		return
	}
	out, err := h.svc.BatchUploadSkillsFromZip(aihubNS(), merged, overwrite)
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, out)
}

func (h *Handler) createDraftCanonical(c *gin.Context) {
	vals := readParams(c)
	name := firstVal(vals, "skillName", "name")
	skill, err := parseDraftSkill(vals, aihubNS(), name)
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	version, err := h.svc.CreateDraft(aihubNS(), name, vals["basedOnVersion"], draftTargetVersion(vals), skill, vals["commitMsg"])
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, version)
}
func (h *Handler) updateDraftCanonical(c *gin.Context) {
	vals := readParams(c)
	name := firstVal(vals, "skillName", "name")
	skill, err := parseDraftSkill(vals, aihubNS(), name)
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	if skill == nil {
		httputil.Fail(c, httputil.BadRequest("skillCard is required"))
		return
	}
	if err := h.svc.UpdateDraft(aihubNS(), *skill, vals["commitMsg"]); err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, "ok")
}
func (h *Handler) deleteDraftCanonical(c *gin.Context) {
	name := firstFormOrQuery(c, "skillName")
	if name == "" {
		name = c.Query("skillName")
	}
	if err := h.svc.DeleteDraft(aihubNS(), name); err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, "ok")
}
func (h *Handler) forcePublishCanonical(c *gin.Context) {
	vals := readParams(c)
	if err := h.svc.Publish(aihubNS(), c.Param("skillName"), firstVal(vals, "version"), true, true); err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, "ok")
}
func (h *Handler) redraftCanonical(c *gin.Context) {
	vals := readParams(c)
	if err := h.svc.Redraft(aihubNS(), c.Param("skillName"), firstVal(vals, "version")); err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, "ok")
}
func (h *Handler) updateBizTagsCanonical(c *gin.Context) {
	vals := readParams(c)
	if err := h.svc.UpdateBizTags(aihubNS(), c.Param("skillName"), vals["bizTags"]); err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, "ok")
}
func (h *Handler) updateMetadataCanonical(c *gin.Context) {
	vals := readParams(c)
	metadata := model.Skill{SkillBase: model.SkillBase{SkillSet: vals["skillSet"], Groups: parseCSVorJSON(vals["groups"]), Keywords: parseCSVorJSON(vals["keywords"]), ModelName: vals["modelName"], ModelDescription: vals["modelDescription"], MatchHint: vals["matchHint"], Activation: vals["activation"]}}
	if vals["priority"] != "" {
		p, err := strconv.Atoi(vals["priority"])
		if err != nil {
			httputil.Fail(c, httputil.BadRequest("priority must be integer"))
			return
		}
		metadata.Priority = &p
	}
	if err := h.svc.UpdateMetadata(aihubNS(), c.Param("skillName"), metadata); err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, "ok")
}
func (h *Handler) updateScopeCanonical(c *gin.Context) {
	vals := readParams(c)
	if err := h.svc.UpdateScope(aihubNS(), c.Param("skillName"), vals["scope"]); err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, "ok")
}
