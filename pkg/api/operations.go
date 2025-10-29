package api

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ProgressCallback is a function that receives progress updates
type ProgressCallback func(percent int, message string)

// OperationManager manages concurrent backup and restore operations
type OperationManager struct {
	operations map[string]*OperationStatus
	mu         sync.RWMutex
	hub        *Hub
}

// NewOperationManager creates a new operation manager
func NewOperationManager(hub *Hub) *OperationManager {
	return &OperationManager{
		operations: make(map[string]*OperationStatus),
		hub:        hub,
	}
}

// StartOperation starts a new async operation and returns its ID
func (om *OperationManager) StartOperation(opType string, fn func(progress ProgressCallback) error) (string, error) {
	id := uuid.New().String()

	status := &OperationStatus{
		ID:        id,
		Type:      opType,
		Status:    "running",
		Progress:  0,
		Message:   "Starting operation...",
		StartTime: time.Now(),
	}

	om.mu.Lock()
	om.operations[id] = status
	om.mu.Unlock()

	// Broadcast initial status
	om.broadcastStatus(status)

	// Run operation in goroutine
	go func() {
		// Create progress callback
		progressCallback := func(percent int, message string) {
			om.updateProgress(id, percent, message)
		}

		// Execute the operation
		err := fn(progressCallback)

		// Update final status
		om.mu.Lock()
		if err != nil {
			status.Status = "failed"
			status.Error = err.Error()
			status.Message = fmt.Sprintf("Operation failed: %v", err)
		} else {
			status.Status = "completed"
			status.Progress = 100
			status.Message = "Operation completed successfully"
		}
		endTime := time.Now()
		status.EndTime = &endTime
		om.mu.Unlock()

		// Broadcast final status
		om.broadcastStatus(status)
	}()

	return id, nil
}

// updateProgress updates the progress of an operation
func (om *OperationManager) updateProgress(id string, percent int, message string) {
	om.mu.Lock()
	if status, exists := om.operations[id]; exists {
		status.Progress = percent
		status.Message = message
		om.broadcastStatus(status)
	}
	om.mu.Unlock()
}

// broadcastStatus sends a status update via WebSocket
func (om *OperationManager) broadcastStatus(status *OperationStatus) {
	if om.hub != nil {
		om.hub.Broadcast(WebSocketMessage{
			Type:    "status",
			Payload: status,
		})
	}
}

// GetStatus retrieves the status of an operation
func (om *OperationManager) GetStatus(id string) (*OperationStatus, error) {
	om.mu.RLock()
	defer om.mu.RUnlock()

	status, exists := om.operations[id]
	if !exists {
		return nil, fmt.Errorf("operation not found: %s", id)
	}

	return status, nil
}

// GetAll returns all operations
func (om *OperationManager) GetAll() []OperationStatus {
	om.mu.RLock()
	defer om.mu.RUnlock()

	result := make([]OperationStatus, 0, len(om.operations))
	for _, status := range om.operations {
		result = append(result, *status)
	}

	return result
}

// SetOutputFile sets the output file for an operation
func (om *OperationManager) SetOutputFile(id string, outputFile string) {
	om.mu.Lock()
	defer om.mu.Unlock()

	if status, exists := om.operations[id]; exists {
		status.OutputFile = outputFile
	}
}
