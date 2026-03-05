package quiz

import (
	"log"
	"math/rand"
	"time"

	"github.com/mats/telegram-quiz-bot/internal/db"
	"github.com/mats/telegram-quiz-bot/internal/llm"
	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	cron    *cron.Cron
	db      *db.DB
	llm     *llm.Client
	plan    *PlanManager
	cronJob cron.EntryID
	OnBatch func([]db.Quiz)
}

type Broadcaster interface {
	BroadcastQuiz(quizzes []db.Quiz)
}

func NewScheduler(database *db.DB, llmClient *llm.Client, planManager *PlanManager) *Scheduler {
	// For demonstration, we'll run it every hour (or we can make it every minute for testing)
	// We'll configure it to every hour natively, but we can trigger it manually on startup.
	return &Scheduler{
		cron: cron.New(), // using standard 5-field cron spec (min, hour, dom, month, dow)
		db:   database,
		llm:  llmClient,
		plan: planManager,
	}
}

func (s *Scheduler) SetOnBatch(fn func([]db.Quiz)) {
	s.OnBatch = fn
}

// Start begins the cron scheduler
func (s *Scheduler) Start(spec string) error {
	id, err := s.cron.AddFunc(spec, func() {
		s.GenerateAndSaveQuiz()
	})
	if err != nil {
		return err
	}
	s.cronJob = id
	s.cron.Start()
	log.Printf("Scheduler started with spec: %s", spec)
	return nil
}

// Stop gracefully stops the scheduler
func (s *Scheduler) Stop() {
	s.cron.Stop()
}

// GenerateAndSaveQuiz is the core logic that the cron job runs
func (s *Scheduler) GenerateAndSaveQuiz() {
	s.GenerateAndBroadcastBatch(5)
}

// GenerateAndBroadcastBatch generates a batch of quizzes and notifies subscribers
func (s *Scheduler) GenerateAndBroadcastBatch(count int) {
	topic := s.plan.GetCurrentTopic()
	log.Printf("Generating %d new quizzes for topic '%s' via LLM...", count, topic)

	// Fetch recent questions to avoid duplicates
	exclude, _ := s.db.GetRecentQuestionsForTopic(topic, 10)

	quizzes, err := s.llm.GenerateSpanishQuizzes(topic, exclude, count)
	if err != nil {
		log.Printf("Error generating quizzes: %v", err)
		return
	}

	var savedQuizzes []db.Quiz
	for _, q := range quizzes {
		// Shuffling options
		rand.Seed(time.Now().UnixNano())
		rand.Shuffle(len(q.Options), func(i, j int) {
			q.Options[i], q.Options[j] = q.Options[j], q.Options[i]
		})

		id, err := s.db.SaveQuiz(q)
		if err != nil {
			log.Printf("Error saving quiz: %v", err)
			continue
		}
		q.ID = id
		savedQuizzes = append(savedQuizzes, q)
		s.plan.RecordQuizGenerated()
	}

	if len(savedQuizzes) > 0 && s.OnBatch != nil {
		log.Printf("Broadcasting %d new quizzes", len(savedQuizzes))
		s.OnBatch(savedQuizzes)
	}
}

// EnsurePoolSufficient checks if a user has enough unanswered quizzes for the current topic.
// If not, it triggers generation of more quizzes until a buffer of 2 is reached.
func (s *Scheduler) EnsurePoolSufficient(telegramID int64) {
	topic := s.plan.GetCurrentTopic()
	count, err := s.db.GetUnansweredCount(telegramID, topic)
	if err != nil {
		log.Printf("[Scheduler] Error checking pool sufficiency for user %d: %v", telegramID, err)
		return
	}

	if count < 2 {
		log.Printf("[Scheduler] Pool low for user %d (topic: %s, count: %d). Seeding to buffer of 2...", telegramID, topic, count)
		go func() {
			for i := count; i < 2; i++ {
				log.Printf("[Scheduler] Background generation %d/2 for user %d", i+1, telegramID)
				s.GenerateAndSaveQuiz()
				// Small delay between generations if seeding multiple
				if i < 1 {
					time.Sleep(1 * time.Second)
				}
			}
			log.Printf("[Scheduler] Background seeding finished for user %d", telegramID)
		}()
	}
}
