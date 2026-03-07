package sqlite

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/mats/telegram-quiz-bot/internal/repository"
	_ "github.com/mattn/go-sqlite3"
)

// Store implements all repositories.
type Store struct {
	db *sql.DB
}

const schema = `
CREATE TABLE IF NOT EXISTS users (
	telegram_id INTEGER PRIMARY KEY,
	username TEXT,
	score INTEGER DEFAULT 0,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS segments (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	title TEXT,
	description TEXT,
	order_index INTEGER DEFAULT 0,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS quizzes (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	segment_id INTEGER,
	title TEXT,
	description TEXT,
	order_index INTEGER DEFAULT 0,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY(segment_id) REFERENCES segments(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS questions (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	quiz_id INTEGER,
	text TEXT,
	options TEXT, -- JSON array
	correct_answer TEXT,
	tts_phrase TEXT DEFAULT '',
	audio_file_id TEXT,
	is_active BOOLEAN DEFAULT 0,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY(quiz_id) REFERENCES quizzes(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS user_answers (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	question_id INTEGER,
	telegram_id INTEGER,
	is_correct BOOLEAN,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(question_id, telegram_id)
);

CREATE TABLE IF NOT EXISTS settings (
	key TEXT PRIMARY KEY,
	value TEXT
);
`

// NewStore initializes an SQLite database and creates tables if they don't exist.
func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
	if err != nil {
		return nil, err
	}

	// 1. Check if old schema exists and migrate
	if err := migrateOldSchema(db); err != nil {
		return nil, fmt.Errorf("failed to migrate old schema: %w", err)
	}

	// 2. Create tables for new schema (if they don't exist)
	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("failed to execute schema: %w", err)
	}

	// 3. Migrate data from old tables to new tables
	if err := migrateData(db); err != nil {
		return nil, fmt.Errorf("failed to migrate data: %w", err)
	}

	// Create default segment/quiz if empty so the bot can work out of the box
	initDefaultPlan(db)

	return &Store{db: db}, nil
}

func migrateOldSchema(db *sql.DB) error {
	// Check if the old 'quizzes' table exists and doesn't have a segment_id column
	var hasOldQuizzes int
	err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='quizzes' AND sql NOT LIKE '%segment_id INTEGER%'").Scan(&hasOldQuizzes)
	if err != nil {
		return err
	}

	if hasOldQuizzes > 0 {
		log.Println("Old schema detected. Renaming tables for migration...")
		// Rename old tables to temporary names so the new schema can be created
		_, err = db.Exec(`
			ALTER TABLE quizzes RENAME TO old_quizzes;
			ALTER TABLE user_answers RENAME TO old_user_answers;
		`)
		if err != nil {
			return err
		}
	}
	return nil
}

func migrateData(db *sql.DB) error {
	// Check if old tables exist to migrate data from
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='old_quizzes'").Scan(&count)
	if err != nil {
		return err
	}

	if count == 0 {
		return nil // Nothing to migrate
	}

	log.Println("Migrating data from old schema to new schema...")

	// 1. Create a default Segment to hold the old quizzes
	res, err := db.Exec("INSERT INTO segments (title, description, order_index) VALUES (?, ?, ?)",
		"Legacy Topics", "Topics migrated from previous version", 0)
	if err != nil {
		return err
	}
	segID, _ := res.LastInsertId()

	// 2. Migrate topics from old_quizzes to new quizzes table
	// We need to group old_quizzes by topic to create a new Quiz for each topic
	rows, err := db.Query("SELECT DISTINCT topic FROM old_quizzes")
	if err != nil {
		return err
	}
	defer rows.Close()

	topicToQuizID := make(map[string]int)
	orderIdx := 0
	for rows.Next() {
		var topic string
		if err := rows.Scan(&topic); err != nil {
			return err
		}

		// Fetch lesson content if it exists
		var lessonContent string
		err := db.QueryRow("SELECT content FROM topic_lessons WHERE topic = ?", topic).Scan(&lessonContent)
		if err == sql.ErrNoRows {
			lessonContent = "Learn about " + topic
		}

		qRes, err := db.Exec("INSERT INTO quizzes (segment_id, title, description, order_index) VALUES (?, ?, ?, ?)",
			segID, topic, lessonContent, orderIdx)
		if err != nil {
			return err
		}
		newQuizID, _ := qRes.LastInsertId()
		topicToQuizID[topic] = int(newQuizID)
		orderIdx++
	}
	rows.Close() // Close early before next query loop

	// 3. Migrate questions and answers
	// To preserve foreign keys in user_answers, we need a mapping from old_quiz_id to new_question_id
	oldQuizToNewQuestion := make(map[int]int)

	qRows, err := db.Query("SELECT id, topic, question, options, correct_answer, audio_file_id, is_active, created_at FROM old_quizzes")
	if err != nil {
		return err
	}
	defer qRows.Close()

	for qRows.Next() {
		var oldID int
		var topic, text, options, correctAnswer, audioFileID string
		var isActive bool
		var createdAt string

		if err := qRows.Scan(&oldID, &topic, &text, &options, &correctAnswer, &audioFileID, &isActive, &createdAt); err != nil {
			return err
		}

		newQuizID := topicToQuizID[topic]

		insRes, err := db.Exec("INSERT INTO questions (quiz_id, text, options, correct_answer, tts_phrase, audio_file_id, is_active, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
			newQuizID, text, options, correctAnswer, "", audioFileID, isActive, createdAt)
		if err != nil {
			return err
		}
		newQuestionID, _ := insRes.LastInsertId()
		oldQuizToNewQuestion[oldID] = int(newQuestionID)
	}
	qRows.Close()

	// 4. Migrate user answers
	ansRows, err := db.Query("SELECT id, quiz_id, telegram_id, is_correct, created_at FROM old_user_answers")
	if err != nil {
		return err
	}
	defer ansRows.Close()

	for ansRows.Next() {
		var id, oldQuizID int
		var telegramID int64
		var isCorrect bool
		var createdAt string

		if err := ansRows.Scan(&id, &oldQuizID, &telegramID, &isCorrect, &createdAt); err != nil {
			return err
		}

		newQuestionID, ok := oldQuizToNewQuestion[oldQuizID]
		if !ok {
			continue // Skip if mapped question not found (shouldn't happen)
		}

		_, err = db.Exec("INSERT INTO user_answers (question_id, telegram_id, is_correct, created_at) VALUES (?, ?, ?, ?)",
			newQuestionID, telegramID, isCorrect, createdAt)
		if err != nil {
			log.Printf("Warning: failed to migrate user answer (old quiz_id: %d): %v", oldQuizID, err)
		}
	}
	ansRows.Close()

	// 5. Cleanup old tables
	log.Println("Data migration complete. Dropping legacy tables...")
	db.Exec("DROP TABLE old_quizzes")
	db.Exec("DROP TABLE old_user_answers")
	db.Exec("DROP TABLE topic_lessons")
	db.Exec("DROP TABLE user_lessons")

	return nil
}

func initDefaultPlan(db *sql.DB) {
	var count int
	db.QueryRow("SELECT COUNT(*) FROM segments").Scan(&count)
	if count > 0 {
		return
	}

	log.Println("Initializing default learning plan in database...")

	segmentsData := []struct {
		Title       string
		Description string
		Order       int
		Topics      []string
	}{
		{
			Title:       "Basics",
			Description: "Core greetings, numbers, and colors.",
			Order:       0,
			Topics: []string{
				"Basic Greetings (Hola, Adiós, Buenos días)",
				"Numbers 1 to 10",
				"Colors (Red, Blue, Green, Yellow)",
			},
		},
		{
			Title:       "Everyday Life",
			Description: "Family, days of the week, and months.",
			Order:       1,
			Topics: []string{
				"Family Members (Padre, Madre, Hermano)",
				"Days of the Week",
				"Months of the Year",
			},
		},
		{
			Title:       "Advanced Basics",
			Description: "Animals, food, verbs, and time.",
			Order:       2,
			Topics: []string{
				"Common Animals (Dog, Cat, Bird)",
				"Basic Foods (Bread, Water, Apple)",
				"Basic Verbs (To be, To have, To go)",
				"Telling Time in Spanish",
			},
		},
		{
			Title:       "Travel & Directions",
			Description: "Navigating cities, transportation, and asking for help.",
			Order:       3,
			Topics: []string{
				"Transportation (Train, Bus, Airport)",
				"Asking for Directions",
				"Booking a Hotel",
			},
		},
		{
			Title:       "Dining Out",
			Description: "Ordering food, interacting with waiters, and understanding menus.",
			Order:       4,
			Topics: []string{
				"Ordering at a Restaurant",
				"Beverages and Drinks",
				"Paying the Bill",
			},
		},
		{
			Title:       "Shopping & Clothing",
			Description: "Buying clothes, asking for prices, and discussing colors/sizes.",
			Order:       5,
			Topics: []string{
				"Types of Clothing",
				"Asking for Prices",
				"Colors and Sizes",
			},
		},
		{
			Title:       "House & Home",
			Description: "Rooms in a house, furniture, and household chores.",
			Order:       6,
			Topics: []string{
				"Rooms of the House",
				"Furniture and Appliances",
				"Household Chores",
			},
		},
		{
			Title:       "Health & Body",
			Description: "Body parts, describing symptoms, and visiting the doctor.",
			Order:       7,
			Topics: []string{
				"Parts of the Body",
				"Common Illnesses and Symptoms",
				"At the Doctor's Office",
			},
		},
		{
			Title:       "Hobbies & Leisure",
			Description: "Activities, sports, and discussing free time.",
			Order:       8,
			Topics: []string{
				"Common Hobbies",
				"Sports and Games",
				"Music and Entertainment",
			},
		},
		{
			Title:       "Work & Profession",
			Description: "Jobs, describing a typical workday, and office vocabulary.",
			Order:       9,
			Topics: []string{
				"Common Professions",
				"Office Tools and Environment",
				"Describing Work Duties",
			},
		},
		{
			Title:       "Grammar Deep Dive I",
			Description: "Present tense, articles, and adjectives.",
			Order:       10,
			Topics: []string{
				"Definite and Indefinite Articles",
				"Regular Verbs in Present Tense (-ar, -er, -ir)",
				"Adjective Agreement",
			},
		},
		{
			Title:       "Grammar Deep Dive II",
			Description: "Past tenses, prepositions, and pronouns.",
			Order:       11,
			Topics: []string{
				"Preterite Tense (Basic Regular Verbs)",
				"Direct and Indirect Object Pronouns",
				"Common Prepositions (Por vs. Para basics)",
			},
		},
	}

	for _, segData := range segmentsData {
		res, err := db.Exec("INSERT INTO segments (title, description, order_index) VALUES (?, ?, ?)",
			segData.Title, segData.Description, segData.Order)
		if err != nil {
			log.Printf("failed to create segment %s: %v", segData.Title, err)
			continue
		}

		segID, _ := res.LastInsertId()

		for i, topic := range segData.Topics {
			db.Exec("INSERT INTO quizzes (segment_id, title, description, order_index) VALUES (?, ?, ?, ?)",
				segID, topic, "Learn about "+topic, i)
		}
	}
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// Repositories returns all repositories initialized with the SQLite connection.
func (s *Store) Repositories() *repository.Repositories {
	return &repository.Repositories{
		Segments:  &SegmentRepo{db: s.db},
		Quizzes:   &QuizRepo{db: s.db},
		Questions: &QuestionRepo{db: s.db},
		Users:     &UserRepo{db: s.db},
		Settings:  &SettingsRepo{db: s.db},
	}
}
