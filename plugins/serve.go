package plugins

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/vosiander/open-webui-backup/pkg/api"
	"github.com/vosiander/open-webui-backup/pkg/config"
	"github.com/vosiander/open-webui-backup/pkg/plugin"
)

// ServePlugin implements the serve command
type ServePlugin struct {
	port int
}

// NewServePlugin creates a new serve plugin
func NewServePlugin() plugin.Plugin {
	return &ServePlugin{}
}

// Name returns the plugin name
func (p *ServePlugin) Name() string {
	return "serve"
}

// Description returns the plugin description
func (p *ServePlugin) Description() string {
	return "Start web dashboard server for backup and restore operations"
}

// SetupFlags adds command-line flags for this plugin
func (p *ServePlugin) SetupFlags(cmd *cobra.Command) {
	cmd.Flags().IntVarP(&p.port, "port", "p", 0, "Server port (default: 3000, or OWUI_SERVER_PORT env)")
}

// Execute runs the serve command
func (p *ServePlugin) Execute(cfg *config.Config) error {
	// Override port if specified via flag
	if p.port != 0 {
		cfg.ServerPort = p.port
	}

	// Create and start server
	server := api.NewServer(cfg)

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle OS signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start server in goroutine
	go func() {
		if err := server.Start(); err != nil {
			logrus.WithError(err).Error("Server error")
			cancel()
		}
	}()

	// Wait for shutdown signal
	select {
	case <-sigChan:
		logrus.Info("Received shutdown signal")
	case <-ctx.Done():
		logrus.Info("Context cancelled")
	}

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Stop(shutdownCtx); err != nil {
		return err
	}

	logrus.Info("Server stopped successfully")
	return nil
}
