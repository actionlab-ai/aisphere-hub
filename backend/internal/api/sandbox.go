package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/actionlab-ai/aisphere-hub/backend/internal/httputil"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/model"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/sandbox"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/service"
	"github.com/gin-gonic/gin"
)

func (h *Handler) ensureSandbox(c *gin.Context) {
	var req model.SandboxEnsureRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.Fail(c, httputil.BadRequest(err.Error()))
		return
	}
	if strings.TrimSpace(req.RuntimeID) == "" {
		req.RuntimeID = principalID(c)
	}
	if strings.TrimSpace(req.OwnerSubject) == "" {
		req.OwnerSubject = catalogSubject(currentPrincipal(c))
	}
	if strings.TrimSpace(req.OwnerSubject) == "" {
		req.OwnerSubject = principalID(c)
	}
	if strings.TrimSpace(req.OrgID) == "" {
		if p := currentPrincipal(c); p != nil {
			req.OrgID = p.Organization
		}
	}
	resourceID := firstNonEmptyString(req.SandboxID, req.SessionID, req.RunID, "new")
	if allowed, access := h.catalogCan(c, "sandbox", resourceID, "run"); !allowed {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "object": h.catalogObject("sandbox", resourceID), "action": "run", "reason": access.Reason})
		return
	}
	st, err := h.svc.EnsureSandbox(c.Request.Context(), req)
	if err != nil {
		failSandbox(c, err)
		return
	}
	c.JSON(http.StatusOK, st)
}

func (h *Handler) listSandboxes(c *gin.Context) {
	resourceID := firstNonEmptyString(c.Query("orgId"), c.Query("projectId"), "all")
	if allowed, access := h.catalogCan(c, "sandbox", resourceID, "read"); !allowed {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "object": h.catalogObject("sandbox", resourceID), "action": "read", "reason": access.Reason})
		return
	}
	items, err := h.svc.ListSandboxes(c.Request.Context(), sandbox.ListQuery{
		OwnerSubject: strings.TrimSpace(c.Query("ownerSubject")),
		OrgID:        strings.TrimSpace(c.Query("orgId")),
		ProjectID:    strings.TrimSpace(c.Query("projectId")),
		SessionID:    strings.TrimSpace(c.Query("sessionId")),
		AgentID:      strings.TrimSpace(c.Query("agentId")),
	})
	if err != nil {
		failSandbox(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": len(items)})
}

func (h *Handler) getSandbox(c *gin.Context) {
	id := strings.TrimSpace(c.Param("sandboxId"))
	if allowed, access := h.catalogCan(c, "sandbox", id, "read"); !allowed {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "object": h.catalogObject("sandbox", id), "action": "read", "reason": access.Reason})
		return
	}
	st, err := h.svc.GetSandbox(c.Request.Context(), id)
	if err != nil {
		failSandbox(c, err)
		return
	}
	c.JSON(http.StatusOK, st)
}

func (h *Handler) restartSandbox(c *gin.Context) {
	id := strings.TrimSpace(c.Param("sandboxId"))
	if allowed, access := h.catalogCan(c, "sandbox", id, "run"); !allowed {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "object": h.catalogObject("sandbox", id), "action": "run", "reason": access.Reason})
		return
	}
	st, err := h.svc.RestartSandbox(c.Request.Context(), id)
	if err != nil {
		failSandbox(c, err)
		return
	}
	c.JSON(http.StatusOK, st)
}

func (h *Handler) deleteSandbox(c *gin.Context) {
	id := strings.TrimSpace(c.Param("sandboxId"))
	if allowed, access := h.catalogCan(c, "sandbox", id, "delete"); !allowed {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "object": h.catalogObject("sandbox", id), "action": "delete", "reason": access.Reason})
		return
	}
	var req model.SandboxDeleteRequest
	_ = c.ShouldBindJSON(&req)
	if err := h.svc.DeleteSandbox(c.Request.Context(), id, req.DeleteWorkspace || strings.EqualFold(c.Query("deleteWorkspace"), "true")); err != nil {
		failSandbox(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"ok": true, "sandboxId": id})
}

func (h *Handler) sandboxLogs(c *gin.Context) {
	id := strings.TrimSpace(c.Param("sandboxId"))
	if allowed, access := h.catalogCan(c, "sandbox", id, "read"); !allowed {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "object": h.catalogObject("sandbox", id), "action": "read", "reason": access.Reason})
		return
	}
	q := model.SandboxLogQuery{Container: strings.TrimSpace(c.Query("container")), TailLines: int64(atoiDefault(c.Query("tailLines"), 200))}
	logs, err := h.svc.SandboxLogs(c.Request.Context(), id, q)
	if err != nil {
		failSandbox(c, err)
		return
	}
	c.String(http.StatusOK, logs)
}

func failSandbox(c *gin.Context, err error) {
	if errors.Is(err, service.ErrSandboxDisabled) {
		httputil.Fail(c, &httputil.AppError{Status: http.StatusServiceUnavailable, Code: 503, Message: err.Error()})
		return
	}
	httputil.Fail(c, err)
}
