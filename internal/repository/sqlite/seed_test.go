package sqlite

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/mats/telegram-quiz-bot/internal/domain"
)

func TestSeedDatabase(t *testing.T) {
	// Create a temporary file for the database
	tmpfile, err := os.CreateTemp("", "quizbot_test_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp db file: %v", err)
	}
	dbPath := tmpfile.Name()
	tmpfile.Close()
	defer os.Remove(dbPath)

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer store.Close()
	repos := store.Repositories()
	ctx := context.Background()

	seedData := `[
      {
        "text": "Which Spanish phrase means 'Good morning'?",
        "options": ["Buenas noches", "Buenos días", "Buenas tardes", "Adiós"],
        "correct_answer": "Buenos días"
      },
      {
        "text": "Which word is used to say goodbye in Spanish?",
        "options": ["Hola", "Por favor", "Adiós", "Gracias"],
        "correct_answer": "Adiós"
      }
    ]`

	var questions []domain.Question
	if err := json.Unmarshal([]byte(seedData), &questions); err != nil {
		t.Fatalf("Failed to unmarshal seed data: %v", err)
	}

	segment := &domain.Segment{Title: "Seed Test Segment", Description: "Test desc"}
	err = repos.Segments.Create(ctx, segment)
	if err != nil {
		t.Fatalf("Failed to save segment: %v", err)
	}

	quiz := &domain.Quiz{
		SegmentID:   segment.ID,
		Title:       "Basic Greetings (Hola, Adiós, Buenos días)",
		Description: "Welcome to your first Spanish lesson!",
	}
	if err := repos.Quizzes.Create(ctx, quiz); err != nil {
		t.Fatalf("Failed to save quiz: %v", err)
	}

	for _, q := range questions {
		q.QuizID = quiz.ID
		q.IsActive = true
		err := repos.Questions.Create(ctx, &q)
		if err != nil {
			t.Errorf("Failed to save question '%s': %v", q.Text, err)
			continue
		}
		t.Logf("Saved question with ID: %d", q.ID)
	}
}
