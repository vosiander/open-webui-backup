package plugin

import (
	"github.com/spf13/cobra"
	"github.com/vosiander/open-webui-backup/pkg/config"
)

// Plugin defines the interface that all plugins must implement
type Plugin interface {
	// Name returns the name of the plugin (used as command name)
	Name() string
	// Description returns a short description of the plugin
	Description() string
	// Execute runs the plugin with the given configuration
	Execute(config *config.Config) error
}

// FlaggablePlugin is an optional interface for plugins that need custom flags
type FlaggablePlugin interface {
	Plugin
	// SetupFlags allows plugins to add custom flags to their command
	SetupFlags(cmd *cobra.Command)
}

// Registry holds all registered plugins
type Registry struct {
	plugins []Plugin
}

// NewRegistry creates a new plugin registry
func NewRegistry() *Registry {
	return &Registry{
		plugins: make([]Plugin, 0),
	}
}

// Register adds a plugin to the registry
func (r *Registry) Register(p Plugin) {
	r.plugins = append(r.plugins, p)
}

// GetPlugins returns all registered plugins
func (r *Registry) GetPlugins() []Plugin {
	return r.plugins
}

// CreateCommand creates a cobra command for a plugin
func CreateCommand(p Plugin, cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   p.Name(),
		Short: p.Description(),
		RunE: func(cmd *cobra.Command, args []string) error {
			return p.Execute(cfg)
		},
	}

	// If the plugin implements FlaggablePlugin, let it add custom flags
	if fp, ok := p.(FlaggablePlugin); ok {
		fp.SetupFlags(cmd)
	}

	return cmd
}
