package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"github.com/mats/telegram-quiz-bot/internal/domain"
)

type SegmentRepo struct { db *sql.DB }
func (r *SegmentRepo) Create(ctx context.Context, segment *domain.Segment) error {
	res, err := r.db.ExecContext(ctx, "INSERT INTO segments (title, description, order_index) VALUES (?, ?, ?)",
		segment.Title, segment.Description, segment.OrderIndex)
	if err != nil { return err }
	id, _ := res.LastInsertId()
	segment.ID = int(id)
	return nil
}
func (r *SegmentRepo) GetByID(ctx context.Context, id int) (*domain.Segment, error) {
	var s domain.Segment
	err := r.db.QueryRowContext(ctx, "SELECT id, title, description, order_index, created_at FROM segments WHERE id = ?", id).
		Scan(&s.ID, &s.Title, &s.Description, &s.OrderIndex, &s.CreatedAt)
	if err == sql.ErrNoRows { return nil, nil }
	return &s, err
}
func (r *SegmentRepo) GetAll(ctx context.Context) ([]domain.Segment, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id, title, description, order_index, created_at FROM segments ORDER BY order_index ASC")
	if err != nil { return nil, err }
	defer rows.Close()
	var segs []domain.Segment
	for rows.Next() {
		var s domain.Segment
		if err := rows.Scan(&s.ID, &s.Title, &s.Description, &s.OrderIndex, &s.CreatedAt); err != nil { return nil, err }
		segs = append(segs, s)
	}
	return segs, nil
}
func (r *SegmentRepo) Update(ctx context.Context, segment *domain.Segment) error {
	_, err := r.db.ExecContext(ctx, "UPDATE segments SET title = ?, description = ?, order_index = ? WHERE id = ?",
		segment.Title, segment.Description, segment.OrderIndex, segment.ID)
	return err
}
func (r *SegmentRepo) Delete(ctx context.Context, id int) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM segments WHERE id = ?", id)
	return err
}

type QuizRepo struct { db *sql.DB }
func (r *QuizRepo) Create(ctx context.Context, quiz *domain.Quiz) error {
	res, err := r.db.ExecContext(ctx, "INSERT INTO quizzes (segment_id, title, description, order_index) VALUES (?, ?, ?, ?)",
		quiz.SegmentID, quiz.Title, quiz.Description, quiz.OrderIndex)
	if err != nil { return err }
	id, _ := res.LastInsertId()
	quiz.ID = int(id)
	return nil
}
func (r *QuizRepo) GetByID(ctx context.Context, id int) (*domain.Quiz, error) {
	var q domain.Quiz
	err := r.db.QueryRowContext(ctx, "SELECT id, segment_id, title, description, order_index, created_at FROM quizzes WHERE id = ?", id).
		Scan(&q.ID, &q.SegmentID, &q.Title, &q.Description, &q.OrderIndex, &q.CreatedAt)
	if err == sql.ErrNoRows { return nil, nil }
	return &q, err
}
func (r *QuizRepo) GetBySegmentID(ctx context.Context, segmentID int) ([]domain.Quiz, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id, segment_id, title, description, order_index, created_at FROM quizzes WHERE segment_id = ? ORDER BY order_index ASC", segmentID)
	if err != nil { return nil, err }
	defer rows.Close()
	var quizzes []domain.Quiz
	for rows.Next() {
		var q domain.Quiz
		if err := rows.Scan(&q.ID, &q.SegmentID, &q.Title, &q.Description, &q.OrderIndex, &q.CreatedAt); err != nil { return nil, err }
		quizzes = append(quizzes, q)
	}
	return quizzes, nil
}
func (r *QuizRepo) Update(ctx context.Context, quiz *domain.Quiz) error {
	_, err := r.db.ExecContext(ctx, "UPDATE quizzes SET segment_id = ?, title = ?, description = ?, order_index = ? WHERE id = ?",
		quiz.SegmentID, quiz.Title, quiz.Description, quiz.OrderIndex, quiz.ID)
	return err
}
func (r *QuizRepo) Delete(ctx context.Context, id int) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM quizzes WHERE id = ?", id)
	return err
}

type QuestionRepo struct { db *sql.DB }
func (r *QuestionRepo) Create(ctx context.Context, q *domain.Question) error {
	opts, _ := json.Marshal(q.Options)
	res, err := r.db.ExecContext(ctx, "INSERT INTO questions (quiz_id, text, options, correct_answer, audio_file_id, is_active) VALUES (?, ?, ?, ?, ?, ?)",
		q.QuizID, q.Text, string(opts), q.CorrectAnswer, q.AudioFileID, q.IsActive)
	if err != nil { return err }
	id, _ := res.LastInsertId()
	q.ID = int(id)
	return nil
}
func (r *QuestionRepo) GetByID(ctx context.Context, id int) (*domain.Question, error) {
	var q domain.Question
	var opts string
	err := r.db.QueryRowContext(ctx, "SELECT id, quiz_id, text, options, correct_answer, audio_file_id, is_active, created_at FROM questions WHERE id = ?", id).
		Scan(&q.ID, &q.QuizID, &q.Text, &opts, &q.CorrectAnswer, &q.AudioFileID, &q.IsActive, &q.CreatedAt)
	if err == sql.ErrNoRows { return nil, nil }
	if err != nil { return nil, err }
	json.Unmarshal([]byte(opts), &q.Options)
	return &q, nil
}
func (r *QuestionRepo) GetByQuizID(ctx context.Context, quizID int) ([]domain.Question, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id, quiz_id, text, options, correct_answer, audio_file_id, is_active, created_at FROM questions WHERE quiz_id = ? ORDER BY id ASC", quizID)
	if err != nil { return nil, err }
	defer rows.Close()
	var qs []domain.Question
	for rows.Next() {
		var q domain.Question
		var opts string
		if err := rows.Scan(&q.ID, &q.QuizID, &q.Text, &opts, &q.CorrectAnswer, &q.AudioFileID, &q.IsActive, &q.CreatedAt); err != nil { return nil, err }
		json.Unmarshal([]byte(opts), &q.Options)
		qs = append(qs, q)
	}
	return qs, nil
}
func (r *QuestionRepo) Update(ctx context.Context, q *domain.Question) error {
	opts, _ := json.Marshal(q.Options)
	_, err := r.db.ExecContext(ctx, "UPDATE questions SET quiz_id = ?, text = ?, options = ?, correct_answer = ?, audio_file_id = ?, is_active = ? WHERE id = ?",
		q.QuizID, q.Text, string(opts), q.CorrectAnswer, q.AudioFileID, q.IsActive, q.ID)
	return err
}
func (r *QuestionRepo) Delete(ctx context.Context, id int) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM questions WHERE id = ?", id)
	return err
}
func (r *QuestionRepo) GetNextUnanswered(ctx context.Context, telegramID int64, quizID int) (*domain.Question, error) {
	var q domain.Question
	var opts string
	err := r.db.QueryRowContext(ctx, `
		SELECT id, quiz_id, text, options, correct_answer, audio_file_id, is_active, created_at
		FROM questions
		WHERE quiz_id = ?
		AND id NOT IN (SELECT question_id FROM user_answers WHERE telegram_id = ?)
		ORDER BY id ASC LIMIT 1
	`, quizID, telegramID).Scan(&q.ID, &q.QuizID, &q.Text, &opts, &q.CorrectAnswer, &q.AudioFileID, &q.IsActive, &q.CreatedAt)
	if err == sql.ErrNoRows {
		// Fallback to any unanswered if none for this quiz (this depends on business logic, but matching old logic)
		err = r.db.QueryRowContext(ctx, `
			SELECT id, quiz_id, text, options, correct_answer, audio_file_id, is_active, created_at
			FROM questions
			WHERE id NOT IN (SELECT question_id FROM user_answers WHERE telegram_id = ?)
			ORDER BY id ASC LIMIT 1
		`, telegramID).Scan(&q.ID, &q.QuizID, &q.Text, &opts, &q.CorrectAnswer, &q.AudioFileID, &q.IsActive, &q.CreatedAt)
		if err == sql.ErrNoRows {
			return nil, nil
		}
	}
	if err != nil { return nil, err }
	json.Unmarshal([]byte(opts), &q.Options)
	return &q, nil
}
func (r *QuestionRepo) GetUnansweredCount(ctx context.Context, telegramID int64, quizID int) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM questions WHERE quiz_id = ? AND id NOT IN (SELECT question_id FROM user_answers WHERE telegram_id = ?)", quizID, telegramID).Scan(&count)
	return count, err
}
func (r *QuestionRepo) GetRecentForQuiz(ctx context.Context, quizID int, limit int) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT text FROM questions WHERE quiz_id = ? ORDER BY id DESC LIMIT ?", quizID, limit)
	if err != nil { return nil, err }
	defer rows.Close()
	var questions []string
	for rows.Next() {
		var text string
		if err := rows.Scan(&text); err != nil { return nil, err }
		questions = append(questions, text)
	}
	return questions, nil
}
func (r *QuestionRepo) RecordAnswer(ctx context.Context, questionID int, telegramID int64, isCorrect bool) error {
	_, err := r.db.ExecContext(ctx, "INSERT INTO user_answers (question_id, telegram_id, is_correct) VALUES (?, ?, ?)", questionID, telegramID, isCorrect)
	if err != nil { return fmt.Errorf("already answered or error: %w", err) }

	if isCorrect {
		_, err = r.db.ExecContext(ctx, "UPDATE users SET score = score + 1 WHERE telegram_id = ?", telegramID)
	}
	return err
}

type UserRepo struct { db *sql.DB }
func (r *UserRepo) RegisterUser(ctx context.Context, telegramID int64, username string) error {
	_, err := r.db.ExecContext(ctx, "INSERT INTO users (telegram_id, username) VALUES (?, ?) ON CONFLICT(telegram_id) DO UPDATE SET username = excluded.username", telegramID, username)
	return err
}
func (r *UserRepo) GetTopUsers(ctx context.Context, limit int) ([]domain.User, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT telegram_id, username, score, created_at FROM users ORDER BY score DESC LIMIT ?", limit)
	if err != nil { return nil, err }
	defer rows.Close()
	var users []domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.TelegramID, &u.Username, &u.Score, &u.CreatedAt); err != nil { return nil, err }
		users = append(users, u)
	}
	return users, nil
}
func (r *UserRepo) GetAllUsers(ctx context.Context) ([]domain.User, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT telegram_id, username, score, created_at FROM users")
	if err != nil { return nil, err }
	defer rows.Close()
	var users []domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.TelegramID, &u.Username, &u.Score, &u.CreatedAt); err != nil { return nil, err }
		users = append(users, u)
	}
	return users, nil
}
func (r *UserRepo) UpdateScore(ctx context.Context, telegramID int64, increment int) error {
	_, err := r.db.ExecContext(ctx, "UPDATE users SET score = score + ? WHERE telegram_id = ?", increment, telegramID)
	return err
}

type SettingsRepo struct { db *sql.DB }
func (r *SettingsRepo) Get(ctx context.Context, key string) (string, error) {
	var val string
	err := r.db.QueryRowContext(ctx, "SELECT value FROM settings WHERE key = ?", key).Scan(&val)
	if err == sql.ErrNoRows { return "", nil }
	return val, err
}
func (r *SettingsRepo) Set(ctx context.Context, key, value string) error {
	log.Printf("Setting key '%s' to '%s'", key, value)
	_, err := r.db.ExecContext(ctx, "INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value", key, value)
	return err
}
