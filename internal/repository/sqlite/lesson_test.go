package sqlite

import (
	"context"
	"os"
	"testing"

	"github.com/mats/telegram-quiz-bot/internal/domain"
)

func TestTopicLessons(t *testing.T) {
	dbPath := "test_lessons.db"
	defer os.Remove(dbPath)

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	repos := store.Repositories()
	ctx := context.Background()

	segment := &domain.Segment{Title: "Test Segment"}
	err = repos.Segments.Create(ctx, segment)
	if err != nil {
		t.Fatalf("Failed to create segment: %v", err)
	}

	quiz := &domain.Quiz{
		SegmentID:   segment.ID,
		Title:       "Test Topic",
		Description: "This is a test lesson.",
	}
	if err := repos.Quizzes.Create(ctx, quiz); err != nil {
		t.Fatalf("Create Quiz failed: %v", err)
	}

	gotQuiz, err := repos.Quizzes.GetByID(ctx, quiz.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if gotQuiz.Description != quiz.Description {
		t.Errorf("Expected lesson content %q, got %q", quiz.Description, gotQuiz.Description)
	}
}
