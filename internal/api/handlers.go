package api

import (
	"encoding/json"
	"net/http"

	"github.com/mats/telegram-quiz-bot/internal/domain"
)

func (s *Server) handleSuggestSegments(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	segments, err := s.repos.Segments.GetAll(ctx)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch segments")
		return
	}

	var topics []string
	for _, seg := range segments {
		topics = append(topics, seg.Title)
	}

	suggestions, err := s.llm.SuggestSections(topics)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to generate suggestions: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, suggestions)
}

func (s *Server) handleGetPlan(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	segments, err := s.repos.Segments.GetAll(ctx)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch segments")
		return
	}

	// We'll build a custom struct to return the hierarchical plan
	type PlanSegment struct {
		Segment domain.Segment `json:"segment"`
		Quizzes []domain.Quiz  `json:"quizzes"`
	}

	var plan []PlanSegment
	for _, seg := range segments {
		quizzes, err := s.repos.Quizzes.GetBySegmentID(ctx, seg.ID)
		if err != nil {
			quizzes = []domain.Quiz{}
		}
		plan = append(plan, PlanSegment{
			Segment: seg,
			Quizzes: quizzes,
		})
	}

	respondJSON(w, http.StatusOK, plan)
}

func (s *Server) handleGetSegments(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	segments, err := s.repos.Segments.GetAll(ctx)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch segments")
		return
	}
	respondJSON(w, http.StatusOK, segments)
}

func (s *Server) handleCreateSegment(w http.ResponseWriter, r *http.Request) {
	var seg domain.Segment
	if err := json.NewDecoder(r.Body).Decode(&seg); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	ctx := r.Context()
	if err := s.repos.Segments.Create(ctx, &seg); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create segment")
		return
	}

	respondJSON(w, http.StatusCreated, seg)
}

func (s *Server) handleGetSegment(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid segment ID")
		return
	}

	ctx := r.Context()
	seg, err := s.repos.Segments.GetByID(ctx, id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Segment not found")
		return
	}

	respondJSON(w, http.StatusOK, seg)
}

func (s *Server) handleUpdateSegment(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid segment ID")
		return
	}

	var seg domain.Segment
	if err := json.NewDecoder(r.Body).Decode(&seg); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	seg.ID = id

	ctx := r.Context()
	if err := s.repos.Segments.Update(ctx, &seg); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update segment")
		return
	}

	respondJSON(w, http.StatusOK, seg)
}

func (s *Server) handleDeleteSegment(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid segment ID")
		return
	}

	ctx := r.Context()
	if err := s.repos.Segments.Delete(ctx, id); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete segment")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleGetSegmentQuizzes(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid segment ID")
		return
	}

	ctx := r.Context()
	quizzes, err := s.repos.Quizzes.GetBySegmentID(ctx, id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch quizzes")
		return
	}

	if quizzes == nil {
		quizzes = []domain.Quiz{}
	}
	respondJSON(w, http.StatusOK, quizzes)
}

func (s *Server) handleCreateSegmentQuiz(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid segment ID")
		return
	}

	var q domain.Quiz
	if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	q.SegmentID = id

	ctx := r.Context()
	if err := s.repos.Quizzes.Create(ctx, &q); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create quiz")
		return
	}

	respondJSON(w, http.StatusCreated, q)
}

func (s *Server) handleGetQuiz(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid quiz ID")
		return
	}

	ctx := r.Context()
	q, err := s.repos.Quizzes.GetByID(ctx, id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Quiz not found")
		return
	}

	respondJSON(w, http.StatusOK, q)
}

func (s *Server) handleUpdateQuiz(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid quiz ID")
		return
	}

	var q domain.Quiz
	if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	q.ID = id

	ctx := r.Context()
	// Fetch existing to preserve SegmentID if not provided
	existing, err := s.repos.Quizzes.GetByID(ctx, id)
	if err == nil && q.SegmentID == 0 {
		q.SegmentID = existing.SegmentID
	}

	if err := s.repos.Quizzes.Update(ctx, &q); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update quiz")
		return
	}

	respondJSON(w, http.StatusOK, q)
}

func (s *Server) handleDeleteQuiz(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid quiz ID")
		return
	}

	ctx := r.Context()
	if err := s.repos.Quizzes.Delete(ctx, id); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete quiz")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleGetQuizQuestions(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid quiz ID")
		return
	}

	ctx := r.Context()
	questions, err := s.repos.Questions.GetByQuizID(ctx, id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch questions")
		return
	}

	if questions == nil {
		questions = []domain.Question{}
	}
	respondJSON(w, http.StatusOK, questions)
}

func (s *Server) handleCreateQuizQuestion(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid quiz ID")
		return
	}

	var q domain.Question
	if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	q.QuizID = id

	ctx := r.Context()
	if err := s.repos.Questions.Create(ctx, &q); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create question")
		return
	}

	respondJSON(w, http.StatusCreated, q)
}

func (s *Server) handleGetQuestion(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid question ID")
		return
	}

	ctx := r.Context()
	q, err := s.repos.Questions.GetByID(ctx, id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Question not found")
		return
	}

	respondJSON(w, http.StatusOK, q)
}

func (s *Server) handleUpdateQuestion(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid question ID")
		return
	}

	var q domain.Question
	if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	q.ID = id

	ctx := r.Context()
	// Fetch existing to preserve QuizID if not provided
	existing, err := s.repos.Questions.GetByID(ctx, id)
	if err == nil && q.QuizID == 0 {
		q.QuizID = existing.QuizID
	}

	if err := s.repos.Questions.Update(ctx, &q); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update question")
		return
	}

	respondJSON(w, http.StatusOK, q)
}

func (s *Server) handleDeleteQuestion(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid question ID")
		return
	}

	ctx := r.Context()
	if err := s.repos.Questions.Delete(ctx, id); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete question")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
