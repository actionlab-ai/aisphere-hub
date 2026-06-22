package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/actionlab-ai/aisphere-hub/backend/internal/httputil"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/model"
	"github.com/gin-gonic/gin"
)

func (h *Handler) listSandboxTools(c *gin.Context) {
	id := strings.TrimSpace(c.Param("sandboxId"))
	if allowed, access := h.catalogCan(c, "sandbox", id, "read"); !allowed {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "object": h.catalogObject("sandbox", id), "action": "read", "reason": access.Reason})
		return
	}
	out, err := h.svc.ListSandboxTools(c.Request.Context(), id)
	if err != nil {
		failSandbox(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) callSandboxTool(c *gin.Context) {
	id := strings.TrimSpace(c.Param("sandboxId"))
	if allowed, access := h.catalogCan(c, "sandbox", id, "run"); !allowed {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "object": h.catalogObject("sandbox", id), "action": "run", "reason": access.Reason})
		return
	}
	var req model.SandboxToolCallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.Fail(c, httputil.BadRequest(err.Error()))
		return
	}
	req.Tool = strings.TrimSpace(firstNonEmptyString(req.Tool, c.Query("tool")))
	if req.Tool == "" {
		httputil.Fail(c, httputil.BadRequest("tool is required"))
		return
	}
	if allowed, access := h.catalogCan(c, "tool", req.Tool, "run"); !allowed {
		forbiddenTool(c, req.Tool, "run", access)
		return
	}
	out, err := h.svc.CallSandboxTool(c.Request.Context(), id, req)
	if err != nil {
		failSandbox(c, err)
		return
	}
	if out != nil && !out.OK {
		_, _ = h.svc.ReportToolFailure(aihubNS(), model.ToolFailureReportRequest{
			ToolID:         req.Tool,
			RuntimeID:      principalID(c),
			RunID:          req.RunID,
			TraceID:        req.TraceID,
			Attempt:        req.Attempt,
			ErrorCode:      sandboxToolErrorCode(out),
			ErrorMessage:   sandboxToolErrorMessage(out),
			InputPreview:   truncateForFailure(req.Input),
			DurationMillis: out.DurationMillis,
			Metadata:       map[string]interface{}{"sandboxId": id},
		}, agentOperator(c), h.catalogObject("tool", req.Tool))
	}
	status := http.StatusOK
	if out != nil && !out.OK {
		status = http.StatusBadGateway
	}
	c.JSON(status, out)
}

func sandboxToolErrorCode(out *model.SandboxToolCallResult) string {
	if out != nil && out.Error != nil && out.Error.Code != "" {
		return out.Error.Code
	}
	return "SANDBOX_TOOL_FAILED"
}

func sandboxToolErrorMessage(out *model.SandboxToolCallResult) string {
	if out != nil && out.Error != nil && out.Error.Message != "" {
		return out.Error.Message
	}
	if out != nil && out.Trace != "" {
		return out.Trace
	}
	return "sandbox tool failed"
}

func truncateForFailure(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	if len(b) > 2048 {
		return string(b[:2048])
	}
	return string(b)
}
