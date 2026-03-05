package repository

import (
	"context"

	"github.com/mats/telegram-quiz-bot/internal/domain"
)

// SegmentRepository handles data operations for learning segments.
type SegmentRepository interface {
	Create(ctx context.Context, segment *domain.Segment) error
	GetByID(ctx context.Context, id int) (*domain.Segment, error)
	GetAll(ctx context.Context) ([]domain.Segment, error)
	Update(ctx context.Context, segment *domain.Segment) error
	Delete(ctx context.Context, id int) error
}

// QuizRepository handles data operations for quizzes within segments.
type QuizRepository interface {
	Create(ctx context.Context, quiz *domain.Quiz) error
	GetByID(ctx context.Context, id int) (*domain.Quiz, error)
	GetBySegmentID(ctx context.Context, segmentID int) ([]domain.Quiz, error)
	Update(ctx context.Context, quiz *domain.Quiz) error
	Delete(ctx context.Context, id int) error
}

// QuestionRepository handles data operations for individual questions.
type QuestionRepository interface {
	Create(ctx context.Context, question *domain.Question) error
	GetByID(ctx context.Context, id int) (*domain.Question, error)
	GetByQuizID(ctx context.Context, quizID int) ([]domain.Question, error)
	Update(ctx context.Context, question *domain.Question) error
	Delete(ctx context.Context, id int) error
	GetNextUnanswered(ctx context.Context, telegramID int64, quizID int) (*domain.Question, error)
	GetUnansweredCount(ctx context.Context, telegramID int64, quizID int) (int, error)
	GetRecentForQuiz(ctx context.Context, quizID int, limit int) ([]string, error)
	RecordAnswer(ctx context.Context, questionID int, telegramID int64, isCorrect bool) error
}

// UserRepository handles user data and scores.
type UserRepository interface {
	RegisterUser(ctx context.Context, telegramID int64, username string) error
	GetTopUsers(ctx context.Context, limit int) ([]domain.User, error)
	GetAllUsers(ctx context.Context) ([]domain.User, error)
	UpdateScore(ctx context.Context, telegramID int64, increment int) error
}

// SettingsRepository handles application settings/state.
type SettingsRepository interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string) error
}

// Repositories aggregates all repositories.
type Repositories struct {
	Segments  SegmentRepository
	Quizzes   QuizRepository
	Questions QuestionRepository
	Users     UserRepository
	Settings  SettingsRepository
}
