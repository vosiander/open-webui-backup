package backup

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vosiander/open-webui-backup/pkg/openwebui"
)

// BackupKnowledge is the main entry point for backing up all knowledge bases
func BackupKnowledge(client *openwebui.Client, outputDir string) error {
	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	logrus.Info("Fetching knowledge bases...")
	knowledgeBases, err := client.ListKnowledge()
	if err != nil {
		return fmt.Errorf("failed to list knowledge bases: %w", err)
	}

	if len(knowledgeBases) == 0 {
		logrus.Info("No knowledge bases found")
		return nil
	}

	logrus.Infof("Found %d knowledge base(s)", len(knowledgeBases))

	// Backup each knowledge base
	for i, kb := range knowledgeBases {
		logrus.Infof("Backing up knowledge base %d/%d: %s", i+1, len(knowledgeBases), kb.Name)
		if err := backupSingleKnowledge(&kb, client, outputDir); err != nil {
			return fmt.Errorf("failed to backup knowledge base '%s': %w", kb.Name, err)
		}
	}

	logrus.Info("All knowledge bases backed up successfully")
	return nil
}

// backupSingleKnowledge backs up a single knowledge base to a ZIP file
func backupSingleKnowledge(kb *openwebui.KnowledgeBase, client *openwebui.Client, outputDir string) error {
	// Generate ZIP filename
	zipFilename := generateZipFilename(kb)
	zipPath := filepath.Join(outputDir, zipFilename)

	// Check if file already exists
	if _, err := os.Stat(zipPath); err == nil {
		return &openwebui.FileExistsError{Path: zipPath}
	}

	// Collect file IDs from the knowledge base
	var fileIDs []string
	if kb.Data != nil && kb.Data.FileIDs != nil {
		fileIDs = kb.Data.FileIDs
	}

	logrus.Infof("  Downloading %d file(s)...", len(fileIDs))

	// Download all files
	files := make(map[string][]byte)
	for _, fileID := range fileIDs {
		fileData, err := client.GetFile(fileID)
		if err != nil {
			logrus.Warnf("  Failed to download file %s: %v", fileID, err)
			continue
		}

		// Extract filename and content
		filename := fileData.Meta.Name
		if filename == "" {
			filename = fileData.Filename
		}
		if filename == "" {
			filename = fmt.Sprintf("file_%s", fileID)
		}

		// Get content from fileData
		var content []byte
		if fileData.Data != nil && fileData.Data.Content != "" {
			content = []byte(fileData.Data.Content)
		}

		files[filename] = content
		logrus.Debugf("  Downloaded: %s (%d bytes)", filename, len(content))
	}

	// Create ZIP archive
	if err := createZipArchive(kb, files, zipPath); err != nil {
		return fmt.Errorf("failed to create ZIP archive: %w", err)
	}

	logrus.Infof("  Created: %s", zipFilename)
	return nil
}

// createZipArchive creates a ZIP file with the knowledge base metadata and files
func createZipArchive(kb *openwebui.KnowledgeBase, files map[string][]byte, outputPath string) error {
	// Create ZIP file
	zipFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create zip file: %w", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Add knowledge_base.json
	kbJSON, err := json.MarshalIndent(kb, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal knowledge base: %w", err)
	}

	kbFile, err := zipWriter.Create("knowledge_base.json")
	if err != nil {
		return fmt.Errorf("failed to create knowledge_base.json in zip: %w", err)
	}
	if _, err := kbFile.Write(kbJSON); err != nil {
		return fmt.Errorf("failed to write knowledge_base.json: %w", err)
	}

	// Add documents folder with all files
	for filename, content := range files {
		docPath := filepath.Join("documents", filename)
		docFile, err := zipWriter.Create(docPath)
		if err != nil {
			return fmt.Errorf("failed to create %s in zip: %w", docPath, err)
		}
		if _, err := docFile.Write(content); err != nil {
			return fmt.Errorf("failed to write %s: %w", docPath, err)
		}
	}

	return nil
}

// generateZipFilename generates a timestamped ZIP filename for a knowledge base
func generateZipFilename(kb *openwebui.KnowledgeBase) string {
	timestamp := time.Now().UTC().Format("20060102_150405")
	sanitizedName := sanitizeFilename(kb.Name)
	return fmt.Sprintf("%s_knowledge_base_%s.zip", timestamp, sanitizedName)
}

// sanitizeFilename sanitizes a knowledge base name for use in a filename
func sanitizeFilename(name string) string {
	// Convert to lowercase
	name = strings.ToLower(name)

	// Replace spaces with underscores
	name = strings.ReplaceAll(name, " ", "_")

	// Remove special characters (keep alphanumeric, underscore, hyphen)
	reg := regexp.MustCompile("[^a-z0-9_-]+")
	name = reg.ReplaceAllString(name, "")

	// Truncate to 50 characters
	if len(name) > 50 {
		name = name[:50]
	}

	// Remove trailing underscores or hyphens
	name = strings.TrimRight(name, "_-")

	// If empty after sanitization, use a default
	if name == "" {
		name = "unnamed"
	}

	return name
}
