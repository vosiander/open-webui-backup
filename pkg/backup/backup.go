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
	"github.com/vosiander/open-webui-backup/pkg/config"
	"github.com/vosiander/open-webui-backup/pkg/openwebui"
)

// ProgressCallback is a function that receives progress updates during backup operations
type ProgressCallback func(percent int, message string)

// SelectiveBackupOptions defines which data types to include in a backup
type SelectiveBackupOptions struct {
	Knowledge bool
	Models    bool
	Tools     bool
	Prompts   bool
	Files     bool
	Chats     bool
	Groups    bool
	Feedbacks bool
	Users     bool
}

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
	if err := createZipArchiveWithMetadata(kb, files, zipPath, client); err != nil {
		return fmt.Errorf("failed to create ZIP archive: %w", err)
	}

	logrus.Infof("  Created: %s", zipFilename)
	return nil
}

// createZipArchiveWithMetadata creates a ZIP file with the knowledge base metadata, files, and owui.json
func createZipArchiveWithMetadata(kb *openwebui.KnowledgeBase, files map[string][]byte, outputPath string, client *openwebui.Client) error {
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

	// Add owui.json metadata
	metadata := generateMetadata(client, "knowledge", 1, false, nil)
	if err := writeMetadataToZip(zipWriter, metadata); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
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

	// Add owui.json metadata
	metadata := generateMetadata(client, "model", 1, false, nil)
	if err := writeMetadataToZip(zipWriter, metadata); err != nil {
		logrus.Warnf("  Failed to write metadata: %v", err)
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

// BackupTools is the main entry point for backing up all tools
func BackupTools(client *openwebui.Client, outputDir string) error {
	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	logrus.Info("Fetching tools...")
	tools, err := client.ExportTools()
	if err != nil {
		return fmt.Errorf("failed to export tools: %w", err)
	}

	if len(tools) == 0 {
		logrus.Info("No tools found")
		return nil
	}

	logrus.Infof("Found %d tool(s)", len(tools))

	// Backup each tool
	for i, tool := range tools {
		logrus.Infof("Backing up tool %d/%d: %s", i+1, len(tools), tool.Name)
		if err := backupSingleTool(&tool, client, outputDir); err != nil {
			return fmt.Errorf("failed to backup tool '%s': %w", tool.Name, err)
		}
	}

	logrus.Info("All tools backed up successfully")
	return nil
}

// backupSingleTool backs up a single tool to a ZIP file
func backupSingleTool(tool *openwebui.Tool, client *openwebui.Client, outputDir string) error {
	// Generate ZIP filename
	timestamp := time.Now().UTC().Format("20060102_150405")
	sanitizedName := sanitizeFilename(tool.Name)
	zipFilename := fmt.Sprintf("%s_tool_%s.zip", timestamp, sanitizedName)
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

	// Add tool.json
	toolJSON, err := json.MarshalIndent(tool, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tool: %w", err)
	}

	toolFile, err := zipWriter.Create("tool.json")
	if err != nil {
		return fmt.Errorf("failed to create tool.json in zip: %w", err)
	}
	if _, err := toolFile.Write(toolJSON); err != nil {
		return fmt.Errorf("failed to write tool.json: %w", err)
	}

	// Add owui.json metadata
	metadata := generateMetadata(client, "tool", 1, false, nil)
	if err := writeMetadataToZip(zipWriter, metadata); err != nil {
		logrus.Warnf("  Failed to write metadata: %v", err)
	}

	logrus.Infof("  Created: %s", zipFilename)
	return nil
}

// BackupPrompts is the main entry point for backing up all prompts
func BackupPrompts(client *openwebui.Client, outputDir string) error {
	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	logrus.Info("Fetching prompts...")
	prompts, err := client.ListPrompts()
	if err != nil {
		return fmt.Errorf("failed to list prompts: %w", err)
	}

	if len(prompts) == 0 {
		logrus.Info("No prompts found")
		return nil
	}

	logrus.Infof("Found %d prompt(s)", len(prompts))

	// Backup each prompt
	for i, prompt := range prompts {
		logrus.Infof("Backing up prompt %d/%d: %s", i+1, len(prompts), prompt.Title)
		if err := backupSinglePrompt(&prompt, client, outputDir); err != nil {
			return fmt.Errorf("failed to backup prompt '%s': %w", prompt.Title, err)
		}
	}

	logrus.Info("All prompts backed up successfully")
	return nil
}

// backupSinglePrompt backs up a single prompt to a ZIP file
func backupSinglePrompt(prompt *openwebui.Prompt, client *openwebui.Client, outputDir string) error {
	// Generate ZIP filename using command as identifier
	timestamp := time.Now().UTC().Format("20060102_150405")
	sanitizedCommand := sanitizeFilename(prompt.Command)
	zipFilename := fmt.Sprintf("%s_prompt_%s.zip", timestamp, sanitizedCommand)
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

	// Add prompt.json
	promptJSON, err := json.MarshalIndent(prompt, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal prompt: %w", err)
	}

	promptFile, err := zipWriter.Create("prompt.json")
	if err != nil {
		return fmt.Errorf("failed to create prompt.json in zip: %w", err)
	}
	if _, err := promptFile.Write(promptJSON); err != nil {
		return fmt.Errorf("failed to write prompt.json: %w", err)
	}

	// Add owui.json metadata
	metadata := generateMetadata(client, "prompt", 1, false, nil)
	if err := writeMetadataToZip(zipWriter, metadata); err != nil {
		logrus.Warnf("  Failed to write metadata: %v", err)
	}

	logrus.Infof("  Created: %s", zipFilename)
	return nil
}

// BackupFiles is the main entry point for backing up all standalone files
func BackupFiles(client *openwebui.Client, outputDir string) error {
	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	logrus.Info("Fetching files...")
	files, err := client.ListFiles()
	if err != nil {
		return fmt.Errorf("failed to list files: %w", err)
	}

	if len(files) == 0 {
		logrus.Info("No standalone files found")
		return nil
	}

	logrus.Infof("Found %d file(s)", len(files))

	// Backup each file
	for i, fileMetadata := range files {
		logrus.Infof("Backing up file %d/%d: %s", i+1, len(files), fileMetadata.Meta.Name)
		if err := backupSingleFile(fileMetadata.ID, client, outputDir); err != nil {
			logrus.Warnf("Failed to backup file '%s': %v", fileMetadata.Meta.Name, err)
			continue
		}
	}

	logrus.Info("All files backed up successfully")
	return nil
}

// backupSingleFile backs up a single file with content to a ZIP file
func backupSingleFile(fileID string, client *openwebui.Client, outputDir string) error {
	// Get file with content
	fileExport, err := client.GetFileWithContent(fileID)
	if err != nil {
		return fmt.Errorf("failed to get file with content: %w", err)
	}

	// Generate ZIP filename
	timestamp := time.Now().UTC().Format("20060102_150405")
	sanitizedName := sanitizeFilename(fileExport.Filename)
	zipFilename := fmt.Sprintf("%s_file_%s.zip", timestamp, sanitizedName)
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

	// Add file.json (metadata)
	fileJSON, err := json.MarshalIndent(fileExport, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal file metadata: %w", err)
	}

	metaFile, err := zipWriter.Create("file.json")
	if err != nil {
		return fmt.Errorf("failed to create file.json in zip: %w", err)
	}
	if _, err := metaFile.Write(fileJSON); err != nil {
		return fmt.Errorf("failed to write file.json: %w", err)
	}

	// Add actual file content
	var content []byte
	if fileExport.Data != nil && fileExport.Data.Content != "" {
		content = []byte(fileExport.Data.Content)
	}

	contentPath := filepath.Join("content", fileExport.Filename)
	contentFile, err := zipWriter.Create(contentPath)
	if err != nil {
		return fmt.Errorf("failed to create content file in zip: %w", err)
	}
	if _, err := contentFile.Write(content); err != nil {
		return fmt.Errorf("failed to write content: %w", err)
	}

	// Add owui.json metadata
	metadata := generateMetadata(client, "file", 1, false, nil)
	if err := writeMetadataToZip(zipWriter, metadata); err != nil {
		logrus.Warnf("  Failed to write metadata: %v", err)
	}

	logrus.Infof("  Created: %s (%d bytes)", zipFilename, len(content))
	return nil
}

// BackupChats is the main entry point for backing up all chats
func BackupChats(client *openwebui.Client, outputDir string) error {
	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	logrus.Info("Fetching chats...")
	chats, err := client.GetAllChats()
	if err != nil {
		return fmt.Errorf("failed to get chats: %w", err)
	}

	if len(chats) == 0 {
		logrus.Info("No chats found")
		return nil
	}

	logrus.Infof("Found %d chat(s)", len(chats))

	// Backup each chat
	for i, chat := range chats {
		logrus.Infof("Backing up chat %d/%d: %s", i+1, len(chats), chat.Title)
		if err := backupSingleChat(&chat, client, outputDir); err != nil {
			logrus.Warnf("Failed to backup chat '%s': %v", chat.Title, err)
			continue
		}
	}

	logrus.Info("All chats backed up successfully")
	return nil
}

// backupSingleChat backs up a single chat to a ZIP file
func backupSingleChat(chat *openwebui.Chat, client *openwebui.Client, outputDir string) error {
	// Generate ZIP filename
	timestamp := time.Now().UTC().Format("20060102_150405")
	sanitizedTitle := sanitizeFilename(chat.Title)
	zipFilename := fmt.Sprintf("%s_chat_%s.zip", timestamp, sanitizedTitle)
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

	// Add chat.json
	chatJSON, err := json.MarshalIndent(chat, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal chat: %w", err)
	}

	chatFile, err := zipWriter.Create("chat.json")
	if err != nil {
		return fmt.Errorf("failed to create chat.json in zip: %w", err)
	}
	if _, err := chatFile.Write(chatJSON); err != nil {
		return fmt.Errorf("failed to write chat.json: %w", err)
	}

	// Add owui.json metadata
	metadata := generateMetadata(client, "chat", 1, false, nil)
	if err := writeMetadataToZip(zipWriter, metadata); err != nil {
		logrus.Warnf("  Failed to write metadata: %v", err)
	}

	logrus.Infof("  Created: %s", zipFilename)
	return nil
}

// backupAllChats backs up all chats into the unified ZIP
func backupAllChats(zipWriter *zip.Writer, client *openwebui.Client) (int, error) {
	chats, err := client.GetAllChats()
	if err != nil {
		return 0, fmt.Errorf("failed to get chats: %w", err)
	}

	for i, chat := range chats {
		logrus.Infof("  Backing up chat %d/%d: %s", i+1, len(chats), chat.Title)
		if err := backupChatToZip(zipWriter, &chat); err != nil {
			logrus.Warnf("  Failed to backup chat '%s': %v", chat.Title, err)
			continue
		}
	}

	return len(chats), nil
}

// backupChatToZip backs up a single chat into an existing ZIP writer
func backupChatToZip(zipWriter *zip.Writer, chat *openwebui.Chat) error {
	// Create chats/{id}/ directory
	chatDir := fmt.Sprintf("chats/%s/", chat.ID)

	// Add chat.json
	chatJSON, err := json.MarshalIndent(chat, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal chat: %w", err)
	}

	chatFile, err := zipWriter.Create(chatDir + "chat.json")
	if err != nil {
		return fmt.Errorf("failed to create chat.json in zip: %w", err)
	}
	if _, err := chatFile.Write(chatJSON); err != nil {
		return fmt.Errorf("failed to write chat.json: %w", err)
	}

	return nil
}

// BackupSelective performs a selective backup based on the provided options
// outputFile should be the full path to the output ZIP file
// progressCallback is an optional callback function for progress updates (can be nil)
func BackupSelective(client *openwebui.Client, outputFile string, options *SelectiveBackupOptions, progressCallback ProgressCallback) error {
	logrus.Info("Starting selective backup...")

	if progressCallback != nil {
		progressCallback(0, "Starting selective backup...")
	}

	// Validate that at least one option is enabled
	if !options.Knowledge && !options.Models && !options.Tools && !options.Prompts && !options.Files && !options.Chats {
		return fmt.Errorf("at least one data type must be selected for backup")
	}

	// Check if file already exists
	if _, err := os.Stat(outputFile); err == nil {
		return &openwebui.FileExistsError{Path: outputFile}
	}

	// Create ZIP file
	zipFile, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create zip file: %w", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Track contained types and total item count
	containedTypes := []string{}
	totalItems := 0

	// Backup selected types
	if options.Knowledge {
		if progressCallback != nil {
			progressCallback(10, "Backing up knowledge bases...")
		}
		logrus.Info("Backing up knowledge bases...")
		kbCount, err := backupAllKnowledgeBases(zipWriter, client)
		if err != nil {
			logrus.Warnf("Failed to backup some knowledge bases: %v", err)
		}
		if kbCount > 0 {
			containedTypes = append(containedTypes, "knowledge")
			totalItems += kbCount
			logrus.Infof("  Backed up %d knowledge base(s)", kbCount)
		}
	}

	if options.Models {
		if progressCallback != nil {
			progressCallback(25, "Backing up models...")
		}
		logrus.Info("Backing up models...")
		modelCount, err := backupAllModels(zipWriter, client)
		if err != nil {
			logrus.Warnf("Failed to backup some models: %v", err)
		}
		if modelCount > 0 {
			containedTypes = append(containedTypes, "model")
			totalItems += modelCount
			logrus.Infof("  Backed up %d model(s)", modelCount)
		}
	}

	if options.Tools {
		if progressCallback != nil {
			progressCallback(40, "Backing up tools...")
		}
		logrus.Info("Backing up tools...")
		toolCount, err := backupAllTools(zipWriter, client)
		if err != nil {
			logrus.Warnf("Failed to backup some tools: %v", err)
		}
		if toolCount > 0 {
			containedTypes = append(containedTypes, "tool")
			totalItems += toolCount
			logrus.Infof("  Backed up %d tool(s)", toolCount)
		}
	}

	if options.Prompts {
		if progressCallback != nil {
			progressCallback(55, "Backing up prompts...")
		}
		logrus.Info("Backing up prompts...")
		promptCount, err := backupAllPrompts(zipWriter, client)
		if err != nil {
			logrus.Warnf("Failed to backup some prompts: %v", err)
		}
		if promptCount > 0 {
			containedTypes = append(containedTypes, "prompt")
			totalItems += promptCount
			logrus.Infof("  Backed up %d prompt(s)", promptCount)
		}
	}

	if options.Files {
		if progressCallback != nil {
			progressCallback(65, "Backing up files...")
		}
		logrus.Info("Backing up files...")
		fileCount, err := backupAllFiles(zipWriter, client)
		if err != nil {
			logrus.Warnf("Failed to backup some files: %v", err)
		}
		if fileCount > 0 {
			containedTypes = append(containedTypes, "file")
			totalItems += fileCount
			logrus.Infof("  Backed up %d file(s)", fileCount)
		}
	}

	if options.Chats {
		if progressCallback != nil {
			progressCallback(75, "Backing up chats...")
		}
		logrus.Info("Backing up chats...")
		chatCount, err := backupAllChats(zipWriter, client)
		if err != nil {
			logrus.Warnf("Failed to backup some chats: %v", err)
		}
		if chatCount > 0 {
			containedTypes = append(containedTypes, "chat")
			totalItems += chatCount
			logrus.Infof("  Backed up %d chat(s)", chatCount)
		}
	}

	if options.Groups {
		if progressCallback != nil {
			progressCallback(82, "Backing up groups...")
		}
		logrus.Info("Backing up groups...")
		groupCount, err := backupAllGroups(zipWriter, client)
		if err != nil {
			logrus.Warnf("Failed to backup some groups: %v", err)
		}
		if groupCount > 0 {
			containedTypes = append(containedTypes, "group")
			totalItems += groupCount
			logrus.Infof("  Backed up %d group(s)", groupCount)
		}
	}

	if options.Feedbacks {
		if progressCallback != nil {
			progressCallback(88, "Backing up feedbacks...")
		}
		logrus.Info("Backing up feedbacks...")
		feedbackCount, err := backupAllFeedbacks(zipWriter, client)
		if err != nil {
			logrus.Warnf("Failed to backup some feedbacks: %v", err)
		}
		if feedbackCount > 0 {
			containedTypes = append(containedTypes, "feedback")
			totalItems += feedbackCount
			logrus.Infof("  Backed up %d feedback(s)", feedbackCount)
		}
	}

	// IMPORTANT: Users must be backed up LAST
	if options.Users {
		if progressCallback != nil {
			progressCallback(93, "Backing up users...")
		}
		logrus.Info("Backing up users...")
		userCount, err := backupAllUsers(zipWriter, client)
		if err != nil {
			logrus.Warnf("Failed to backup some users: %v", err)
		}
		if userCount > 0 {
			containedTypes = append(containedTypes, "user")
			totalItems += userCount
			logrus.Infof("  Backed up %d user(s)", userCount)
		}
	}

	// Determine backup type string
	backupType := "selective"
	if len(containedTypes) == 1 {
		backupType = containedTypes[0]
	}

	// Add unified metadata
	metadata := generateMetadata(client, backupType, totalItems, true, containedTypes)
	if err := writeMetadataToZip(zipWriter, metadata); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	if progressCallback != nil {
		progressCallback(100, "Backup completed successfully")
	}
	logrus.Infof("Created selective backup: %s (%d total items)", filepath.Base(outputFile), totalItems)
	logrus.Info("Selective backup completed successfully")
	return nil
}

// BackupAll backs up all data types into a single unified ZIP file
func BackupAll(client *openwebui.Client, outputDir string) error {
	logrus.Info("Starting unified full backup...")

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate unified ZIP filename
	timestamp := time.Now().UTC().Format("20060102_150405")
	zipFilename := fmt.Sprintf("%s_owui_full_backup.zip", timestamp)
	zipPath := filepath.Join(outputDir, zipFilename)

	// Check if file already exists
	if _, err := os.Stat(zipPath); err == nil {
		return &openwebui.FileExistsError{Path: zipPath}
	}

	// Create ZIP file
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return fmt.Errorf("failed to create zip file: %w", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Track all contained types and total item count
	containedTypes := []string{}
	totalItems := 0

	// Step 1: Backup knowledge bases
	logrus.Info("Step 1/5: Backing up knowledge bases...")
	kbCount, err := backupAllKnowledgeBases(zipWriter, client)
	if err != nil {
		logrus.Warnf("Failed to backup some knowledge bases: %v", err)
	}
	if kbCount > 0 {
		containedTypes = append(containedTypes, "knowledge")
		totalItems += kbCount
		logrus.Infof("  Backed up %d knowledge base(s)", kbCount)
	}

	// Step 2: Backup models
	logrus.Info("Step 2/5: Backing up models...")
	modelCount, err := backupAllModels(zipWriter, client)
	if err != nil {
		logrus.Warnf("Failed to backup some models: %v", err)
	}
	if modelCount > 0 {
		containedTypes = append(containedTypes, "model")
		totalItems += modelCount
		logrus.Infof("  Backed up %d model(s)", modelCount)
	}

	// Step 3: Backup tools
	logrus.Info("Step 3/5: Backing up tools...")
	toolCount, err := backupAllTools(zipWriter, client)
	if err != nil {
		logrus.Warnf("Failed to backup some tools: %v", err)
	}
	if toolCount > 0 {
		containedTypes = append(containedTypes, "tool")
		totalItems += toolCount
		logrus.Infof("  Backed up %d tool(s)", toolCount)
	}

	// Step 4: Backup prompts
	logrus.Info("Step 4/5: Backing up prompts...")
	promptCount, err := backupAllPrompts(zipWriter, client)
	if err != nil {
		logrus.Warnf("Failed to backup some prompts: %v", err)
	}
	if promptCount > 0 {
		containedTypes = append(containedTypes, "prompt")
		totalItems += promptCount
		logrus.Infof("  Backed up %d prompt(s)", promptCount)
	}

	// Step 5: Backup files
	logrus.Info("Step 5/6: Backing up files...")
	fileCount, err := backupAllFiles(zipWriter, client)
	if err != nil {
		logrus.Warnf("Failed to backup some files: %v", err)
	}
	if fileCount > 0 {
		containedTypes = append(containedTypes, "file")
		totalItems += fileCount
		logrus.Infof("  Backed up %d file(s)", fileCount)
	}

	// Step 6: Backup chats
	logrus.Info("Step 6/9: Backing up chats...")
	chatCount, err := backupAllChats(zipWriter, client)
	if err != nil {
		logrus.Warnf("Failed to backup some chats: %v", err)
	}
	if chatCount > 0 {
		containedTypes = append(containedTypes, "chat")
		totalItems += chatCount
		logrus.Infof("  Backed up %d chat(s)", chatCount)
	}

	// Step 7: Backup groups
	logrus.Info("Step 7/9: Backing up groups...")
	groupCount, err := backupAllGroups(zipWriter, client)
	if err != nil {
		logrus.Warnf("Failed to backup some groups: %v", err)
	}
	if groupCount > 0 {
		containedTypes = append(containedTypes, "group")
		totalItems += groupCount
		logrus.Infof("  Backed up %d group(s)", groupCount)
	}

	// Step 8: Backup feedbacks
	logrus.Info("Step 8/9: Backing up feedbacks...")
	feedbackCount, err := backupAllFeedbacks(zipWriter, client)
	if err != nil {
		logrus.Warnf("Failed to backup some feedbacks: %v", err)
	}
	if feedbackCount > 0 {
		containedTypes = append(containedTypes, "feedback")
		totalItems += feedbackCount
		logrus.Infof("  Backed up %d feedback(s)", feedbackCount)
	}

	// Step 9: Backup users (MUST be LAST)
	logrus.Info("Step 9/9: Backing up users...")
	userCount, err := backupAllUsers(zipWriter, client)
	if err != nil {
		logrus.Warnf("Failed to backup some users: %v", err)
	}
	if userCount > 0 {
		containedTypes = append(containedTypes, "user")
		totalItems += userCount
		logrus.Infof("  Backed up %d user(s)", userCount)
	}

	// Add unified metadata
	metadata := generateMetadata(client, "all", totalItems, true, containedTypes)
	if err := writeMetadataToZip(zipWriter, metadata); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	logrus.Infof("Created unified backup: %s (%d total items)", zipFilename, totalItems)
	logrus.Info("Full backup completed successfully")
	return nil
}

// backupAllKnowledgeBases backs up all knowledge bases into the unified ZIP
func backupAllKnowledgeBases(zipWriter *zip.Writer, client *openwebui.Client) (int, error) {
	knowledgeBases, err := client.ListKnowledge()
	if err != nil {
		return 0, fmt.Errorf("failed to list knowledge bases: %w", err)
	}

	for i, kb := range knowledgeBases {
		logrus.Infof("  Backing up knowledge base %d/%d: %s", i+1, len(knowledgeBases), kb.Name)
		if err := backupKnowledgeToZip(zipWriter, &kb, client); err != nil {
			logrus.Warnf("  Failed to backup knowledge base '%s': %v", kb.Name, err)
			continue
		}
	}

	return len(knowledgeBases), nil
}

// backupKnowledgeToZip backs up a single knowledge base into an existing ZIP writer
func backupKnowledgeToZip(zipWriter *zip.Writer, kb *openwebui.KnowledgeBase, client *openwebui.Client) error {
	// Create knowledge-bases/{id}/ directory
	kbDir := fmt.Sprintf("knowledge-bases/%s/", kb.ID)

	// Add knowledge_base.json
	kbJSON, err := json.MarshalIndent(kb, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal knowledge base: %w", err)
	}

	kbFile, err := zipWriter.Create(kbDir + "knowledge_base.json")
	if err != nil {
		return fmt.Errorf("failed to create knowledge_base.json in zip: %w", err)
	}
	if _, err := kbFile.Write(kbJSON); err != nil {
		return fmt.Errorf("failed to write knowledge_base.json: %w", err)
	}

	// Collect and download files
	var fileIDs []string
	if kb.Data != nil && kb.Data.FileIDs != nil {
		fileIDs = kb.Data.FileIDs
	}

	for _, fileID := range fileIDs {
		fileData, err := client.GetFile(fileID)
		if err != nil {
			logrus.Warnf("    Failed to download file %s: %v", fileID, err)
			continue
		}

		filename := fileData.Meta.Name
		if filename == "" {
			filename = fileData.Filename
		}
		if filename == "" {
			filename = fmt.Sprintf("file_%s", fileID)
		}

		var content []byte
		if fileData.Data != nil && fileData.Data.Content != "" {
			content = []byte(fileData.Data.Content)
		}

		docPath := kbDir + "documents/" + filename
		docFile, err := zipWriter.Create(docPath)
		if err != nil {
			logrus.Warnf("    Failed to create %s in zip: %v", docPath, err)
			continue
		}
		if _, err := docFile.Write(content); err != nil {
			logrus.Warnf("    Failed to write %s: %v", docPath, err)
			continue
		}
	}

	return nil
}

// backupAllModels backs up all models into the unified ZIP
func backupAllModels(zipWriter *zip.Writer, client *openwebui.Client) (int, error) {
	models, err := client.ExportModels()
	if err != nil {
		return 0, fmt.Errorf("failed to export models: %w", err)
	}

	for i, model := range models {
		logrus.Infof("  Backing up model %d/%d: %s", i+1, len(models), model.Name)
		if err := backupModelToZip(zipWriter, &model, client); err != nil {
			logrus.Warnf("  Failed to backup model '%s': %v", model.Name, err)
			continue
		}
	}

	return len(models), nil
}

// backupModelToZip backs up a single model into an existing ZIP writer
func backupModelToZip(zipWriter *zip.Writer, model *openwebui.Model, client *openwebui.Client) error {
	// Create models/{id}/ directory
	modelDir := fmt.Sprintf("models/%s/", model.ID)

	// Add model.json
	modelJSON, err := json.MarshalIndent(model, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal model: %w", err)
	}

	modelFile, err := zipWriter.Create(modelDir + "model.json")
	if err != nil {
		return fmt.Errorf("failed to create model.json in zip: %w", err)
	}
	if _, err := modelFile.Write(modelJSON); err != nil {
		return fmt.Errorf("failed to write model.json: %w", err)
	}

	// Backup embedded knowledge bases if present
	if len(model.Meta.Knowledge) > 0 {
		for _, item := range model.Meta.Knowledge {
			itemType, ok := item["type"].(string)
			if !ok {
				continue
			}

			switch itemType {
			case "file":
				if err := backupModelFileItem(zipWriter, modelDir, item, client); err != nil {
					logrus.Warnf("    Failed to backup file item: %v", err)
				}
			case "collection":
				if err := backupModelCollectionItem(zipWriter, modelDir, item, client); err != nil {
					logrus.Warnf("    Failed to backup collection item: %v", err)
				}
			}
		}
	}

	return nil
}

// backupModelFileItem backs up a file-type knowledge item for a model
func backupModelFileItem(zipWriter *zip.Writer, modelDir string, item map[string]interface{}, client *openwebui.Client) error {
	fileID, ok := item["id"].(string)
	if !ok {
		return fmt.Errorf("file item missing 'id' field")
	}

	// Save metadata
	itemJSON, err := json.MarshalIndent(item, "", "  ")
	if err != nil {
		return err
	}

	metaPath := modelDir + fmt.Sprintf("model-files/%s/metadata.json", fileID)
	metaFile, err := zipWriter.Create(metaPath)
	if err != nil {
		return err
	}
	if _, err := metaFile.Write(itemJSON); err != nil {
		return err
	}

	// Download and save file content
	fileData, err := client.GetFile(fileID)
	if err != nil {
		return err
	}

	filename := fileData.Meta.Name
	if filename == "" {
		filename = fileData.Filename
	}
	if filename == "" {
		filename = fmt.Sprintf("file_%s", fileID)
	}

	var content []byte
	if fileData.Data != nil && fileData.Data.Content != "" {
		content = []byte(fileData.Data.Content)
	}

	contentPath := modelDir + fmt.Sprintf("model-files/%s/%s", fileID, filename)
	contentFile, err := zipWriter.Create(contentPath)
	if err != nil {
		return err
	}
	if _, err := contentFile.Write(content); err != nil {
		return err
	}

	return nil
}

// backupModelCollectionItem backs up a collection-type knowledge item for a model
func backupModelCollectionItem(zipWriter *zip.Writer, modelDir string, item map[string]interface{}, client *openwebui.Client) error {
	kbID, ok := item["id"].(string)
	if !ok {
		return fmt.Errorf("collection item missing 'id' field")
	}

	kbDir := modelDir + fmt.Sprintf("knowledge-bases/%s/", kbID)

	// Save knowledge base metadata
	kbJSON, err := json.MarshalIndent(item, "", "  ")
	if err != nil {
		return err
	}

	kbFile, err := zipWriter.Create(kbDir + "knowledge_base.json")
	if err != nil {
		return err
	}
	if _, err := kbFile.Write(kbJSON); err != nil {
		return err
	}

	// Extract and download files
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

	for _, fileID := range fileIDs {
		fileData, err := client.GetFile(fileID)
		if err != nil {
			logrus.Warnf("      Failed to download file %s: %v", fileID, err)
			continue
		}

		filename := fileData.Meta.Name
		if filename == "" {
			filename = fileData.Filename
		}
		if filename == "" {
			filename = fmt.Sprintf("file_%s", fileID)
		}

		var content []byte
		if fileData.Data != nil && fileData.Data.Content != "" {
			content = []byte(fileData.Data.Content)
		}

		filePath := kbDir + "documents/" + filename
		docFile, err := zipWriter.Create(filePath)
		if err != nil {
			continue
		}
		docFile.Write(content)
	}

	return nil
}

// backupAllTools backs up all tools into the unified ZIP
func backupAllTools(zipWriter *zip.Writer, client *openwebui.Client) (int, error) {
	tools, err := client.ExportTools()
	if err != nil {
		return 0, fmt.Errorf("failed to export tools: %w", err)
	}

	for i, tool := range tools {
		logrus.Infof("  Backing up tool %d/%d: %s", i+1, len(tools), tool.Name)
		if err := backupToolToZip(zipWriter, &tool); err != nil {
			logrus.Warnf("  Failed to backup tool '%s': %v", tool.Name, err)
			continue
		}
	}

	return len(tools), nil
}

// backupToolToZip backs up a single tool into an existing ZIP writer
func backupToolToZip(zipWriter *zip.Writer, tool *openwebui.Tool) error {
	// Create tools/{id}/ directory
	toolDir := fmt.Sprintf("tools/%s/", tool.ID)

	// Add tool.json
	toolJSON, err := json.MarshalIndent(tool, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tool: %w", err)
	}

	toolFile, err := zipWriter.Create(toolDir + "tool.json")
	if err != nil {
		return fmt.Errorf("failed to create tool.json in zip: %w", err)
	}
	if _, err := toolFile.Write(toolJSON); err != nil {
		return fmt.Errorf("failed to write tool.json: %w", err)
	}

	return nil
}

// backupAllPrompts backs up all prompts into the unified ZIP
func backupAllPrompts(zipWriter *zip.Writer, client *openwebui.Client) (int, error) {
	prompts, err := client.ListPrompts()
	if err != nil {
		return 0, fmt.Errorf("failed to list prompts: %w", err)
	}

	for i, prompt := range prompts {
		logrus.Infof("  Backing up prompt %d/%d: %s", i+1, len(prompts), prompt.Title)
		if err := backupPromptToZip(zipWriter, &prompt); err != nil {
			logrus.Warnf("  Failed to backup prompt '%s': %v", prompt.Title, err)
			continue
		}
	}

	return len(prompts), nil
}

// backupPromptToZip backs up a single prompt into an existing ZIP writer
func backupPromptToZip(zipWriter *zip.Writer, prompt *openwebui.Prompt) error {
	// Create prompts/{command}/ directory
	sanitizedCommand := sanitizeFilename(prompt.Command)
	promptDir := fmt.Sprintf("prompts/%s/", sanitizedCommand)

	// Add prompt.json
	promptJSON, err := json.MarshalIndent(prompt, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal prompt: %w", err)
	}

	promptFile, err := zipWriter.Create(promptDir + "prompt.json")
	if err != nil {
		return fmt.Errorf("failed to create prompt.json in zip: %w", err)
	}
	if _, err := promptFile.Write(promptJSON); err != nil {
		return fmt.Errorf("failed to write prompt.json: %w", err)
	}

	return nil
}

// backupAllFiles backs up all files into the unified ZIP
func backupAllFiles(zipWriter *zip.Writer, client *openwebui.Client) (int, error) {
	files, err := client.ListFiles()
	if err != nil {
		return 0, fmt.Errorf("failed to list files: %w", err)
	}

	for i, fileMeta := range files {
		logrus.Infof("  Backing up file %d/%d: %s", i+1, len(files), fileMeta.Meta.Name)
		if err := backupFileToZip(zipWriter, fileMeta.ID, client); err != nil {
			logrus.Warnf("  Failed to backup file '%s': %v", fileMeta.Meta.Name, err)
			continue
		}
	}

	return len(files), nil
}

// backupFileToZip backs up a single file into an existing ZIP writer
func backupFileToZip(zipWriter *zip.Writer, fileID string, client *openwebui.Client) error {
	// Get file with content
	fileExport, err := client.GetFileWithContent(fileID)
	if err != nil {
		return fmt.Errorf("failed to get file with content: %w", err)
	}

	// Create files/{id}/ directory
	fileDir := fmt.Sprintf("files/%s/", fileID)

	// Add file.json
	fileJSON, err := json.MarshalIndent(fileExport, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal file: %w", err)
	}

	metaFile, err := zipWriter.Create(fileDir + "file.json")
	if err != nil {
		return fmt.Errorf("failed to create file.json in zip: %w", err)
	}
	if _, err := metaFile.Write(fileJSON); err != nil {
		return fmt.Errorf("failed to write file.json: %w", err)
	}

	// Add file content
	var content []byte
	if fileExport.Data != nil && fileExport.Data.Content != "" {
		content = []byte(fileExport.Data.Content)
	}

	contentPath := fileDir + "content/" + fileExport.Filename
	contentFile, err := zipWriter.Create(contentPath)
	if err != nil {
		return fmt.Errorf("failed to create content file in zip: %w", err)
	}
	if _, err := contentFile.Write(content); err != nil {
		return fmt.Errorf("failed to write content: %w", err)
	}

	return nil
}

// BackupGroups is the main entry point for backing up all groups
func BackupGroups(client *openwebui.Client, outputDir string) error {
	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	logrus.Info("Fetching groups...")
	groups, err := client.GetAllGroups()
	if err != nil {
		return fmt.Errorf("failed to get groups: %w", err)
	}

	if len(groups) == 0 {
		logrus.Info("No groups found")
		return nil
	}

	logrus.Infof("Found %d group(s)", len(groups))

	// Backup all groups to a single ZIP file
	timestamp := time.Now().UTC().Format("20060102_150405")
	zipFilename := fmt.Sprintf("%s_groups_backup.zip", timestamp)
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

	// Backup each group
	for i, group := range groups {
		logrus.Infof("Backing up group %d/%d: %s", i+1, len(groups), group.Name)
		if err := backupGroupToZip(zipWriter, &group); err != nil {
			logrus.Warnf("Failed to backup group '%s': %v", group.Name, err)
			continue
		}
	}

	// Add owui.json metadata
	metadata := generateMetadata(client, "group", len(groups), false, nil)
	if err := writeMetadataToZip(zipWriter, metadata); err != nil {
		logrus.Warnf("  Failed to write metadata: %v", err)
	}

	logrus.Infof("  Created: %s", zipFilename)
	logrus.Info("All groups backed up successfully")
	return nil
}

// backupAllGroups backs up all groups into the unified ZIP
func backupAllGroups(zipWriter *zip.Writer, client *openwebui.Client) (int, error) {
	groups, err := client.GetAllGroups()
	if err != nil {
		return 0, fmt.Errorf("failed to get groups: %w", err)
	}

	for i, group := range groups {
		logrus.Infof("  Backing up group %d/%d: %s", i+1, len(groups), group.Name)
		if err := backupGroupToZip(zipWriter, &group); err != nil {
			logrus.Warnf("  Failed to backup group '%s': %v", group.Name, err)
			continue
		}
	}

	return len(groups), nil
}

// backupGroupToZip backs up a single group into an existing ZIP writer
func backupGroupToZip(zipWriter *zip.Writer, group *openwebui.Group) error {
	// Create groups/{id}/ directory
	groupDir := fmt.Sprintf("groups/%s/", group.ID)

	// Add group.json
	groupJSON, err := json.MarshalIndent(group, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal group: %w", err)
	}

	groupFile, err := zipWriter.Create(groupDir + "group.json")
	if err != nil {
		return fmt.Errorf("failed to create group.json in zip: %w", err)
	}
	if _, err := groupFile.Write(groupJSON); err != nil {
		return fmt.Errorf("failed to write group.json: %w", err)
	}

	return nil
}

// BackupFeedbacks is the main entry point for backing up all feedbacks
func BackupFeedbacks(client *openwebui.Client, outputDir string) error {
	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	logrus.Info("Fetching feedbacks...")
	feedbacks, err := client.GetAllFeedbacks()
	if err != nil {
		return fmt.Errorf("failed to get feedbacks: %w", err)
	}

	if len(feedbacks) == 0 {
		logrus.Info("No feedbacks found")
		return nil
	}

	logrus.Infof("Found %d feedback(s)", len(feedbacks))

	// Backup all feedbacks to a single ZIP file
	timestamp := time.Now().UTC().Format("20060102_150405")
	zipFilename := fmt.Sprintf("%s_feedbacks_backup.zip", timestamp)
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

	// Backup each feedback
	for i, feedback := range feedbacks {
		logrus.Infof("Backing up feedback %d/%d (ID: %s)", i+1, len(feedbacks), feedback.ID)
		if err := backupFeedbackToZip(zipWriter, &feedback); err != nil {
			logrus.Warnf("Failed to backup feedback '%s': %v", feedback.ID, err)
			continue
		}
	}

	// Add owui.json metadata
	metadata := generateMetadata(client, "feedback", len(feedbacks), false, nil)
	if err := writeMetadataToZip(zipWriter, metadata); err != nil {
		logrus.Warnf("  Failed to write metadata: %v", err)
	}

	logrus.Infof("  Created: %s", zipFilename)
	logrus.Info("All feedbacks backed up successfully")
	return nil
}

// backupAllFeedbacks backs up all feedbacks into the unified ZIP
func backupAllFeedbacks(zipWriter *zip.Writer, client *openwebui.Client) (int, error) {
	feedbacks, err := client.GetAllFeedbacks()
	if err != nil {
		return 0, fmt.Errorf("failed to get feedbacks: %w", err)
	}

	for i, feedback := range feedbacks {
		logrus.Infof("  Backing up feedback %d/%d (ID: %s)", i+1, len(feedbacks), feedback.ID)
		if err := backupFeedbackToZip(zipWriter, &feedback); err != nil {
			logrus.Warnf("  Failed to backup feedback '%s': %v", feedback.ID, err)
			continue
		}
	}

	return len(feedbacks), nil
}

// backupFeedbackToZip backs up a single feedback into an existing ZIP writer
func backupFeedbackToZip(zipWriter *zip.Writer, feedback *openwebui.Feedback) error {
	// Create feedbacks/{id}/ directory
	feedbackDir := fmt.Sprintf("feedbacks/%s/", feedback.ID)

	// Add feedback.json
	feedbackJSON, err := json.MarshalIndent(feedback, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal feedback: %w", err)
	}

	feedbackFile, err := zipWriter.Create(feedbackDir + "feedback.json")
	if err != nil {
		return fmt.Errorf("failed to create feedback.json in zip: %w", err)
	}
	if _, err := feedbackFile.Write(feedbackJSON); err != nil {
		return fmt.Errorf("failed to write feedback.json: %w", err)
	}

	return nil
}

// BackupUsers is the main entry point for backing up all users
func BackupUsers(client *openwebui.Client, outputDir string) error {
	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	logrus.Info("Fetching users...")
	users, err := client.GetAllUsers()
	if err != nil {
		return fmt.Errorf("failed to get users: %w", err)
	}

	if len(users) == 0 {
		logrus.Info("No users found")
		return nil
	}

	logrus.Infof("Found %d user(s)", len(users))

	// Backup all users to a single ZIP file
	timestamp := time.Now().UTC().Format("20060102_150405")
	zipFilename := fmt.Sprintf("%s_users_backup.zip", timestamp)
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

	// Backup each user
	for i, user := range users {
		logrus.Infof("Backing up user %d/%d: %s", i+1, len(users), user.Name)
		if err := backupUserToZip(zipWriter, &user); err != nil {
			logrus.Warnf("Failed to backup user '%s': %v", user.Name, err)
			continue
		}
	}

	// Add owui.json metadata
	metadata := generateMetadata(client, "user", len(users), false, nil)
	if err := writeMetadataToZip(zipWriter, metadata); err != nil {
		logrus.Warnf("  Failed to write metadata: %v", err)
	}

	logrus.Infof("  Created: %s", zipFilename)
	logrus.Info("All users backed up successfully")
	return nil
}

// backupAllUsers backs up all users into the unified ZIP
func backupAllUsers(zipWriter *zip.Writer, client *openwebui.Client) (int, error) {
	users, err := client.GetAllUsers()
	if err != nil {
		return 0, fmt.Errorf("failed to get users: %w", err)
	}

	for i, user := range users {
		logrus.Infof("  Backing up user %d/%d: %s", i+1, len(users), user.Name)
		if err := backupUserToZip(zipWriter, &user); err != nil {
			logrus.Warnf("  Failed to backup user '%s': %v", user.Name, err)
			continue
		}
	}

	return len(users), nil
}

// backupUserToZip backs up a single user into an existing ZIP writer
func backupUserToZip(zipWriter *zip.Writer, user *openwebui.User) error {
	// Create users/{id}/ directory
	userDir := fmt.Sprintf("users/%s/", user.ID)

	// Add user.json
	userJSON, err := json.MarshalIndent(user, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal user: %w", err)
	}

	userFile, err := zipWriter.Create(userDir + "user.json")
	if err != nil {
		return fmt.Errorf("failed to create user.json in zip: %w", err)
	}
	if _, err := userFile.Write(userJSON); err != nil {
		return fmt.Errorf("failed to write user.json: %w", err)
	}

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

// generateMetadata creates backup metadata for a given backup operation
func generateMetadata(client *openwebui.Client, backupType string, itemCount int, unifiedBackup bool, containedTypes []string) *openwebui.BackupMetadata {
	return &openwebui.BackupMetadata{
		OpenWebUIURL:      client.GetBaseURL(),
		OpenWebUIVersion:  "", // Will be populated if API provides version info
		BackupToolVersion: config.BackupToolVersion,
		BackupTimestamp:   time.Now().UTC().Format(time.RFC3339),
		BackupType:        backupType,
		ItemCount:         itemCount,
		UnifiedBackup:     unifiedBackup,
		ContainedTypes:    containedTypes,
	}
}

// writeMetadataToZip writes the owui.json metadata file to the ZIP archive
func writeMetadataToZip(zipWriter *zip.Writer, metadata *openwebui.BackupMetadata) error {
	metadataJSON, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	metadataFile, err := zipWriter.Create("owui.json")
	if err != nil {
		return fmt.Errorf("failed to create owui.json in zip: %w", err)
	}

	if _, err := metadataFile.Write(metadataJSON); err != nil {
		return fmt.Errorf("failed to write owui.json: %w", err)
	}

	return nil
}
