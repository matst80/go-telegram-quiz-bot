package quiz

import (
	"testing"

	"github.com/mats/telegram-quiz-bot/internal/llm"
	"github.com/mats/telegram-quiz-bot/internal/repository"
)

func TestSchedulerStart(t *testing.T) {
	s := NewScheduler(&repository.Repositories{}, &llm.Client{}, &PlanManager{})

	err := s.Start("0 0 * * *")
	if err != nil {
		t.Errorf("Failed to start scheduler with 5-field spec: %v", err)
	}
	s.Stop()
}
