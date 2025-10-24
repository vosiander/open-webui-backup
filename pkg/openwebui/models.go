package openwebui

// KnowledgeBase represents a knowledge base from the API
type KnowledgeBase struct {
	ID            string                 `json:"id"`
	UserID        string                 `json:"user_id"`
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	Data          *KnowledgeData         `json:"data"`
	Meta          map[string]interface{} `json:"meta"`
	AccessControl map[string]interface{} `json:"access_control"`
	CreatedAt     int64                  `json:"created_at"`
	UpdatedAt     int64                  `json:"updated_at"`
	Files         []FileMetadata         `json:"files"`
}

// KnowledgeData contains file IDs associated with the knowledge base
type KnowledgeData struct {
	FileIDs []string `json:"file_ids"`
}

// FileMetadata represents file metadata from the API
type FileMetadata struct {
	ID        string   `json:"id"`
	Meta      FileMeta `json:"meta"`
	CreatedAt int64    `json:"created_at"`
	UpdatedAt int64    `json:"updated_at"`
}

// FileMeta contains file-specific metadata
type FileMeta struct {
	Name           string                 `json:"name"`
	ContentType    string                 `json:"content_type"`
	Size           int64                  `json:"size"`
	Data           map[string]interface{} `json:"data"`
	CollectionName string                 `json:"collection_name"`
}

// FileData represents a complete file with content from the API
type FileData struct {
	ID            string                 `json:"id"`
	UserID        string                 `json:"user_id"`
	Hash          string                 `json:"hash"`
	Filename      string                 `json:"filename"`
	Path          string                 `json:"path"`
	Data          *FileContent           `json:"data"`
	Meta          FileMeta               `json:"meta"`
	AccessControl map[string]interface{} `json:"access_control"`
	CreatedAt     int64                  `json:"created_at"`
	UpdatedAt     int64                  `json:"updated_at"`
}

// FileContent contains the actual file content
type FileContent struct {
	Status  string `json:"status"`
	Content string `json:"content"`
}

// KnowledgeForm represents the request body for creating/updating knowledge
type KnowledgeForm struct {
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	Data          *KnowledgeData         `json:"data,omitempty"`
	AccessControl map[string]interface{} `json:"access_control,omitempty"`
}

// KnowledgeCreateResponse represents the response from creating a knowledge base
type KnowledgeCreateResponse struct {
	ID            string                 `json:"id"`
	UserID        string                 `json:"user_id"`
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	Data          *KnowledgeData         `json:"data"`
	Meta          map[string]interface{} `json:"meta"`
	AccessControl map[string]interface{} `json:"access_control"`
	CreatedAt     int64                  `json:"created_at"`
	UpdatedAt     int64                  `json:"updated_at"`
}

// FileUploadResponse represents the response from uploading a file
type FileUploadResponse struct {
	ID        string   `json:"id"`
	UserID    string   `json:"user_id"`
	Filename  string   `json:"filename"`
	Meta      FileMeta `json:"meta"`
	CreatedAt int64    `json:"created_at"`
	UpdatedAt int64    `json:"updated_at"`
}

// AddFileRequest represents the request to add a file to knowledge base
type AddFileRequest struct {
	FileID string `json:"file_id"`
}

// Model represents a model from the Open WebUI API
type Model struct {
	ID            string                 `json:"id"`
	UserID        string                 `json:"user_id"`
	BaseModelID   *string                `json:"base_model_id"`
	Name          string                 `json:"name"`
	Params        map[string]interface{} `json:"params"`
	Meta          ModelMeta              `json:"meta"`
	AccessControl map[string]interface{} `json:"access_control"`
	IsActive      bool                   `json:"is_active"`
	UpdatedAt     int64                  `json:"updated_at"`
	CreatedAt     int64                  `json:"created_at"`
}

// ModelMeta contains model metadata including knowledge base references
type ModelMeta struct {
	ProfileImageURL string                   `json:"profile_image_url,omitempty"`
	Description     *string                  `json:"description,omitempty"`
	Capabilities    map[string]interface{}   `json:"capabilities,omitempty"`
	Knowledge       []map[string]interface{} `json:"knowledge,omitempty"` // Can be files or collections
	Tags            []string                 `json:"tags,omitempty"`
}

// KnowledgeItem represents a single knowledge item (file or collection)
type KnowledgeItem struct {
	Type string                 `json:"type"` // "file" or "collection"
	Data map[string]interface{} `json:"-"`    // Raw data for flexibility
}

// ModelForm for creating/updating models
type ModelForm struct {
	ID            string                 `json:"id"`
	BaseModelID   *string                `json:"base_model_id,omitempty"`
	Name          string                 `json:"name"`
	Params        map[string]interface{} `json:"params"`
	Meta          ModelMeta              `json:"meta"`
	AccessControl map[string]interface{} `json:"access_control,omitempty"`
	IsActive      bool                   `json:"is_active"`
}
