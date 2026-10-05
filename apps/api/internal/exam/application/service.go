package application

import (
	"context"
	"errors"
	"strings"
)

var (
	ErrNotFound      = errors.New("exam or attempt not found")
	ErrAttemptClosed = errors.New("attempt is closed or its time has expired")
	ErrInvalidAnswer = errors.New("answer must contain between one and ten values")
)

type Repository interface {
	ListExams(ctx context.Context) ([]ExamSummary, error)
	GetExam(ctx context.Context, examID string) (Exam, error)
	StartAttempt(ctx context.Context, userID, versionID string) (Attempt, error)
	GetAttempt(ctx context.Context, userID, attemptID string) (Attempt, error)
	SaveAnswer(ctx context.Context, userID, attemptID, questionID string, values []string) (SavedAnswer, error)
	SubmitAttempt(ctx context.Context, userID, attemptID string) (Attempt, error)
	ListHistory(ctx context.Context, userID string, limit, offset int32) ([]HistoryItem, error)
	GetProgress(ctx context.Context, userID string) ([]SkillProgress, error)
}

func (s *Service) GetProgress(ctx context.Context, userID string) ([]SkillProgress, error) {
	return s.repository.GetProgress(ctx, userID)
}

type Service struct{ repository Repository }

func NewService(repository Repository) *Service { return &Service{repository: repository} }

func (s *Service) ListExams(ctx context.Context) ([]ExamSummary, error) {
	return s.repository.ListExams(ctx)
}

func (s *Service) GetExam(ctx context.Context, examID string) (Exam, error) {
	return s.repository.GetExam(ctx, examID)
}

func (s *Service) StartAttempt(ctx context.Context, userID, examID string) (Attempt, error) {
	exam, err := s.repository.GetExam(ctx, examID)
	if err != nil {
		return Attempt{}, err
	}
	return s.repository.StartAttempt(ctx, userID, exam.VersionID)
}

func (s *Service) GetAttempt(ctx context.Context, userID, attemptID string) (Attempt, error) {
	return s.repository.GetAttempt(ctx, userID, attemptID)
}

func (s *Service) SaveAnswer(ctx context.Context, userID, attemptID, questionID string, values []string) (SavedAnswer, error) {
	if len(values) == 0 || len(values) > 10 {
		return SavedAnswer{}, ErrInvalidAnswer
	}
	cleaned := make([]string, len(values))
	for i, value := range values {
		cleaned[i] = strings.TrimSpace(value)
		if cleaned[i] == "" || len(cleaned[i]) > 500 {
			return SavedAnswer{}, ErrInvalidAnswer
		}
	}
	return s.repository.SaveAnswer(ctx, userID, attemptID, questionID, cleaned)
}

func (s *Service) SubmitAttempt(ctx context.Context, userID, attemptID string) (Attempt, error) {
	return s.repository.SubmitAttempt(ctx, userID, attemptID)
}

func (s *Service) ListHistory(ctx context.Context, userID string, limit, offset int32) ([]HistoryItem, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return s.repository.ListHistory(ctx, userID, limit, offset)
}
