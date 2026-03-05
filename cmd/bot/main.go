package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/mats/telegram-quiz-bot/internal/bot"
	"github.com/mats/telegram-quiz-bot/internal/llm"
	"github.com/mats/telegram-quiz-bot/internal/quiz"
	"github.com/mats/telegram-quiz-bot/internal/repository/sqlite"
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

	// 4. Initialize Scheduler and Plan
	planManager := quiz.NewPlanManager(repos)
	scheduler := quiz.NewScheduler(repos, llmClient, planManager)

	// Schedule a quiz generation every 5 minutes during daytime (8 AM - 8 PM)
	err = scheduler.Start("*/5 8-20 * * *")
	if err != nil {
		log.Fatalf("Failed to start scheduler: %v", err)
	}
	defer scheduler.Stop()

	// 5. Initialize and Start Bot
	quizBot, err := bot.New(botToken, repos, llmClient, scheduler, planManager)
	if err != nil {
		log.Fatalf("Failed to initialize bot: %v", err)
	}

	go quizBot.Start()
	defer quizBot.Stop()

	// 6. Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down bot...")
}
