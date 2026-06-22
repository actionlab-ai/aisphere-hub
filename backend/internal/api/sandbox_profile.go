package api

import (
	"github.com/actionlab-ai/aisphere-hub/backend/internal/httputil"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/model"
	"github.com/gin-gonic/gin"
)

func (h *Handler) listSandboxProfiles(c *gin.Context) {
	out, err := h.svc.ListSandboxProfiles(model.DefaultNamespace)
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, out)
}
func (h *Handler) getSandboxProfile(c *gin.Context) {
	out, err := h.svc.GetSandboxProfile(model.DefaultNamespace, c.Param("profileId"))
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, out)
}
func (h *Handler) saveSandboxProfile(c *gin.Context) {
	var p model.SandboxProfile
	if err := c.ShouldBindJSON(&p); err != nil {
		httputil.Fail(c, httputil.BadRequest(err.Error()))
		return
	}
	if p.ID == "" {
		p.ID = c.Param("profileId")
	}
	out, err := h.svc.SaveSandboxProfile(model.DefaultNamespace, &p)
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, out)
}
func (h *Handler) deleteSandboxProfile(c *gin.Context) {
	if err := h.svc.DeleteSandboxProfile(model.DefaultNamespace, c.Param("profileId")); err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, "ok")
}
func (h *Handler) listSandboxPolicies(c *gin.Context) {
	out, err := h.svc.ListSandboxPolicies(model.DefaultNamespace)
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, out)
}
func (h *Handler) getSandboxPolicy(c *gin.Context) {
	out, err := h.svc.GetSandboxPolicy(model.DefaultNamespace, c.Param("policyId"))
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, out)
}
func (h *Handler) saveSandboxPolicy(c *gin.Context) {
	var p model.SandboxPolicy
	if err := c.ShouldBindJSON(&p); err != nil {
		httputil.Fail(c, httputil.BadRequest(err.Error()))
		return
	}
	if p.ID == "" {
		p.ID = c.Param("policyId")
	}
	out, err := h.svc.SaveSandboxPolicy(model.DefaultNamespace, &p)
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, out)
}
func (h *Handler) deleteSandboxPolicy(c *gin.Context) {
	if err := h.svc.DeleteSandboxPolicy(model.DefaultNamespace, c.Param("policyId")); err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, "ok")
}
