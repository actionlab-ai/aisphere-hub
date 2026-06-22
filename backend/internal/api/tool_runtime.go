package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/actionlab-ai/aisphere-hub/backend/internal/httputil"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/model"
	"github.com/gin-gonic/gin"
)

type resolveToolRuntimeRequest struct {
	RuntimeID string `json:"runtimeId"`
	SessionID string `json:"sessionId"`
	Version   string `json:"version,omitempty"`
	Label     string `json:"label,omitempty"`
}

type toolRuntimeSnapshotItem struct {
	Name          string                        `json:"name"`
	Version       string                        `json:"version"`
	Revision      string                        `json:"revision"`
	Object        string                        `json:"object"`
	Status        string                        `json:"status,omitempty"`
	Runtime       model.ToolRuntimeDefinition   `json:"runtime"`
	Execution     model.ToolExecutionDefinition `json:"execution,omitempty"`
	InputSchema   map[string]interface{}        `json:"inputSchema,omitempty"`
	OutputSchema  map[string]interface{}        `json:"outputSchema,omitempty"`
	TimeoutMillis int64                         `json:"timeoutMillis,omitempty"`
	Retry         *model.ToolRetryPolicy        `json:"retry,omitempty"`
	Metadata      map[string]interface{}        `json:"metadata,omitempty"`
}

type toolRuntimeSnapshot struct {
	SnapshotID  string                  `json:"snapshotId"`
	RuntimeID   string                  `json:"runtimeId"`
	SessionID   string                  `json:"sessionId"`
	GeneratedAt string                  `json:"generatedAt"`
	Tool        toolRuntimeSnapshotItem `json:"tool"`
}

func (h *Handler) resolveToolRuntime(c *gin.Context) {
	id := strings.TrimSpace(c.Param("toolId"))
	if allowed, access := h.catalogCan(c, "tool", id, "run"); !allowed {
		forbiddenTool(c, id, "run", access)
		return
	}
	var req resolveToolRuntimeRequest
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
	item, err := h.resolveAgentTool(c, model.AgentToolRef{Name: id, Version: req.Version, Label: req.Label, Required: true})
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, toolRuntimeSnapshot{
		SnapshotID:  "tool_snap_" + shortHash(req.RuntimeID+"|"+req.SessionID+"|"+item.Name+"|"+item.Version+"|"+item.Revision),
		RuntimeID:   req.RuntimeID,
		SessionID:   req.SessionID,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Tool:        *item,
	})
}

func (h *Handler) reportToolFailure(c *gin.Context) {
	var req model.ToolFailureReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.Fail(c, httputil.BadRequest(err.Error()))
		return
	}
	toolID := strings.TrimSpace(req.ToolID)
	if toolID == "" {
		toolID = strings.TrimSpace(c.Param("toolId"))
		req.ToolID = toolID
	}
	if toolID == "" {
		httputil.Fail(c, httputil.BadRequest("tool id is required"))
		return
	}
	if allowed, access := h.catalogCan(c, "tool", toolID, "run"); !allowed {
		forbiddenTool(c, toolID, "run", access)
		return
	}
	if strings.TrimSpace(req.AgentID) != "" {
		if allowed, access := h.catalogCan(c, "agent", req.AgentID, "run"); !allowed {
			forbiddenAgent(c, req.AgentID, "run", access)
			return
		}
	}
	rec, err := h.svc.ReportToolFailure(aihubNS(), req, agentOperator(c), h.catalogObject("tool", toolID))
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	c.JSON(http.StatusAccepted, rec)
}

func (h *Handler) listToolFailures(c *gin.Context) {
	q := model.ToolFailureQuery{
		ToolID:     strings.TrimSpace(firstQuery(c, "toolId", "tool")),
		AgentID:    strings.TrimSpace(firstQuery(c, "agentId", "agent")),
		RuntimeID:  strings.TrimSpace(c.Query("runtimeId")),
		SessionID:  strings.TrimSpace(c.Query("sessionId")),
		RunID:      strings.TrimSpace(c.Query("runId")),
		TraceID:    strings.TrimSpace(c.Query("traceId")),
		SnapshotID: strings.TrimSpace(c.Query("snapshotId")),
		Limit:      atoiDefault(c.Query("limit"), 100),
	}
	if q.ToolID != "" {
		if allowed, access := h.catalogCan(c, "tool", q.ToolID, "read"); !allowed {
			forbiddenTool(c, q.ToolID, "read", access)
			return
		}
	}
	items, total, err := h.svc.ListToolFailures(q)
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	filtered := make([]*model.ToolFailureRecord, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		if q.ToolID == "" {
			if allowed, _ := h.catalogCan(c, "tool", item.ToolID, "read"); !allowed {
				continue
			}
		}
		filtered = append(filtered, item)
	}
	visibleTotal := total
	if q.ToolID == "" {
		visibleTotal = int64(len(filtered))
	}
	c.JSON(http.StatusOK, gin.H{"items": filtered, "total": visibleTotal, "visible": len(filtered)})
}
