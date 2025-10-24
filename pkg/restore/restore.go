package restore

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/vosiander/open-webui-backup/pkg/openwebui"
)

// RestoreKnowledge restores a knowledge base from a backup ZIP file
// If overwrite is true, existing files will be replaced; otherwise they are skipped
func RestoreKnowledge(client *openwebui.Client, zipPath string, overwrite bool) error {
	logrus.Infof("Starting restore from: %s", zipPath)

	// Extract knowledge base and files from ZIP
	kb, files, err := extractZipFile(zipPath)
	if err != nil {
		return fmt.Errorf("failed to extract ZIP file: %w", err)
	}

	logrus.Infof("Extracted knowledge base: %s", kb.Name)
	logrus.Infof("Found %d files to restore", len(files))

	// Check if knowledge base already exists
	existingKB, err := findKnowledgeByName(client, kb.Name)
	if err != nil {
		return fmt.Errorf("failed to check existing knowledge bases: %w", err)
	}

	var knowledgeID string
	if existingKB == nil {
		// Create new knowledge base
		logrus.Infof("Creating knowledge base: %s", kb.Name)
		form := &openwebui.KnowledgeForm{
			Name:          kb.Name,
			Description:   kb.Description,
			AccessControl: kb.AccessControl,
		}

		createResp, err := client.CreateKnowledge(form)
		if err != nil {
			return fmt.Errorf("failed to create knowledge base: %w", err)
		}

		knowledgeID = createResp.ID
		logrus.Infof("Knowledge base created with ID: %s", knowledgeID)

		// Upload and link all files
		fileIDMap, err := uploadFiles(client, files)
		if err != nil {
			return fmt.Errorf("failed to upload files: %w", err)
		}

		fileIDs := make([]string, 0, len(fileIDMap))
		for _, fileID := range fileIDMap {
			fileIDs = append(fileIDs, fileID)
		}

		if err := linkFilesToKnowledge(client, knowledgeID, fileIDs); err != nil {
			return fmt.Errorf("failed to link files to knowledge base: %w", err)
		}

		logrus.Infof("Successfully restored knowledge base: %s", kb.Name)
	} else {
		// Update existing knowledge base
		logrus.Infof("Knowledge base '%s' already exists (ID: %s)", kb.Name, existingKB.ID)
		knowledgeID = existingKB.ID

		// Update metadata if different
		if err := updateKnowledgeMetadata(client, existingKB, kb); err != nil {
			return fmt.Errorf("failed to update knowledge base metadata: %w", err)
		}

		// Get existing files
		existingFiles := getExistingFileMap(existingKB)
		logrus.Infof("Found %d existing files in knowledge base", len(existingFiles))

		// Sync files (upload new, overwrite if requested, skip otherwise)
		stats, err := syncFiles(client, knowledgeID, files, existingFiles, overwrite)
		if err != nil {
			return fmt.Errorf("failed to sync files: %w", err)
		}

		logrus.Infof("File sync completed: %d new, %d overwritten, %d skipped",
			stats.New, stats.Overwritten, stats.Skipped)
		logrus.Infof("Successfully restored knowledge base: %s", kb.Name)
	}

	return nil
}

// extractZipFile extracts knowledge base metadata and document files from ZIP
func extractZipFile(zipPath string) (*openwebui.KnowledgeBase, map[string][]byte, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open ZIP file: %w", err)
	}
	defer r.Close()

	var kb *openwebui.KnowledgeBase
	files := make(map[string][]byte)

	for _, f := range r.File {
		// Read knowledge_base.json
		if f.Name == "knowledge_base.json" {
			rc, err := f.Open()
			if err != nil {
				return nil, nil, fmt.Errorf("failed to open knowledge_base.json: %w", err)
			}

			data, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				return nil, nil, fmt.Errorf("failed to read knowledge_base.json: %w", err)
			}

			kb = &openwebui.KnowledgeBase{}
			if err := json.Unmarshal(data, kb); err != nil {
				return nil, nil, fmt.Errorf("failed to parse knowledge_base.json: %w", err)
			}
			continue
		}

		// Read document files from documents/ folder
		if strings.HasPrefix(f.Name, "documents/") && !f.FileInfo().IsDir() {
			rc, err := f.Open()
			if err != nil {
				return nil, nil, fmt.Errorf("failed to open file %s: %w", f.Name, err)
			}

			content, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				return nil, nil, fmt.Errorf("failed to read file %s: %w", f.Name, err)
			}

			// Use only the filename, not the full path
			filename := filepath.Base(f.Name)
			files[filename] = content
		}
	}

	if kb == nil {
		return nil, nil, fmt.Errorf("knowledge_base.json not found in ZIP file")
	}

	return kb, files, nil
}

// checkKnowledgeExists checks if a knowledge base with the given name already exists
func checkKnowledgeExists(client *openwebui.Client, name string) (bool, error) {
	knowledgeBases, err := client.ListKnowledge()
	if err != nil {
		return false, err
	}

	for _, kb := range knowledgeBases {
		if strings.EqualFold(kb.Name, name) {
			return true, nil
		}
	}

	return false, nil
}

// uploadFiles uploads all document files and returns a map of filename to file ID
func uploadFiles(client *openwebui.Client, files map[string][]byte) (map[string]string, error) {
	fileIDMap := make(map[string]string)

	for filename, content := range files {
		logrus.Infof("Uploading file: %s (%d bytes)", filename, len(content))

		resp, err := client.UploadFile(filename, content)
		if err != nil {
			return nil, fmt.Errorf("failed to upload file %s: %w", filename, err)
		}

		fileIDMap[filename] = resp.ID
		logrus.Infof("File uploaded successfully: %s (ID: %s)", filename, resp.ID)
	}

	return fileIDMap, nil
}

// linkFilesToKnowledge associates all uploaded files with the knowledge base
func linkFilesToKnowledge(client *openwebui.Client, knowledgeID string, fileIDs []string) error {
	logrus.Infof("Linking %d files to knowledge base", len(fileIDs))

	for i, fileID := range fileIDs {
		logrus.Infof("Linking file %d/%d (ID: %s)", i+1, len(fileIDs), fileID)

		if err := client.AddFileToKnowledge(knowledgeID, fileID); err != nil {
			return fmt.Errorf("failed to link file %s: %w", fileID, err)
		}
	}

	logrus.Info("All files linked successfully")
	return nil
}

// findKnowledgeByName finds a knowledge base by name and returns it with full details
func findKnowledgeByName(client *openwebui.Client, name string) (*openwebui.KnowledgeBase, error) {
	knowledgeBases, err := client.ListKnowledge()
	if err != nil {
		return nil, err
	}

	for _, kb := range knowledgeBases {
		if strings.EqualFold(kb.Name, name) {
			// Get full details including files
			return client.GetKnowledgeByID(kb.ID)
		}
	}

	return nil, nil
}

// getExistingFileMap builds a map of filename to fileID from existing knowledge base
func getExistingFileMap(kb *openwebui.KnowledgeBase) map[string]string {
	fileMap := make(map[string]string)

	for _, file := range kb.Files {
		// Use the filename from meta.name
		if file.Meta.Name != "" {
			fileMap[file.Meta.Name] = file.ID
		}
	}

	return fileMap
}

// updateKnowledgeMetadata updates the knowledge base metadata if it differs from backup
func updateKnowledgeMetadata(client *openwebui.Client, existing, backup *openwebui.KnowledgeBase) error {
	needsUpdate := false

	if existing.Description != backup.Description {
		logrus.Infof("Description changed, updating...")
		needsUpdate = true
	}

	// Compare access control (simple comparison)
	if fmt.Sprintf("%v", existing.AccessControl) != fmt.Sprintf("%v", backup.AccessControl) {
		logrus.Infof("Access control changed, updating...")
		needsUpdate = true
	}

	if needsUpdate {
		form := &openwebui.KnowledgeForm{
			Name:          backup.Name,
			Description:   backup.Description,
			AccessControl: backup.AccessControl,
		}
		return client.UpdateKnowledge(existing.ID, form)
	}

	return nil
}

// SyncStats tracks file synchronization statistics
type SyncStats struct {
	New         int
	Overwritten int
	Skipped     int
}

// syncFiles synchronizes files between backup and existing knowledge base
func syncFiles(client *openwebui.Client, knowledgeID string, backupFiles map[string][]byte, existingFiles map[string]string, overwrite bool) (*SyncStats, error) {
	stats := &SyncStats{}

	for filename, content := range backupFiles {
		existingFileID, exists := existingFiles[filename]

		if !exists {
			// File doesn't exist, upload it
			logrus.Infof("Uploading new file: %s (%d bytes)", filename, len(content))

			resp, err := client.UploadFile(filename, content)
			if err != nil {
				return stats, fmt.Errorf("failed to upload file %s: %w", filename, err)
			}

			if err := client.AddFileToKnowledge(knowledgeID, resp.ID); err != nil {
				return stats, fmt.Errorf("failed to link file %s: %w", filename, err)
			}

			logrus.Infof("New file added: %s (ID: %s)", filename, resp.ID)
			stats.New++
		} else if overwrite {
			// File exists and overwrite is enabled
			logrus.Infof("Overwriting existing file: %s (%d bytes)", filename, len(content))

			// Remove old file from knowledge base
			if err := client.RemoveFileFromKnowledge(knowledgeID, existingFileID); err != nil {
				logrus.Warnf("Failed to remove old file %s: %v (continuing anyway)", filename, err)
			}

			// Upload new file
			resp, err := client.UploadFile(filename, content)
			if err != nil {
				return stats, fmt.Errorf("failed to upload file %s: %w", filename, err)
			}

			// Link new file
			if err := client.AddFileToKnowledge(knowledgeID, resp.ID); err != nil {
				return stats, fmt.Errorf("failed to link file %s: %w", filename, err)
			}

			logrus.Infof("File overwritten: %s (new ID: %s)", filename, resp.ID)
			stats.Overwritten++
		} else {
			// File exists but overwrite is disabled, skip it
			logrus.Infof("File already exists, skipping: %s", filename)
			stats.Skipped++
		}
	}

	return stats, nil
}

// RestoreModel restores a model from a backup ZIP file
// If overwrite is true, existing model with same ID will be replaced
func RestoreModel(client *openwebui.Client, zipPath string, overwrite bool) error {
	logrus.Infof("Starting model restore from: %s", zipPath)

	// Extract model from ZIP
	model, err := extractModelFromZip(zipPath)
	if err != nil {
		return fmt.Errorf("failed to extract model from ZIP: %w", err)
	}

	logrus.Infof("Extracted model: %s (ID: %s)", model.Name, model.ID)

	// Check if model already exists
	existingModel, err := client.GetModelByID(model.ID)
	if err == nil && existingModel != nil && !overwrite {
		return fmt.Errorf("model with ID %s already exists (use --overwrite to replace)", model.ID)
	}

	// Restore knowledge bases and get ID mapping
	kbIDMap, err := restoreKnowledgeBasesFromZip(client, zipPath, overwrite)
	if err != nil {
		return fmt.Errorf("failed to restore knowledge bases: %w", err)
	}

	// Update model's knowledge base references with new IDs
	if len(kbIDMap) > 0 {
		updateModelKnowledgeReferences(model, kbIDMap)
	}

	// Import the model
	logrus.Infof("Importing model: %s", model.Name)
	models := []openwebui.Model{*model}
	if err := client.ImportModels(models); err != nil {
		return fmt.Errorf("failed to import model: %w", err)
	}

	logrus.Infof("Successfully restored model: %s", model.Name)
	return nil
}

// extractModelFromZip extracts the model.json from the ZIP file root
func extractModelFromZip(zipPath string) (*openwebui.Model, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open ZIP file: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		if f.Name == "model.json" {
			rc, err := f.Open()
			if err != nil {
				return nil, fmt.Errorf("failed to open model.json: %w", err)
			}

			data, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				return nil, fmt.Errorf("failed to read model.json: %w", err)
			}

			var model openwebui.Model
			if err := json.Unmarshal(data, &model); err != nil {
				return nil, fmt.Errorf("failed to parse model.json: %w", err)
			}

			return &model, nil
		}
	}

	return nil, fmt.Errorf("model.json not found in ZIP file")
}

// restoreKnowledgeBasesFromZip restores all knowledge items (files and collections) from ZIP
// Returns a map of old item ID to new item ID
func restoreKnowledgeBasesFromZip(client *openwebui.Client, zipPath string, overwrite bool) (map[string]string, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open ZIP file: %w", err)
	}
	defer r.Close()

	idMap := make(map[string]string)

	// Restore file-type knowledge items
	fileIDMap, err := restoreFileKnowledgeItems(r, client)
	if err != nil {
		logrus.Warnf("Failed to restore file knowledge items: %v", err)
	}
	for oldID, newID := range fileIDMap {
		idMap[oldID] = newID
	}

	// Restore collection-type knowledge items (full KBs)
	collectionIDMap, err := restoreCollectionKnowledgeItems(r, client, overwrite)
	if err != nil {
		logrus.Warnf("Failed to restore collection knowledge items: %v", err)
	}
	for oldID, newID := range collectionIDMap {
		idMap[oldID] = newID
	}

	return idMap, nil
}

// restoreFileKnowledgeItems restores file-type knowledge items from model-files/ directory
func restoreFileKnowledgeItems(r *zip.ReadCloser, client *openwebui.Client) (map[string]string, error) {
	fileIDMap := make(map[string]string)

	// Map to track file items: fileID -> {metadata, content}
	fileData := make(map[string]struct {
		metadata map[string]interface{}
		content  []byte
		filename string
	})

	// Extract all file metadata and content
	for _, f := range r.File {
		if !strings.HasPrefix(f.Name, "model-files/") {
			continue
		}

		parts := strings.Split(f.Name, "/")
		if len(parts) < 2 {
			continue
		}

		fileID := parts[1] // model-files/{file-id}/...

		// Initialize file entry if needed
		if _, exists := fileData[fileID]; !exists {
			fileData[fileID] = struct {
				metadata map[string]interface{}
				content  []byte
				filename string
			}{
				metadata: nil,
				content:  nil,
				filename: "",
			}
		}

		entry := fileData[fileID]

		// Read metadata.json
		if strings.HasSuffix(f.Name, "/metadata.json") {
			rc, err := f.Open()
			if err != nil {
				logrus.Warnf("Failed to open %s: %v", f.Name, err)
				continue
			}

			data, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				logrus.Warnf("Failed to read %s: %v", f.Name, err)
				continue
			}

			var metadata map[string]interface{}
			if err := json.Unmarshal(data, &metadata); err != nil {
				logrus.Warnf("Failed to parse %s: %v", f.Name, err)
				continue
			}

			entry.metadata = metadata
		}

		// Read file content
		if len(parts) >= 3 && parts[2] != "metadata.json" && !f.FileInfo().IsDir() {
			rc, err := f.Open()
			if err != nil {
				logrus.Warnf("Failed to open file %s: %v", f.Name, err)
				continue
			}

			content, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				logrus.Warnf("Failed to read file %s: %v", f.Name, err)
				continue
			}

			entry.content = content
			entry.filename = filepath.Base(f.Name)
		}

		fileData[fileID] = entry
	}

	// Upload each file and build ID mapping
	for oldFileID, entry := range fileData {
		if entry.metadata == nil || entry.content == nil {
			logrus.Warnf("Incomplete file data for ID %s, skipping", oldFileID)
			continue
		}

		filename := entry.filename
		if filename == "" {
			filename = fmt.Sprintf("file_%s", oldFileID)
		}

		logrus.Infof("Restoring file: %s (original ID: %s)", filename, oldFileID)

		// Upload the file
		resp, err := client.UploadFile(filename, entry.content)
		if err != nil {
			logrus.Warnf("Failed to upload file %s: %v", filename, err)
			continue
		}

		fileIDMap[oldFileID] = resp.ID
		logrus.Infof("File uploaded with new ID: %s", resp.ID)
	}

	return fileIDMap, nil
}

// restoreCollectionKnowledgeItems restores collection-type knowledge items from knowledge-bases/ directory
func restoreCollectionKnowledgeItems(r *zip.ReadCloser, client *openwebui.Client, overwrite bool) (map[string]string, error) {
	// Map to track knowledge bases found in ZIP: kbID -> {kb data, files}
	kbData := make(map[string]struct {
		kb    *openwebui.KnowledgeBase
		files map[string][]byte
	})

	// Extract all knowledge bases and their files
	for _, f := range r.File {
		if !strings.HasPrefix(f.Name, "knowledge-bases/") {
			continue
		}

		parts := strings.Split(f.Name, "/")
		if len(parts) < 2 {
			continue
		}

		kbID := parts[1] // knowledge-bases/{kb-id}/...

		// Initialize KB entry if needed
		if _, exists := kbData[kbID]; !exists {
			kbData[kbID] = struct {
				kb    *openwebui.KnowledgeBase
				files map[string][]byte
			}{
				kb:    nil,
				files: make(map[string][]byte),
			}
		}

		entry := kbData[kbID]

		// Read knowledge_base.json
		if strings.HasSuffix(f.Name, "/knowledge_base.json") {
			rc, err := f.Open()
			if err != nil {
				logrus.Warnf("Failed to open %s: %v", f.Name, err)
				continue
			}

			data, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				logrus.Warnf("Failed to read %s: %v", f.Name, err)
				continue
			}

			var kb openwebui.KnowledgeBase
			if err := json.Unmarshal(data, &kb); err != nil {
				logrus.Warnf("Failed to parse %s: %v", f.Name, err)
				continue
			}

			entry.kb = &kb
		}

		// Read document files
		if strings.Contains(f.Name, "/documents/") && !f.FileInfo().IsDir() {
			rc, err := f.Open()
			if err != nil {
				logrus.Warnf("Failed to open file %s: %v", f.Name, err)
				continue
			}

			content, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				logrus.Warnf("Failed to read file %s: %v", f.Name, err)
				continue
			}

			filename := filepath.Base(f.Name)
			entry.files[filename] = content
		}

		kbData[kbID] = entry
	}

	// Restore each knowledge base and build ID mapping
	kbIDMap := make(map[string]string)

	for oldKBID, entry := range kbData {
		if entry.kb == nil {
			logrus.Warnf("No knowledge_base.json found for KB ID %s, skipping", oldKBID)
			continue
		}

		logrus.Infof("Restoring knowledge base: %s (original ID: %s)", entry.kb.Name, oldKBID)

		// Check if knowledge base already exists by name
		existingKB, err := findKnowledgeByName(client, entry.kb.Name)
		if err != nil {
			logrus.Warnf("Failed to check for existing KB %s: %v", entry.kb.Name, err)
			continue
		}

		var newKBID string
		if existingKB == nil {
			// Create new knowledge base
			form := &openwebui.KnowledgeForm{
				Name:          entry.kb.Name,
				Description:   entry.kb.Description,
				AccessControl: entry.kb.AccessControl,
			}

			createResp, err := client.CreateKnowledge(form)
			if err != nil {
				logrus.Warnf("Failed to create KB %s: %v", entry.kb.Name, err)
				continue
			}

			newKBID = createResp.ID
			logrus.Infof("Created KB: %s (ID: %s)", entry.kb.Name, newKBID)

			// Upload files
			if len(entry.files) > 0 {
				fileIDMap, err := uploadFiles(client, entry.files)
				if err != nil {
					logrus.Warnf("Failed to upload files for KB %s: %v", entry.kb.Name, err)
				} else {
					fileIDs := make([]string, 0, len(fileIDMap))
					for _, fileID := range fileIDMap {
						fileIDs = append(fileIDs, fileID)
					}

					if err := linkFilesToKnowledge(client, newKBID, fileIDs); err != nil {
						logrus.Warnf("Failed to link files to KB %s: %v", entry.kb.Name, err)
					}
				}
			}
		} else {
			// Use existing knowledge base
			newKBID = existingKB.ID
			logrus.Infof("KB %s already exists (ID: %s)", entry.kb.Name, newKBID)

			// Update metadata if needed
			if err := updateKnowledgeMetadata(client, existingKB, entry.kb); err != nil {
				logrus.Warnf("Failed to update KB metadata for %s: %v", entry.kb.Name, err)
			}

			// Sync files if any
			if len(entry.files) > 0 {
				existingFiles := getExistingFileMap(existingKB)
				stats, err := syncFiles(client, newKBID, entry.files, existingFiles, overwrite)
				if err != nil {
					logrus.Warnf("Failed to sync files for KB %s: %v", entry.kb.Name, err)
				} else {
					logrus.Infof("File sync for KB %s: %d new, %d overwritten, %d skipped",
						entry.kb.Name, stats.New, stats.Overwritten, stats.Skipped)
				}
			}
		}

		// Map old ID to new ID
		kbIDMap[oldKBID] = newKBID
	}

	logrus.Infof("Restored %d knowledge bases", len(kbIDMap))
	return kbIDMap, nil
}

// updateModelKnowledgeReferences updates the model's knowledge base references with new IDs
func updateModelKnowledgeReferences(model *openwebui.Model, kbIDMap map[string]string) {
	if len(model.Meta.Knowledge) == 0 {
		return
	}

	logrus.Infof("Updating knowledge base references in model")

	// Update each knowledge base ID in the model
	for i := range model.Meta.Knowledge {
		item := model.Meta.Knowledge[i]
		oldID, ok := item["id"].(string)
		if !ok {
			logrus.Warnf("Knowledge item missing or invalid 'id' field, skipping")
			continue
		}

		if newID, found := kbIDMap[oldID]; found {
			logrus.Infof("Updating KB reference: %s -> %s", oldID, newID)
			model.Meta.Knowledge[i]["id"] = newID
		} else {
			logrus.Warnf("No mapping found for KB ID %s, keeping original", oldID)
		}
	}
}

// RestoreAll restores all knowledge bases and models from a directory
// Order: models first (which restore their KBs), then standalone KBs
// This ensures models get their dependencies first
func RestoreAll(client *openwebui.Client, inputDir string, overwrite bool) error {
	logrus.Info("Starting full restore (models + knowledge bases)...")

	// Get all ZIP files in directory
	files, err := filepath.Glob(filepath.Join(inputDir, "*.zip"))
	if err != nil {
		return fmt.Errorf("failed to list ZIP files: %w", err)
	}

	if len(files) == 0 {
		logrus.Info("No ZIP files found in directory")
		return nil
	}

	logrus.Infof("Found %d ZIP file(s)", len(files))

	// Separate model and knowledge base files
	var modelFiles []string
	var kbFiles []string

	for _, file := range files {
		basename := filepath.Base(file)
		if strings.Contains(basename, "_model_") {
			modelFiles = append(modelFiles, file)
		} else if strings.Contains(basename, "_knowledge_base_") {
			kbFiles = append(kbFiles, file)
		}
	}

	logrus.Infof("Found %d model(s) and %d knowledge base(s)", len(modelFiles), len(kbFiles))

	// Step 1: Restore all models (which will restore their embedded KBs)
	if len(modelFiles) > 0 {
		logrus.Info("Step 1/2: Restoring models...")
		for i, modelFile := range modelFiles {
			logrus.Infof("Restoring model %d/%d: %s", i+1, len(modelFiles), filepath.Base(modelFile))
			if err := RestoreModel(client, modelFile, overwrite); err != nil {
				logrus.Warnf("Failed to restore model from %s: %v", filepath.Base(modelFile), err)
				continue
			}
		}
	}

	// Step 2: Restore standalone knowledge bases
	if len(kbFiles) > 0 {
		logrus.Info("Step 2/2: Restoring standalone knowledge bases...")
		for i, kbFile := range kbFiles {
			logrus.Infof("Restoring knowledge base %d/%d: %s", i+1, len(kbFiles), filepath.Base(kbFile))
			if err := RestoreKnowledge(client, kbFile, overwrite); err != nil {
				logrus.Warnf("Failed to restore knowledge base from %s: %v", filepath.Base(kbFile), err)
				continue
			}
		}
	}

	logrus.Info("Full restore completed successfully")
	return nil
}
