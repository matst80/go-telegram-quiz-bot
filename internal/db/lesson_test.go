package db

import (
	"os"
	"testing"
)

func TestTopicLessons(t *testing.T) {
	dbPath := "test_lessons.db"
	defer os.Remove(dbPath)

	database, err := New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer database.Close()

	topic := "Test Topic"
	content := "This is a test lesson."

	// Test SaveLesson and GetLesson
	if err := database.SaveLesson(topic, content); err != nil {
		t.Fatalf("SaveLesson failed: %v", err)
	}

	got, err := database.GetLesson(topic)
	if err != nil {
		t.Fatalf("GetLesson failed: %v", err)
	}
	if got != content {
		t.Errorf("Expected lesson content %q, got %q", content, got)
	}

	// Test HasSeenLesson and MarkLessonSeen
	userID := int64(12345)
	seen, err := database.HasSeenLesson(userID, topic)
	if err != nil {
		t.Fatalf("HasSeenLesson failed: %v", err)
	}
	if seen {
		t.Errorf("Expected seen=false for new user")
	}

	if err := database.MarkLessonSeen(userID, topic); err != nil {
		t.Fatalf("MarkLessonSeen failed: %v", err)
	}

	seen, err = database.HasSeenLesson(userID, topic)
	if err != nil {
		t.Fatalf("HasSeenLesson failed: %v", err)
	}
	if !seen {
		t.Errorf("Expected seen=true after MarkLessonSeen")
	}
}
