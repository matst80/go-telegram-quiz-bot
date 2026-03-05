package bot

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/mats/telegram-quiz-bot/internal/db"
	"github.com/mats/telegram-quiz-bot/internal/quiz"
	"gopkg.in/telebot.v3"
)

func (b *Bot) handleStart(c telebot.Context) error {
	defer func() {
		// Attempt to register user in DB when they start the bot
		if err := b.db.RegisterUser(c.Sender().ID, c.Sender().Username); err != nil {
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
	users, err := b.db.GetTopUsers(10)
	if err != nil {
		return c.Send("Failed to fetch leaderboard.")
	}

	if len(users) == 0 {
		return c.Send("No one has scored yet! Be the first!")
	}

	msg := "🏆 **Leaderboard** 🏆\n\n"
	for i, u := range users {
		name := u.Username
		if name == "" {
			name = fmt.Sprintf("User%d", u.TelegramID)
		}
		msg += fmt.Sprintf("%d. %s - %d pts\n", i+1, name, u.Score)
	}

	return c.Send(msg, telebot.ModeMarkdown)
}

func (b *Bot) handlePlan(c telebot.Context) error {
	topic := b.plan.GetCurrentTopic()
	count := b.plan.GetCurrentQuizzesGenerated()
	idx := b.plan.GetCurrentStepIndex()

	msg := fmt.Sprintf("📚 **Current Learning Plan** 📚\n\n"+
		"**Step:** %d\n"+
		"**Topic:** %s\n"+
		"**Progress:** %d/%d quizzes generated for this topic.\n\n"+
		"Use /nextstep to skip to the next topic.",
		idx+1, topic, count, quiz.QuizzesPerStep)

	return c.Send(msg, telebot.ModeMarkdown)
}

func (b *Bot) handleNextStep(c telebot.Context) error {
	b.plan.AdvancePlan()

	topic := b.plan.GetCurrentTopic()
	msg := fmt.Sprintf("⏩ **Advanced to Next Step** ⏩\n\n"+
		"The new topic is: **%s**\n\n"+
		"The next quiz generated will be about this topic.", topic)

	return c.Send(msg, telebot.ModeMarkdown)
}

func (b *Bot) handleQuiz(c telebot.Context) error {
	userID := c.Sender().ID
	topic := b.plan.GetCurrentTopic()
	log.Printf("[Bot] User %d requested quiz for topic: %s", userID, topic)

	// NEW: Ensure lesson is shown before the first quiz of a topic
	if err := b.ensureLessonShown(c, userID, topic); err != nil {
		log.Printf("[Bot] Failed to ensure lesson shown for user %d: %v", userID, err)
	}

	q, err := b.db.GetNextUnansweredQuiz(userID, topic)
	if err != nil {
		log.Printf("[Bot] Error fetching unanswered quiz for user %d: %v", userID, err)
		return c.Send("Oops, something went wrong fetching the quiz.")
	}

	if q == nil {
		log.Printf("[Bot] No unanswered quizzes for user %d in topic '%s'. Triggering seed.", userID, topic)
		b.scheduler.EnsurePoolSufficient(userID)
		return c.Send("I'm preparing more questions for you on **"+topic+"**. Please try again in a few seconds!", telebot.ModeMarkdown)
	}

	log.Printf("[Bot] Serving quiz ID %d to user %d", q.ID, userID)

	// Make sure the user is registered
	if err := b.db.RegisterUser(c.Sender().ID, c.Sender().Username); err != nil {
		log.Printf("[Bot] Failed to register user %d: %v", userID, err)
	}

	// Ensure we have enough questions seeded for the future
	b.scheduler.EnsurePoolSufficient(userID)

	menu := &telebot.ReplyMarkup{}
	var rows []telebot.Row

	// Create buttons for each option.
	for _, opt := range q.Options {
		// btn := menu.Data(opt, "ans", fmt.Sprintf("%d|%s", q.ID, opt))
		// We use a simpler data format to avoid telebot's limit and parsing issues
		data := fmt.Sprintf("%d|%s", q.ID, opt)
		btn := menu.Data(opt, "ans", data)
		rows = append(rows, menu.Row(btn))
	}

	menu.Inline(rows...)

	msg := fmt.Sprintf("📝 **Topic:** %s\n\n**%s**", q.Topic, q.Question)
	return c.Send(msg, menu, telebot.ModeMarkdown)
}

func (b *Bot) handleCallback(c telebot.Context) error {
	raw := c.Callback().Data
	userID := c.Sender().ID
	log.Printf("[Bot] Callback from user %d: %s", userID, raw)

	// telebot often prepends the unique key (e.g., "ans") and sometimes a separator like \f.
	// We skip all non-digit characters at the start to find the Quiz ID.
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

	quizID, err := strconv.Atoi(parts[0])
	if err != nil {
		log.Printf("[Bot] Failed to parse quiz ID from '%s' (raw: %s) for user %d: %v", parts[0], raw, userID, err)
		return c.Respond(&telebot.CallbackResponse{Text: "Invalid quiz ID.", ShowAlert: true})
	}
	selectedOption := parts[1]
	log.Printf("[Bot] User %d selected '%s' for quiz %d", userID, selectedOption, quizID)

	// Fetch the specific quiz to verify answer
	var q db.Quiz
	err = b.db.QueryRow("SELECT correct_answer FROM quizzes WHERE id = ?", quizID).Scan(&q.CorrectAnswer)
	if err != nil {
		log.Printf("[Bot] Quiz %d not found in DB for user %d: %v", quizID, userID, err)
		return c.Respond(&telebot.CallbackResponse{Text: "Quiz not found or expired.", ShowAlert: true})
	}

	isCorrect := (selectedOption == q.CorrectAnswer)

	// Record answer in db
	err = b.db.RecordAnswer(quizID, c.Sender().ID, isCorrect)
	if err != nil {
		// Likely already answered
		return c.Respond(&telebot.CallbackResponse{Text: "You already answered this quiz!", ShowAlert: true})
	}

	// Notify user
	msg := fmt.Sprintf("❌ Incorrect. The correct answer was: %s", q.CorrectAnswer)
	if isCorrect {
		msg = "✅ Correct! +1 Point!"
	}

	// Edit the message to remove the keyboard and show the result
	newText := fmt.Sprintf("%s\n\n%s", c.Message().Text, msg)
	c.Bot().Edit(c.Message(), newText)

	// Ensure user is caught up in the db if they weren't somehow
	b.db.RegisterUser(c.Sender().ID, c.Sender().Username)

	// After answering, check if we need to seed more questions for this user
	b.scheduler.EnsurePoolSufficient(c.Sender().ID)

	// AUTOMATICALLY SEND NEXT QUESTION
	go func() {
		// Small delay for better UX
		time.Sleep(1 * time.Second)

		topic := b.plan.GetCurrentTopic()
		// NEW: Ensure lesson is shown before the next quiz if topic changed
		if err := b.ensureLessonShown(c, userID, topic); err != nil {
			log.Printf("[Bot] Failed to ensure lesson shown for user %d in callback: %v", userID, err)
		}

		qNext, err := b.db.GetNextUnansweredQuiz(userID, topic)
		if err == nil && qNext != nil {
			log.Printf("[Bot] Auto-serving next quiz ID %d to user %d", qNext.ID, userID)

			menu := &telebot.ReplyMarkup{}
			var rows []telebot.Row
			for _, opt := range qNext.Options {
				data := fmt.Sprintf("%d|%s", qNext.ID, opt)
				btn := menu.Data(opt, "ans", data)
				rows = append(rows, menu.Row(btn))
			}
			menu.Inline(rows...)

			msg := fmt.Sprintf("📝 **Topic:** %s\n\n**%s**", qNext.Topic, qNext.Question)
			b.teleBot.Send(c.Sender(), msg, menu, telebot.ModeMarkdown)
		}
	}()

	return c.Respond(&telebot.CallbackResponse{Text: msg, ShowAlert: true})
}

// ensureLessonShown checks if a user has seen the lesson for a topic and sends it if not.
func (b *Bot) ensureLessonShown(c telebot.Context, userID int64, topic string) error {
	seen, err := b.db.HasSeenLesson(userID, topic)
	if err != nil {
		return err
	}
	if seen {
		return nil
	}

	lesson, err := b.db.GetLesson(topic)
	if err != nil {
		return err
	}

	if lesson != "" {
		msg := fmt.Sprintf("📖 **Lesson: %s** 📖\n\n%s", topic, lesson)
		if _, err := c.Bot().Send(c.Sender(), msg, telebot.ModeMarkdown); err != nil {
			return err
		}
		// Small delay to let the user read before the quiz pops up
		time.Sleep(2 * time.Second)
	}

	return b.db.MarkLessonSeen(userID, topic)
}
