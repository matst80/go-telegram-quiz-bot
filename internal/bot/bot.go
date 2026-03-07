package bot

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/mats/telegram-quiz-bot/internal/domain"
	"github.com/mats/telegram-quiz-bot/internal/llm"
	"github.com/mats/telegram-quiz-bot/internal/quiz"
	"github.com/mats/telegram-quiz-bot/internal/repository"
	"gopkg.in/telebot.v3"
)

type Bot struct {
	teleBot   *telebot.Bot
	repos     *repository.Repositories
	llmClient *llm.Client
	scheduler *quiz.Scheduler
	plan      *quiz.PlanManager
}

func New(token string, repos *repository.Repositories, llmClient *llm.Client, scheduler *quiz.Scheduler, planManager *quiz.PlanManager) (*Bot, error) {
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
		repos:     repos,
		llmClient: llmClient,
		scheduler: scheduler,
		plan:      planManager,
	}

	appBot.registerHandlers()

	scheduler.SetOnBatch(appBot.BroadcastQuestion)

	return appBot, nil
}

func (b *Bot) registerHandlers() {
	b.teleBot.Handle("/start", b.handleStart)
	b.teleBot.Handle("/quiz", b.handleQuiz)
	b.teleBot.Handle("/leaderboard", b.handleLeaderboard)
	b.teleBot.Handle("/plan", b.handlePlan)
	b.teleBot.Handle("/nextstep", b.handleNextStep)

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

func (b *Bot) BroadcastQuestion(questions []domain.Question) {
	if len(questions) == 0 {
		return
	}

	ctx := context.Background()
	users, err := b.repos.Users.GetAllUsers(ctx)
	if err != nil {
		log.Printf("[Bot] Failed to get users for broadcast: %v", err)
		return
	}

	for _, user := range users {
		log.Printf("[Bot] Broadcasting first question of batch (size %d) to user %d", len(questions), user.TelegramID)

		q := questions[0]
		target := &telebot.User{ID: user.TelegramID}

		menu := &telebot.ReplyMarkup{}
		var rows []telebot.Row
		for _, opt := range q.Options {
			data := fmt.Sprintf("%d|%s", q.ID, opt)
			btn := menu.Data(opt, "ans", data)
			rows = append(rows, menu.Row(btn))
		}
		menu.Inline(rows...)

		quizObj, _ := b.repos.Quizzes.GetByID(ctx, q.QuizID)
		topic := "Unknown"
		if quizObj != nil {
			topic = escapeMarkdown(quizObj.Title)
		}
		msg := fmt.Sprintf("🔔 **New Quizzes Available!** 🔔\n\n📝 **Topic:** %s\n\n**%s**", topic, escapeMarkdown(q.Text))

		var err error
		if q.AudioFileID != "" {
			audio := &telebot.Voice{File: telebot.FromDisk(q.AudioFileID), Caption: msg}
			_, err = b.teleBot.Send(target, audio, menu, telebot.ModeMarkdown)
		} else {
			_, err = b.teleBot.Send(target, msg, menu, telebot.ModeMarkdown)
		}
		if err != nil {
			log.Printf("[Bot] Failed to send broadcast to user %d: %v", user.TelegramID, err)
		}
	}
}
