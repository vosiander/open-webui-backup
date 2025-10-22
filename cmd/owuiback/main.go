package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/vosiander/open-webui-backup/pkg/config"
	"github.com/vosiander/open-webui-backup/pkg/plugin"
	"github.com/vosiander/open-webui-backup/plugins"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Load configuration from environment variables
	cfg := config.Load()

	// Create plugin registry
	registry := plugin.NewRegistry()

	// Register plugins
	registry.Register(plugins.NewBackupKnowledgePlugin())
	registry.Register(plugins.NewRestoreKnowledgePlugin())

	// Create root command
	rootCmd := &cobra.Command{
		Use:   "owuiback",
		Short: "Open WebUI Backup Tool",
		Long:  "A tool to backup and restore various important information from an Open WebUI application",
	}

	// Register all plugin commands
	for _, p := range registry.GetPlugins() {
		cmd := plugin.CreateCommand(p, cfg)
		rootCmd.AddCommand(cmd)
	}

	// Execute root command
	return rootCmd.Execute()
}
