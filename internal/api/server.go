package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/mats/telegram-quiz-bot/internal/llm"
	"github.com/mats/telegram-quiz-bot/internal/repository"
)

// Server represents the HTTP server for the management API and frontend.
type Server struct {
	server *http.Server
	repos  *repository.Repositories
	llm    *llm.Client
}

// NewServer initializes a new HTTP server.
func NewServer(port string, repos *repository.Repositories, llmClient *llm.Client) *Server {
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()

	s := &Server{
		server: &http.Server{
			Addr:    ":" + port,
			Handler: mux,
		},
		repos: repos,
		llm:   llmClient,
	}

	s.registerRoutes(mux)

	return s
}

// Start runs the HTTP server in a blocking manner.
func (s *Server) Start() error {
	log.Printf("Starting HTTP server on %s", s.server.Addr)
	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Stop gracefully shuts down the HTTP server.
func (s *Server) Stop(ctx context.Context) error {
	log.Println("Shutting down HTTP server...")
	return s.server.Shutdown(ctx)
}

func (s *Server) registerRoutes(mux *http.ServeMux) {
	// Serve static files from frontend/dist
	frontendDist := "frontend/dist"

	// Create a file server for the static files
	fs := http.FileServer(http.Dir(frontendDist))

	// Handle all routes not starting with /api/
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Clean the path to prevent directory traversal
		path := filepath.Clean(r.URL.Path)

		// Full path to the requested file
		fullPath := filepath.Join(frontendDist, path)

		// Check if the file exists and is not a directory
		info, err := os.Stat(fullPath)
		if os.IsNotExist(err) || info.IsDir() {
			// Serve index.html for SPA routing fallback
			http.ServeFile(w, r, filepath.Join(frontendDist, "index.html"))
			return
		}

		// Serve the actual file (JS, CSS, images, etc)
		fs.ServeHTTP(w, r)
	})

	// Plan
	mux.HandleFunc("GET /api/plan", s.handleGetPlan)

	// Segments
	mux.HandleFunc("POST /api/segments/suggest", s.handleSuggestSegments)
	mux.HandleFunc("GET /api/segments", s.handleGetSegments)
	mux.HandleFunc("POST /api/segments", s.handleCreateSegment)
	mux.HandleFunc("GET /api/segments/{id}", s.handleGetSegment)
	mux.HandleFunc("PUT /api/segments/{id}", s.handleUpdateSegment)
	mux.HandleFunc("DELETE /api/segments/{id}", s.handleDeleteSegment)

	// Quizzes (nested under segments for creation/listing)
	mux.HandleFunc("GET /api/segments/{id}/quizzes", s.handleGetSegmentQuizzes)
	mux.HandleFunc("POST /api/segments/{id}/quizzes", s.handleCreateSegmentQuiz)

	// Quizzes (direct access for update/delete/get)
	mux.HandleFunc("GET /api/quizzes/{id}", s.handleGetQuiz)
	mux.HandleFunc("PUT /api/quizzes/{id}", s.handleUpdateQuiz)
	mux.HandleFunc("DELETE /api/quizzes/{id}", s.handleDeleteQuiz)

	// Questions (nested under quizzes for creation/listing)
	mux.HandleFunc("GET /api/quizzes/{id}/questions", s.handleGetQuizQuestions)
	mux.HandleFunc("POST /api/quizzes/{id}/questions", s.handleCreateQuizQuestion)

	// Questions (direct access for update/delete/get)
	mux.HandleFunc("GET /api/questions/{id}", s.handleGetQuestion)
	mux.HandleFunc("PUT /api/questions/{id}", s.handleUpdateQuestion)
	mux.HandleFunc("DELETE /api/questions/{id}", s.handleDeleteQuestion)
}

// Helper functions for HTTP responses
func respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if payload != nil {
		json.NewEncoder(w).Encode(payload)
	}
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

func parseID(r *http.Request) (int, error) {
	idStr := r.PathValue("id")
	return strconv.Atoi(idStr)
}
