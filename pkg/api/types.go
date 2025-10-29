package api

import "time"

// BackupRequest represents a request to create a backup
type BackupRequest struct {
	OutputFilename    string            `json:"outputFilename"`
	EncryptRecipients []string          `json:"encryptRecipients"`
	DataTypes         DataTypeSelection `json:"dataTypes"`
}

// RestoreRequest represents a request to restore from a backup
type RestoreRequest struct {
	InputFilename   string            `json:"inputFilename"`
	DecryptIdentity string            `json:"decryptIdentity"`
	DataTypes       DataTypeSelection `json:"dataTypes"`
	Overwrite       bool              `json:"overwrite"`
}

// DataTypeSelection specifies which data types to include in backup/restore
type DataTypeSelection struct {
	Prompts   bool `json:"prompts"`
	Tools     bool `json:"tools"`
	Knowledge bool `json:"knowledge"`
	Models    bool `json:"models"`
	Files     bool `json:"files"`
	Chats     bool `json:"chats"`
	Users     bool `json:"users"`
	Groups    bool `json:"groups"`
	Feedbacks bool `json:"feedbacks"`
}

// OperationStatus represents the current status of a backup or restore operation
type OperationStatus struct {
	ID         string     `json:"id"`
	Type       string     `json:"type"`     // "backup" or "restore"
	Status     string     `json:"status"`   // "running", "completed", "failed"
	Progress   int        `json:"progress"` // 0-100
	Message    string     `json:"message"`
	StartTime  time.Time  `json:"startTime"`
	EndTime    *time.Time `json:"endTime,omitempty"`
	Error      string     `json:"error,omitempty"`
	OutputFile string     `json:"outputFile,omitempty"`
}

// ConfigResponse represents the application configuration for the frontend
type ConfigResponse struct {
	OpenWebUIURL         string   `json:"openWebUIURL"`
	APIKey               string   `json:"apiKey,omitempty"`
	ServerPort           int      `json:"serverPort"`
	BackupsDir           string   `json:"backupsDir"`
	DefaultRecipient     string   `json:"defaultRecipient"`
	DefaultIdentity      string   `json:"defaultIdentity"`
	DefaultAgeIdentity   string   `json:"defaultAgeIdentity,omitempty"`
	DefaultAgeRecipients string   `json:"defaultAgeRecipients,omitempty"`
	AvailableBackups     []string `json:"availableBackups"`
}

// UpdateConfigRequest represents a request to update the configuration
type UpdateConfigRequest struct {
	OpenWebUIURL *string `json:"openWebUIURL,omitempty"`
	APIKey       *string `json:"apiKey,omitempty"`
}

// WebSocketMessage represents a message sent over the WebSocket connection
type WebSocketMessage struct {
	Type    string      `json:"type"` // "status", "progress", "log"
	Payload interface{} `json:"payload"`
}

// OperationStartResponse represents the response when starting an operation
type OperationStartResponse struct {
	OperationID string `json:"operationId"`
}
