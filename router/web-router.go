package router

import (
	"embed"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/gin-contrib/gzip"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
)

// WebAssets holds the embedded dashboard frontend assets.
type WebAssets struct {
	BuildFS   embed.FS
	IndexPage []byte
}

func serveCanvasIndex(c *gin.Context, assets WebAssets) {
	canvasIndex, err := assets.BuildFS.ReadFile("web/dist/canvas-app/index.html")
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	c.Header("Cache-Control", "no-cache")
	c.Data(http.StatusOK, "text/html; charset=utf-8", canvasIndex)
}

func SetWebRouter(router *gin.Engine, assets WebAssets) {
	frontendFS := common.EmbedFolder(assets.BuildFS, "web/dist")

	router.Use(gzip.Gzip(gzip.DefaultCompression))
	router.Use(middleware.GlobalWebRateLimit())
	router.Use(middleware.Cache())
	// The parent application owns /canvas. The canvas bundle lives under its
	// private static prefix so direct dashboard routes never bypass the shell.
	router.GET("/canvas", func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache")
		c.Data(http.StatusOK, "text/html; charset=utf-8", assets.IndexPage)
	})
	router.GET("/canvas/", func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache")
		c.Data(http.StatusOK, "text/html; charset=utf-8", assets.IndexPage)
	})
	router.GET("/canvas/index.html", func(c *gin.Context) {
		c.Redirect(http.StatusTemporaryRedirect, "/canvas")
	})
	router.GET("/canvas-app", func(c *gin.Context) {
		c.Redirect(http.StatusTemporaryRedirect, "/canvas-app/")
	})
	router.GET("/canvas-app/", func(c *gin.Context) {
		serveCanvasIndex(c, assets)
	})
	router.GET("/canvas-app/index.html", func(c *gin.Context) {
		c.Redirect(http.StatusTemporaryRedirect, "/canvas-app/")
	})
	router.Use(static.Serve("/", frontendFS))
	router.NoRoute(func(c *gin.Context) {
		c.Set(middleware.RouteTagKey, "web")
		requestPath := c.Request.URL.Path
		if strings.HasPrefix(requestPath, "/canvas-app/") && strings.Contains(c.GetHeader("Accept"), "text/html") {
			serveCanvasIndex(c, assets)
			return
		}
		if strings.HasPrefix(requestPath, "/v1") || strings.HasPrefix(requestPath, "/api") || strings.HasPrefix(requestPath, "/assets") {
			controller.RelayNotFound(c)
			return
		}
		c.Header("Cache-Control", "no-cache")
		c.Data(http.StatusOK, "text/html; charset=utf-8", assets.IndexPage)
	})
}
