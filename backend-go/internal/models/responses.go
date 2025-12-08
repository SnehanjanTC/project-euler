package models

// UploadResponse is returned after successful file upload
type UploadResponse struct {
	Message     string   `json:"message"`
	Rows        int      `json:"rows"`
	Columns     int      `json:"columns"`
	ColumnNames []string `json:"column_names"`
}

// FileStatus represents status of a loaded file
type FileStatus struct {
	Loaded   bool   `json:"loaded"`
	Rows     int    `json:"rows"`
	Columns  int    `json:"columns"`
	Filename string `json:"filename,omitempty"`
}

// StatusResponse is returned by /status endpoint
type StatusResponse struct {
	File1Loaded bool       `json:"file1_loaded"`
	File2Loaded bool       `json:"file2_loaded"`
	File1       FileStatus `json:"file1"`
	File2       FileStatus `json:"file2"`
}

// KPI represents a key performance indicator
type KPI struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
	Avg   float64 `json:"avg"`
	Type  string  `json:"type"`
}

// ColumnSimilarity represents similarity between two columns
type ColumnSimilarity struct {
	File1Column    string  `json:"file1_column"`
	File2Column    string  `json:"file2_column"`
	Confidence     float64 `json:"confidence"`
	NameSimilarity float64 `json:"name_similarity"`
	DataSimilarity float64 `json:"data_similarity"`
}

// SimilarityResponse is returned by /column-similarity endpoint
type SimilarityResponse struct {
	Similarities []ColumnSimilarity     `json:"similarities"`
	ContextUsed  map[string]bool        `json:"context_used"`
	Correlations []CorrelationResult    `json:"correlations,omitempty"`
}

// CorrelationResult represents correlation between column pair
type CorrelationResult struct {
	Column1        string  `json:"column1"`
	Column2        string  `json:"column2"`
	Correlation    float64 `json:"correlation"`
	Interpretation string  `json:"interpretation"`
}

// ContextStatusResponse for /context/status
type ContextStatusResponse struct {
	File1 ContextStatusItem `json:"file1"`
	File2 ContextStatusItem `json:"file2"`
}

// ContextStatusItem represents context status for a file
type ContextStatusItem struct {
	HasContext     bool                   `json:"has_context"`
	ContextSummary map[string]interface{} `json:"context_summary,omitempty"`
}

// OllamaConfig for /config/ollama endpoint
type OllamaConfig struct {
	BaseURL string `json:"baseUrl"`
	Model   string `json:"model"`
}

// QuestionsResponse for /context/questions
type QuestionsResponse struct {
	Success        bool                   `json:"success"`
	Questions      map[string][]Question  `json:"questions"`
	TotalQuestions int                    `json:"total_questions"`
}

// ContextSubmitRequest for /context/submit
type ContextSubmitRequest struct {
	FileIndex   int                    `json:"file_index"`
	ContextData map[string]interface{} `json:"context_data"`
}

// FilterCondition for /filter endpoint
type FilterCondition struct {
	Column   string `json:"column"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
}

// FilterRequest for /filter endpoint
type FilterRequest struct {
	Conditions []FilterCondition `json:"conditions"`
}

// FilterResponse for /filter endpoint
type FilterResponse struct {
	Rows int                      `json:"rows"`
	Data []map[string]interface{} `json:"data"`
}
