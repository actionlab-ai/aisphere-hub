package httputil

import (
	"errors"
	"net/http"

	"github.com/actionlab-ai/aisphere-hub/backend/internal/model"
	"github.com/gin-gonic/gin"
)

type AppError struct {
	Status  int
	Code    int
	Message string
}

func (e *AppError) Error() string { return e.Message }

func BadRequest(msg string) *AppError {
	return &AppError{Status: http.StatusBadRequest, Code: 400, Message: msg}
}
func NotFound(msg string) *AppError {
	return &AppError{Status: http.StatusNotFound, Code: 404, Message: msg}
}
func Conflict(msg string) *AppError {
	return &AppError{Status: http.StatusConflict, Code: 409, Message: msg}
}
func Internal(msg string) *AppError {
	return &AppError{Status: http.StatusInternalServerError, Code: 500, Message: msg}
}

func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, model.Result{Code: 0, Message: "success", Data: data})
}

func Fail(c *gin.Context, err error) {
	var app *AppError
	if errors.As(err, &app) {
		c.JSON(app.Status, model.Result{Code: app.Code, Message: app.Message})
		return
	}
	c.JSON(http.StatusInternalServerError, model.Result{Code: 500, Message: err.Error()})
}

// Namespace is kept only as a compatibility shim for old Nacos-style APIs.
// The independent AIHub registry is group-first and namespace-free at the
// skill resource layer, so any incoming namespaceId is intentionally ignored.
// Access isolation must be enforced by IAM/ABAC policies, not by skill keys.
func Namespace(v string) string {
	return model.DefaultNamespace
}

// DeprecatedNamespaceWarningHeader is emitted by legacy Nacos-compatible
// endpoints when a namespaceId query/path parameter is supplied.
const DeprecatedNamespaceWarningHeader = "X-AIHub-Warning"
