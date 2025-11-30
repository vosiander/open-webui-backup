package plugins

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/vosiander/open-webui-backup/pkg/config"
	"github.com/vosiander/open-webui-backup/pkg/encryption"
)

// NewIdentityPlugin generates a new age identity keypair and saves it to files
type NewIdentityPlugin struct {
	path string
}

// NewNewIdentityPlugin creates a new instance of the NewIdentityPlugin
func NewNewIdentityPlugin() *NewIdentityPlugin {
	return &NewIdentityPlugin{}
}

// Name returns the command name
func (p *NewIdentityPlugin) Name() string {
	return "new-identity"
}

// Description returns the command description
func (p *NewIdentityPlugin) Description() string {
	return "Generate a new age identity keypair and save to identity.txt and recipient.txt"
}

// SetupFlags configures the command-line flags
func (p *NewIdentityPlugin) SetupFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&p.path, "path", "", "Directory to save identity files (required)")
	cmd.MarkFlagRequired("path")
}

// Execute generates the identity pair and saves to files
func (p *NewIdentityPlugin) Execute(cfg *config.Config) error {
	log := logrus.WithField("plugin", p.Name())

	// Validate path directory
	if err := os.MkdirAll(p.path, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	identityPath := filepath.Join(p.path, "identity.txt")
	recipientPath := filepath.Join(p.path, "recipient.txt")

	// Check if files already exist
	if _, err := os.Stat(identityPath); err == nil {
		return fmt.Errorf("identity.txt already exists at %s (will not overwrite)", identityPath)
	}
	if _, err := os.Stat(recipientPath); err == nil {
		return fmt.Errorf("recipient.txt already exists at %s (will not overwrite)", recipientPath)
	}

	log.Info("Generating new age identity keypair...")

	// Generate identity
	identity, err := encryption.GenerateIdentity()
	if err != nil {
		return fmt.Errorf("failed to generate identity: %w", err)
	}

	// Save private key (identity) with restrictive permissions
	privateKey := identity.String()
	if err := os.WriteFile(identityPath, []byte(privateKey+"\n"), 0600); err != nil {
		return fmt.Errorf("failed to write identity.txt: %w", err)
	}
	log.Infof("Saved private key to: %s", identityPath)

	// Save public key (recipient)
	publicKey := identity.Recipient().String()
	if err := os.WriteFile(recipientPath, []byte(publicKey+"\n"), 0644); err != nil {
		// Clean up identity file if recipient save fails
		os.Remove(identityPath)
		return fmt.Errorf("failed to write recipient.txt: %w", err)
	}
	log.Infof("Saved public key to: %s", recipientPath)

	// Print usage information
	fmt.Println("\n✓ Identity keypair generated successfully!")
	fmt.Println("\nTo use with backup commands, set these environment variables:")
	fmt.Printf("  export BACKUP_ENCRYPTION_RECIPIENT=\"%s\"\n", publicKey)
	fmt.Printf("  export BACKUP_ENCRYPTION_IDENTITY=\"%s\"\n", identityPath)
	fmt.Println("\nOr use the new 'full-backup' command which uses these files automatically:")
	fmt.Printf("  owuiback full-backup --path %s\n", p.path)
	fmt.Println("\nIMPORTANT: Keep identity.txt secure - it's needed to decrypt your backups!")

	return nil
}
