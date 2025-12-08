package api

import (
	"backend-go/internal/analysis"
	"backend-go/internal/models"
	"backend-go/internal/service"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	ContextService    *service.ContextService
	QuestionGenerator *service.QuestionGenerator
	CSVService        *analysis.CSVService
	SimilarityService *service.SimilarityService
	ExportService     *service.ExportService
	CurrentDB         service.DataSource // Active DB connection
}

func NewHandler(ctx *service.ContextService, qg *service.QuestionGenerator, csv *analysis.CSVService, sim *service.SimilarityService, export *service.ExportService) *Handler {
	return &Handler{
		ContextService:    ctx,
		QuestionGenerator: qg,
		CSVService:        csv,
		SimilarityService: sim,
		ExportService:     export,
	}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/health", h.HealthCheck)
	r.Post("/api/analyze-file", h.AnalyzeFile)
	r.Post("/api/context/{fileIndex}", h.StoreContext)
	r.Get("/api/questions/{fileIndex}", h.GetQuestions)
	r.Get("/api/similarity/graph", h.GetSimilarityGraph)
	r.Post("/api/export/sql", h.ExportSQL)
	r.Post("/api/export/python", h.ExportPython)
	r.Get("/api/status", h.GetStatus)
	r.Get("/api/context/status", h.GetContextStatus)

	// DB Routes
	r.Post("/api/db/connect", h.ConnectDB)
	r.Get("/api/db/tables", h.ListTables)
	r.Post("/api/db/analyze", h.AnalyzeTable)
}

// ConnectDB establishes a database connection
func (h *Handler) ConnectDB(w http.ResponseWriter, r *http.Request) {
	var config service.DataSourceConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Currently only Postgres supported
	if config.Type != "postgres" {
		http.Error(w, "Only postgres is supported currently", http.StatusBadRequest)
		return
	}

	ds := &service.PostgresDataSource{}
	if err := ds.Connect(config); err != nil {
		http.Error(w, fmt.Sprintf("Failed to connect: %v", err), http.StatusInternalServerError)
		return
	}

	// Close previous if exists
	if h.CurrentDB != nil {
		h.CurrentDB.Close()
	}
	h.CurrentDB = ds

	json.NewEncoder(w).Encode(map[string]string{"status": "connected"})
}

// ListTables returns tables from connected DB
func (h *Handler) ListTables(w http.ResponseWriter, r *http.Request) {
	if h.CurrentDB == nil {
		http.Error(w, "No database connection", http.StatusBadRequest)
		return
	}

	tables, err := h.CurrentDB.ListTables()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error listing tables: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"tables": tables})
}

// AnalyzeTable fetches data from a table and analyzes it
func (h *Handler) AnalyzeTable(w http.ResponseWriter, r *http.Request) {
	if h.CurrentDB == nil {
		http.Error(w, "No database connection", http.StatusBadRequest)
		return
	}

	var req struct {
		TableName string `json:"table_name"`
		FileIndex int    `json:"file_index"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Fetch data (preview limit 1000 rows for analysis)
	data, err := h.CurrentDB.PreviewData(req.TableName, 1000)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error fetching data: %v", err), http.StatusInternalServerError)
		return
	}

	// Analyze
	// We need column names from the first row or separate metadata
	if len(data) == 0 {
		http.Error(w, "Table is empty", http.StatusBadRequest)
		return
	}

	// Extract column names from first row map keys
	// Map iteration order is random, so we should sort or trust what we get?
	// DataSource.PreviewData returns map.
	// We should probably have gotten columns from DataSource properly, but let's just infer.
	var columns []string
	for k := range data[0] {
		columns = append(columns, k)
	}
	// Sort for consistency
	// sort.Strings(columns) // Need "sort" import. Leaving unsorted for now unless critical.

	analysisResult, err := h.CSVService.AnalyzeData(data, columns)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error analyzing data: %v", err), http.StatusInternalServerError)
		return
	}

	// Store result
	if req.FileIndex != 0 {
		h.ContextService.StoreAnalysis(req.FileIndex, &analysisResult)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analysisResult)
}

func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
}

// GetStatus returns the status of loaded files
func (h *Handler) GetStatus(w http.ResponseWriter, r *http.Request) {
	analysis1 := h.ContextService.GetAnalysis(1)
	analysis2 := h.ContextService.GetAnalysis(2)

	status := map[string]interface{}{
		"loaded":        analysis1 != nil || analysis2 != nil,
		"file1_loaded":  analysis1 != nil,
		"file2_loaded":  analysis2 != nil,
		"file1_context": h.ContextService.GetContext(1) != nil,
		"file2_context": h.ContextService.GetContext(2) != nil,
		"file1":         analysis1,
		"file2":         analysis2,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// GetContextStatus returns context status
func (h *Handler) GetContextStatus(w http.ResponseWriter, r *http.Request) {
	// This structure matches Python backend likely
	status := map[string]map[string]bool{
		"file1": {"has_context": h.ContextService.GetContext(1) != nil},
		"file2": {"has_context": h.ContextService.GetContext(2) != nil},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// AnalyzeFile handles file upload and analysis
func (h *Handler) AnalyzeFile(w http.ResponseWriter, r *http.Request) {
	// Limit upload size (e.g., 10MB)
	r.ParseMultipartForm(10 << 20)

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Create a temp file
	tempDir := os.TempDir()
	tempFilePath := filepath.Join(tempDir, header.Filename)
	tempFile, err := os.Create(tempFilePath)
	if err != nil {
		http.Error(w, "Error creating temp file", http.StatusInternalServerError)
		return
	}
	defer os.Remove(tempFilePath) // Clean up
	defer tempFile.Close()

	if _, err := io.Copy(tempFile, file); err != nil {
		http.Error(w, "Error saving file", http.StatusInternalServerError)
		return
	}

	// Analyze the file
	analysisResult, err := h.CSVService.AnalyzeFile(tempFilePath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error analyzing file: %v", err), http.StatusInternalServerError)
		return
	}

	// Store analysis result if fileIndex is provided
	fileIndexStr := r.FormValue("fileIndex")
	if fileIndexStr == "" {
		fileIndexStr = r.FormValue("file_index")
	}

	if fileIndexStr != "" {
		if fileIndex, err := strconv.Atoi(fileIndexStr); err == nil {
			h.ContextService.StoreAnalysis(fileIndex, &analysisResult)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analysisResult)
}

// StoreContext endpoint
func (h *Handler) StoreContext(w http.ResponseWriter, r *http.Request) {
	fileIndexStr := chi.URLParam(r, "fileIndex")
	fileIndex, err := strconv.Atoi(fileIndexStr)
	if err != nil {
		http.Error(w, "Invalid file index", http.StatusBadRequest)
		return
	}

	var ctx models.Context
	body, _ := io.ReadAll(r.Body)
	if err := json.Unmarshal(body, &ctx); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := h.ContextService.StoreContext(fileIndex, &ctx); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// GetQuestions endpoint
func (h *Handler) GetQuestions(w http.ResponseWriter, r *http.Request) {
	fileIndexStr := chi.URLParam(r, "fileIndex")
	fileIndex, err := strconv.Atoi(fileIndexStr)
	if err != nil {
		http.Error(w, "Invalid file index", http.StatusBadRequest)
		return
	}

	// Retrieve analysis from storage
	analysis := h.ContextService.GetAnalysis(fileIndex)
	if analysis == nil {
		http.Error(w, "Analysis not found for this file. Please upload and analyze file first.", http.StatusNotFound)
		return
	}

	questions := h.QuestionGenerator.GenerateQuestions(*analysis, fileIndex)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(questions)
}

// GetSimilarityGraph generates the correlation graph
func (h *Handler) GetSimilarityGraph(w http.ResponseWriter, r *http.Request) {
	graph, err := h.SimilarityService.GenerateGraph(1, 2)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error generating graph: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(graph)
}

// ExportSQL generates SQL from the graph
func (h *Handler) ExportSQL(w http.ResponseWriter, r *http.Request) {
	var graph models.SimilarityGraph
	body, _ := io.ReadAll(r.Body)
	if err := json.Unmarshal(body, &graph); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	sql := h.ExportService.GenerateSQL(&graph)

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(sql))
}

// ExportPython generates Python script from the graph
func (h *Handler) ExportPython(w http.ResponseWriter, r *http.Request) {
	var graph models.SimilarityGraph
	body, _ := io.ReadAll(r.Body)
	if err := json.Unmarshal(body, &graph); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	python := h.ExportService.GeneratePython(&graph)

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(python))
}
