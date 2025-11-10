package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sirupsen/logrus"
	"github.com/vosiander/open-webui-backup/pkg/config"
	"github.com/vosiander/open-webui-backup/pkg/web"
)

// Server represents the HTTP server
type Server struct {
	config *config.Config
	echo   *echo.Echo
	hub    *Hub
	opMgr  *OperationManager
}

// NewServer creates a new HTTP server instance
func NewServer(cfg *config.Config) *Server {
	e := echo.New()
	e.HideBanner = true

	// Create WebSocket hub
	hub := NewHub()

	// Create operation manager
	opMgr := NewOperationManager(hub)

	server := &Server{
		config: cfg,
		echo:   e,
		hub:    hub,
		opMgr:  opMgr,
	}

	// Setup routes and middleware
	server.setupRoutes()

	return server
}

// setupRoutes configures all routes and middleware
func (s *Server) setupRoutes() {
	// Middleware
	s.echo.Use(s.customLogger())
	s.echo.Use(middleware.Recover())
	s.echo.Use(middleware.CORS())

	// API routes
	api := s.echo.Group("/api")
	{
		api.GET("/config", s.handleGetConfig)
		api.PUT("/config", s.handleUpdateConfig)
		api.POST("/backup", s.handleStartBackup)
		api.POST("/restore", s.handleStartRestore)
		api.GET("/status/:id", s.handleGetStatus)
		api.GET("/backups", s.handleListBackups)
		api.POST("/backups/upload", s.handleUploadBackup)
		api.POST("/backups/verify", s.handleVerifyBackup)
		api.GET("/backups/:filename", s.handleDownloadBackup)
		api.DELETE("/backups/:filename", s.handleDeleteBackup)
		api.POST("/identity/generate", s.handleGenerateIdentity)
	}

	// WebSocket route
	s.echo.GET("/ws", s.hub.HandleWebSocket)

	// Serve embedded frontend
	s.serveEmbeddedFrontend()
}

// serveEmbeddedFrontend serves the embedded Vue frontend
func (s *Server) serveEmbeddedFrontend() {
	// Get the embedded filesystem
	distFS, err := web.GetDistFS()
	if err != nil {
		logrus.WithError(err).Fatal("Failed to access embedded files")
	}

	// Create filesystem handler
	fsHandler := http.FileServer(http.FS(distFS))

	// Serve static files with SPA fallback
	s.echo.GET("/*", echo.WrapHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to serve the requested file
		path := r.URL.Path

		// Check if file exists
		if _, err := distFS.Open(path[1:]); err != nil {
			// File not found, serve index.html for SPA routing
			r.URL.Path = "/"
		}

		fsHandler.ServeHTTP(w, r)
	})))
}

// Start starts the HTTP server
func (s *Server) Start() error {
	// Start WebSocket hub in background
	go s.hub.Run()

	addr := fmt.Sprintf(":%d", s.config.ServerPort)
	logrus.Infof("Starting server on http://localhost%s", addr)
	logrus.Infof("Open WebUI URL: %s", s.config.OpenWebUIURL)
	logrus.Infof("Backups directory: %s", s.config.BackupsDir)

	return s.echo.Start(addr)
}

// customLogger returns a custom logging middleware that uses logrus format
func (s *Server) customLogger() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			res := c.Response()

			err := next(c)

			// Log in simple text format matching logrus
			logrus.Infof("%s %s %d %s",
				req.Method,
				req.URL.Path,
				res.Status,
				formatBytes(res.Size))

			return err
		}
	}
}

// formatBytes formats byte size for logging
func formatBytes(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%dB", bytes)
	}
	kb := float64(bytes) / 1024
	if kb < 1024 {
		return fmt.Sprintf("%.1fKB", kb)
	}
	mb := kb / 1024
	return fmt.Sprintf("%.1fMB", mb)
}

// Stop performs graceful shutdown
func (s *Server) Stop(ctx context.Context) error {
	logrus.Info("Shutting down server...")
	return s.echo.Shutdown(ctx)
}
