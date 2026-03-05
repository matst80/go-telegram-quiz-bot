package quiz

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/mats/telegram-quiz-bot/internal/domain"
	"github.com/mats/telegram-quiz-bot/internal/repository"
)

// QuizzesPerStep determines how many questions to generate for each topic (quiz)
// before automatically advancing to the next step (quiz).
const QuizzesPerStep = 5

// PlanManager handles loading the learning plan and tracking progress.
type PlanManager struct {
	repos *repository.Repositories
}

// NewPlanManager creates a manager that reads directly from the database segments/quizzes.
func NewPlanManager(repos *repository.Repositories) *PlanManager {
	return &PlanManager{
		repos: repos,
	}
}

// GetCurrentQuiz retrieves the currently active quiz from the database based on settings.
// This replaces reading from LEARNINGPLAN.md.
func (pm *PlanManager) GetCurrentQuiz(ctx context.Context) (*domain.Quiz, error) {
	// 1. Get current step index from settings
	valStr, err := pm.repos.Settings.Get(ctx, "current_learning_step")
	idx := 0
	if err == nil && valStr != "" {
		idx, _ = strconv.Atoi(valStr)
	}

	// 2. Fetch all segments, ordered by index
	segments, err := pm.repos.Segments.GetAll(ctx)
	if err != nil || len(segments) == 0 {
		return nil, fmt.Errorf("no segments found in database")
	}

	// 3. To find the Nth quiz overall, we iterate through segments
	currentQuizIndex := 0
	for _, seg := range segments {
		quizzes, err := pm.repos.Quizzes.GetBySegmentID(ctx, seg.ID)
		if err != nil {
			continue
		}

		for _, q := range quizzes {
			if currentQuizIndex == idx {
				return &q, nil
			}
			currentQuizIndex++
		}
	}

	// If we've exhausted all quizzes, wrap around back to 0
	if idx > 0 && currentQuizIndex > 0 {
		log.Println("Wrapped around learning plan to the beginning.")
		pm.repos.Settings.Set(ctx, "current_learning_step", "0")
		pm.repos.Settings.Set(ctx, "current_step_quizzes_generated", "0")
		return pm.GetCurrentQuiz(ctx)
	}

	return nil, fmt.Errorf("no quizzes found in any segment")
}

// GetCurrentQuestionsGenerated returns how many questions have been generated for the current quiz.
func (pm *PlanManager) GetCurrentQuestionsGenerated(ctx context.Context) int {
	valStr, err := pm.repos.Settings.Get(ctx, "current_step_quizzes_generated")
	if err != nil || valStr == "" {
		return 0
	}
	count, err := strconv.Atoi(valStr)
	if err != nil {
		return 0
	}
	return count
}

// RecordQuestionGenerated increments the counter for the current step.
// If it reaches the threshold, it advances to the next step and resets the counter.
func (pm *PlanManager) RecordQuestionGenerated(ctx context.Context) {
	count := pm.GetCurrentQuestionsGenerated(ctx)
	count++

	if count >= QuizzesPerStep {
		// Advance to next step
		valStr, _ := pm.repos.Settings.Get(ctx, "current_learning_step")
		idx := 0
		if valStr != "" {
			idx, _ = strconv.Atoi(valStr)
		}
		idx++
		if err := pm.repos.Settings.Set(ctx, "current_learning_step", strconv.Itoa(idx)); err != nil {
			log.Printf("Failed to increment learning step: %v", err)
		}

		// Reset counter
		if err := pm.repos.Settings.Set(ctx, "current_step_quizzes_generated", "0"); err != nil {
			log.Printf("Failed to reset quiz counter: %v", err)
		}

		log.Printf("Learning plan advanced to step %d. Counter reset.", idx)
	} else {
		// Just update counter
		if err := pm.repos.Settings.Set(ctx, "current_step_quizzes_generated", strconv.Itoa(count)); err != nil {
			log.Printf("Failed to update quiz counter: %v", err)
		}
		log.Printf("Question generated for current step. Counter: %d/%d", count, QuizzesPerStep)
	}
}

// AdvancePlan manually moves the plan to the next step, resetting the counter.
func (pm *PlanManager) AdvancePlan(ctx context.Context) {
	valStr, _ := pm.repos.Settings.Get(ctx, "current_learning_step")
	idx := 0
	if valStr != "" {
		idx, _ = strconv.Atoi(valStr)
	}
	idx++
	pm.repos.Settings.Set(ctx, "current_learning_step", strconv.Itoa(idx))
	pm.repos.Settings.Set(ctx, "current_step_quizzes_generated", "0")
}
