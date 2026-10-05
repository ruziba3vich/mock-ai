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

var questionTypes = map[string]struct{}{
	"multiple_choice": {}, "multiple_select": {}, "true_false_not_given": {},
	"yes_no_not_given": {}, "short_answer": {}, "sentence_completion": {},
	"summary_completion": {}, "matching": {}, "map_labeling": {},
}

type Repository interface {
	ListExams(ctx context.Context) ([]ExamSummary, error)
	GetExam(ctx context.Context, examID string) (Exam, error)
	StartAttempt(ctx context.Context, userID, versionID string) (Attempt, error)
	GetAttempt(ctx context.Context, userID, attemptID string) (Attempt, error)
	SaveAnswer(ctx context.Context, userID, attemptID, questionID string, values []string) (SavedAnswer, error)
	SubmitAttempt(ctx context.Context, userID, attemptID string) (Attempt, error)
	ListHistory(ctx context.Context, userID string, limit, offset int32) ([]HistoryItem, error)
	GetProgress(ctx context.Context, userID string) ([]SkillProgress, error)
	ListPracticeQuestions(ctx context.Context, skill, questionType string, limit int32) ([]PracticeQuestion, error)
	CheckPracticeAnswer(ctx context.Context, questionID string, values []string) (PracticeFeedback, error)
	GetAttemptReview(ctx context.Context, userID, attemptID string) (AttemptReview, error)
}

func (s *Service) GetProgress(ctx context.Context, userID string) ([]SkillProgress, error) {
	return s.repository.GetProgress(ctx, userID)
}

func (s *Service) ListPracticeQuestions(ctx context.Context, skill, questionType string, limit int32) ([]PracticeQuestion, error) {
	if skill != "" && skill != "reading" && skill != "listening" {
		return nil, ErrNotFound
	}
	if questionType != "" {
		if _, ok := questionTypes[questionType]; !ok {
			return nil, ErrNotFound
		}
	}
	if limit <= 0 || limit > 10 {
		limit = 5
	}
	return s.repository.ListPracticeQuestions(ctx, skill, questionType, limit)
}

func (s *Service) CheckPracticeAnswer(ctx context.Context, questionID string, values []string) (PracticeFeedback, error) {
	cleaned, err := cleanAnswers(values)
	if err != nil {
		return PracticeFeedback{}, err
	}
	return s.repository.CheckPracticeAnswer(ctx, questionID, cleaned)
}

func (s *Service) GetAttemptReview(ctx context.Context, userID, attemptID string) (AttemptReview, error) {
	return s.repository.GetAttemptReview(ctx, userID, attemptID)
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
	cleaned, err := cleanAnswers(values)
	if err != nil {
		return SavedAnswer{}, err
	}
	return s.repository.SaveAnswer(ctx, userID, attemptID, questionID, cleaned)
}

func cleanAnswers(values []string) ([]string, error) {
	if len(values) == 0 || len(values) > 10 {
		return nil, ErrInvalidAnswer
	}
	cleaned := make([]string, len(values))
	for i, value := range values {
		cleaned[i] = strings.TrimSpace(value)
		if cleaned[i] == "" || len(cleaned[i]) > 500 {
			return nil, ErrInvalidAnswer
		}
	}
	return cleaned, nil
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
