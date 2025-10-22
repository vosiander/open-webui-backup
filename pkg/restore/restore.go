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
