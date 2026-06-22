package casdoorremote

import (
	"net/http"

	core "github.com/actionlab-ai/aisphere-hub/backend/internal/auth"
	"github.com/gin-gonic/gin"
)

type evaluateRequest struct {
	Subject string `json:"subject"`
	Object  string `json:"object"`
	Action  string `json:"action"`
}

func RegisterAccessRoutes(r *gin.Engine, az *Authorizer) {
	h := &handler{az: az}
	g := r.Group("/v3/admin/access")
	g.GET("/overview", h.overview)
	g.GET("/resources", h.resources)
	g.GET("/links", h.links)
	g.POST("/evaluate", h.evaluate)
}

type handler struct{ az *Authorizer }

func (h *handler) overview(c *gin.Context) {
	p := currentPrincipal(c)
	c.JSON(http.StatusOK, gin.H{"data": h.az.Overview(p)})
}

func (h *handler) resources(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"items": ResourceTemplates(), "total": len(ResourceTemplates())}})
}

func (h *handler) links(c *gin.Context) {
	links := h.az.QuickLinks()
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"items": links, "total": len(links)}})
}

func (h *handler) evaluate(c *gin.Context) {
	var req evaluateRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Subject == "" || req.Object == "" || req.Action == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "subject, object and action are required"})
		return
	}
	allowed, err := h.az.Enforce(c.Request.Context(), req.Subject, req.Object, req.Action)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "casdoor_enforce_failed", "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"allowed": allowed, "subject": req.Subject, "object": req.Object, "action": req.Action, "provider": "casdoor-remote"}})
}

func currentPrincipal(c *gin.Context) *core.Principal {
	v, ok := c.Get(core.PrincipalContextKey)
	if !ok {
		return nil
	}
	switch p := v.(type) {
	case core.Principal:
		return &p
	case *core.Principal:
		return p
	default:
		return nil
	}
}
