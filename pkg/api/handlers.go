package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	"github.com/vosiander/open-webui-backup/pkg/backup"
	"github.com/vosiander/open-webui-backup/pkg/openwebui"
	"github.com/vosiander/open-webui-backup/pkg/restore"
)

// handleGetConfig returns the current configuration
func (s *Server) handleGetConfig(c echo.Context) error {
	// Get default values from environment
	defaultRecipient := os.Getenv("AGE_RECIPIENT")
	defaultIdentity := os.Getenv("AGE_IDENTITY")

	// List available backup files
	backups, err := listBackupFiles(s.config.BackupsDir)
	if err != nil {
		logrus.WithError(err).Warn("Failed to list backup files")
		backups = []string{}
	}

	response := ConfigResponse{
		OpenWebUIURL:         s.config.OpenWebUIURL,
		APIKey:               s.config.OpenWebUIAPIKey,
		ServerPort:           s.config.ServerPort,
		BackupsDir:           s.config.BackupsDir,
		DefaultRecipient:     defaultRecipient,
		DefaultIdentity:      defaultIdentity,
		DefaultAgeIdentity:   os.Getenv("AGE_IDENTITY"),
		DefaultAgeRecipients: os.Getenv("AGE_RECIPIENTS"),
		AvailableBackups:     backups,
	}

	return c.JSON(http.StatusOK, response)
}

// handleUpdateConfig updates the configuration
func (s *Server) handleUpdateConfig(c echo.Context) error {
	var req UpdateConfigRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Update the configuration
	s.config.Update(req.OpenWebUIURL, req.APIKey)

	logrus.Info("Configuration updated")

	// Return the updated configuration
	return s.handleGetConfig(c)
}

// handleStartBackup starts a new backup operation
func (s *Server) handleStartBackup(c echo.Context) error {
	var req BackupRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Validate request
	if req.OutputFilename == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Output filename is required",
		})
	}

	// Create backup directory if it doesn't exist
	if err := os.MkdirAll(s.config.BackupsDir, 0755); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to create backups directory: %v", err),
		})
	}

	// Build full output path
	outputFile := filepath.Join(s.config.BackupsDir, req.OutputFilename)
	if !strings.HasSuffix(outputFile, ".age") {
		outputFile += ".age"
	}

	// Create OpenWebUI client
	client := openwebui.NewClient(s.config.OpenWebUIURL, s.config.OpenWebUIAPIKey)

	// Convert request data types to backup options
	options := &backup.SelectiveBackupOptions{
		Knowledge: req.DataTypes.Knowledge,
		Models:    req.DataTypes.Models,
		Tools:     req.DataTypes.Tools,
		Prompts:   req.DataTypes.Prompts,
		Files:     req.DataTypes.Files,
		Chats:     req.DataTypes.Chats,
		Users:     req.DataTypes.Users,
		Groups:    req.DataTypes.Groups,
		Feedbacks: req.DataTypes.Feedbacks,
	}

	// Start the backup operation asynchronously
	operationID, err := s.opMgr.StartOperation("backup", func(progress ProgressCallback) error {
		// Wrap progress callback to match backup.ProgressCallback signature
		backupProgress := func(percent int, message string) {
			progress(percent, message)
		}

		// Perform the backup
		if err := backup.BackupSelective(client, outputFile, options, backupProgress); err != nil {
			return err
		}

		// Encrypt the backup if recipients are provided
		if len(req.EncryptRecipients) > 0 {
			progress(95, "Encrypting backup...")
			// The backup is already encrypted by BackupSelective
			// (encryption happens inline based on .age extension)
		}

		return nil
	})

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to start backup: %v", err),
		})
	}

	// Store the output file in operation status
	s.opMgr.SetOutputFile(operationID, filepath.Base(outputFile))

	return c.JSON(http.StatusOK, OperationStartResponse{
		OperationID: operationID,
	})
}

// handleStartRestore starts a new restore operation
func (s *Server) handleStartRestore(c echo.Context) error {
	var req RestoreRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Validate request
	if req.InputFilename == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Input filename is required",
		})
	}

	// Build full input path
	inputFile := filepath.Join(s.config.BackupsDir, req.InputFilename)

	// Check if file exists
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Backup file not found",
		})
	}

	// Create OpenWebUI client
	client := openwebui.NewClient(s.config.OpenWebUIURL, s.config.OpenWebUIAPIKey)

	// Convert request data types to restore options
	options := &restore.SelectiveRestoreOptions{
		Knowledge: req.DataTypes.Knowledge,
		Models:    req.DataTypes.Models,
		Tools:     req.DataTypes.Tools,
		Prompts:   req.DataTypes.Prompts,
		Files:     req.DataTypes.Files,
		Chats:     req.DataTypes.Chats,
		Users:     req.DataTypes.Users,
		Groups:    req.DataTypes.Groups,
		Feedbacks: req.DataTypes.Feedbacks,
	}

	// Start the restore operation asynchronously
	operationID, err := s.opMgr.StartOperation("restore", func(progress ProgressCallback) error {
		// Wrap progress callback to match restore.ProgressCallback signature
		restoreProgress := func(percent int, message string) {
			progress(percent, message)
		}

		// Decrypt if needed (handled by restore function based on .age extension)
		if req.DecryptIdentity != "" {
			restoreProgress(5, "Decrypting backup...")
			// Decryption happens inline in restore function
		}

		// Perform the restore
		return restore.RestoreSelective(client, inputFile, options, req.Overwrite, restoreProgress)
	})

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to start restore: %v", err),
		})
	}

	return c.JSON(http.StatusOK, OperationStartResponse{
		OperationID: operationID,
	})
}

// handleGetStatus returns the status of an operation
func (s *Server) handleGetStatus(c echo.Context) error {
	operationID := c.Param("id")

	status, err := s.opMgr.GetStatus(operationID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Operation not found",
		})
	}

	return c.JSON(http.StatusOK, status)
}

// handleListBackups lists all available backup files
func (s *Server) handleListBackups(c echo.Context) error {
	backups, err := listBackupFilesWithMetadata(s.config.BackupsDir)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to list backups: %v", err),
		})
	}

	return c.JSON(http.StatusOK, backups)
}

// handleDownloadBackup serves a backup file for download
func (s *Server) handleDownloadBackup(c echo.Context) error {
	filename := c.Param("filename")

	// Validate filename (prevent path traversal)
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid filename",
		})
	}

	filePath := filepath.Join(s.config.BackupsDir, filename)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "File not found",
		})
	}

	return c.File(filePath)
}

// handleDeleteBackup deletes a backup file
func (s *Server) handleDeleteBackup(c echo.Context) error {
	filename := c.Param("filename")

	// Validate filename (prevent path traversal)
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid filename",
		})
	}

	filePath := filepath.Join(s.config.BackupsDir, filename)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "File not found",
		})
	}

	// Delete the file
	if err := os.Remove(filePath); err != nil {
		logrus.WithError(err).Errorf("Failed to delete backup file: %s", filename)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to delete file: %v", err),
		})
	}

	logrus.Infof("Deleted backup file: %s", filename)

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Backup deleted successfully",
	})
}

// listBackupFiles returns a list of backup files in the specified directory
func listBackupFiles(dir string) ([]string, error) {
	// Ensure directory exists
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var backups []string
	for _, entry := range entries {
		if !entry.IsDir() && (strings.HasSuffix(entry.Name(), ".zip") || strings.HasSuffix(entry.Name(), ".age")) {
			backups = append(backups, entry.Name())
		}
	}

	return backups, nil
}

// BackupFileInfo represents metadata about a backup file
type BackupFileInfo struct {
	Name        string `json:"name"`
	Size        int64  `json:"size"`
	ModTime     string `json:"modTime"`
	DownloadURL string `json:"downloadUrl"`
}

// listBackupFilesWithMetadata returns a list of backup files with their metadata
func listBackupFilesWithMetadata(dir string) ([]BackupFileInfo, error) {
	// Ensure directory exists
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var backups []BackupFileInfo
	for _, entry := range entries {
		if !entry.IsDir() && (strings.HasSuffix(entry.Name(), ".zip") || strings.HasSuffix(entry.Name(), ".age")) {
			info, err := entry.Info()
			if err != nil {
				logrus.WithError(err).Warnf("Failed to get info for file: %s", entry.Name())
				continue
			}

			backups = append(backups, BackupFileInfo{
				Name:        entry.Name(),
				Size:        info.Size(),
				ModTime:     info.ModTime().Format("2006-01-02T15:04:05Z07:00"),
				DownloadURL: fmt.Sprintf("/api/backups/%s", entry.Name()),
			})
		}
	}

	return backups, nil
}
