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

// BackupModels is the main entry point for backing up all models
func BackupModels(client *openwebui.Client, outputDir string) error {
	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	logrus.Info("Fetching models...")
	models, err := client.ExportModels()
	if err != nil {
		return fmt.Errorf("failed to export models: %w", err)
	}

	if len(models) == 0 {
		logrus.Info("No models found")
		return nil
	}

	logrus.Infof("Found %d model(s)", len(models))

	// Backup each model
	for i, model := range models {
		logrus.Infof("Backing up model %d/%d: %s", i+1, len(models), model.Name)
		if err := backupSingleModel(&model, client, outputDir); err != nil {
			return fmt.Errorf("failed to backup model '%s': %w", model.Name, err)
		}
	}

	logrus.Info("All models backed up successfully")
	return nil
}

// backupSingleModel backs up a single model to a ZIP file
func backupSingleModel(model *openwebui.Model, client *openwebui.Client, outputDir string) error {
	// Generate ZIP filename
	zipFilename := generateModelZipFilename(model)
	zipPath := filepath.Join(outputDir, zipFilename)

	// Check if file already exists
	if _, err := os.Stat(zipPath); err == nil {
		return &openwebui.FileExistsError{Path: zipPath}
	}

	// Create ZIP archive
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return fmt.Errorf("failed to create zip file: %w", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Add model.json
	modelJSON, err := json.MarshalIndent(model, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal model: %w", err)
	}

	modelFile, err := zipWriter.Create("model.json")
	if err != nil {
		return fmt.Errorf("failed to create model.json in zip: %w", err)
	}
	if _, err := modelFile.Write(modelJSON); err != nil {
		return fmt.Errorf("failed to write model.json: %w", err)
	}

	// Backup knowledge bases if present
	if len(model.Meta.Knowledge) > 0 {
		logrus.Infof("  Backing up %d knowledge base(s)...", len(model.Meta.Knowledge))
		if err := backupModelKnowledgeBases(zipWriter, model.Meta.Knowledge, client); err != nil {
			logrus.Warnf("  Failed to backup some knowledge bases: %v", err)
		}
	}

	logrus.Infof("  Created: %s", zipFilename)
	return nil
}

// backupModelKnowledgeBases backs up knowledge items referenced by a model
func backupModelKnowledgeBases(zipWriter *zip.Writer, knowledge []map[string]interface{}, client *openwebui.Client) error {
	for _, item := range knowledge {
		itemType, ok := item["type"].(string)
		if !ok {
			logrus.Warnf("    Knowledge item missing 'type' field, skipping")
			continue
		}

		switch itemType {
		case "file":
			// Handle file-type knowledge item
			if err := backupFileKnowledgeItem(zipWriter, item, client); err != nil {
				logrus.Warnf("    Failed to backup file knowledge item: %v", err)
			}

		case "collection":
			// Handle collection-type knowledge item (full KB)
			if err := backupCollectionKnowledgeItem(zipWriter, item, client); err != nil {
				logrus.Warnf("    Failed to backup collection knowledge item: %v", err)
			}

		default:
			logrus.Warnf("    Unknown knowledge item type: %s, skipping", itemType)
		}
	}

	return nil
}

// backupFileKnowledgeItem backs up a file-type knowledge item
func backupFileKnowledgeItem(zipWriter *zip.Writer, item map[string]interface{}, client *openwebui.Client) error {
	// Extract file ID
	fileID, ok := item["id"].(string)
	if !ok {
		return fmt.Errorf("file item missing 'id' field")
	}

	logrus.Infof("    Backing up file: %s", fileID)

	// Save the full item metadata
	itemJSON, err := json.MarshalIndent(item, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal file item: %w", err)
	}

	metaPath := fmt.Sprintf("model-files/%s/metadata.json", fileID)
	metaFile, err := zipWriter.Create(metaPath)
	if err != nil {
		return fmt.Errorf("failed to create metadata file in zip: %w", err)
	}
	if _, err := metaFile.Write(itemJSON); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	// Download and save the actual file content
	fileData, err := client.GetFile(fileID)
	if err != nil {
		return fmt.Errorf("failed to download file %s: %w", fileID, err)
	}

	// Extract filename
	filename := fileData.Meta.Name
	if filename == "" {
		filename = fileData.Filename
	}
	if filename == "" {
		filename = fmt.Sprintf("file_%s", fileID)
	}

	// Get content
	var content []byte
	if fileData.Data != nil && fileData.Data.Content != "" {
		content = []byte(fileData.Data.Content)
	}

	// Save file content
	contentPath := fmt.Sprintf("model-files/%s/%s", fileID, filename)
	contentFile, err := zipWriter.Create(contentPath)
	if err != nil {
		return fmt.Errorf("failed to create content file in zip: %w", err)
	}
	if _, err := contentFile.Write(content); err != nil {
		return fmt.Errorf("failed to write content: %w", err)
	}

	logrus.Infof("      Saved file: %s (%d bytes)", filename, len(content))
	return nil
}

// backupCollectionKnowledgeItem backs up a collection-type knowledge item (full KB)
func backupCollectionKnowledgeItem(zipWriter *zip.Writer, item map[string]interface{}, client *openwebui.Client) error {
	// Extract KB ID
	kbID, ok := item["id"].(string)
	if !ok {
		return fmt.Errorf("collection item missing 'id' field")
	}

	kbName, _ := item["name"].(string)
	logrus.Infof("    Backing up collection: %s (ID: %s)", kbName, kbID)

	// Create knowledge-bases/{kb-id}/ directory
	kbDir := fmt.Sprintf("knowledge-bases/%s/", kbID)

	// Save the full item as knowledge_base.json
	kbJSON, err := json.MarshalIndent(item, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal KB %s: %w", kbID, err)
	}

	kbFile, err := zipWriter.Create(kbDir + "knowledge_base.json")
	if err != nil {
		return fmt.Errorf("failed to create KB json in zip: %w", err)
	}
	if _, err := kbFile.Write(kbJSON); err != nil {
		return fmt.Errorf("failed to write KB json: %w", err)
	}

	// Extract file IDs from data.file_ids
	var fileIDs []string
	if data, ok := item["data"].(map[string]interface{}); ok {
		if fileIDsRaw, ok := data["file_ids"].([]interface{}); ok {
			for _, idRaw := range fileIDsRaw {
				if id, ok := idRaw.(string); ok {
					fileIDs = append(fileIDs, id)
				}
			}
		}
	}

	logrus.Infof("      Downloading %d file(s)...", len(fileIDs))

	// Download and add files
	for _, fileID := range fileIDs {
		fileData, err := client.GetFile(fileID)
		if err != nil {
			logrus.Warnf("      Failed to download file %s: %v", fileID, err)
			continue
		}

		// Extract filename
		filename := fileData.Meta.Name
		if filename == "" {
			filename = fileData.Filename
		}
		if filename == "" {
			filename = fmt.Sprintf("file_%s", fileID)
		}

		// Get content
		var content []byte
		if fileData.Data != nil && fileData.Data.Content != "" {
			content = []byte(fileData.Data.Content)
		}

		// Add to ZIP
		filePath := kbDir + "documents/" + filename
		docFile, err := zipWriter.Create(filePath)
		if err != nil {
			logrus.Warnf("      Failed to create %s in zip: %v", filePath, err)
			continue
		}
		if _, err := docFile.Write(content); err != nil {
			logrus.Warnf("      Failed to write %s: %v", filePath, err)
			continue
		}

		logrus.Debugf("      Downloaded: %s (%d bytes)", filename, len(content))
	}

	return nil
}

// generateModelZipFilename generates a timestamped ZIP filename for a model
func generateModelZipFilename(model *openwebui.Model) string {
	timestamp := time.Now().UTC().Format("20060102_150405")
	sanitizedName := sanitizeFilename(model.Name)
	return fmt.Sprintf("%s_model_%s.zip", timestamp, sanitizedName)
}

// BackupAll backs up all knowledge bases and models
// Order: knowledge bases first, then models
func BackupAll(client *openwebui.Client, outputDir string) error {
	logrus.Info("Starting full backup (knowledge bases + models)...")

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Step 1: Backup all knowledge bases
	logrus.Info("Step 1/2: Backing up knowledge bases...")
	if err := BackupKnowledge(client, outputDir); err != nil {
		return fmt.Errorf("failed to backup knowledge bases: %w", err)
	}

	// Step 2: Backup all models
	logrus.Info("Step 2/2: Backing up models...")
	if err := BackupModels(client, outputDir); err != nil {
		return fmt.Errorf("failed to backup models: %w", err)
	}

	logrus.Info("Full backup completed successfully")
	return nil
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
