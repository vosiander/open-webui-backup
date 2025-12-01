package plugins

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/vosiander/open-webui-backup/pkg/config"
	"github.com/vosiander/open-webui-backup/pkg/encryption"
	"github.com/vosiander/open-webui-backup/pkg/openwebui"
)

// VerifyPlugin verifies that a backup can be decrypted and optionally validates contents
type VerifyPlugin struct {
	path           string
	file           string
	onlyEncryption bool
}

// NewVerifyPlugin creates a new instance of the VerifyPlugin
func NewVerifyPlugin() *VerifyPlugin {
	return &VerifyPlugin{}
}

// Name returns the command name
func (p *VerifyPlugin) Name() string {
	return "verify"
}

// Description returns the command description
func (p *VerifyPlugin) Description() string {
	return "Verify that a backup file can be decrypted and optionally validate its contents"
}

// SetupFlags configures the command-line flags
func (p *VerifyPlugin) SetupFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&p.path, "path", "", "Directory containing identity.txt and backup files (required)")
	cmd.Flags().StringVar(&p.file, "file", "", "Specific backup file to verify (optional, auto-detects newest .age file if not provided)")
	cmd.Flags().BoolVar(&p.onlyEncryption, "only-encryption", false, "Only verify decryption, skip content validation")
	cmd.MarkFlagRequired("path")
}

// Execute verifies the backup file
func (p *VerifyPlugin) Execute(cfg *config.Config) error {
	log := logrus.WithField("plugin", p.Name())

	// Load identity from path/identity.txt
	identityPath := filepath.Join(p.path, "identity.txt")
	identityContent, err := os.ReadFile(identityPath)
	if err != nil {
		return fmt.Errorf("failed to read identity file %s: %w (use 'new-identity' command to create one)", identityPath, err)
	}

	// Determine backup file to verify
	var backupFile string
	if p.file != "" {
		// Use explicit file
		if filepath.IsAbs(p.file) {
			backupFile = p.file
		} else {
			backupFile = filepath.Join(p.path, p.file)
		}
	} else {
		// Auto-detect newest .age file
		found, err := findNewestBackup(p.path)
		if err != nil {
			return fmt.Errorf("failed to find backup file: %w", err)
		}
		backupFile = found
		log.Infof("Auto-detected backup file: %s", filepath.Base(backupFile))
	}

	// Check if file exists
	if _, err := os.Stat(backupFile); os.IsNotExist(err) {
		return fmt.Errorf("backup file not found: %s", backupFile)
	}

	// Check if file is encrypted
	if !encryption.IsEncrypted(backupFile) {
		logrus.Warn("⚠️  Backup file is not encrypted")
		if p.onlyEncryption {
			return nil
		}
		// For unencrypted files, validate directly
		return p.validateBackupContents(backupFile, log)
	}

	// Decrypt to temporary file
	log.Info("Verifying backup decryption...")
	tempFile := backupFile + ".verify.tmp"
	defer os.Remove(tempFile) // Ensure cleanup

	if err := encryption.DecryptFileWithIdentities(backupFile, tempFile, []string{string(identityContent)}); err != nil {
		logrus.Error("❌ Verification FAILED: Unable to decrypt backup")
		return fmt.Errorf("decryption failed: %w", err)
	}

	logrus.Info("✓ Decryption successful - identity key is correct")

	// If only checking encryption, we're done
	if p.onlyEncryption {
		os.Remove(tempFile)
		return nil
	}

	// Validate backup contents
	return p.validateBackupContents(tempFile, log)
}

// validateBackupContents validates the ZIP structure and contents
func (p *VerifyPlugin) validateBackupContents(zipPath string, log *logrus.Entry) error {
	log.Info("Validating backup contents...")

	// Open ZIP file
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		logrus.Error("❌ Verification FAILED: Invalid ZIP file")
		return fmt.Errorf("failed to open ZIP file: %w", err)
	}
	defer r.Close()

	// Read metadata
	metadata, err := readBackupMetadata(r)
	if err != nil {
		logrus.Warn("⚠️  Warning: Could not read backup metadata")
		log.Warnf("Failed to read metadata: %v", err)
	}

	// Count items by type
	itemCounts := countBackupItems(r)

	// Print results
	logrus.Info("✓ Backup contents validated successfully")
	logrus.Info("=== Backup Information ===")

	if metadata != nil {
		logrus.Infof("Backup Type: %s", getBackupType(metadata))
		if metadata.BackupTimestamp != "" {
			logrus.Infof("Created: %s", metadata.BackupTimestamp)
		}
		if metadata.BackupToolVersion != "" {
			logrus.Infof("Tool Version: %s", metadata.BackupToolVersion)
		}
		logrus.Infof("Total Items: %d", metadata.ItemCount)
		if len(metadata.ContainedTypes) > 0 {
			logrus.Infof("Data Types: %s", strings.Join(metadata.ContainedTypes, ", "))
		}
	}

	if len(itemCounts) > 0 {
		logrus.Info("=== Item Counts ===")
		// Sort keys for consistent output
		types := make([]string, 0, len(itemCounts))
		for t := range itemCounts {
			types = append(types, t)
		}
		sort.Strings(types)

		for _, t := range types {
			count := itemCounts[t]
			logrus.Infof("%s: %d", strings.Title(t), count)
		}
	}

	logrus.Info("")
	return nil
}

// findNewestBackup finds the most recent .age file in the directory
func findNewestBackup(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("failed to read directory: %w", err)
	}

	var newestFile string
	var newestTime int64

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".age") {
			continue
		}

		fullPath := filepath.Join(dir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Unix() > newestTime {
			newestTime = info.ModTime().Unix()
			newestFile = fullPath
		}
	}

	if newestFile == "" {
		return "", fmt.Errorf("no .age backup files found in %s", dir)
	}

	return newestFile, nil
}

// readBackupMetadata extracts and parses owui.json from backup
func readBackupMetadata(r *zip.ReadCloser) (*openwebui.BackupMetadata, error) {
	for _, f := range r.File {
		if f.Name == "owui.json" {
			rc, err := f.Open()
			if err != nil {
				return nil, fmt.Errorf("failed to open owui.json: %w", err)
			}
			defer rc.Close()

			data, err := io.ReadAll(rc)
			if err != nil {
				return nil, fmt.Errorf("failed to read owui.json: %w", err)
			}

			var metadata openwebui.BackupMetadata
			if err := json.Unmarshal(data, &metadata); err != nil {
				return nil, fmt.Errorf("failed to parse owui.json: %w", err)
			}

			return &metadata, nil
		}
	}

	return nil, fmt.Errorf("owui.json not found in backup")
}

// countBackupItems counts items of each type in the backup
func countBackupItems(r *zip.ReadCloser) map[string]int {
	counts := make(map[string]int)

	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}

		// Count by directory prefix
		parts := strings.Split(f.Name, "/")
		if len(parts) < 2 {
			continue
		}

		dirName := parts[0]
		switch dirName {
		case "knowledge-bases":
			// Count knowledge_base.json files
			if strings.HasSuffix(f.Name, "/knowledge_base.json") {
				counts["knowledge"]++
			}
		case "models":
			// Count model.json files
			if strings.HasSuffix(f.Name, "/model.json") {
				counts["models"]++
			}
		case "tools":
			// Count tool.json files
			if strings.HasSuffix(f.Name, "/tool.json") {
				counts["tools"]++
			}
		case "prompts":
			// Count prompt.json files
			if strings.HasSuffix(f.Name, "/prompt.json") {
				counts["prompts"]++
			}
		case "files":
			// Count file.json files
			if strings.HasSuffix(f.Name, "/file.json") {
				counts["files"]++
			}
		case "chats":
			// Count chat.json files
			if strings.HasSuffix(f.Name, "/chat.json") {
				counts["chats"]++
			}
		case "users":
			// Count user.json files
			if strings.HasSuffix(f.Name, "/user.json") {
				counts["users"]++
			}
		case "groups":
			// Count group.json files
			if strings.HasSuffix(f.Name, "/group.json") {
				counts["groups"]++
			}
		case "feedbacks":
			// Count feedback.json files
			if strings.HasSuffix(f.Name, "/feedback.json") {
				counts["feedbacks"]++
			}
		}
	}

	return counts
}

// getBackupType returns a human-readable backup type description
func getBackupType(metadata *openwebui.BackupMetadata) string {
	if metadata.UnifiedBackup {
		return "Unified Backup"
	}
	return "Legacy Backup"
}
