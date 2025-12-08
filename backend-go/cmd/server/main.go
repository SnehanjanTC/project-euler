package main

import (
	"log"
	"net/http"
	"os"

	"backend-go/internal/analysis"
	"backend-go/internal/api"
	"backend-go/internal/llm"
	"backend-go/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func main() {
	// Initialize Services
	llmService := llm.NewService("", "")
	ctxService := service.NewContextService()
	qgService := service.NewQuestionGenerator(llmService)
	csvService := analysis.NewCSVService()
	simService := service.NewSimilarityService(ctxService)
	exportService := service.NewExportService()

	// Initialize Handler
	handler := api.NewHandler(ctxService, qgService, csvService, simService, exportService)

	// Router Setup
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// CORS
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "http://localhost:3001", "http://localhost:3002"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Project Euler Go Backend is Running"))
	})

	// Register Routes
	handler.RegisterRoutes(r)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8001"
	}

	log.Printf("Starting Go Backend on port %s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
