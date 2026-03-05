package bot

import (
	"fmt"
	"log"
	"time"

	"github.com/mats/telegram-quiz-bot/internal/db"
	"github.com/mats/telegram-quiz-bot/internal/llm"
	"github.com/mats/telegram-quiz-bot/internal/quiz"
	"gopkg.in/telebot.v3"
)

type Bot struct {
	teleBot   *telebot.Bot
	db        *db.DB
	llmClient *llm.Client
	scheduler *quiz.Scheduler
	plan      *quiz.PlanManager
}

func New(token string, database *db.DB, llmClient *llm.Client, scheduler *quiz.Scheduler, planManager *quiz.PlanManager) (*Bot, error) {
	pref := telebot.Settings{
		Token:  token,
		Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
	}

	b, err := telebot.NewBot(pref)
	if err != nil {
		return nil, err
	}

	appBot := &Bot{
		teleBot:   b,
		db:        database,
		llmClient: llmClient,
		scheduler: scheduler,
		plan:      planManager,
	}

	appBot.registerHandlers()

	// Set the scheduler callback for broadcasting
	scheduler.SetOnBatch(appBot.BroadcastQuiz)

	return appBot, nil
}

func (b *Bot) registerHandlers() {
	b.teleBot.Handle("/start", b.handleStart)
	b.teleBot.Handle("/quiz", b.handleQuiz)
	b.teleBot.Handle("/leaderboard", b.handleLeaderboard)
	b.teleBot.Handle("/plan", b.handlePlan)
	b.teleBot.Handle("/nextstep", b.handleNextStep)

	// Handle all callback queries from inline buttons
	b.teleBot.Handle(telebot.OnCallback, b.handleCallback)
}

func (b *Bot) Start() {
	log.Println("Starting Telegram Bot...")
	b.teleBot.Start()
}

func (b *Bot) Stop() {
	log.Println("Stopping Telegram Bot...")
	b.teleBot.Stop()
}

func (b *Bot) BroadcastQuiz(quizzes []db.Quiz) {
	users, err := b.db.GetAllUsers()
	if err != nil {
		log.Printf("[Bot] Failed to get users for broadcast: %v", err)
		return
	}

	for _, user := range users {
		log.Printf("[Bot] Broadcasting batch of %d quizzes to user %d", len(quizzes), user.TelegramID)
		// We just send the first one, the sequence will be handled by handleCallback
		// Actually, the user wants "present the user with a sequence of questions"
		// If we send them all at once, it might be messy.
		// If we send the first one, they can start the sequence.
		q := quizzes[0]
		target := &telebot.User{ID: user.TelegramID}

		menu := &telebot.ReplyMarkup{}
		var rows []telebot.Row
		for _, opt := range q.Options {
			data := fmt.Sprintf("%d|%s", q.ID, opt)
			btn := menu.Data(opt, "ans", data)
			rows = append(rows, menu.Row(btn))
		}
		menu.Inline(rows...)

		msg := fmt.Sprintf("🔔 **New Quizzes Available!** 🔔\n\n📝 **Topic:** %s\n\n**%s**", q.Topic, q.Question)
		_, err := b.teleBot.Send(target, msg, menu, telebot.ModeMarkdown)
		if err != nil {
			log.Printf("[Bot] Failed to send broadcast to user %d: %v", user.TelegramID, err)
		}
	}
}
