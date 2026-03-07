package main

import (
	"context"
	"fmt"
	"log"

	"github.com/mats/telegram-quiz-bot/internal/repository/sqlite"
)

func main() {
	store, err := sqlite.NewStore("quizbot.db")
	if err != nil {
		log.Fatalf("Failed to open store: %v", err)
	}
	defer store.Close()

	repos := store.Repositories()
	segs, err := repos.Segments.GetAll(context.Background())
	if err != nil {
		log.Fatalf("GetAll error: %v", err)
	}

	fmt.Printf("Found %d segments:\n", len(segs))
	for _, s := range segs {
		fmt.Printf("- [%d] %s: %s (Created: %v)\n", s.ID, s.Title, s.Description, s.CreatedAt)
		quizzes, err := repos.Quizzes.GetBySegmentID(context.Background(), s.ID)
		if err != nil {
			fmt.Printf("  Error fetching quizzes: %v\n", err)
		} else {
			fmt.Printf("  Found %d quizzes\n", len(quizzes))
		}
	}
}
