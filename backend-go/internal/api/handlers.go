package api

import (
	"backend-go/internal/analysis"
	"backend-go/internal/llm"
	"backend-go/internal/models"
	"backend-go/internal/service"
	"backend-go/internal/state"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

const (
	UploadDir   = "./uploads"
	MaxFileSize = 100 * 1024 * 1024 // 100MB
)

type Handler struct {
	ContextService            *service.ContextService
	QuestionGenerator         *service.QuestionGenerator
	CSVService                *analysis.CSVService
	SimilarityService         *service.SimilarityService
	ExportService             *service.ExportService
	EnhancedSimilarityService *service.EnhancedSimilarityService
	AISemanticMatcher         *service.AISemanticMatcher
	LLMService                *llm.Service
	CurrentDB                 service.DataSource // Active DB connection
}

func NewHandler(ctx *service.ContextService, qg *service.QuestionGenerator, csv *analysis.CSVService, sim *service.SimilarityService, export *service.ExportService, llmSvc *llm.Service) *Handler {
	return &Handler{
		ContextService:            ctx,
		QuestionGenerator:         qg,
		CSVService:                csv,
		SimilarityService:         sim,
		ExportService:             export,
		EnhancedSimilarityService: service.NewEnhancedSimilarityService(ctx),
		AISemanticMatcher:         service.NewAISemanticMatcher(llmSvc, ctx),
		LLMService:                llmSvc,
	}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	// API V2 Routes (My Migration)
	r.Get("/health", h.HealthCheck)
	r.Post("/api/analyze-file", h.AnalyzeFile)
	r.Post("/api/context/{fileIndex}", h.StoreContext)
	r.Get("/api/questions/{fileIndex}", h.GetQuestions)
	r.Get("/api/similarity/graph", h.GetSimilarityGraph)
	r.Post("/api/export/sql", h.ExportSQL)
	r.Post("/api/export/python", h.ExportPython)
	r.Get("/api/status", h.GetAnalysisStatus)
	r.Get("/api/context/status", h.GetAnalysisContextStatus)

	// DB Routes
	r.Post("/api/db/connect", h.ConnectDB)
	r.Get("/api/db/tables", h.ListTables)
	r.Post("/api/db/analyze", h.AnalyzeTable)

	// Upstream/Legacy Routes
	r.Post("/upload", h.Upload)
	r.Get("/status", h.GetStatus)
	r.Get("/preview", h.GetPreview)
	r.Get("/column-types", h.GetColumnTypes)
	r.Get("/kpis", h.GetKPIs)

	r.Get("/column-similarity", h.GetColumnSimilarity)
	r.Get("/correlation", h.GetCorrelation)
	r.Post("/filter", h.FilterData)
	r.Post("/query", h.Query)

	r.Post("/context/questions", h.GenerateContextQuestions)
	r.Post("/context/submit", h.SubmitContext)
	r.Get("/context/{fileIndex}", h.GetContext)
	r.Delete("/context/{fileIndex}", h.DeleteContext)
	r.Get("/context/status", h.GetContextStatus)

	r.Get("/config/ollama", h.GetOllamaConfig)
	r.Post("/config/ollama", h.SaveOllamaConfig)

	r.Post("/feedback/match", h.SubmitMatchFeedback)
	r.Get("/feedback/stats", h.GetFeedbackStats)
}

// ============================================================================
// Health
// ============================================================================

func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
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
	if len(data) == 0 {
		http.Error(w, "Table is empty", http.StatusBadRequest)
		return
	}

	var columns []string
	for k := range data[0] {
		columns = append(columns, k)
	}

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

// GetAnalysisStatus returns the status of loaded files (My V2 impl)
func (h *Handler) GetAnalysisStatus(w http.ResponseWriter, r *http.Request) {
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

// GetAnalysisContextStatus returns context status (My V2 impl)
func (h *Handler) GetAnalysisContextStatus(w http.ResponseWriter, r *http.Request) {
	// This structure matches Python backend likely
	status := map[string]map[string]bool{
		"file1": {"has_context": h.ContextService.GetContext(1) != nil},
		"file2": {"has_context": h.ContextService.GetContext(2) != nil},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// AnalyzeFile handles file upload and analysis (My V2 impl)
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

func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form (max 100MB)
	if err := r.ParseMultipartForm(MaxFileSize); err != nil {
		http.Error(w, "File too large", http.StatusBadRequest)
		return
	}

	// Get file_index parameter
	fileIndexStr := r.FormValue("file_index")
	if fileIndexStr == "" {
		fileIndexStr = "1"
	}
	fileIndex, err := strconv.Atoi(fileIndexStr)
	if err != nil || (fileIndex != 1 && fileIndex != 2) {
		http.Error(w, "file_index must be 1 or 2", http.StatusBadRequest)
		return
	}

	// Get file from form
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "No file uploaded", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Validate file extension
	if !strings.HasSuffix(strings.ToLower(header.Filename), ".csv") {
		http.Error(w, "Only CSV files are allowed", http.StatusBadRequest)
		return
	}

	// Create upload directory
	os.MkdirAll(UploadDir, 0755)

	// Save file
	filename := fmt.Sprintf("file%d_%s", fileIndex, filepath.Base(header.Filename))
	filePath := filepath.Join(UploadDir, filename)

	dst, err := os.Create(filePath)
	if err != nil {
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}

	// Parse CSV
	df, err := parseCSVFile(filePath)
	if err != nil {
		os.Remove(filePath)
		http.Error(w, fmt.Sprintf("Failed to parse CSV: %v", err), http.StatusBadRequest)
		return
	}
	df.FileName = header.Filename
	df.FilePath = filePath

	// Store in state
	state.State.SetDataFrame(fileIndex, df)

	// Return response
	resp := models.UploadResponse{
		Message:     fmt.Sprintf("File '%s' uploaded successfully", header.Filename),
		Rows:        len(df.Rows),
		Columns:     len(df.Headers),
		ColumnNames: df.Headers,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func parseCSVFile(filePath string) (*state.DataFrame, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1 // Allow variable fields
	reader.LazyQuotes = true    // Allow bare quotes in non-quoted fields
	reader.TrimLeadingSpace = true

	// Try to read headers
	headers, err := reader.Read()
	if err != nil {
		// Try with semicolon separator
		file.Seek(0, 0)
		reader = csv.NewReader(file)
		reader.Comma = ';'
		reader.FieldsPerRecord = -1
		reader.LazyQuotes = true
		reader.TrimLeadingSpace = true
		headers, err = reader.Read()
		if err != nil {
			return nil, fmt.Errorf("failed to read headers: %v", err)
		}
	}

	// Clean headers
	for i, h := range headers {
		headers[i] = strings.TrimSpace(h)
	}

	// Read all rows
	rows := [][]string{}
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			// Try to continue on malformed rows
			continue
		}
		rows = append(rows, record)
	}

	return &state.DataFrame{
		Headers:  headers,
		Rows:     rows,
		FilePath: filePath,
	}, nil
}

// ============================================================================
// Status
// ============================================================================

func (h *Handler) GetStatus(w http.ResponseWriter, r *http.Request) {
	df1 := state.State.GetDataFrame(1)
	df2 := state.State.GetDataFrame(2)

	resp := models.StatusResponse{
		File1Loaded: df1 != nil,
		File2Loaded: df2 != nil,
		File1: models.FileStatus{
			Loaded: df1 != nil,
		},
		File2: models.FileStatus{
			Loaded: df2 != nil,
		},
	}

	if df1 != nil {
		resp.File1.Rows = len(df1.Rows)
		resp.File1.Columns = len(df1.Headers)
		resp.File1.Filename = df1.FileName
	}
	if df2 != nil {
		resp.File2.Rows = len(df2.Rows)
		resp.File2.Columns = len(df2.Headers)
		resp.File2.Filename = df2.FileName
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// ============================================================================
// Preview
// ============================================================================

func (h *Handler) GetPreview(w http.ResponseWriter, r *http.Request) {
	fileIndex := getIntParam(r, "file_index", 1)
	rows := getIntParam(r, "rows", 10)

	df := state.State.GetDataFrame(fileIndex)
	if df == nil {
		http.Error(w, fmt.Sprintf("File %d not loaded", fileIndex), http.StatusBadRequest)
		return
	}

	// Convert rows to []map[string]interface{}
	limit := rows
	if limit > len(df.Rows) {
		limit = len(df.Rows)
	}

	data := make([]map[string]interface{}, limit)
	for i := 0; i < limit; i++ {
		row := make(map[string]interface{})
		for j, header := range df.Headers {
			if j < len(df.Rows[i]) {
				row[header] = df.Rows[i][j]
			} else {
				row[header] = ""
			}
		}
		data[i] = row
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// ============================================================================
// Column Types
// ============================================================================

func (h *Handler) GetColumnTypes(w http.ResponseWriter, r *http.Request) {
	fileIndex := getIntParam(r, "file_index", 1)

	df := state.State.GetDataFrame(fileIndex)
	if df == nil {
		http.Error(w, fmt.Sprintf("File %d not loaded", fileIndex), http.StatusBadRequest)
		return
	}

	types := make(map[string]string)
	numericCols := df.GetNumericColumnIndices()

	for i, header := range df.Headers {
		if numericCols[i] {
			types[header] = "numeric"
		} else if isDateColumn(df, i) {
			types[header] = "datetime"
		} else {
			types[header] = "categorical"
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(types)
}

func isDateColumn(df *state.DataFrame, colIdx int) bool {
	dateFormats := []string{
		time.RFC3339, "2006-01-02", "02/01/2006", "01/02/2006",
		"2006/01/02", "Jan 2, 2006", "January 2, 2006",
	}

	checkRows := 5
	if len(df.Rows) < checkRows {
		checkRows = len(df.Rows)
	}

	for i := 0; i < checkRows; i++ {
		if colIdx >= len(df.Rows[i]) {
			continue
		}
		val := df.Rows[i][colIdx]
		if val == "" {
			continue
		}
		parsed := false
		for _, format := range dateFormats {
			if _, err := time.Parse(format, val); err == nil {
				parsed = true
				break
			}
		}
		if parsed {
			return true
		}
	}
	return false
}

// ============================================================================
// KPIs
// ============================================================================

func (h *Handler) GetKPIs(w http.ResponseWriter, r *http.Request) {
	fileIndex := getIntParam(r, "file_index", 1)

	df := state.State.GetDataFrame(fileIndex)
	if df == nil {
		http.Error(w, fmt.Sprintf("File %d not loaded", fileIndex), http.StatusBadRequest)
		return
	}

	kpis := []models.KPI{}
	numericCols := df.GetNumericColumnIndices()

	for colIdx, isNumeric := range numericCols {
		if !isNumeric || colIdx >= len(df.Headers) {
			continue
		}

		colName := df.Headers[colIdx]
		values := []float64{}

		for _, row := range df.Rows {
			if colIdx >= len(row) {
				continue
			}
			if val, err := strconv.ParseFloat(row[colIdx], 64); err == nil {
				values = append(values, val)
			}
		}

		if len(values) == 0 {
			continue
		}

		sum := 0.0
		for _, v := range values {
			sum += v
		}

		kpis = append(kpis, models.KPI{
			Name:  colName,
			Value: sum,
			Avg:   sum / float64(len(values)),
			Type:  "sum",
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(kpis)
}

// ============================================================================
// Column Similarity
// ============================================================================

func (h *Handler) GetColumnSimilarity(w http.ResponseWriter, r *http.Request) {
	df1 := state.State.GetDataFrame(1)
	df2 := state.State.GetDataFrame(2)

	if df1 == nil || df2 == nil {
		http.Error(w, "Both files must be loaded to calculate similarity", http.StatusBadRequest)
		return
	}

	ctx1 := state.State.GetContext(1)
	ctx2 := state.State.GetContext(2)

	// Build nodes for graph
	nodes := []map[string]interface{}{}
	for _, col := range df1.Headers {
		nodes = append(nodes, map[string]interface{}{
			"id":    fmt.Sprintf("file1_%s", col),
			"label": col,
			"group": "file1",
		})
	}
	for _, col := range df2.Headers {
		nodes = append(nodes, map[string]interface{}{
			"id":    fmt.Sprintf("file2_%s", col),
			"label": col,
			"group": "file2",
		})
	}

	// Check if AI matching is requested
	useAI := r.URL.Query().Get("use_ai") == "true"

	// Convert to response format
	type SimilarityItem struct {
		File1Column            string  `json:"file1_column"`
		File2Column            string  `json:"file2_column"`
		Similarity             float64 `json:"similarity"`
		Confidence             float64 `json:"confidence"`
		Type                   string  `json:"type"`
		DataSimilarity         float64 `json:"data_similarity"`
		NameSimilarity         float64 `json:"name_similarity"`
		DistributionSimilarity float64 `json:"distribution_similarity"`
		JSONConfidence         float64 `json:"json_confidence"`
		LLMSemanticScore       float64 `json:"llm_semantic_score"`
		Reason                 string  `json:"reason,omitempty"`
		TokenSimilarity        float64 `json:"token_similarity,omitempty"`
		SynonymMatch           bool    `json:"synonym_match,omitempty"`
		PatternMatch           string  `json:"pattern_match,omitempty"`
		ValueOverlap           float64 `json:"value_overlap,omitempty"`
		AIExplanation          string  `json:"ai_explanation,omitempty"`
	}

	similarities := []SimilarityItem{}

	if useAI && h.AISemanticMatcher != nil {
		// Use AI-powered matching
		log.Println("[API] Using AI-powered semantic matching via Ollama")
		aiResults := h.AISemanticMatcher.MatchColumns(df1, df2, ctx1, ctx2)
		for _, r := range aiResults {
			similarities = append(similarities, SimilarityItem{
				File1Column:            r.File1Column,
				File2Column:            r.File2Column,
				Similarity:             r.Confidence / 100,
				Confidence:             r.Confidence,
				Type:                   r.MatchType,
				DataSimilarity:         r.DataSimilarity,
				NameSimilarity:         r.NameSimilarity,
				DistributionSimilarity: r.DistributionSimilarity,
				LLMSemanticScore:       r.SemanticScore,
				Reason:                 r.Reason,
				ValueOverlap:           r.ValueOverlap,
				AIExplanation:          r.AIExplanation,
			})
		}
	} else {
		// Use Enhanced heuristic matching (default)
		enhancedResults := h.EnhancedSimilarityService.CalculateEnhancedSimilarity(df1, df2, ctx1, ctx2)
		for _, r := range enhancedResults {
			similarities = append(similarities, SimilarityItem{
				File1Column:            r.File1Column,
				File2Column:            r.File2Column,
				Similarity:             r.Similarity,
				Confidence:             r.Confidence,
				Type:                   r.Type,
				DataSimilarity:         r.DataSimilarity,
				NameSimilarity:         r.NameSimilarity,
				DistributionSimilarity: r.DistributionSimilarity,
				JSONConfidence:         r.JSONConfidence,
				LLMSemanticScore:       r.LLMSemanticScore,
				Reason:                 r.Reason,
				TokenSimilarity:        r.TokenSimilarity,
				SynonymMatch:           r.SynonymMatch,
				PatternMatch:           r.PatternMatch,
				ValueOverlap:           r.ValueOverlap,
			})
		}
	}

	totalRelationships := len(similarities)

	// Limit to top 15 for display
	if len(similarities) > 15 {
		similarities = similarities[:15]
	}

	// Build edges from top similarities
	edges := []map[string]interface{}{}
	for _, sim := range similarities {
		edges = append(edges, map[string]interface{}{
			"source":     fmt.Sprintf("file1_%s", sim.File1Column),
			"target":     fmt.Sprintf("file2_%s", sim.File2Column),
			"value":      sim.Confidence,
			"similarity": sim.Similarity,
			"type":       sim.Type,
			"label":      fmt.Sprintf("%d%%", int(sim.Confidence)),
		})
	}

	// Calculate correlations for ALL numeric column pairs (not just name-matched)
	type CorrelationItem struct {
		File1Column         string  `json:"file1_column"`
		File2Column         string  `json:"file2_column"`
		PearsonCorrelation  float64 `json:"pearson_correlation"`
		SpearmanCorrelation float64 `json:"spearman_correlation"`
		Strength            string  `json:"strength"`
		SampleSize          int     `json:"sample_size"`
	}

	correlations := []CorrelationItem{}
	numericCols1 := df1.GetNumericColumnIndices()
	numericCols2 := df2.GetNumericColumnIndices()

	// Calculate correlations for ALL numeric column pairs
	for col1Idx, isNumeric1 := range numericCols1 {
		if !isNumeric1 || col1Idx >= len(df1.Headers) {
			continue
		}
		col1Name := df1.Headers[col1Idx]

		for col2Idx, isNumeric2 := range numericCols2 {
			if !isNumeric2 || col2Idx >= len(df2.Headers) {
				continue
			}
			col2Name := df2.Headers[col2Idx]

			// Get values
			vals1 := getNumericValues(df1, col1Idx)
			vals2 := getNumericValues(df2, col2Idx)

			if len(vals1) == 0 || len(vals2) == 0 {
				continue
			}

			// Use minimum length
			minLen := len(vals1)
			if len(vals2) < minLen {
				minLen = len(vals2)
			}
			if minLen < 2 {
				continue
			}

			vals1 = vals1[:minLen]
			vals2 = vals2[:minLen]

			pearson := pearsonCorrelation(vals1, vals2)
			spearman := spearmanCorrelation(vals1, vals2)

			// Skip if correlation is very weak (less than 0.1)
			if math.Abs(pearson) < 0.1 {
				continue
			}

			// Determine strength
			absPearson := math.Abs(pearson)
			strength := "None"
			if absPearson >= 0.7 {
				strength = "Strong"
			} else if absPearson >= 0.4 {
				strength = "Moderate"
			} else if absPearson >= 0.2 {
				strength = "Weak"
			}

			correlations = append(correlations, CorrelationItem{
				File1Column:         col1Name,
				File2Column:         col2Name,
				PearsonCorrelation:  pearson,
				SpearmanCorrelation: spearman,
				Strength:            strength,
				SampleSize:          minLen,
			})
		}
	}

	// Sort correlations by absolute Pearson value
	sort.Slice(correlations, func(i, j int) bool {
		return math.Abs(correlations[i].PearsonCorrelation) > math.Abs(correlations[j].PearsonCorrelation)
	})

	// Limit to top 20 correlations
	if len(correlations) > 20 {
		correlations = correlations[:20]
	}

	// Return response in Python backend format
	resp := map[string]interface{}{
		"nodes":               nodes,
		"edges":               edges,
		"similarities":        similarities,
		"total_relationships": totalRelationships,
		"correlations":        correlations,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func getColumnIndex(headers []string, col string) int {
	for i, h := range headers {
		if h == col {
			return i
		}
	}
	return -1
}

func getPatternScore(col1, col2 string) float64 {
	patterns := map[string][]string{
		"email":   {"email", "e-mail", "mail"},
		"id":      {"id", "identifier", "key", "code", "number"},
		"date":    {"date", "time", "year", "month", "day", "created", "updated", "timestamp"},
		"name":    {"name", "first", "last", "full", "surname"},
		"phone":   {"phone", "mobile", "cell", "contact", "tel"},
		"address": {"address", "city", "state", "zip", "postal", "country", "location"},
		"price":   {"price", "cost", "amount", "value", "total", "revenue", "fee"},
		"status":  {"status", "state", "condition", "flag"},
	}

	c1Lower := strings.ToLower(col1)
	c2Lower := strings.ToLower(col2)

	for _, keywords := range patterns {
		match1, match2 := false, false
		for _, kw := range keywords {
			if strings.Contains(c1Lower, kw) {
				match1 = true
			}
			if strings.Contains(c2Lower, kw) {
				match2 = true
			}
		}
		if match1 && match2 {
			return 0.9
		}
	}
	return 0.0
}

func estimateDataSimilarity(df1, df2 *state.DataFrame, col1Idx, col2Idx int) float64 {
	// Quick heuristic: compare types and sample values
	if len(df1.Rows) == 0 || len(df2.Rows) == 0 {
		return 0.0
	}

	// Check if both are numeric
	num1 := df1.GetNumericColumnIndices()
	num2 := df2.GetNumericColumnIndices()

	isNum1 := num1[col1Idx]
	isNum2 := num2[col2Idx]

	if isNum1 && isNum2 {
		// Both numeric - assume some similarity
		return 0.4
	} else if !isNum1 && !isNum2 {
		// Both string - check Jaccard of unique values
		set1 := make(map[string]bool)
		set2 := make(map[string]bool)

		limit := 100
		if len(df1.Rows) < limit {
			limit = len(df1.Rows)
		}
		for i := 0; i < limit; i++ {
			if col1Idx < len(df1.Rows[i]) {
				set1[df1.Rows[i][col1Idx]] = true
			}
		}

		limit = 100
		if len(df2.Rows) < limit {
			limit = len(df2.Rows)
		}
		for i := 0; i < limit; i++ {
			if col2Idx < len(df2.Rows[i]) {
				set2[df2.Rows[i][col2Idx]] = true
			}
		}

		// Calculate Jaccard
		intersection := 0
		for k := range set1 {
			if set2[k] {
				intersection++
			}
		}

		union := len(set1) + len(set2) - intersection
		if union > 0 {
			return float64(intersection) / float64(union)
		}
	}

	return 0.0
}

// ============================================================================
// Correlation
// ============================================================================

func (h *Handler) GetCorrelation(w http.ResponseWriter, r *http.Request) {
	col1 := r.URL.Query().Get("col1")
	col2 := r.URL.Query().Get("col2")

	// If no params provided, return correlations for all matched column pairs
	if col1 == "" && col2 == "" {
		h.GetAllCorrelations(w, r)
		return
	}

	fileIndex := getIntParam(r, "file_index", 1)

	df := state.State.GetDataFrame(fileIndex)
	if df == nil {
		http.Error(w, fmt.Sprintf("File %d not loaded", fileIndex), http.StatusBadRequest)
		return
	}

	// Find column indices
	col1Idx, col2Idx := -1, -1
	for i, h := range df.Headers {
		if h == col1 {
			col1Idx = i
		}
		if h == col2 {
			col2Idx = i
		}
	}

	if col1Idx == -1 || col2Idx == -1 {
		http.Error(w, "Column not found", http.StatusNotFound)
		return
	}

	// Calculate correlation
	vals1, vals2 := []float64{}, []float64{}
	for _, row := range df.Rows {
		if col1Idx >= len(row) || col2Idx >= len(row) {
			continue
		}
		v1, err1 := strconv.ParseFloat(row[col1Idx], 64)
		v2, err2 := strconv.ParseFloat(row[col2Idx], 64)
		if err1 == nil && err2 == nil {
			vals1 = append(vals1, v1)
			vals2 = append(vals2, v2)
		}
	}

	if len(vals1) < 2 {
		http.Error(w, "Not enough numeric values for correlation", http.StatusBadRequest)
		return
	}

	corr := pearsonCorrelation(vals1, vals2)
	interpretation := "Weak/None"
	if corr > 0.7 {
		interpretation = "Strong positive"
	} else if corr < -0.7 {
		interpretation = "Strong negative"
	} else if corr > 0.3 {
		interpretation = "Moderate positive"
	} else if corr < -0.3 {
		interpretation = "Moderate negative"
	}

	resp := models.CorrelationResult{
		Column1:        col1,
		Column2:        col2,
		Correlation:    corr,
		Interpretation: interpretation,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// GetAllCorrelations returns correlations for all numeric column pairs between two files
func (h *Handler) GetAllCorrelations(w http.ResponseWriter, r *http.Request) {
	df1 := state.State.GetDataFrame(1)
	df2 := state.State.GetDataFrame(2)

	if df1 == nil || df2 == nil {
		http.Error(w, "Both files must be loaded to calculate correlations", http.StatusBadRequest)
		return
	}

	// Get numeric columns from both files
	numericCols1 := df1.GetNumericColumnIndices()
	numericCols2 := df2.GetNumericColumnIndices()

	type CorrelationItem struct {
		File1Column         string  `json:"file1_column"`
		File2Column         string  `json:"file2_column"`
		Correlation         float64 `json:"correlation"`
		PearsonCorrelation  float64 `json:"pearson_correlation"`
		SpearmanCorrelation float64 `json:"spearman_correlation"`
		Strength            string  `json:"strength"`
		SampleSize          int     `json:"sample_size"`
		File1Rows           int     `json:"file1_rows"`
		File2Rows           int     `json:"file2_rows"`
	}

	correlations := []CorrelationItem{}

	// Calculate correlations for matching numeric columns
	for col1Idx := range numericCols1 {
		if col1Idx >= len(df1.Headers) {
			continue
		}
		col1Name := df1.Headers[col1Idx]

		for col2Idx := range numericCols2 {
			if col2Idx >= len(df2.Headers) {
				continue
			}
			col2Name := df2.Headers[col2Idx]

			// Get values
			vals1 := getNumericValues(df1, col1Idx)
			vals2 := getNumericValues(df2, col2Idx)

			if len(vals1) == 0 || len(vals2) == 0 {
				continue
			}

			// Use minimum length
			minLen := len(vals1)
			if len(vals2) < minLen {
				minLen = len(vals2)
			}
			if minLen < 2 {
				continue
			}

			vals1 = vals1[:minLen]
			vals2 = vals2[:minLen]

			corr := pearsonCorrelation(vals1, vals2)
			spearman := spearmanCorrelation(vals1, vals2)

			// Determine strength
			absCorr := math.Abs(corr)
			strength := "None"
			if absCorr >= 0.7 {
				strength = "Strong"
			} else if absCorr >= 0.4 {
				strength = "Moderate"
			} else if absCorr >= 0.2 {
				strength = "Weak"
			}

			// Only include if there's some correlation
			if absCorr >= 0.1 {
				correlations = append(correlations, CorrelationItem{
					File1Column:         col1Name,
					File2Column:         col2Name,
					Correlation:         corr,
					PearsonCorrelation:  corr,
					SpearmanCorrelation: spearman,
					Strength:            strength,
					SampleSize:          minLen,
					File1Rows:           len(df1.Rows),
					File2Rows:           len(df2.Rows),
				})
			}
		}
	}

	// Sort by absolute correlation descending
	sort.Slice(correlations, func(i, j int) bool {
		return math.Abs(correlations[i].Correlation) > math.Abs(correlations[j].Correlation)
	})

	// Limit to top 50
	if len(correlations) > 50 {
		correlations = correlations[:50]
	}

	// Get column lists
	file1Cols := []string{}
	file2Cols := []string{}
	for i, h := range df1.Headers {
		if numericCols1[i] {
			file1Cols = append(file1Cols, h)
		}
	}
	for i, h := range df2.Headers {
		if numericCols2[i] {
			file2Cols = append(file2Cols, h)
		}
	}

	resp := map[string]interface{}{
		"total_correlations": len(correlations),
		"correlations":       correlations,
		"file1_columns":      file1Cols,
		"file2_columns":      file2Cols,
		"file1_rows":         len(df1.Rows),
		"file2_rows":         len(df2.Rows),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func getNumericValues(df *state.DataFrame, colIdx int) []float64 {
	values := []float64{}
	for _, row := range df.Rows {
		if colIdx >= len(row) {
			continue
		}
		if val, err := strconv.ParseFloat(row[colIdx], 64); err == nil {
			values = append(values, val)
		}
	}
	return values
}

func spearmanCorrelation(x, y []float64) float64 {
	// Simple Spearman: convert to ranks and compute Pearson
	n := len(x)
	if n == 0 {
		return 0
	}

	rankX := computeRanks(x)
	rankY := computeRanks(y)

	return pearsonCorrelation(rankX, rankY)
}

func computeRanks(vals []float64) []float64 {
	n := len(vals)
	type indexedVal struct {
		val   float64
		index int
	}

	indexed := make([]indexedVal, n)
	for i, v := range vals {
		indexed[i] = indexedVal{v, i}
	}

	sort.Slice(indexed, func(i, j int) bool {
		return indexed[i].val < indexed[j].val
	})

	ranks := make([]float64, n)
	for rank, iv := range indexed {
		ranks[iv.index] = float64(rank + 1)
	}
	return ranks
}

func pearsonCorrelation(x, y []float64) float64 {
	n := float64(len(x))
	if n == 0 {
		return 0
	}

	sumX, sumY, sumXY, sumX2, sumY2 := 0.0, 0.0, 0.0, 0.0, 0.0
	for i := range x {
		sumX += x[i]
		sumY += y[i]
		sumXY += x[i] * y[i]
		sumX2 += x[i] * x[i]
		sumY2 += y[i] * y[i]
	}

	num := n*sumXY - sumX*sumY
	den := math.Sqrt((n*sumX2 - sumX*sumX) * (n*sumY2 - sumY*sumY))

	if den == 0 {
		return 0
	}
	return num / den
}

// ============================================================================
// Filter
// ============================================================================

func (h *Handler) FilterData(w http.ResponseWriter, r *http.Request) {
	df := state.State.GetDataFrame(1)
	if df == nil {
		http.Error(w, "No CSV file loaded", http.StatusBadRequest)
		return
	}

	var req models.FilterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Build column index map
	colIdx := make(map[string]int)
	for i, h := range df.Headers {
		colIdx[h] = i
	}

	// Filter rows
	filtered := [][]string{}
	for _, row := range df.Rows {
		match := true
		for _, cond := range req.Conditions {
			idx, ok := colIdx[cond.Column]
			if !ok || idx >= len(row) {
				continue
			}
			val := row[idx]

			switch cond.Operator {
			case "equals":
				if val != cond.Value {
					match = false
				}
			case "contains":
				if !strings.Contains(strings.ToLower(val), strings.ToLower(cond.Value)) {
					match = false
				}
			case "greater_than":
				fVal, err1 := strconv.ParseFloat(val, 64)
				fCond, err2 := strconv.ParseFloat(cond.Value, 64)
				if err1 != nil || err2 != nil || fVal <= fCond {
					match = false
				}
			case "less_than":
				fVal, err1 := strconv.ParseFloat(val, 64)
				fCond, err2 := strconv.ParseFloat(cond.Value, 64)
				if err1 != nil || err2 != nil || fVal >= fCond {
					match = false
				}
			}
		}
		if match {
			filtered = append(filtered, row)
		}
	}

	// Convert to response format (limit to 100)
	limit := 100
	if len(filtered) < limit {
		limit = len(filtered)
	}

	data := make([]map[string]interface{}, limit)
	for i := 0; i < limit; i++ {
		row := make(map[string]interface{})
		for j, header := range df.Headers {
			if j < len(filtered[i]) {
				row[header] = filtered[i][j]
			}
		}
		data[i] = row
	}

	resp := models.FilterResponse{
		Rows: len(filtered),
		Data: data,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// ============================================================================
// Query (LLM-based data analysis)
// ============================================================================

type QueryRequest struct {
	Question string `json:"question"`
}

type QueryResponse struct {
	Answer      string                   `json:"answer"`
	Explanation string                   `json:"explanation"`
	RawResponse string                   `json:"raw_response,omitempty"`
	Result      string                   `json:"result,omitempty"`
	ResultData  []map[string]interface{} `json:"result_data,omitempty"`
	ResultType  string                   `json:"result_type,omitempty"`
	Error       string                   `json:"error,omitempty"`
}

func (h *Handler) Query(w http.ResponseWriter, r *http.Request) {
	df := state.State.GetDataFrame(1)
	if df == nil {
		http.Error(w, "No CSV file loaded. Please upload a file first.", http.StatusBadRequest)
		return
	}

	var req QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Question == "" {
		http.Error(w, "Question is required", http.StatusBadRequest)
		return
	}

	// Process the query
	question := strings.ToLower(req.Question)
	resp := QueryResponse{}

	// Simple query processing without LLM (fallback mode)
	if strings.Contains(question, "average") || strings.Contains(question, "mean") {
		resp = h.processAverageQuery(df, question)
	} else if strings.Contains(question, "sum") || strings.Contains(question, "total") {
		resp = h.processSumQuery(df, question)
	} else if strings.Contains(question, "count") || strings.Contains(question, "how many") {
		resp = h.processCountQuery(df, question)
	} else if strings.Contains(question, "max") || strings.Contains(question, "maximum") || strings.Contains(question, "highest") {
		resp = h.processMaxQuery(df, question)
	} else if strings.Contains(question, "min") || strings.Contains(question, "minimum") || strings.Contains(question, "lowest") {
		resp = h.processMinQuery(df, question)
	} else if strings.Contains(question, "overview") || strings.Contains(question, "summary") || strings.Contains(question, "describe") {
		resp = h.processOverviewQuery(df)
	} else if strings.Contains(question, "top") {
		resp = h.processTopQuery(df, question)
	} else {
		// Default: provide overview
		resp = h.processOverviewQuery(df)
		resp.Explanation = fmt.Sprintf("I understood your question: '%s'. Here's an overview of the data. For specific queries, try asking about averages, sums, counts, or statistics.", req.Question)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) processAverageQuery(df *state.DataFrame, question string) QueryResponse {
	numericCols := df.GetNumericColumnIndices()
	results := []string{}

	for colIdx, isNumeric := range numericCols {
		if !isNumeric || colIdx >= len(df.Headers) {
			continue
		}
		colName := df.Headers[colIdx]
		if strings.Contains(question, strings.ToLower(colName)) || strings.Contains(question, "all") {
			sum, count := 0.0, 0
			for _, row := range df.Rows {
				if colIdx < len(row) {
					if val, err := strconv.ParseFloat(row[colIdx], 64); err == nil {
						sum += val
						count++
					}
				}
			}
			if count > 0 {
				avg := sum / float64(count)
				results = append(results, fmt.Sprintf("%s: %.2f", colName, avg))
			}
		}
	}

	if len(results) == 0 {
		// Calculate for all numeric columns
		for colIdx, isNumeric := range numericCols {
			if !isNumeric || colIdx >= len(df.Headers) {
				continue
			}
			colName := df.Headers[colIdx]
			sum, count := 0.0, 0
			for _, row := range df.Rows {
				if colIdx < len(row) {
					if val, err := strconv.ParseFloat(row[colIdx], 64); err == nil {
						sum += val
						count++
					}
				}
			}
			if count > 0 {
				avg := sum / float64(count)
				results = append(results, fmt.Sprintf("%s: %.2f", colName, avg))
			}
		}
	}

	return QueryResponse{
		Answer:      fmt.Sprintf("Average values:\n%s", strings.Join(results, "\n")),
		Explanation: "Calculated average for numeric columns.",
		Result:      strings.Join(results, "\n"),
		ResultType:  "statistics",
	}
}

func (h *Handler) processSumQuery(df *state.DataFrame, question string) QueryResponse {
	numericCols := df.GetNumericColumnIndices()
	results := []string{}

	for colIdx := range numericCols {
		if colIdx >= len(df.Headers) {
			continue
		}
		colName := df.Headers[colIdx]
		sum := 0.0
		for _, row := range df.Rows {
			if colIdx < len(row) {
				if val, err := strconv.ParseFloat(row[colIdx], 64); err == nil {
					sum += val
				}
			}
		}
		results = append(results, fmt.Sprintf("%s: %.2f", colName, sum))
	}

	return QueryResponse{
		Answer:      fmt.Sprintf("Sum of values:\n%s", strings.Join(results, "\n")),
		Explanation: "Calculated sum for numeric columns.",
		Result:      strings.Join(results, "\n"),
		ResultType:  "statistics",
	}
}

func (h *Handler) processCountQuery(df *state.DataFrame, question string) QueryResponse {
	return QueryResponse{
		Answer:      fmt.Sprintf("Total rows: %d\nTotal columns: %d", len(df.Rows), len(df.Headers)),
		Explanation: "Counted records in the dataset.",
		Result:      fmt.Sprintf("Rows: %d, Columns: %d", len(df.Rows), len(df.Headers)),
		ResultType:  "count",
	}
}

func (h *Handler) processMaxQuery(df *state.DataFrame, question string) QueryResponse {
	numericCols := df.GetNumericColumnIndices()
	results := []string{}

	for colIdx := range numericCols {
		if colIdx >= len(df.Headers) {
			continue
		}
		colName := df.Headers[colIdx]
		maxVal := math.Inf(-1)
		for _, row := range df.Rows {
			if colIdx < len(row) {
				if val, err := strconv.ParseFloat(row[colIdx], 64); err == nil {
					if val > maxVal {
						maxVal = val
					}
				}
			}
		}
		if maxVal != math.Inf(-1) {
			results = append(results, fmt.Sprintf("%s: %.2f", colName, maxVal))
		}
	}

	return QueryResponse{
		Answer:      fmt.Sprintf("Maximum values:\n%s", strings.Join(results, "\n")),
		Explanation: "Found maximum for numeric columns.",
		Result:      strings.Join(results, "\n"),
		ResultType:  "statistics",
	}
}

func (h *Handler) processMinQuery(df *state.DataFrame, question string) QueryResponse {
	numericCols := df.GetNumericColumnIndices()
	results := []string{}

	for colIdx := range numericCols {
		if colIdx >= len(df.Headers) {
			continue
		}
		colName := df.Headers[colIdx]
		minVal := math.Inf(1)
		for _, row := range df.Rows {
			if colIdx < len(row) {
				if val, err := strconv.ParseFloat(row[colIdx], 64); err == nil {
					if val < minVal {
						minVal = val
					}
				}
			}
		}
		if minVal != math.Inf(1) {
			results = append(results, fmt.Sprintf("%s: %.2f", colName, minVal))
		}
	}

	return QueryResponse{
		Answer:      fmt.Sprintf("Minimum values:\n%s", strings.Join(results, "\n")),
		Explanation: "Found minimum for numeric columns.",
		Result:      strings.Join(results, "\n"),
		ResultType:  "statistics",
	}
}

func (h *Handler) processOverviewQuery(df *state.DataFrame) QueryResponse {
	numericCols := df.GetNumericColumnIndices()

	summary := fmt.Sprintf("📊 Dataset Overview:\n• Rows: %d\n• Columns: %d\n• Column names: %s\n\n",
		len(df.Rows), len(df.Headers), strings.Join(df.Headers, ", "))

	numericHeaders := []string{}
	for colIdx := range numericCols {
		if colIdx < len(df.Headers) {
			numericHeaders = append(numericHeaders, df.Headers[colIdx])
		}
	}

	if len(numericHeaders) > 0 {
		summary += fmt.Sprintf("📈 Numeric columns: %s\n", strings.Join(numericHeaders, ", "))
	}

	return QueryResponse{
		Answer:      summary,
		Explanation: "Generated overview of the dataset.",
		ResultType:  "overview",
	}
}

func (h *Handler) processTopQuery(df *state.DataFrame, question string) QueryResponse {
	// Default to top 5
	n := 5
	if strings.Contains(question, "10") {
		n = 10
	} else if strings.Contains(question, "3") {
		n = 3
	}

	if n > len(df.Rows) {
		n = len(df.Rows)
	}

	data := make([]map[string]interface{}, n)
	for i := 0; i < n; i++ {
		row := make(map[string]interface{})
		for j, header := range df.Headers {
			if j < len(df.Rows[i]) {
				row[header] = df.Rows[i][j]
			}
		}
		data[i] = row
	}

	return QueryResponse{
		Answer:      fmt.Sprintf("Top %d records from the dataset", n),
		Explanation: fmt.Sprintf("Showing first %d rows.", n),
		ResultData:  data,
		ResultType:  "dataframe",
	}
}

// ============================================================================
// Context Management
// ============================================================================

func (h *Handler) GenerateContextQuestions(w http.ResponseWriter, r *http.Request) {
	df1 := state.State.GetDataFrame(1)
	df2 := state.State.GetDataFrame(2)

	if df1 == nil || df2 == nil {
		http.Error(w, "Both files must be loaded to generate context questions", http.StatusBadRequest)
		return
	}

	// Generate questions for both files
	analysis1 := h.analyzeDataFrame(df1)
	analysis2 := h.analyzeDataFrame(df2)

	questions1 := h.QuestionGenerator.GenerateQuestions(analysis1, 1)
	questions2 := h.QuestionGenerator.GenerateQuestions(analysis2, 2)

	// Relationship questions
	relationshipQuestions := []models.Question{
		{
			ID:       "rel_type",
			Type:     models.QuestionTypeRelationships,
			Text:     "How are these two datasets related?",
			Options:  []string{"One-to-One", "One-to-Many", "Many-to-Many", "Hierarchical", "Temporal sequence", "Unknown"},
			Required: false,
		},
		{
			ID:       "rel_keys",
			Type:     models.QuestionTypeCustomMappings,
			Text:     "Which columns should be used to join these datasets?",
			Options:  append(df1.Headers, df2.Headers...),
			Required: false,
		},
	}

	resp := models.QuestionsResponse{
		Success: true,
		Questions: map[string][]models.Question{
			"file1_questions":        questions1,
			"file2_questions":        questions2,
			"relationship_questions": relationshipQuestions,
		},
		TotalQuestions: len(questions1) + len(questions2) + len(relationshipQuestions),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) analyzeDataFrame(df *state.DataFrame) models.DataAnalysisResult {
	result := models.DataAnalysisResult{
		NumRows:     len(df.Rows),
		NumColumns:  len(df.Headers),
		ColumnNames: df.Headers,
		ColumnTypes: make(map[string]string),
	}

	numericCols := df.GetNumericColumnIndices()
	for i, header := range df.Headers {
		if numericCols[i] {
			result.HasNumeric = true
			headerLower := strings.ToLower(header)
			if strings.Contains(headerLower, "id") || strings.Contains(headerLower, "key") {
				result.PotentialIDs = append(result.PotentialIDs, header)
			}
			if strings.Contains(headerLower, "amount") || strings.Contains(headerLower, "price") {
				result.PotentialAmounts = append(result.PotentialAmounts, header)
			}
			result.ColumnTypes[header] = "numeric"
		} else if isDateColumn(df, i) {
			result.HasDates = true
			result.PotentialDates = append(result.PotentialDates, header)
			result.ColumnTypes[header] = "date"
		} else {
			result.HasText = true
			result.ColumnTypes[header] = "string"
		}
	}

	return result
}

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

func (h *Handler) SubmitContext(w http.ResponseWriter, r *http.Request) {
	var req models.ContextSubmitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.FileIndex != 1 && req.FileIndex != 2 {
		http.Error(w, "file_index must be 1 or 2", http.StatusBadRequest)
		return
	}

	// Convert context data to Context model
	ctx := models.NewContext()
	if purpose, ok := req.ContextData["dataset_purpose"].(string); ok {
		ctx.DatasetPurpose = purpose
	}
	if domain, ok := req.ContextData["business_domain"].(string); ok {
		ctx.BusinessDomain = domain
	}
	if entities, ok := req.ContextData["key_entities"].([]interface{}); ok {
		for _, e := range entities {
			if s, ok := e.(string); ok {
				ctx.KeyEntities = append(ctx.KeyEntities, s)
			}
		}
	}
	if temporal, ok := req.ContextData["temporal_context"].(string); ok {
		ctx.TemporalContext = temporal
	}
	if exclusions, ok := req.ContextData["exclusions"].([]interface{}); ok {
		for _, e := range exclusions {
			if s, ok := e.(string); ok {
				ctx.Exclusions = append(ctx.Exclusions, s)
			}
		}
	}

	state.State.SetContext(req.FileIndex, ctx)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Context stored for File %d", req.FileIndex),
		"context": ctx,
	})
}

func (h *Handler) GetContext(w http.ResponseWriter, r *http.Request) {
	fileIndexStr := chi.URLParam(r, "fileIndex")
	fileIndex, err := strconv.Atoi(fileIndexStr)
	if err != nil || (fileIndex != 1 && fileIndex != 2) {
		http.Error(w, "fileIndex must be 1 or 2", http.StatusBadRequest)
		return
	}

	ctx := state.State.GetContext(fileIndex)

	resp := map[string]interface{}{
		"success":     true,
		"has_context": ctx != nil,
		"context":     ctx,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// GetQuestions endpoint (My V2 impl)
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

func (h *Handler) DeleteContext(w http.ResponseWriter, r *http.Request) {
	fileIndexStr := chi.URLParam(r, "fileIndex")
	fileIndex, err := strconv.Atoi(fileIndexStr)
	if err != nil || (fileIndex != 1 && fileIndex != 2) {
		http.Error(w, "fileIndex must be 1 or 2", http.StatusBadRequest)
		return
	}

	state.State.ClearContext(&fileIndex)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Context cleared for File %d", fileIndex),
	})
}

// GetSimilarityGraph generates the correlation graph (My V2 impl)
func (h *Handler) GetSimilarityGraph(w http.ResponseWriter, r *http.Request) {
	graph, err := h.SimilarityService.GenerateGraph(1, 2)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error generating graph: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(graph)
}

func (h *Handler) GetContextStatus(w http.ResponseWriter, r *http.Request) {
	ctx1 := state.State.GetContext(1)
	ctx2 := state.State.GetContext(2)

	resp := models.ContextStatusResponse{
		File1: models.ContextStatusItem{
			HasContext: ctx1 != nil,
		},
		File2: models.ContextStatusItem{
			HasContext: ctx2 != nil,
		},
	}

	if ctx1 != nil {
		resp.File1.ContextSummary = map[string]interface{}{
			"dataset_purpose": ctx1.DatasetPurpose,
			"business_domain": ctx1.BusinessDomain,
		}
	}
	if ctx2 != nil {
		resp.File2.ContextSummary = map[string]interface{}{
			"dataset_purpose": ctx2.DatasetPurpose,
			"business_domain": ctx2.BusinessDomain,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// ============================================================================
// Ollama Config
// ============================================================================

func (h *Handler) GetOllamaConfig(w http.ResponseWriter, r *http.Request) {
	resp := models.OllamaConfig{
		BaseURL: state.State.OllamaBaseURL,
		Model:   state.State.OllamaModel,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) SaveOllamaConfig(w http.ResponseWriter, r *http.Request) {
	var config models.OllamaConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if config.BaseURL != "" {
		state.State.OllamaBaseURL = config.BaseURL
	}
	if config.Model != "" {
		state.State.OllamaModel = config.Model
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Ollama configuration saved successfully",
		"config": models.OllamaConfig{
			BaseURL: state.State.OllamaBaseURL,
			Model:   state.State.OllamaModel,
		},
	})
}

// ============================================================================
// Feedback Learning
// ============================================================================

// SubmitMatchFeedback handles POST /feedback/match
func (h *Handler) SubmitMatchFeedback(w http.ResponseWriter, r *http.Request) {
	var req struct {
		File1Column    string  `json:"file1_column"`
		File2Column    string  `json:"file2_column"`
		IsCorrect      bool    `json:"is_correct"`
		CorrectMatch   string  `json:"correct_match,omitempty"`
		UserNote       string  `json:"user_note,omitempty"`
		NameSimilarity float64 `json:"name_similarity"`
		DataSimilarity float64 `json:"data_similarity"`
		PatternScore   float64 `json:"pattern_score"`
		Confidence     float64 `json:"confidence"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.File1Column == "" || req.File2Column == "" {
		http.Error(w, "file1_column and file2_column are required", http.StatusBadRequest)
		return
	}

	feedbackSystem := service.GetFeedbackSystem()

	entry := service.FeedbackEntry{
		File1Column:    req.File1Column,
		File2Column:    req.File2Column,
		IsCorrect:      req.IsCorrect,
		CorrectMatch:   req.CorrectMatch,
		UserNote:       req.UserNote,
		NameSimilarity: req.NameSimilarity,
		DataSimilarity: req.DataSimilarity,
		PatternScore:   req.PatternScore,
		Confidence:     req.Confidence,
	}

	result, err := feedbackSystem.AddFeedback(entry)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error recording feedback: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"message":  "Feedback recorded successfully",
		"feedback": result,
	})
}

// GetFeedbackStats handles GET /feedback/stats
func (h *Handler) GetFeedbackStats(w http.ResponseWriter, r *http.Request) {
	feedbackSystem := service.GetFeedbackSystem()
	stats := feedbackSystem.GetStats()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
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

// ============================================================================
// Helpers
// ============================================================================

func getIntParam(r *http.Request, name string, defaultVal int) int {
	valStr := r.URL.Query().Get(name)
	if valStr == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return defaultVal
	}
	return val
}
