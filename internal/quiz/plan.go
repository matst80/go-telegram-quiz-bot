package quiz

import (
	"bufio"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/mats/telegram-quiz-bot/internal/db"
)

// QuizzesPerStep determines how many quizzes to generate for each topic
// before automatically advancing to the next step.
const QuizzesPerStep = 5

// PlanManager handles loading the learning plan and tracking progress.
type PlanManager struct {
	db     *db.DB
	topics []string
}

// NewPlanManager creates a manager and loads topics from the provided file.
func NewPlanManager(db *db.DB, filePath string) *PlanManager {
	pm := &PlanManager{
		db: db,
	}
	pm.loadPlan(filePath)
	return pm
}

func (pm *PlanManager) loadPlan(filePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		log.Printf("Warning: Could not open learning plan at %s: %v. Using fallback plan.", filePath, err)
		pm.topics = []string{"Basic Spanish Vocabulary"} // fallback
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			pm.topics = append(pm.topics, line)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading learning plan: %v", err)
	}

	if len(pm.topics) == 0 {
		pm.topics = []string{"Basic Spanish Vocabulary"}
	}
}

// GetCurrentStepIndex returns the zero-based index of the current learning step.
func (pm *PlanManager) GetCurrentStepIndex() int {
	valStr, err := pm.db.GetSetting("current_learning_step")
	if err != nil || valStr == "" {
		return 0
	}
	idx, err := strconv.Atoi(valStr)
	if err != nil {
		return 0
	}
	return idx
}

// GetCurrentTopic returns the topic for the current step.
func (pm *PlanManager) GetCurrentTopic() string {
	idx := pm.GetCurrentStepIndex()
	// Wrap around if we reach the end of the plan
	if len(pm.topics) == 0 {
		return "Basic Spanish Vocabulary"
	}
	safeIdx := idx % len(pm.topics)
	return pm.topics[safeIdx]
}

// GetCurrentQuizzesGenerated returns how many quizzes have been generated for the current step.
func (pm *PlanManager) GetCurrentQuizzesGenerated() int {
	valStr, err := pm.db.GetSetting("current_step_quizzes_generated")
	if err != nil || valStr == "" {
		return 0
	}
	count, err := strconv.Atoi(valStr)
	if err != nil {
		return 0
	}
	return count
}

// RecordQuizGenerated increments the counter for the current step.
// If it reaches the threshold, it advances to the next step and resets the counter.
func (pm *PlanManager) RecordQuizGenerated() {
	count := pm.GetCurrentQuizzesGenerated()
	count++

	if count >= QuizzesPerStep {
		// Advance to next step
		idx := pm.GetCurrentStepIndex()
		idx++
		if err := pm.db.SetSetting("current_learning_step", strconv.Itoa(idx)); err != nil {
			log.Printf("Failed to increment learning step: %v", err)
		}

		// Reset counter
		if err := pm.db.SetSetting("current_step_quizzes_generated", "0"); err != nil {
			log.Printf("Failed to reset quiz counter: %v", err)
		}

		log.Printf("Learning plan advanced to step %d. Counter reset.", idx)
	} else {
		// Just update counter
		if err := pm.db.SetSetting("current_step_quizzes_generated", strconv.Itoa(count)); err != nil {
			log.Printf("Failed to update quiz counter: %v", err)
		}
		log.Printf("Quiz generated for current step. Counter: %d/%d", count, QuizzesPerStep)
	}
}

// AdvancePlan manually moves the plan to the next step, resetting the counter.
func (pm *PlanManager) AdvancePlan() {
	idx := pm.GetCurrentStepIndex()
	idx++
	pm.db.SetSetting("current_learning_step", strconv.Itoa(idx))
	pm.db.SetSetting("current_step_quizzes_generated", "0")
}
