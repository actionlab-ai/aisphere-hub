package api

import (
	"net/http"
	"strings"

	"github.com/actionlab-ai/aisphere-hub/backend/internal/httputil"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/model"
	"github.com/gin-gonic/gin"
)

type toolListItem struct {
	ID            string        `json:"id"`
	DisplayName   string        `json:"displayName,omitempty"`
	Description   string        `json:"description,omitempty"`
	Status        string        `json:"status"`
	Scope         string        `json:"scope,omitempty"`
	LatestVersion string        `json:"latestVersion"`
	RuntimeType   string        `json:"runtimeType,omitempty"`
	RuntimeName   string        `json:"runtimeName,omitempty"`
	Object        string        `json:"object"`
	UpdateTime    int64         `json:"updateTime"`
	Access        catalogAccess `json:"access"`
}

func (h *Handler) listTools(c *gin.Context) {
	records, err := h.svc.ListTools(aihubNS())
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	query := strings.ToLower(strings.TrimSpace(firstQuery(c, "q", "search")))
	items := make([]toolListItem, 0, len(records))
	for _, record := range records {
		if record == nil {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(record.ID+" "+record.DisplayName+" "+record.Description), query) {
			continue
		}
		allowed, access := h.catalogCan(c, "tool", record.ID, "read")
		if !allowed {
			continue
		}
		latest := record.Versions[record.LatestVersion]
		item := toolListItem{
			ID: record.ID, DisplayName: record.DisplayName, Description: record.Description,
			Status: record.Status, Scope: record.Scope, LatestVersion: record.LatestVersion,
			Object: h.catalogObject("tool", record.ID), UpdateTime: record.UpdateTime, Access: access,
		}
		if latest != nil {
			item.RuntimeType = latest.Definition.Runtime.Type
			item.RuntimeName = firstNonEmptyString(latest.Definition.Runtime.Name, latest.Definition.Runtime.EntryPoint, latest.Definition.Runtime.URL)
		}
		items = append(items, item)
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": len(items)})
}

func (h *Handler) createTool(c *gin.Context) {
	var req model.ToolUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.Fail(c, httputil.BadRequest(err.Error()))
		return
	}
	if allowed, access := h.catalogCan(c, "tool", req.ID, "create"); !allowed {
		forbiddenTool(c, req.ID, "create", access)
		return
	}
	record, err := h.svc.CreateTool(aihubNS(), req, agentOperator(c))
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, toolResponse(h, record))
}

func (h *Handler) getTool(c *gin.Context) {
	id := strings.TrimSpace(c.Param("toolId"))
	if allowed, access := h.catalogCan(c, "tool", id, "read"); !allowed {
		forbiddenTool(c, id, "read", access)
		return
	}
	record, err := h.svc.GetTool(aihubNS(), id)
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, toolResponse(h, record))
}

func (h *Handler) updateTool(c *gin.Context) {
	id := strings.TrimSpace(c.Param("toolId"))
	var req model.ToolUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.Fail(c, httputil.BadRequest(err.Error()))
		return
	}
	if allowed, access := h.catalogCan(c, "tool", id, "update"); !allowed {
		forbiddenTool(c, id, "update", access)
		return
	}
	record, err := h.svc.UpdateTool(aihubNS(), id, req, agentOperator(c))
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, toolResponse(h, record))
}

func (h *Handler) deleteTool(c *gin.Context) {
	id := strings.TrimSpace(c.Param("toolId"))
	if allowed, access := h.catalogCan(c, "tool", id, "delete"); !allowed {
		forbiddenTool(c, id, "delete", access)
		return
	}
	if err := h.svc.DeleteTool(aihubNS(), id); err != nil {
		httputil.Fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func toolResponse(h *Handler, record *model.ToolRecord) gin.H {
	return gin.H{"tool": record, "object": h.catalogObject("tool", record.ID), "latestVersion": record.LatestVersion}
}

func forbiddenTool(c *gin.Context, id, action string, access catalogAccess) {
	c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "object": "aihub:tool:" + id, "action": action, "reason": access.Reason})
}

func (h *Handler) listToolShares(c *gin.Context) {
	h.listResourceShares(c, "tool", c.Param("toolId"))
}
func (h *Handler) createToolShare(c *gin.Context) {
	h.createResourceShare(c, "tool", c.Param("toolId"))
}
