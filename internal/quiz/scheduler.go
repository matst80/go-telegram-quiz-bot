package quiz

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/mats/telegram-quiz-bot/internal/domain"
	"github.com/mats/telegram-quiz-bot/internal/llm"
	"github.com/mats/telegram-quiz-bot/internal/repository"
	"github.com/mats/telegram-quiz-bot/internal/tts"
	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	cron    *cron.Cron
	repos   *repository.Repositories
	llm     *llm.Client
	plan    *PlanManager
	tts     tts.Service
	cronJob cron.EntryID
	OnBatch func([]domain.Question)

	mu          sync.Mutex
	activeSeeds map[int64]bool
}

func NewScheduler(repos *repository.Repositories, llmClient *llm.Client, planManager *PlanManager, ttsService tts.Service) *Scheduler {
	return &Scheduler{
		cron:        cron.New(),
		repos:       repos,
		llm:         llmClient,
		plan:        planManager,
		tts:         ttsService,
		activeSeeds: make(map[int64]bool),
	}
}

func (s *Scheduler) SetOnBatch(fn func([]domain.Question)) {
	s.OnBatch = fn
}

func (s *Scheduler) Start(spec string) error {
	id, err := s.cron.AddFunc(spec, func() {
		s.GenerateAndSaveQuestion()
	})
	if err != nil {
		return err
	}
	s.cronJob = id
	s.cron.Start()
	log.Printf("Scheduler started with spec: %s", spec)
	return nil
}

func (s *Scheduler) Stop() {
	s.cron.Stop()
}

func (s *Scheduler) GenerateAndSaveQuestion() {
	s.GenerateAndBroadcastBatch(5)
}

func (s *Scheduler) GenerateAndBroadcastBatch(count int) {
	ctx := context.Background()
	currentQuiz, err := s.plan.GetCurrentQuiz(ctx)
	if err != nil {
		log.Printf("Scheduler: Failed to get current quiz from plan: %v", err)
		return
	}

	topic := currentQuiz.Title
	log.Printf("Generating %d new questions for quiz topic '%s' via LLM...", count, topic)

	exclude, _ := s.repos.Questions.GetRecentForQuiz(ctx, currentQuiz.ID, 10)

	questions, err := s.llm.GenerateSpanishQuestions(topic, exclude, count)
	if err != nil {
		log.Printf("Error generating questions: %v", err)
		return
	}

	var savedQuestions []domain.Question
	for _, q := range questions {
		q.QuizID = currentQuiz.ID
		q.IsActive = true

		// Shuffling options
		rand.Seed(time.Now().UnixNano())
		rand.Shuffle(len(q.Options), func(i, j int) {
			q.Options[i], q.Options[j] = q.Options[j], q.Options[i]
		})

		err := s.repos.Questions.Create(ctx, &q)
		if err != nil {
			log.Printf("Error saving question: %v", err)
			continue
		}

		// Generate TTS audio
		if s.tts != nil && q.TTSPhrase != "" {
			audioPath := fmt.Sprintf("storage/audio/q_%d.ogg", q.ID)
			err = s.tts.GenerateSpeech(q.TTSPhrase, audioPath)
			if err != nil {
				log.Printf("[Scheduler] Error generating TTS for question %d: %v", q.ID, err)
			} else {
				q.AudioFileID = audioPath
				// Update question with audio file ID
				if updateErr := s.repos.Questions.Update(ctx, &q); updateErr != nil {
					log.Printf("[Scheduler] Error updating question %d with audio file: %v", q.ID, updateErr)
				}
			}
		}

		savedQuestions = append(savedQuestions, q)
		s.plan.RecordQuestionGenerated(ctx)
	}

	if len(savedQuestions) > 0 && s.OnBatch != nil {
		log.Printf("Broadcasting %d new questions", len(savedQuestions))
		s.OnBatch(savedQuestions)
	}
}

func (s *Scheduler) EnsurePoolSufficient(telegramID int64) {
	ctx := context.Background()
	currentQuiz, err := s.plan.GetCurrentQuiz(ctx)
	if err != nil {
		log.Printf("[Scheduler] Error getting current quiz for pool sufficiency: %v", err)
		return
	}

	count, err := s.repos.Questions.GetUnansweredCount(ctx, telegramID, currentQuiz.ID)
	if err != nil {
		log.Printf("[Scheduler] Error checking pool sufficiency for user %d: %v", telegramID, err)
		return
	}

	if count < 2 {
		s.mu.Lock()
		if s.activeSeeds[telegramID] {
			s.mu.Unlock()
			return
		}
		s.activeSeeds[telegramID] = true
		s.mu.Unlock()

		log.Printf("[Scheduler] Pool low for user %d (quiz: %s, count: %d). Seeding to buffer of 2...", telegramID, currentQuiz.Title, count)
		go func() {
			defer func() {
				s.mu.Lock()
				delete(s.activeSeeds, telegramID)
				s.mu.Unlock()
			}()

			for i := count; i < 2; i++ {
				log.Printf("[Scheduler] Background generation %d/2 for user %d", i+1, telegramID)
				s.GenerateAndSaveQuestion()
				if i < 1 {
					time.Sleep(1 * time.Second)
				}
			}
			log.Printf("[Scheduler] Background seeding finished for user %d", telegramID)
		}()
	}
}
