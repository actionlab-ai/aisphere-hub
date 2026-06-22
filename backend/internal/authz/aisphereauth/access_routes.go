package aisphereauth

import (
	"net/http"

	core "github.com/actionlab-ai/aisphere-hub/backend/internal/auth"
	"github.com/gin-gonic/gin"
)

// RegisterAccessRoutes mounts a small diagnostics surface that mirrors
// the casdoor-remote Access page. It lets the AIHub console show
// whether the aisphere-auth integration is healthy and what subject /
// object / action a given request would resolve to.
func RegisterAccessRoutes(r *gin.Engine, a *Authorizer) {
	if a == nil || a.client == nil {
		return
	}
	grp := r.Group("/v3/admin/access/aisphere")
	grp.GET("/overview", func(c *gin.Context) {
		p := CurrentPrincipal(c)
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "success",
			"data":    a.Overview(p),
		})
	})
	grp.POST("/evaluate", func(c *gin.Context) {
		var req struct {
			Subject string `json:"subject"`
			Object  string `json:"object"`
			Action  string `json:"action"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
			return
		}
		allowed, reason, err := a.client.Check(req.Subject, req.Object, req.Action)
		resp := gin.H{
			"code": 0,
			"data": gin.H{
				"allow":   allowed,
				"reason":  reason,
				"subject": req.Subject,
				"object":  req.Object,
				"action":  req.Action,
			},
		}
		if err != nil {
			resp["error"] = err.Error()
		}
		c.JSON(http.StatusOK, resp)
	})
}

// CurrentPrincipal pulls the principal stored by the auth middleware.
// Kept here to avoid an import cycle on authhttp.
func CurrentPrincipal(c *gin.Context) *core.Principal {
	v, ok := c.Get(core.PrincipalContextKey)
	if !ok {
		return nil
	}
	if p, ok := v.(core.Principal); ok {
		return &p
	}
	return nil
}
