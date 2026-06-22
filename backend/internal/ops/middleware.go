package ops

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/actionlab-ai/aisphere-auth/pkg/aisphereauth"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/aisphereclient"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/auth"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/model"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/ports"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/service"
	"github.com/gin-gonic/gin"
)

type Metrics struct {
	start         time.Time
	mu            sync.Mutex
	RequestsTotal int64
	ErrorsTotal   int64
	ByPath        map[string]int64
	ByStatus      map[string]int64
}

func NewMetrics() *Metrics {
	return &Metrics{start: time.Now(), ByPath: map[string]int64{}, ByStatus: map[string]int64{}}
}
func (m *Metrics) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		m.mu.Lock()
		defer m.mu.Unlock()
		m.RequestsTotal++
		if c.Writer.Status() >= 400 {
			m.ErrorsTotal++
		}
		m.ByPath[c.FullPath()]++
		m.ByStatus[strconv.Itoa(c.Writer.Status())]++
	}
}
func (m *Metrics) Snapshot() model.MetricsSnapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	bp := map[string]int64{}
	bs := map[string]int64{}
	for k, v := range m.ByPath {
		bp[k] = v
	}
	for k, v := range m.ByStatus {
		bs[k] = v
	}
	return model.MetricsSnapshot{UptimeSeconds: int64(time.Since(m.start).Seconds()), RequestsTotal: m.RequestsTotal, ErrorsTotal: m.ErrorsTotal, ByPath: bp, ByStatus: bs}
}
func (m *Metrics) Register(r *gin.Engine) {
	r.GET("/metrics", func(c *gin.Context) {
		snap := m.Snapshot()
		c.String(200, "# TYPE aihub_requests_total counter\naihub_requests_total %d\n# TYPE aihub_errors_total counter\naihub_errors_total %d\n# TYPE aihub_uptime_seconds gauge\naihub_uptime_seconds %d\n", snap.RequestsTotal, snap.ErrorsTotal, snap.UptimeSeconds)
	})
	r.GET("/v3/admin/metrics", func(c *gin.Context) { c.JSON(200, gin.H{"code": 0, "message": "success", "data": m.Snapshot()}) })
}

type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string][]time.Time
	limit   int
	window  time.Duration
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	if limit <= 0 {
		limit = 120
	}
	if window <= 0 {
		window = time.Minute
	}
	return &RateLimiter{buckets: map[string][]time.Time{}, limit: limit, window: window}
}
func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodGet {
			c.Next()
			return
		}
		key := c.ClientIP()
		now := time.Now()
		rl.mu.Lock()
		arr := rl.buckets[key]
		keep := arr[:0]
		for _, t := range arr {
			if now.Sub(t) < rl.window {
				keep = append(keep, t)
			}
		}
		if len(keep) >= rl.limit {
			rl.buckets[key] = keep
			rl.mu.Unlock()
			c.AbortWithStatusJSON(429, gin.H{"code": 429, "message": "rate limit exceeded"})
			return
		}
		keep = append(keep, now)
		rl.buckets[key] = keep
		rl.mu.Unlock()
		c.Next()
	}
}

type captureWriter struct {
	gin.ResponseWriter
	buf bytes.Buffer
}

func (w *captureWriter) Write(b []byte) (int, error) {
	w.buf.Write(b)
	return w.ResponseWriter.Write(b)
}
func IdempotencyMiddleware(svc *service.Service, ttl time.Duration) gin.HandlerFunc {
	if ttl <= 0 {
		ttl = time.Hour
	}
	return func(c *gin.Context) {
		key := c.GetHeader("Idempotency-Key")
		if key == "" || (c.Request.Method != http.MethodPost && c.Request.Method != http.MethodPut && c.Request.Method != http.MethodDelete) {
			c.Next()
			return
		}
		body, _ := io.ReadAll(c.Request.Body)
		c.Request.Body = io.NopCloser(bytes.NewReader(body))
		sum := sha256.Sum256(append([]byte(c.Request.Method+":"+c.Request.URL.Path+":"), body...))
		reqHash := hex.EncodeToString(sum[:])
		if rec, err := svc.LoadIdempotency(key); err == nil && rec != nil {
			if rec.RequestHash != reqHash {
				c.AbortWithStatusJSON(409, gin.H{"code": 409, "message": "idempotency key reused with different request"})
				return
			}
			c.Header("X-Idempotent-Replay", "true")
			c.Data(rec.StatusCode, "application/json; charset=utf-8", []byte(rec.ResponseBody))
			return
		}
		cw := &captureWriter{ResponseWriter: c.Writer}
		c.Writer = cw
		c.Next()
		status := cw.Status()
		if status >= 200 && status < 300 {
			_ = svc.SaveIdempotency(&model.IdempotencyRecord{Key: key, Method: c.Request.Method, Path: c.Request.URL.Path, RequestHash: reqHash, StatusCode: status, ResponseBody: cw.buf.String(), CreateTime: model.NowMillis(), ExpiresAt: model.NowMillis() + int64(ttl/time.Millisecond)})
		}
		c.Header("X-Idempotency-Key", fmt.Sprint(key))
	}
}

// DistributedRateLimiter uses the configured Cache port. With Redis this becomes
// a process-independent global limiter; with memory cache it remains local.
type DistributedRateLimiter struct {
	cache  ports.Cache
	limit  int64
	window time.Duration
}

func NewDistributedRateLimiter(c ports.Cache, limit int, window time.Duration) *DistributedRateLimiter {
	if limit <= 0 {
		limit = 120
	}
	if window <= 0 {
		window = time.Minute
	}
	return &DistributedRateLimiter{cache: c, limit: int64(limit), window: window}
}
func (rl *DistributedRateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodGet || rl.cache == nil {
			c.Next()
			return
		}
		bucket := time.Now().Unix() / int64(rl.window.Seconds())
		key := fmt.Sprintf("ratelimit:write:%s:%d", c.ClientIP(), bucket)
		n, err := rl.cache.IncrBy(c.Request.Context(), key, 1)
		if err == nil && n == 1 {
			_ = rl.cache.Expire(c.Request.Context(), key, rl.window+5*time.Second)
		}
		if err == nil && n > rl.limit {
			c.AbortWithStatusJSON(429, gin.H{"code": 429, "message": "rate limit exceeded", "scope": "global"})
			return
		}
		c.Next()
	}
}

func AuditMiddleware(svc *service.Service) gin.HandlerFunc {
	return AuditMiddlewareWithAISphere(svc, nil)
}

func AuditMiddlewareWithAISphere(svc *service.Service, aisphere *aisphereclient.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodGet || svc == nil {
			c.Next()
			return
		}
		c.Next()
		if c.Writer.Status() < 200 || c.Writer.Status() >= 300 {
			return
		}
		operator := "anonymous"
		actorName := ""
		if v, ok := c.Get("principal"); ok {
			if p, ok := v.(auth.Principal); ok && p.SubjectID != "" {
				operator = firstNonEmpty(p.ExternalSubject, p.SubjectID)
				actorName = firstNonEmpty(p.Username, p.Email, p.SubjectID)
			}
		}
		ns := c.Query("namespaceId")
		if ns == "" {
			ns = c.PostForm("namespaceId")
		}
		if ns == "" {
			ns = c.Param("namespaceId")
		}
		name := c.Query("skillName")
		if name == "" {
			name = c.Query("name")
		}
		if name == "" {
			name = c.PostForm("skillName")
		}
		resourceType := inferAuditResource(c.Request.URL.Path)
		action := auditAction(c.Request.Method, c.FullPath(), c.Request.URL.Path)
		_ = svc.AppendAudit(model.AuditLog{NamespaceID: ns, ResourceType: resourceType, ResourceName: name, Action: action, Operator: operator, RequestID: c.GetHeader("X-Request-ID"), Detail: map[string]interface{}{"path": c.Request.URL.Path, "status": c.Writer.Status()}})
		if aisphere != nil {
			_, _ = aisphere.WriteAudit(c.Request.Context(), aisphereauth.AuditEvent{
				TraceID:       c.GetHeader("X-Request-ID"),
				ActorSubject:  operator,
				ActorName:     actorName,
				ResourceType:  resourceType,
				ResourceID:    name,
				Action:        action,
				Result:        aisphereauth.AuditResultSuccess,
				IP:            c.ClientIP(),
				UserAgent:     c.Request.UserAgent(),
				RequestPath:   c.Request.URL.Path,
				RequestMethod: c.Request.Method,
				Metadata: map[string]string{
					"namespaceId": ns,
					"status":      strconv.Itoa(c.Writer.Status()),
				},
			})
		}
	}
}
func inferAuditResource(path string) string {
	if strings.Contains(path, "skill-proposals") {
		return "proposal"
	}
	if strings.Contains(path, "skill-groups") {
		return "group"
	}
	if strings.Contains(path, "namespaces") {
		return "namespace"
	}
	if strings.Contains(path, "iam") {
		return "iam"
	}
	if strings.Contains(path, "skills") {
		return "skill"
	}
	return "http"
}

func auditAction(method, route, path string) string {
	method = strings.ToUpper(method)
	p := strings.ToLower(path + " " + route)
	switch {
	case strings.Contains(p, "approve"):
		return "proposal.approve"
	case strings.Contains(p, "reject"):
		return "proposal.reject"
	case strings.Contains(p, "publish"):
		return "skill.publish"
	case strings.Contains(p, "rollback"):
		return "skill.rollback"
	case strings.Contains(p, "skill-proposals") && method == http.MethodPost:
		return "proposal.create"
	case strings.Contains(p, "skill-proposals"):
		return "proposal.read"
	case strings.Contains(p, "skill-groups") || strings.Contains(p, "/group"):
		return "group." + methodAction(method)
	case strings.Contains(p, "skills") || strings.Contains(p, "/skill"):
		return "skill." + methodAction(method)
	default:
		return strings.ToLower(method) + " " + route
	}
}

func methodAction(method string) string {
	switch method {
	case http.MethodPost:
		return "create"
	case http.MethodPut, http.MethodPatch:
		return "update"
	case http.MethodDelete:
		return "delete"
	default:
		return "read"
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
