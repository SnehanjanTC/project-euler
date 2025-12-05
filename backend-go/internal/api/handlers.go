package api

import (
	"backend-go/internal/analysis"
	"backend-go/internal/models"
	"backend-go/internal/service"
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	ContextService    *service.ContextService
	QuestionGenerator *service.QuestionGenerator
	CSVService        *analysis.CSVService
	SimilarityService *service.SimilarityService
}

func NewHandler(ctx *service.ContextService, qg *service.QuestionGenerator, csv *analysis.CSVService, sim *service.SimilarityService) *Handler {
	return &Handler{
		ContextService:    ctx,
		QuestionGenerator: qg,
		CSVService:        csv,
		SimilarityService: sim,
	}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/health", h.HealthCheck)
	r.Post("/api/analyze-file", h.AnalyzeFile)
	r.Post("/api/context/{fileIndex}", h.StoreContext)
	r.Get("/api/questions/{fileIndex}", h.GetQuestions)
}

func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
}

// AnalyzeFile - Mocking file upload analysis for now,
// normally would handle multipart upload and call CSVService
func (h *Handler) AnalyzeFile(w http.ResponseWriter, r *http.Request) {
	// In a real app, read file from request.
	// For this rewrite verify, we assume files are local or handling logic later.
	// We will just return a mocked success for the structure check.
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "File received (Go Backend)"})
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

// GetQuestions endpoint (Simulates generating questions based on stored file context state)
func (h *Handler) GetQuestions(w http.ResponseWriter, r *http.Request) {
	fileIndexStr := chi.URLParam(r, "fileIndex")
	fileIndex, err := strconv.Atoi(fileIndexStr)
	if err != nil {
		http.Error(w, "Invalid file index", http.StatusBadRequest)
		return
	}

	// Mock analysis data for now since we don't have the file physically uploaded in this request flow
	// In real app, we'd retrieve the analysis from state/DB
	mockAnalysis := models.DataAnalysisResult{
		NumRows:        100,
		ColumnNames:    []string{"id", "name", "email", "date_joined", "total_spend"},
		PotentialDates: []string{"date_joined"},
		HasDates:       true,
	}

	questions := h.QuestionGenerator.GenerateQuestions(mockAnalysis, fileIndex)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(questions)
}
