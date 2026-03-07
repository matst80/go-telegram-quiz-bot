package domain

import "time"

// Segment represents a high-level learning category (e.g., "Basics", "Intermediate").
type Segment struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	OrderIndex  int       `json:"order_index"`
	CreatedAt   time.Time `json:"created_at"`
}

// Quiz represents a specific topic within a segment (e.g., "Greetings", "Numbers 1-10").
type Quiz struct {
	ID          int       `json:"id"`
	SegmentID   int       `json:"segment_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	OrderIndex  int       `json:"order_index"`
	CreatedAt   time.Time `json:"created_at"`
}

// Question represents an individual multiple-choice question belonging to a Quiz.
type Question struct {
	ID            int       `json:"id"`
	QuizID        int       `json:"quiz_id"`
	Text          string    `json:"text"`
	Options       []string  `json:"options"`
	CorrectAnswer string    `json:"correct_answer"`
	TTSPhrase     string    `json:"tts_phrase,omitempty"`
	AudioFileID   string    `json:"audio_file_id,omitempty"`
	IsActive      bool      `json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
}

// User represents a Telegram user participating in the quizzes.
type User struct {
	TelegramID int64     `json:"telegram_id"`
	Username   string    `json:"username"`
	Score      int       `json:"score"`
	CreatedAt  time.Time `json:"created_at"`
}

// UserAnswer tracks which user answered which question and if they were correct.
type UserAnswer struct {
	ID         int       `json:"id"`
	QuestionID int       `json:"question_id"`
	TelegramID int64     `json:"telegram_id"`
	IsCorrect  bool      `json:"is_correct"`
	CreatedAt  time.Time `json:"created_at"`
}
