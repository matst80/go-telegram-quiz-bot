package sqlite

import (
	"context"
	"testing"

	"github.com/mats/telegram-quiz-bot/internal/domain"
)

func TestSeedPlan(t *testing.T) {
	// Use the main database file for seeding
	dbPath := "../../../quizbot.db"
	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer store.Close()
	repos := store.Repositories()
	ctx := context.Background()

	// 1. Clear existing data to ensure a fresh seed (Optional, but cleaner for a defined plan)
	// Note: We use raw Exec for simplicity in this specific seeder test
	_, _ = store.db.Exec("DELETE FROM questions")
	_, _ = store.db.Exec("DELETE FROM quizzes")
	_, _ = store.db.Exec("DELETE FROM segments")

	// 2. Define Segments
	segments := []*domain.Segment{
		{Title: "Basics", Description: "Core greetings, numbers, and colors.", OrderIndex: 0},
		{Title: "Everyday Life", Description: "Family, days of the week, and months.", OrderIndex: 1},
		{Title: "Advanced Basics", Description: "Animals, food, verbs, and time.", OrderIndex: 2},
	}

	for _, s := range segments {
		if err := repos.Segments.Create(ctx, s); err != nil {
			t.Fatalf("Failed to create segment %s: %v", s.Title, err)
		}
	}

	// 3. Define Quizzes (Lessons) for each segment
	plan := map[string][]string{
		"Basics": {
			"Basic Greetings (Hola, Adiós, Buenos días)",
			"Numbers 1 to 10",
			"Colors (Red, Blue, Green, Yellow)",
		},
		"Everyday Life": {
			"Family Members (Padre, Madre, Hermano)",
			"Days of the Week",
			"Months of the Year",
		},
		"Advanced Basics": {
			"Common Animals (Dog, Cat, Bird)",
			"Basic Foods (Bread, Water, Apple)",
			"Basic Verbs (To be, To have, To go)",
			"Telling Time in Spanish",
		},
	}

	segmentMap := make(map[string]int)
	for _, s := range segments {
		segmentMap[s.Title] = s.ID
	}

	quizToID := make(map[string]int)

	for segTitle, topics := range plan {
		segID := segmentMap[segTitle]
		for i, topic := range topics {
			q := &domain.Quiz{
				SegmentID:   segID,
				Title:       topic,
				Description: "Learn about " + topic,
				OrderIndex:  i,
			}
			if err := repos.Quizzes.Create(ctx, q); err != nil {
				t.Fatalf("Failed to create quiz %s: %v", topic, err)
			}
			quizToID[topic] = q.ID
		}
	}

	// 4. Seed sample questions for the first few topics
	seedQuestions := []struct {
		Topic     string
		Questions []domain.Question
	}{
		{
			Topic: "Basic Greetings (Hola, Adiós, Buenos días)",
			Questions: []domain.Question{
				{
					Text:          "Which Spanish phrase means 'Good morning'?",
					Options:       []string{"Buenas noches", "Buenos días", "Buenas tardes", "Adiós"},
					CorrectAnswer: "Buenos días",
				},
				{
					Text:          "Which word is used to say goodbye in Spanish?",
					Options:       []string{"Hola", "Por favor", "Adiós", "Gracias"},
					CorrectAnswer: "Adiós",
				},
			},
		},
		{
			Topic: "Numbers 1 to 10",
			Questions: []domain.Question{
				{
					Text:          "How do you say 'Three' in Spanish?",
					Options:       []string{"Uno", "Dos", "Tres", "Cuatro"},
					CorrectAnswer: "Tres",
				},
				{
					Text:          "What is 'Siete' in English?",
					Options:       []string{"Six", "Seven", "Eight", "Five"},
					CorrectAnswer: "Seven",
				},
			},
		},
	}

	for _, sq := range seedQuestions {
		quizID := quizToID[sq.Topic]
		for _, q := range sq.Questions {
			q.QuizID = quizID
			q.IsActive = true
			if err := repos.Questions.Create(ctx, &q); err != nil {
				t.Errorf("Failed to create question '%s' for topic %s: %v", q.Text, sq.Topic, err)
			}
		}
	}

	t.Logf("Seeding complete. Created %d segments, %d quizzes.", len(segments), len(quizToID))
}
