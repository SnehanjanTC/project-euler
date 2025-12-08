package main

import (
	"log"
	"net/http"
	"os"

	"backend-go/internal/analysis"
	"backend-go/internal/api"
	"backend-go/internal/llm"
	"backend-go/internal/service"
	"backend-go/internal/state"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func main() {
	// Initialize Services
	llmService := llm.NewService(state.State.OllamaBaseURL, state.State.OllamaModel)
	ctxService := service.NewContextService()
	qgService := service.NewQuestionGenerator(llmService)
	csvService := analysis.NewCSVService()
	simService := service.NewSimilarityService(ctxService)
	exportService := service.NewExportService()

	// Initialize Handler
	handler := api.NewHandler(ctxService, qgService, csvService, simService, exportService, llmService)

	// Router Setup
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)

	// CORS - Allow frontend
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"http://localhost:3000", "http://localhost:3001", "http://localhost:3002", "http://127.0.0.1:3000"},

		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Root endpoint
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Project Euler Go Backend is Running"))
	})

	// Register all API Routes
	handler.RegisterRoutes(r)

	// Get port from environment or default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8001"
	}

	log.Printf("🚀 Starting Go Backend on http://localhost:%s", port)
	log.Printf("📡 CORS enabled for: http://localhost:3000")
	log.Printf("📁 Upload directory: ./uploads")

	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
