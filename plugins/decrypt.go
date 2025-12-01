package plugins

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/vosiander/open-webui-backup/pkg/config"
	"github.com/vosiander/open-webui-backup/pkg/encryption"
)

// DecryptPlugin decrypts all .age files in a directory
type DecryptPlugin struct {
	path  string
	force bool
}

// NewDecryptPlugin creates a new instance of the DecryptPlugin
func NewDecryptPlugin() *DecryptPlugin {
	return &DecryptPlugin{}
}

// Name returns the command name
func (p *DecryptPlugin) Name() string {
	return "decrypt"
}

// Description returns the command description
func (p *DecryptPlugin) Description() string {
	return "Decrypt all .age encrypted files in a directory using identity.txt"
}

// SetupFlags configures the command-line flags
func (p *DecryptPlugin) SetupFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&p.path, "path", "", "Directory containing identity.txt and .age files to decrypt (required)")
	cmd.Flags().BoolVar(&p.force, "force", false, "Overwrite existing decrypted files")
	cmd.MarkFlagRequired("path")
}

// Execute decrypts all .age files in the directory
func (p *DecryptPlugin) Execute(cfg *config.Config) error {
	log := logrus.WithField("plugin", p.Name())

	// Load identity from path/identity.txt
	identityPath := filepath.Join(p.path, "identity.txt")
	identityContent, err := os.ReadFile(identityPath)
	if err != nil {
		return fmt.Errorf("failed to read identity file %s: %w (use 'new-identity' command to create one)", identityPath, err)
	}

	// Trim whitespace from identity
	identityStr := strings.TrimSpace(string(identityContent))

	// Find all .age files in directory
	encryptedFiles, err := findEncryptedFiles(p.path)
	if err != nil {
		return fmt.Errorf("failed to scan directory: %w", err)
	}

	if len(encryptedFiles) == 0 {
		logrus.Warnf("⚠️  No .age files found in %s", p.path)
		return nil
	}

	logrus.Infof("Decrypting %d file(s) from %s\n", len(encryptedFiles), p.path)

	// Decrypt each file
	stats := &decryptStats{}
	for _, encryptedFile := range encryptedFiles {
		if err := p.decryptSingleFile(encryptedFile, identityStr, log, stats); err != nil {
			log.Warnf("Failed to decrypt %s: %v", filepath.Base(encryptedFile), err)
		}
	}

	// Print summary
	logrus.Info("" + strings.Repeat("─", 50))
	logrus.Infof("Summary: %d decrypted, %d skipped, %d failed", stats.decrypted, stats.skipped, stats.failed)
	if stats.failed > 0 {
		logrus.Warn("⚠️  Some files failed to decrypt. Check the logs above for details.")
	}

	return nil
}

// decryptSingleFile decrypts a single .age file
func (p *DecryptPlugin) decryptSingleFile(encryptedPath, identityContent string, log *logrus.Entry, stats *decryptStats) error {
	basename := filepath.Base(encryptedPath)

	// Check if file is actually encrypted
	if !encryption.IsEncrypted(encryptedPath) {
		logrus.Infof("⊘ %s → skipped (not encrypted)", basename)
		stats.skipped++
		return nil
	}

	// Determine output path (remove .age extension)
	outputPath := strings.TrimSuffix(encryptedPath, ".age")
	outputBasename := filepath.Base(outputPath)

	// Check if output file already exists
	if _, err := os.Stat(outputPath); err == nil && !p.force {
		logrus.Infof("⊘ %s → skipped (%s already exists, use --force)", basename, outputBasename)
		stats.skipped++
		return nil
	}

	// Decrypt the file
	if err := encryption.DecryptFileWithIdentities(encryptedPath, outputPath, []string{identityContent}); err != nil {
		logrus.Errorf("❌ %s → failed (%v)", basename, err)
		stats.failed++
		return err
	}

	logrus.Infof("✓ %s → %s", basename, outputBasename)
	stats.decrypted++
	return nil
}

// findEncryptedFiles finds all .age files in the directory
func findEncryptedFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if strings.HasSuffix(entry.Name(), ".age") {
			fullPath := filepath.Join(dir, entry.Name())
			files = append(files, fullPath)
		}
	}

	return files, nil
}

// decryptStats tracks decryption statistics
type decryptStats struct {
	decrypted int
	skipped   int
	failed    int
}
