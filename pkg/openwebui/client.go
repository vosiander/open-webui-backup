package openwebui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
)

// Client represents an HTTP client for the Open WebUI API
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewClient creates a new API client instance
func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     apiKey,
		httpClient: &http.Client{},
	}
}

// ListKnowledge fetches all knowledge bases from the API
func (c *Client) ListKnowledge() ([]KnowledgeBase, error) {
	resp, err := c.doRequest("GET", "/api/v1/knowledge/list", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
		}
	}

	var knowledgeBases []KnowledgeBase
	if err := json.NewDecoder(resp.Body).Decode(&knowledgeBases); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return knowledgeBases, nil
}

// GetFile fetches a single file by ID
func (c *Client) GetFile(fileID string) (*FileData, error) {
	path := fmt.Sprintf("/api/v1/files/%s", fileID)
	resp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
		}
	}

	var fileData FileData
	if err := json.NewDecoder(resp.Body).Decode(&fileData); err != nil {
		return nil, fmt.Errorf("failed to decode file response: %w", err)
	}

	return &fileData, nil
}

// CreateKnowledge creates a new knowledge base
func (c *Client) CreateKnowledge(form *KnowledgeForm) (*KnowledgeCreateResponse, error) {
	jsonData, err := json.Marshal(form)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal knowledge form: %w", err)
	}

	resp, err := c.doRequest("POST", "/api/v1/knowledge/create", bytes.NewReader(jsonData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
		}
	}

	var result KnowledgeCreateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// UploadFile uploads a file via multipart/form-data
func (c *Client) UploadFile(filename string, content []byte) (*FileUploadResponse, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Create form file field
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := part.Write(content); err != nil {
		return nil, fmt.Errorf("failed to write file content: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Make the request with process=true and process_in_background=false
	// This ensures the file is fully processed before we try to link it
	url := c.baseURL + "/api/v1/files/?process=true&process_in_background=false"
	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
		}
	}

	var result FileUploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// GetKnowledgeByID fetches a specific knowledge base with its files
func (c *Client) GetKnowledgeByID(id string) (*KnowledgeBase, error) {
	path := fmt.Sprintf("/api/v1/knowledge/%s", id)
	resp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
		}
	}

	var kb KnowledgeBase
	if err := json.NewDecoder(resp.Body).Decode(&kb); err != nil {
		return nil, fmt.Errorf("failed to decode knowledge base response: %w", err)
	}

	return &kb, nil
}

// UpdateKnowledge updates an existing knowledge base
func (c *Client) UpdateKnowledge(id string, form *KnowledgeForm) error {
	jsonData, err := json.Marshal(form)
	if err != nil {
		return fmt.Errorf("failed to marshal knowledge form: %w", err)
	}

	path := fmt.Sprintf("/api/v1/knowledge/%s/update", id)
	resp, err := c.doRequest("POST", path, bytes.NewReader(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
		}
	}

	return nil
}

// AddFileToKnowledge associates an uploaded file with a knowledge base
func (c *Client) AddFileToKnowledge(knowledgeID, fileID string) error {
	reqBody := AddFileRequest{FileID: fileID}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	path := fmt.Sprintf("/api/v1/knowledge/%s/file/add", knowledgeID)
	resp, err := c.doRequest("POST", path, bytes.NewReader(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
		}
	}

	return nil
}

// RemoveFileFromKnowledge removes a file from a knowledge base
func (c *Client) RemoveFileFromKnowledge(knowledgeID, fileID string) error {
	reqBody := AddFileRequest{FileID: fileID}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	path := fmt.Sprintf("/api/v1/knowledge/%s/file/remove", knowledgeID)
	resp, err := c.doRequest("POST", path, bytes.NewReader(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
		}
	}

	return nil
}

// doRequest makes an authenticated HTTP request to the API
func (c *Client) doRequest(method, path string, body io.Reader) (*http.Response, error) {
	url := c.baseURL + path

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication header
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	return resp, nil
}
