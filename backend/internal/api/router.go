package api

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/awu123-create/http-bench/internal/static"
	"github.com/awu123-create/http-bench/internal/task"
	"github.com/gin-gonic/gin"
)

func NewRouter(m *task.Manager) *gin.Engine {
	r := gin.Default()
	h := NewHandler(m)

	api := r.Group("/api")
	{
		api.POST("/tests", h.CreateTest)
		api.POST("/tests/:id/start", h.StartTest)
		api.POST("/tests/:id/stop", h.StopTest)
		api.GET("/tests/:id", h.GetTest)
		api.GET("/tests/:id/metrics", h.GetMetrics)
		api.GET("/tests/:id/events", h.StreamMetrics)
		api.GET("/tests/:id/result", h.GetTestResult)
		api.GET("/tests", h.ListTests)
	}

	// 静态前端托管：从 embed 的 dist 提供静态文件，找不到则回退 index.html（SPA）
	distFS, err := fs.Sub(static.DistFS, "dist")
	if err == nil {
		r.NoRoute(func(c *gin.Context) {
			// API 路径未命中则 404
			if strings.HasPrefix(c.Request.URL.Path, "/api/") {
				c.JSON(http.StatusNotFound, gin.H{"message": "not found"})
				return
			}
			// 尝试静态文件（assets 等）
			p := strings.TrimPrefix(c.Request.URL.Path, "/")
			if p != "" && p != "index.html" {
				if f, err := distFS.Open(p); err == nil {
					f.Close()
					c.FileFromFS(p, http.FS(distFS))
					return
				}
			}
			// SPA fallback：直接返回 index.html 内容，交由前端路由处理
			c.Header("Content-Type", "text/html; charset=utf-8")
			if data, err := fs.ReadFile(distFS, "index.html"); err == nil {
				c.Data(http.StatusOK, "text/html; charset=utf-8", data)
				return
			}
			c.String(http.StatusNotFound, "frontend not built")
		})
	}

	return r
}
