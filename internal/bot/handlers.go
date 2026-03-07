package bot

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/mats/telegram-quiz-bot/internal/domain"
	"github.com/mats/telegram-quiz-bot/internal/quiz"
	"gopkg.in/telebot.v3"
)

func (b *Bot) handleStart(c telebot.Context) error {
	ctx := context.Background()
	defer func() {
		if err := b.repos.Users.RegisterUser(ctx, c.Sender().ID, c.Sender().Username); err != nil {
			log.Printf("Failed to register user %d: %v", c.Sender().ID, err)
		}
	}()

	msg := "Bienvenido! I am your Spanish Quiz Bot.\n\n" +
		"Use /quiz to see the current active question.\n" +
		"Use /leaderboard to see the top scorers.\n\n" +
		"A new quiz is generated automatically every day!"
	return c.Send(msg)
}

func (b *Bot) handleLeaderboard(c telebot.Context) error {
	ctx := context.Background()
	users, err := b.repos.Users.GetTopUsers(ctx, 10)
	if err != nil {
		return c.Send("Failed to fetch leaderboard.")
	}

	if len(users) == 0 {
		return c.Send("No one has scored yet! Be the first!")
	}

	msg := "🏆 **Leaderboard** 🏆\n\n"
	for i, u := range users {
		name := escapeMarkdown(u.Username)
		if name == "" {
			name = fmt.Sprintf("User%d", u.TelegramID)
		}
		msg += fmt.Sprintf("%d. %s - %d pts\n", i+1, name, u.Score)
	}

	return c.Send(msg, telebot.ModeMarkdown)
}

func (b *Bot) handlePlan(c telebot.Context) error {
	ctx := context.Background()
	currentQuiz, err := b.plan.GetCurrentQuiz(ctx)
	if err != nil {
		return c.Send("Error fetching learning plan.")
	}

	count := b.plan.GetCurrentQuestionsGenerated(ctx)

	msg := fmt.Sprintf("📚 **Current Learning Plan** 📚\n\n"+
		"**Topic:** %s\n"+
		"**Description:** %s\n"+
		"**Progress:** %d/%d questions generated for this topic.\n\n"+
		"Use /nextstep to skip to the next topic.",
		escapeMarkdown(currentQuiz.Title), escapeMarkdown(currentQuiz.Description), count, quiz.QuizzesPerStep)

	return c.Send(msg, telebot.ModeMarkdown)
}

func (b *Bot) handleNextStep(c telebot.Context) error {
	ctx := context.Background()
	b.plan.AdvancePlan(ctx)

	currentQuiz, _ := b.plan.GetCurrentQuiz(ctx)
	topic := "Unknown"
	if currentQuiz != nil {
		topic = currentQuiz.Title
	}

	msg := fmt.Sprintf("⏩ **Advanced to Next Step** ⏩\n\n"+
		"The new topic is: **%s**\n\n"+
		"The next quiz generated will be about this topic.", escapeMarkdown(topic))

	return c.Send(msg, telebot.ModeMarkdown)
}

func (b *Bot) handleQuiz(c telebot.Context) error {
	ctx := context.Background()
	userID := c.Sender().ID

	currentQuiz, err := b.plan.GetCurrentQuiz(ctx)
	if err != nil {
		return c.Send("Error finding current topic.")
	}

	topic := currentQuiz.Title
	log.Printf("[Bot] User %d requested quiz for topic: %s", userID, topic)

	// ensure lesson could be reimplemented later, omitting for simplicity of refactoring
	if err := b.ensureLessonShown(c, userID, currentQuiz); err != nil {
		log.Printf("[Bot] Failed to ensure lesson shown for user %d: %v", userID, err)
	}

	q, err := b.repos.Questions.GetNextUnanswered(ctx, userID, currentQuiz.ID)
	if err != nil {
		log.Printf("[Bot] Error fetching unanswered question for user %d: %v", userID, err)
		return c.Send("Oops, something went wrong fetching the quiz.")
	}

	if q == nil {
		log.Printf("[Bot] No unanswered questions for user %d in topic '%s'. Triggering seed.", userID, topic)
		b.scheduler.EnsurePoolSufficient(userID)
		return c.Send("I'm preparing more questions for you on **"+escapeMarkdown(topic)+"**. Please try again in a few seconds!", telebot.ModeMarkdown)
	}

	log.Printf("[Bot] Serving question ID %d to user %d", q.ID, userID)

	if err := b.repos.Users.RegisterUser(ctx, c.Sender().ID, c.Sender().Username); err != nil {
		log.Printf("[Bot] Failed to register user %d: %v", userID, err)
	}

	b.scheduler.EnsurePoolSufficient(userID)

	menu := &telebot.ReplyMarkup{}
	var rows []telebot.Row

	for _, opt := range q.Options {
		data := fmt.Sprintf("%d|%s", q.ID, opt)
		btn := menu.Data(opt, "ans", data)
		rows = append(rows, menu.Row(btn))
	}

	menu.Inline(rows...)

	msg := fmt.Sprintf("📝 **Topic:** %s\n\n**%s**", escapeMarkdown(topic), escapeMarkdown(q.Text))
	if q.AudioFileID != "" {
		audio := &telebot.Voice{File: telebot.FromDisk(q.AudioFileID), Caption: msg}
		return c.Send(audio, menu, telebot.ModeMarkdown)
	}
	return c.Send(msg, menu, telebot.ModeMarkdown)
}

func (b *Bot) handleCallback(c telebot.Context) error {
	ctx := context.Background()
	raw := c.Callback().Data
	userID := c.Sender().ID
	log.Printf("[Bot] Callback from user %d: %s", userID, raw)

	data := raw
	for len(data) > 0 && (data[0] < '0' || data[0] > '9') {
		data = data[1:]
	}
	log.Printf("[Bot] Cleaned callback data: %s", data)

	parts := strings.SplitN(data, "|", 2)
	if len(parts) < 2 {
		log.Printf("[Bot] Malformed callback data (no pipe) from user %d: %s", userID, raw)
		return c.Respond(&telebot.CallbackResponse{Text: "Invalid option format.", ShowAlert: true})
	}

	questionID, err := strconv.Atoi(parts[0])
	if err != nil {
		log.Printf("[Bot] Failed to parse question ID from '%s' (raw: %s) for user %d: %v", parts[0], raw, userID, err)
		return c.Respond(&telebot.CallbackResponse{Text: "Invalid quiz ID.", ShowAlert: true})
	}
	selectedOption := parts[1]
	log.Printf("[Bot] User %d selected '%s' for question %d", userID, selectedOption, questionID)

	q, err := b.repos.Questions.GetByID(ctx, questionID)
	if err != nil || q == nil {
		log.Printf("[Bot] Question %d not found in DB for user %d: %v", questionID, userID, err)
		return c.Respond(&telebot.CallbackResponse{Text: "Quiz not found or expired.", ShowAlert: true})
	}

	isCorrect := (selectedOption == q.CorrectAnswer)

	err = b.repos.Questions.RecordAnswer(ctx, questionID, c.Sender().ID, isCorrect)
	if err != nil {
		return c.Respond(&telebot.CallbackResponse{Text: "You already answered this quiz!", ShowAlert: true})
	}

	msg := fmt.Sprintf("❌ Incorrect. The correct answer was: %s", escapeMarkdown(q.CorrectAnswer))
	if isCorrect {
		msg = "✅ Correct! +1 Point!"
	}

	newText := fmt.Sprintf("%s\n\n%s", c.Message().Text, msg)
	c.Bot().Edit(c.Message(), newText)

	b.repos.Users.RegisterUser(ctx, c.Sender().ID, c.Sender().Username)
	b.scheduler.EnsurePoolSufficient(c.Sender().ID)

	go func() {
		time.Sleep(1 * time.Second)

		currentQuiz, _ := b.plan.GetCurrentQuiz(context.Background())
		if currentQuiz == nil {
			return
		}
		topic := currentQuiz.Title

		if err := b.ensureLessonShown(c, userID, currentQuiz); err != nil {
			log.Printf("[Bot] Failed to ensure lesson shown for user %d in callback: %v", userID, err)
		}

		qNext, err := b.repos.Questions.GetNextUnanswered(context.Background(), userID, currentQuiz.ID)
		if err == nil && qNext != nil {
			log.Printf("[Bot] Auto-serving next question ID %d to user %d", qNext.ID, userID)

			menu := &telebot.ReplyMarkup{}
			var rows []telebot.Row
			for _, opt := range qNext.Options {
				data := fmt.Sprintf("%d|%s", qNext.ID, opt)
				btn := menu.Data(opt, "ans", data)
				rows = append(rows, menu.Row(btn))
			}
			menu.Inline(rows...)

			msg := fmt.Sprintf("📝 **Topic:** %s\n\n**%s**", escapeMarkdown(topic), escapeMarkdown(qNext.Text))
			if qNext.AudioFileID != "" {
				audio := &telebot.Voice{File: telebot.FromDisk(qNext.AudioFileID), Caption: msg}
				b.teleBot.Send(c.Sender(), audio, menu, telebot.ModeMarkdown)
			} else {
				b.teleBot.Send(c.Sender(), msg, menu, telebot.ModeMarkdown)
			}
		}
	}()

	return c.Respond(&telebot.CallbackResponse{Text: msg, ShowAlert: true})
}

// ensureLessonShown checks if a user has seen the lesson for a topic and sends it if not.
func (b *Bot) ensureLessonShown(c telebot.Context, userID int64, currentQuiz *domain.Quiz) error {
	ctx := context.Background()
	key := fmt.Sprintf("lesson_seen_%d_%d", userID, currentQuiz.ID)
	seenStr, err := b.repos.Settings.Get(ctx, key)
	if err != nil {
		return err
	}
	if seenStr == "true" {
		return nil
	}

	lesson := currentQuiz.Description
	if lesson != "" {
		msg := fmt.Sprintf("📖 **Lesson: %s** 📖\n\n%s", escapeMarkdown(currentQuiz.Title), escapeMarkdown(lesson))
		if _, err := c.Bot().Send(c.Sender(), msg, telebot.ModeMarkdown); err != nil {
			return err
		}
		// Small delay to let the user read before the quiz pops up
		time.Sleep(2 * time.Second)
	}

	return b.repos.Settings.Set(ctx, key, "true")
}
