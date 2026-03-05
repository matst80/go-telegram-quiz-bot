package quiz

import (
	"testing"

	"github.com/mats/telegram-quiz-bot/internal/db"
	"github.com/mats/telegram-quiz-bot/internal/llm"
)

func TestSchedulerStart(t *testing.T) {
	// We don't need a real DB or LLM for just testing the cron parser
	s := NewScheduler(&db.DB{}, &llm.Client{}, &PlanManager{})

	err := s.Start("0 0 * * *")
	if err != nil {
		t.Errorf("Failed to start scheduler with 5-field spec: %v", err)
	}
	s.Stop()
}
