package api

import (
	"time"
)

// Config represents the application configuration
type Config struct {
	OpenWebUIBaseURL string `json:"openWebUIBaseURL"`
	AgeIdentity      string `json:"ageIdentity"`
	AgeRecipients    string `json:"ageRecipients"`
	Passphrase       string `json:"passphrase"`
}

// DataTypeSelection represents the selection of data types for backup/restore
type DataTypeSelection struct {
	Knowledge bool `json:"knowledge"`
	Models    bool `json:"models"`
	Tools     bool `json:"tools"`
	Prompts   bool `json:"prompts"`
	Files     bool `json:"files"`
	Chats     bool `json:"chats"`
	Users     bool `json:"users"`
	Groups    bool `json:"groups"`
	Feedbacks bool `json:"feedbacks"`
}

// BackupRequest represents a backup operation request
type BackupRequest struct {
	OutputFilename    string            `json:"outputFilename"`
	EncryptRecipients []string          `json:"encryptRecipients"`
	DataTypes         DataTypeSelection `json:"dataTypes"`
}

// RestoreRequest represents a restore operation request
type RestoreRequest struct {
	InputFilename   string            `json:"inputFilename"`
	DecryptIdentity string            `json:"decryptIdentity"`
	DataTypes       DataTypeSelection `json:"dataTypes"`
	Overwrite       bool              `json:"overwrite"`
}

// OperationStartResponse represents the response when starting an operation
type OperationStartResponse struct {
	OperationID string `json:"operationId"`
}

// OperationStatus represents the status of an operation
type OperationStatus struct {
	ID         string     `json:"id"`
	Type       string     `json:"type"`
	Status     string     `json:"status"`
	Progress   int        `json:"progress"`
	Message    string     `json:"message"`
	StartTime  time.Time  `json:"startTime"`
	EndTime    *time.Time `json:"endTime,omitempty"`
	Error      string     `json:"error,omitempty"`
	OutputFile string     `json:"outputFile,omitempty"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// ConfigResponse represents the configuration response
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

// UpdateConfigRequest represents a configuration update request
type UpdateConfigRequest struct {
	OpenWebUIURL string `json:"openWebUIURL,omitempty"`
	APIKey       string `json:"apiKey,omitempty"`
}

// WebSocketMessage represents a WebSocket message
type WebSocketMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// GenerateIdentityResponse contains a newly generated age identity pair
type GenerateIdentityResponse struct {
	Identity  string `json:"identity"`  // Private key (AGE-SECRET-KEY-...)
	Recipient string `json:"recipient"` // Public key (age1...)
}
