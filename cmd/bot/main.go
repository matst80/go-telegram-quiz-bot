package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mats/telegram-quiz-bot/internal/api"
	"github.com/mats/telegram-quiz-bot/internal/bot"
	"github.com/mats/telegram-quiz-bot/internal/llm"
	"github.com/mats/telegram-quiz-bot/internal/quiz"
	"github.com/mats/telegram-quiz-bot/internal/repository/sqlite"
	"github.com/mats/telegram-quiz-bot/internal/tts"
)

func main() {
	// 1. Load configuration
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN environment variable is required")
	}

	ollamaURL := os.Getenv("OLLAMA_URL")
	if ollamaURL == "" {
		ollamaURL = "http://localhost:11434"
	}

	modelName := os.Getenv("OLLAMA_MODEL")
	if modelName == "" {
		modelName = "qwen3.5:4b"
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "quizbot.db"
	}

	// 2. Initialize Database & Repositories
	store, err := sqlite.NewStore(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer store.Close()
	log.Println("Database initialized.")

	repos := store.Repositories()

	// 3. Initialize LLM Client
	llmClient := llm.NewClient(ollamaURL, modelName)
	log.Printf("LLM Client initialized (Model: %s at %s)", modelName, ollamaURL)

	// Initialize TTS Service
	modelDir := "storage/tts_model"
	piperConfig, err := tts.EnsureDefaultModel(modelDir)
	if err != nil {
		log.Fatalf("Failed to initialize TTS model: %v", err)
	}
	ttsService, err := tts.NewPiperService(piperConfig)
	if err != nil {
		log.Fatalf("Failed to create TTS service: %v", err)
	}

	// Make sure the audio storage directory exists
	if err := os.MkdirAll("storage/audio", 0755); err != nil {
		log.Fatalf("Failed to create audio storage directory: %v", err)
	}

	// 4. Initialize Scheduler and Plan
	planManager := quiz.NewPlanManager(repos)
	scheduler := quiz.NewScheduler(repos, llmClient, planManager, ttsService)

	// Schedule a quiz generation every 5 minutes during daytime (8 AM - 8 PM)
	err = scheduler.Start("0 0 * * *")
	if err != nil {
		log.Fatalf("Failed to start scheduler: %v", err)
	}
	defer scheduler.Stop()

	// 5. Initialize and Start API Server
	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = "8080"
	}
	apiServer := api.NewServer(httpPort, repos, llmClient)
	go func() {
		if err := apiServer.Start(); err != nil {
			log.Fatalf("API Server failed: %v", err)
		}
	}()

	// 6. Initialize and Start Bot
	quizBot, err := bot.New(botToken, repos, llmClient, scheduler, planManager)
	if err != nil {
		log.Fatalf("Failed to initialize bot: %v", err)
	}

	go quizBot.Start()
	defer quizBot.Stop()

	// 7. Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down servers...")

	// Gracefully shutdown API server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := apiServer.Stop(ctx); err != nil {
		log.Printf("API Server shutdown error: %v", err)
	}
}
