package main

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"watch-party/pkg/logger"

	"github.com/gin-gonic/gin"
)

//go:embed dist/*
var frontendFS embed.FS

// setupFrontendRoutes sets up routes to serve the embedded React app
func setupFrontendRoutes(r *gin.Engine) error {
	// Get the subdirectory from the embedded filesystem
	frontendStaticFS, err := fs.Sub(frontendFS, "dist")
	if err != nil {
		logger.Error(err, "Failed to create frontend filesystem")
		return err
	}

	// Get the assets subdirectory specifically
	assetsFS, err := fs.Sub(frontendStaticFS, "assets")
	if err != nil {
		logger.Error(err, "Failed to create assets filesystem")
		return err
	}

	// Serve static files (CSS, JS, images, etc.) from the assets directory
	r.StaticFS("/assets", http.FS(assetsFS))

	// serve the favicon and other public files
	r.GET("/vite.svg", func(c *gin.Context) {
		data, err := frontendFS.ReadFile("dist/vite.svg")
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		c.Header("Content-Type", "image/svg+xml")
		c.Data(http.StatusOK, "image/svg+xml", data)
	})

	// serve index.html for all frontend routes (SPA routing)
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path

		// don't serve index.html for API routes
		if strings.HasPrefix(path, "/api/") ||
			strings.HasPrefix(path, "/ws/") ||
			strings.HasPrefix(path, "/health") {
			c.Status(http.StatusNotFound)
			return
		}

		// for all other routes, serve index.html (React Router will handle routing)
		data, err := frontendFS.ReadFile("dist/index.html")
		if err != nil {
			logger.Error(err, "Failed to read index.html")
			c.Status(http.StatusInternalServerError)
			return
		}

		c.Header("Content-Type", "text/html")
		c.Data(http.StatusOK, "text/html", data)
	})

	logger.Info("Frontend routes configured successfully")
	return nil
}
