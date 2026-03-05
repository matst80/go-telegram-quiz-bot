package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	*sql.DB
}

type User struct {
	TelegramID int64
	Username   string
	Score      int
}

type Quiz struct {
	ID            int      `json:"id"`
	Topic         string   `json:"topic"`
	Question      string   `json:"question"`
	Options       []string `json:"options"`
	CorrectAnswer string   `json:"correct_answer"`
	AudioFileID   string   `json:"audio_file_id"`
	IsActive      bool     `json:"is_active"`
}

const schema = `
CREATE TABLE IF NOT EXISTS users (
	telegram_id INTEGER PRIMARY KEY,
	username TEXT,
	score INTEGER DEFAULT 0
);

CREATE TABLE IF NOT EXISTS quizzes (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	topic TEXT,
	question TEXT,
	options TEXT, -- JSON array
	correct_answer TEXT,
	audio_file_id TEXT,
	is_active BOOLEAN DEFAULT 0,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS user_answers (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	quiz_id INTEGER,
	telegram_id INTEGER,
	is_correct BOOLEAN,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(quiz_id, telegram_id)
);

CREATE TABLE IF NOT EXISTS settings (
	key TEXT PRIMARY KEY,
	value TEXT
);

CREATE TABLE IF NOT EXISTS topic_lessons (
	topic TEXT PRIMARY KEY,
	content TEXT
);

CREATE TABLE IF NOT EXISTS user_lessons (
	telegram_id INTEGER,
	topic TEXT,
	PRIMARY KEY(telegram_id, topic)
);
`

func New(dbPath string) (*DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("failed to execute schema: %w", err)
	}

	return &DB{db}, nil
}

// RegisterUser adds a user if they don't exist
func (db *DB) RegisterUser(telegramID int64, username string) error {
	_, err := db.Exec(`
		INSERT INTO users (telegram_id, username) 
		VALUES (?, ?) 
		ON CONFLICT(telegram_id) DO UPDATE SET username = excluded.username`,
		telegramID, username)
	return err
}

// GetTopUsers returns top N users by score
func (db *DB) GetTopUsers(limit int) ([]User, error) {
	rows, err := db.Query(`SELECT telegram_id, username, score FROM users ORDER BY score DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.TelegramID, &u.Username, &u.Score); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// GetAllUsers returns all registered users
func (db *DB) GetAllUsers() ([]User, error) {
	rows, err := db.Query(`SELECT telegram_id, username, score FROM users`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.TelegramID, &u.Username, &u.Score); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// SaveQuiz stores a new quiz and sets it as active
func (db *DB) SaveQuiz(q Quiz) (int, error) {
	optionsJSON, _ := json.Marshal(q.Options)
	res, err := db.Exec(`
		INSERT INTO quizzes (topic, question, options, correct_answer, audio_file_id, is_active)
		VALUES (?, ?, ?, ?, ?, 1)
	`, q.Topic, q.Question, string(optionsJSON), q.CorrectAnswer, q.AudioFileID)

	if err != nil {
		return 0, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(id), nil
}

// GetNextUnansweredQuiz returns the oldest unanswered quiz for a user in a given topic
func (db *DB) GetNextUnansweredQuiz(telegramID int64, topic string) (*Quiz, error) {
	var q Quiz
	var optionsStr string

	err := db.QueryRow(`
		SELECT id, topic, question, options, correct_answer, audio_file_id 
		FROM quizzes 
		WHERE topic = ? 
		AND id NOT IN (SELECT quiz_id FROM user_answers WHERE telegram_id = ?)
		ORDER BY id ASC LIMIT 1
	`, topic, telegramID).Scan(&q.ID, &q.Topic, &q.Question, &optionsStr, &q.CorrectAnswer, &q.AudioFileID)

	if err == sql.ErrNoRows {
		// Fallback: try to find ANY unanswered quiz if topic-specific ones are exhausted
		err = db.QueryRow(`
			SELECT id, topic, question, options, correct_answer, audio_file_id 
			FROM quizzes 
			WHERE id NOT IN (SELECT quiz_id FROM user_answers WHERE telegram_id = ?)
			ORDER BY id ASC LIMIT 1
		`, telegramID).Scan(&q.ID, &q.Topic, &q.Question, &optionsStr, &q.CorrectAnswer, &q.AudioFileID)

		if err == sql.ErrNoRows {
			return nil, nil // Truly no unanswered quizzes
		}
	}

	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(optionsStr), &q.Options); err != nil {
		log.Printf("Failed to unmarshal quiz options: %v", err)
	}

	q.IsActive = true
	return &q, nil
}

// GetUnansweredCount returns how many quizzes the user hasn't answered for a topic
func (db *DB) GetUnansweredCount(telegramID int64, topic string) (int, error) {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM quizzes 
		WHERE topic = ? 
		AND id NOT IN (SELECT quiz_id FROM user_answers WHERE telegram_id = ?)
	`, topic, telegramID).Scan(&count)
	return count, err
}

// GetRecentQuestionsForTopic returns the last N questions for a given topic
func (db *DB) GetRecentQuestionsForTopic(topic string, limit int) ([]string, error) {
	rows, err := db.Query("SELECT question FROM quizzes WHERE topic = ? ORDER BY id DESC LIMIT ?", topic, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var questions []string
	for rows.Next() {
		var q string
		if err := rows.Scan(&q); err != nil {
			return nil, err
		}
		questions = append(questions, q)
	}
	return questions, rows.Err()
}

// GetActiveQuiz returns the currently active quiz
func (db *DB) GetActiveQuiz() (*Quiz, error) {
	var q Quiz
	var optionsStr string

	err := db.QueryRow(`
		SELECT id, topic, question, options, correct_answer, audio_file_id 
		FROM quizzes 
		WHERE is_active = 1 
		ORDER BY id DESC LIMIT 1
	`).Scan(&q.ID, &q.Topic, &q.Question, &optionsStr, &q.CorrectAnswer, &q.AudioFileID)

	if err == sql.ErrNoRows {
		return nil, nil // No active quiz
	} else if err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(optionsStr), &q.Options); err != nil {
		log.Printf("Failed to unmarshal quiz options: %v", err)
	}

	q.IsActive = true
	return &q, nil
}

// RecordAnswer records a user's answer and updates score if correct
func (db *DB) RecordAnswer(quizID int, telegramID int64, isCorrect bool) error {
	// Attempt to insert the answer (will fail if user already answered due to UNIQUE constraint)
	_, err := db.Exec(`
		INSERT INTO user_answers (quiz_id, telegram_id, is_correct)
		VALUES (?, ?, ?)
	`, quizID, telegramID, isCorrect)

	if err != nil {
		// Possibly a unique constraint violation (already answered)
		return fmt.Errorf("you have already answered this quiz")
	}

	if isCorrect {
		_, err = db.Exec(`UPDATE users SET score = score + 1 WHERE telegram_id = ?`, telegramID)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetSetting retrieves a configuration value from the settings table
func (db *DB) GetSetting(key string) (string, error) {
	var val string
	err := db.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&val)
	if err == sql.ErrNoRows {
		return "", nil // Return empty string if not found, let caller handle defaults
	}
	return val, err
}

// SetSetting stores or updates a configuration value in the settings table
func (db *DB) SetSetting(key, value string) error {
	_, err := db.Exec(`
		INSERT INTO settings (key, value) 
		VALUES (?, ?) 
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		key, value)
	return err
}

// SaveLesson stores or updates a lesson for a topic
func (db *DB) SaveLesson(topic, content string) error {
	_, err := db.Exec(`
		INSERT INTO topic_lessons (topic, content) 
		VALUES (?, ?) 
		ON CONFLICT(topic) DO UPDATE SET content = excluded.content`,
		topic, content)
	return err
}

// GetLesson retrieves the content for a topic lesson
func (db *DB) GetLesson(topic string) (string, error) {
	var content string
	err := db.QueryRow("SELECT content FROM topic_lessons WHERE topic = ?", topic).Scan(&content)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return content, err
}

// HasSeenLesson checks if a user has already seen a lesson
func (db *DB) HasSeenLesson(telegramID int64, topic string) (bool, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM user_lessons WHERE telegram_id = ? AND topic = ?", telegramID, topic).Scan(&count)
	return count > 0, err
}

// MarkLessonSeen records that a user has seen a lesson
func (db *DB) MarkLessonSeen(telegramID int64, topic string) error {
	_, err := db.Exec(`INSERT OR IGNORE INTO user_lessons (telegram_id, topic) VALUES (?, ?)`, telegramID, topic)
	return err
}
