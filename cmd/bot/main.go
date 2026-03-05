package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/mats/telegram-quiz-bot/internal/bot"
	"github.com/mats/telegram-quiz-bot/internal/db"
	"github.com/mats/telegram-quiz-bot/internal/llm"
	"github.com/mats/telegram-quiz-bot/internal/quiz"
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

	// 2. Initialize Database
	database, err := db.New(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()
	log.Println("Database initialized.")

	// 3. Initialize LLM Client
	llmClient := llm.NewClient(ollamaURL, modelName)
	log.Printf("LLM Client initialized (Model: %s at %s)", modelName, ollamaURL)

	// 4. Initialize Scheduler and Plan
	planPath := os.Getenv("PLAN_PATH")
	if planPath == "" {
		planPath = "LEARNINGPLAN.md"
	}
	planManager := quiz.NewPlanManager(database, planPath)
	scheduler := quiz.NewScheduler(database, llmClient, planManager)

	// Schedule a quiz generation every 5 minutes during daytime (8 AM - 8 PM)
	// "*/5 8-20 * * *" = every 5 minutes from 8:00 to 20:59
	err = scheduler.Start("*/5 8-20 * * *")
	if err != nil {
		log.Fatalf("Failed to start scheduler: %v", err)
	}
	defer scheduler.Stop()

	// Check if there is an active quiz, if not, generate one immediately so it works on first run
	activeQuiz, _ := database.GetActiveQuiz()
	if activeQuiz == nil {
		log.Println("No active quiz found on startup. Generating a batch of 5 questions now...")
		go scheduler.GenerateAndBroadcastBatch(5)
	}

	// 5. Initialize and Start Bot
	quizBot, err := bot.New(botToken, database, llmClient, scheduler, planManager)
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
