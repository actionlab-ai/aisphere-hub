package api

import (
	"github.com/actionlab-ai/aisphere-hub/backend/internal/httputil"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/model"
	"github.com/gin-gonic/gin"
)

func (h *Handler) listModelProfiles(c *gin.Context) {
	out, err := h.svc.ListModelProfiles(model.DefaultNamespace)
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, out)
}
func (h *Handler) getModelProfile(c *gin.Context) {
	out, err := h.svc.GetModelProfile(model.DefaultNamespace, c.Param("profileId"))
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, out)
}
func (h *Handler) saveModelProfile(c *gin.Context) {
	var p model.ModelProfile
	if err := c.ShouldBindJSON(&p); err != nil {
		httputil.Fail(c, httputil.BadRequest(err.Error()))
		return
	}
	if p.ID == "" {
		p.ID = c.Param("profileId")
	}
	out, err := h.svc.SaveModelProfile(model.DefaultNamespace, &p)
	if err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, out)
}
func (h *Handler) deleteModelProfile(c *gin.Context) {
	if err := h.svc.DeleteModelProfile(model.DefaultNamespace, c.Param("profileId")); err != nil {
		httputil.Fail(c, err)
		return
	}
	httputil.OK(c, "ok")
}
