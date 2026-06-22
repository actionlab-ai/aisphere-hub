package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// RegisterWeb serves the standalone AIHub Console built by ../front.
//
// Important for Gin:
// Do not register /ui/*path together with /ui or /ui/assets/*filepath.
// Gin treats catch-all routes very strictly and will panic on conflicts.
// We only register exact /ui and /ui/ plus /ui/assets/*filepath. SPA
// fallback for client-side routes is handled by NoRoute.
func RegisterWeb(r *gin.Engine, routePrefix, dir string, indexFallback bool) {
	if routePrefix == "" {
		routePrefix = "/ui"
	}
	if !strings.HasPrefix(routePrefix, "/") {
		routePrefix = "/" + routePrefix
	}
	routePrefix = strings.TrimRight(routePrefix, "/")
	index := filepath.Join(dir, "index.html")

	r.GET(routePrefix, func(c *gin.Context) { c.Redirect(http.StatusFound, routePrefix+"/") })
	r.GET(routePrefix+"/", func(c *gin.Context) { c.File(index) })
	r.Static(routePrefix+"/assets", filepath.Join(dir, "assets"))

	if indexFallback {
		r.NoRoute(func(c *gin.Context) {
			path := c.Request.URL.Path
			if path == routePrefix || strings.HasPrefix(path, routePrefix+"/") {
				rel := strings.TrimPrefix(path, routePrefix+"/")
				rel = filepath.Clean(rel)
				if rel != "." && rel != string(os.PathSeparator) {
					candidate := filepath.Join(dir, rel)
					if st, err := os.Stat(candidate); err == nil && !st.IsDir() {
						c.File(candidate)
						return
					}
				}
				c.File(index)
				return
			}
			c.Status(http.StatusNotFound)
		})
	}
}
