package api

import (
	"net/http"
	"strings"

	"github.com/actionlab-ai/aisphere-hub/backend/internal/httputil"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/model"
	"github.com/gin-gonic/gin"
)

type agentListItem struct {
	ID            string        `json:"id"`
	DisplayName   string        `json:"displayName,omitempty"`
	Description   string        `json:"description,omitempty"`
	Status        string        `json:"status"`
	Scope         string        `json:"scope,omitempty"`
	LatestVersion string        `json:"latestVersion"`
	Object        string        `json:"object"`
	UpdateTime    int64         `json:"updateTime"`
	Access        catalogAccess `json:"access"`
}

func (h *Handler) listAgents(c *gin.Context) {
	records, err := h.svc.ListAgents(aihubNS())
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	items := make([]agentListItem, 0, len(records))
	for _, record := range records {
		if record == nil {
			continue
		}
		allowed, access := h.catalogCan(c, "agent", record.ID, "read")
		if !allowed {
			continue
		}
		items = append(items, agentListItem{
			ID: record.ID, DisplayName: record.DisplayName, Description: record.Description,
			Status: record.Status, Scope: record.Scope, LatestVersion: record.LatestVersion,
			Object: h.catalogObject("agent", record.ID), UpdateTime: record.UpdateTime, Access: access,
		})
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": len(items)})
}

func (h *Handler) createAgent(c *gin.Context) {
	var req model.AgentUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.Fail(c, httputil.BadRequest(err.Error()))
		return
	}
	if allowed, access := h.catalogCan(c, "agent", req.ID, "create"); !allowed {
		forbiddenAgent(c, req.ID, "create", access)
		return
	}
	record, err := h.svc.CreateAgent(aihubNS(), req, agentOperator(c))
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, agentResponse(h, record))
}

func (h *Handler) getAgent(c *gin.Context) {
	id := strings.TrimSpace(c.Param("agentId"))
	if allowed, access := h.catalogCan(c, "agent", id, "read"); !allowed {
		forbiddenAgent(c, id, "read", access)
		return
	}
	record, err := h.svc.GetAgent(aihubNS(), id)
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, agentResponse(h, record))
}

func (h *Handler) updateAgent(c *gin.Context) {
	id := strings.TrimSpace(c.Param("agentId"))
	var req model.AgentUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.Fail(c, httputil.BadRequest(err.Error()))
		return
	}
	if allowed, access := h.catalogCan(c, "agent", id, "update"); !allowed {
		forbiddenAgent(c, id, "update", access)
		return
	}
	record, err := h.svc.UpdateAgent(aihubNS(), id, req, agentOperator(c))
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, agentResponse(h, record))
}

func (h *Handler) deleteAgent(c *gin.Context) {
	id := strings.TrimSpace(c.Param("agentId"))
	if allowed, access := h.catalogCan(c, "agent", id, "delete"); !allowed {
		forbiddenAgent(c, id, "delete", access)
		return
	}
	if err := h.svc.DeleteAgent(aihubNS(), id); err != nil {
		httputil.Fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func agentResponse(h *Handler, record *model.AgentRecord) gin.H {
	return gin.H{"agent": record, "object": h.catalogObject("agent", record.ID), "latestVersion": record.LatestVersion}
}

func agentOperator(c *gin.Context) string {
	if subject := catalogSubject(currentPrincipal(c)); subject != "" {
		return subject
	}
	return principalID(c)
}

func forbiddenAgent(c *gin.Context, id, action string, access catalogAccess) {
	c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "object": "aihub:agent:" + id, "action": action, "reason": access.Reason})
}
