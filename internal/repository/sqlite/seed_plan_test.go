package sqlite

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/mats/telegram-quiz-bot/internal/domain"
	"github.com/mats/telegram-quiz-bot/internal/tts"
)

func TestSeedPlan(t *testing.T) {
	dbPath := os.Getenv("SEED_DB_PATH")
	if dbPath == "" {
		// Create a temporary file for the database
		tmpfile, err := os.CreateTemp("", "quizbot_plan_test_*.db")
		if err != nil {
			t.Fatalf("Failed to create temp db file: %v", err)
		}
		dbPath = tmpfile.Name()
		tmpfile.Close()
		defer os.Remove(dbPath)
	} else {
		// Just clear it if it exists so we get a fresh seed
		os.Remove(dbPath)
	}

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer store.Close()
	repos := store.Repositories()
	ctx := context.Background()

	// Set up audio storage directory
	audioDir := "../../../storage/audio"
	if err := os.MkdirAll(audioDir, 0755); err != nil {
		t.Fatalf("Failed to create audio directory: %v", err)
	}

	// Set up TTS Service
	modelDir := "../../../storage/tts_model"
	if err := os.MkdirAll(modelDir, 0755); err != nil {
		t.Fatalf("Failed to create model directory: %v", err)
	}
	piperConfig, err := tts.EnsureDefaultModel(modelDir)
	if err != nil {
		t.Fatalf("Failed to ensure default model: %v", err)
	}
	ttsService, err := tts.NewPiperService(piperConfig)
	if err != nil {
		t.Fatalf("Failed to create Piper service: %v", err)
	}

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
		{Title: "Travel & Directions", Description: "Navigating cities, transportation, and asking for help.", OrderIndex: 3},
		{Title: "Dining Out", Description: "Ordering food, interacting with waiters, and understanding menus.", OrderIndex: 4},
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
		"Travel & Directions": {
			"Transportation (Train, Bus, Airport)",
			"Asking for Directions",
			"Booking a Hotel",
		},
		"Dining Out": {
			"Ordering at a Restaurant",
			"Beverages and Drinks",
			"Paying the Bill",
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
					TTSPhrase:     "Buenos días, ¿cómo estás?",
				},
				{
					Text:          "Which word is used to say goodbye in Spanish?",
					Options:       []string{"Hola", "Por favor", "Adiós", "Gracias"},
					CorrectAnswer: "Adiós",
					TTSPhrase:     "Adiós, hasta luego.",
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
					TTSPhrase:     "Tengo tres gatos.",
				},
				{
					Text:          "What is 'Siete' in English?",
					Options:       []string{"Six", "Seven", "Eight", "Five"},
					CorrectAnswer: "Seven",
					TTSPhrase:     "Son las siete de la tarde.",
				},
			},
		},
		{
			Topic: "Colors (Red, Blue, Green, Yellow)",
			Questions: []domain.Question{
				{
					Text:          "What is the Spanish word for 'Red'?",
					Options:       []string{"Rojo", "Azul", "Verde", "Amarillo"},
					CorrectAnswer: "Rojo",
					TTSPhrase:     "El coche es rojo.",
				},
				{
					Text:          "How do you say 'Blue' in Spanish?",
					Options:       []string{"Blanco", "Negro", "Azul", "Gris"},
					CorrectAnswer: "Azul",
					TTSPhrase:     "El cielo es azul.",
				},
			},
		},
		{
			Topic: "Transportation (Train, Bus, Airport)",
			Questions: []domain.Question{
				{
					Text:          "Which word means 'Train' in Spanish?",
					Options:       []string{"Autobús", "Avión", "Coche", "Tren"},
					CorrectAnswer: "Tren",
					TTSPhrase:     "El tren sale a las cinco.",
				},
				{
					Text:          "How do you say 'Airport'?",
					Options:       []string{"Estación", "Aeropuerto", "Puerto", "Parada"},
					CorrectAnswer: "Aeropuerto",
					TTSPhrase:     "Voy al aeropuerto.",
				},
			},
		},
		{
			Topic: "Ordering at a Restaurant",
			Questions: []domain.Question{
				{
					Text:          "How do you ask for 'The bill, please'?",
					Options:       []string{"La cuenta, por favor", "El menú, por favor", "Más agua, por favor", "La mesa, por favor"},
					CorrectAnswer: "La cuenta, por favor",
					TTSPhrase:     "La cuenta, por favor.",
				},
				{
					Text:          "Which word means 'Waiter'?",
					Options:       []string{"Cocinero", "Camarero", "Cliente", "Gerente"},
					CorrectAnswer: "Camarero",
					TTSPhrase:     "El camarero es muy amable.",
				},
			},
		},
	}

	for _, sq := range seedQuestions {
		quizID := quizToID[sq.Topic]
		for i, q := range sq.Questions {
			q.QuizID = quizID
			q.IsActive = true

			// Generate TTS audio for the correct answer
			audioFilename := fmt.Sprintf("q_%d_%d.wav", quizID, i)
			audioPath := filepath.Join(audioDir, audioFilename)

			if q.TTSPhrase != "" {
				if err := ttsService.GenerateSpeech(q.TTSPhrase, audioPath); err != nil {
					t.Logf("Failed to generate TTS for question '%s': %v", q.Text, err)
				} else {
					// The Bot will run from the repository root, so we store the path relative to it
					q.AudioFileID = filepath.Join("storage", "audio", audioFilename)
				}
			}

			if err := repos.Questions.Create(ctx, &q); err != nil {
				t.Errorf("Failed to create question '%s' for topic %s: %v", q.Text, sq.Topic, err)
			}
		}
	}

	t.Logf("Seeding complete. Created %d segments, %d quizzes.", len(segments), len(quizToID))
}
