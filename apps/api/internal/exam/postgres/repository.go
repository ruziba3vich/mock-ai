package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/own/mock-ai/apps/api/internal/exam/application"
	db "github.com/own/mock-ai/apps/api/internal/platform/postgres/sqlc"
)

type Repository struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool, queries: db.New(pool)}
}

func (r *Repository) ListExams(ctx context.Context) ([]application.ExamSummary, error) {
	rows, err := r.queries.ListPublishedExams(ctx)
	if err != nil {
		return nil, fmt.Errorf("list published exams: %w", err)
	}
	exams := make([]application.ExamSummary, len(rows))
	for i, row := range rows {
		exams[i] = application.ExamSummary{
			ID:              uuidString(row.ID),
			Slug:            row.Slug,
			Title:           row.Title,
			Description:     row.Description,
			Skill:           string(row.Skill),
			DurationSeconds: row.DurationSeconds,
			QuestionCount:   row.QuestionCount,
			PublishedAt:     row.PublishedAt.Time,
		}
	}
	return exams, nil
}

func (r *Repository) GetExam(ctx context.Context, examID string) (application.Exam, error) {
	id, err := parseUUID(examID)
	if err != nil {
		return application.Exam{}, application.ErrNotFound
	}
	row, err := r.queries.GetPublishedExam(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return application.Exam{}, application.ErrNotFound
	}
	if err != nil {
		return application.Exam{}, fmt.Errorf("get published exam: %w", err)
	}
	sections, err := r.loadSections(ctx, row.VersionID)
	if err != nil {
		return application.Exam{}, err
	}
	questionCount := int32(0)
	for _, section := range sections {
		questionCount += int32(len(section.Questions))
	}
	return application.Exam{
		ExamSummary: application.ExamSummary{
			ID:              uuidString(row.ID),
			Slug:            row.Slug,
			Title:           row.Title,
			Description:     row.Description,
			Skill:           string(row.Skill),
			DurationSeconds: row.DurationSeconds,
			QuestionCount:   questionCount,
			PublishedAt:     row.PublishedAt.Time,
		},
		VersionID:    uuidString(row.VersionID),
		Version:      row.Version,
		Instructions: row.Instructions,
		Sections:     sections,
	}, nil
}

func (r *Repository) loadSections(ctx context.Context, versionID pgtype.UUID) ([]application.Section, error) {
	rows, err := r.queries.ListExamSections(ctx, versionID)
	if err != nil {
		return nil, fmt.Errorf("list exam sections: %w", err)
	}
	sections := make([]application.Section, 0, len(rows))
	for _, row := range rows {
		questionRows, err := r.queries.ListSectionQuestions(ctx, row.ID)
		if err != nil {
			return nil, fmt.Errorf("list section questions: %w", err)
		}
		questions := make([]application.Question, 0, len(questionRows))
		for _, questionRow := range questionRows {
			optionRows, err := r.queries.ListQuestionOptions(ctx, questionRow.ID)
			if err != nil {
				return nil, fmt.Errorf("list question options: %w", err)
			}
			options := make([]application.Option, len(optionRows))
			for i, option := range optionRows {
				options[i] = application.Option{ID: uuidString(option.ID), Position: option.Position, Label: option.Label, Value: option.Value}
			}
			questions = append(questions, application.Question{
				ID:       uuidString(questionRow.ID),
				Position: questionRow.Position,
				Type:     string(questionRow.Type),
				Prompt:   questionRow.Prompt,
				Points:   numericFloat(questionRow.Points),
				Config:   questionRow.Config,
				Options:  options,
			})
		}
		sections = append(sections, application.Section{
			ID:           uuidString(row.ID),
			Position:     row.Position,
			Title:        row.Title,
			Instructions: row.Instructions,
			AudioURL:     textString(row.AudioUrl),
			Passage:      textString(row.Passage),
			Questions:    questions,
		})
	}
	return sections, nil
}

func (r *Repository) StartAttempt(ctx context.Context, userID, versionID string) (application.Attempt, error) {
	userUUID, err := parseUUID(userID)
	if err != nil {
		return application.Attempt{}, application.ErrNotFound
	}
	versionUUID, err := parseUUID(versionID)
	if err != nil {
		return application.Attempt{}, application.ErrNotFound
	}
	params := db.GetActiveAttemptParams{UserID: userUUID, ExamVersionID: versionUUID}
	if err := r.queries.ExpireStaleAttemptsForExam(ctx, db.ExpireStaleAttemptsForExamParams(params)); err != nil {
		return application.Attempt{}, fmt.Errorf("expire stale attempt: %w", err)
	}
	active, err := r.queries.GetActiveAttempt(ctx, params)
	if err == nil {
		return r.attemptWithAnswers(ctx, r.queries, active)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return application.Attempt{}, fmt.Errorf("get active attempt: %w", err)
	}
	created, err := r.queries.CreateAttempt(ctx, db.CreateAttemptParams{UserID: userUUID, ID: versionUUID})
	if errors.Is(err, pgx.ErrNoRows) {
		return application.Attempt{}, application.ErrNotFound
	}
	if err != nil {
		return application.Attempt{}, fmt.Errorf("create attempt: %w", err)
	}
	return toAttempt(created, nil), nil
}

func (r *Repository) GetAttempt(ctx context.Context, userID, attemptID string) (application.Attempt, error) {
	params, err := attemptParams(userID, attemptID)
	if err != nil {
		return application.Attempt{}, application.ErrNotFound
	}
	attempt, err := r.queries.GetAttempt(ctx, params)
	if errors.Is(err, pgx.ErrNoRows) {
		return application.Attempt{}, application.ErrNotFound
	}
	if err != nil {
		return application.Attempt{}, fmt.Errorf("get attempt: %w", err)
	}
	return r.attemptWithAnswers(ctx, r.queries, attempt)
}

func (r *Repository) SaveAnswer(ctx context.Context, userID, attemptID, questionID string, values []string) (application.SavedAnswer, error) {
	userUUID, err := parseUUID(userID)
	if err != nil {
		return application.SavedAnswer{}, application.ErrNotFound
	}
	attemptUUID, err := parseUUID(attemptID)
	if err != nil {
		return application.SavedAnswer{}, application.ErrNotFound
	}
	questionUUID, err := parseUUID(questionID)
	if err != nil {
		return application.SavedAnswer{}, application.ErrNotFound
	}
	answerJSON, _ := json.Marshal(map[string][]string{"values": values})
	answer, err := r.queries.SaveAttemptAnswer(ctx, db.SaveAttemptAnswerParams{
		AttemptID: attemptUUID, QuestionID: questionUUID, Answer: answerJSON, UserID: userUUID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return application.SavedAnswer{}, application.ErrAttemptClosed
	}
	if err != nil {
		return application.SavedAnswer{}, fmt.Errorf("save attempt answer: %w", err)
	}
	return toSavedAnswer(answer), nil
}

func (r *Repository) SubmitAttempt(ctx context.Context, userID, attemptID string) (application.Attempt, error) {
	params, err := attemptParams(userID, attemptID)
	if err != nil {
		return application.Attempt{}, application.ErrNotFound
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return application.Attempt{}, fmt.Errorf("begin scoring: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := r.queries.WithTx(tx)
	attempt, err := q.GetAttemptForUpdate(ctx, db.GetAttemptForUpdateParams(params))
	if errors.Is(err, pgx.ErrNoRows) {
		return application.Attempt{}, application.ErrNotFound
	}
	if err != nil {
		return application.Attempt{}, fmt.Errorf("lock attempt: %w", err)
	}
	if attempt.Status == db.AttemptStatusScored {
		if err := tx.Commit(ctx); err != nil {
			return application.Attempt{}, err
		}
		return r.attemptWithAnswers(ctx, r.queries, attempt)
	}
	if attempt.Status != db.AttemptStatusInProgress {
		return application.Attempt{}, application.ErrAttemptClosed
	}

	rows, err := q.ListAttemptScoringRows(ctx, attempt.ID)
	if err != nil {
		return application.Attempt{}, fmt.Errorf("load answer key: %w", err)
	}
	rawScore, err := scoreRows(ctx, q, attempt.ID, rows)
	if err != nil {
		return application.Attempt{}, err
	}
	rawNumeric := numeric(rawScore)
	band, err := q.GetBandForRawScore(ctx, db.GetBandForRawScoreParams{
		ExamVersionID:   attempt.ExamVersionID,
		MinimumRawScore: rawNumeric,
	})
	if err != nil {
		return application.Attempt{}, fmt.Errorf("find band score: %w", err)
	}
	completed, err := q.CompleteAttempt(ctx, db.CompleteAttemptParams{
		ID: attempt.ID, UserID: attempt.UserID, RawScore: rawNumeric, BandScore: band,
	})
	if err != nil {
		return application.Attempt{}, fmt.Errorf("complete attempt: %w", err)
	}
	answers, err := q.ListAttemptAnswers(ctx, attempt.ID)
	if err != nil {
		return application.Attempt{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return application.Attempt{}, fmt.Errorf("commit score: %w", err)
	}
	return toAttempt(completed, answers), nil
}

func (r *Repository) ListHistory(ctx context.Context, userID string, limit, offset int32) ([]application.HistoryItem, error) {
	userUUID, err := parseUUID(userID)
	if err != nil {
		return nil, application.ErrNotFound
	}
	rows, err := r.queries.ListStudentAttempts(ctx, db.ListStudentAttemptsParams{UserID: userUUID, Limit: limit, Offset: offset})
	if err != nil {
		return nil, fmt.Errorf("list attempt history: %w", err)
	}
	history := make([]application.HistoryItem, len(rows))
	for i, row := range rows {
		history[i] = application.HistoryItem{
			Attempt: toAttempt(db.Attempt{
				ID: row.ID, UserID: row.UserID, ExamVersionID: row.ExamVersionID, Status: row.Status,
				StartedAt: row.StartedAt, DeadlineAt: row.DeadlineAt, SubmittedAt: row.SubmittedAt,
				ScoredAt: row.ScoredAt, RawScore: row.RawScore, BandScore: row.BandScore, QuestionCount: row.QuestionCount,
			}, nil),
			Title: row.Title,
			Skill: string(row.Skill),
		}
	}
	return history, nil
}

func (r *Repository) GetProgress(ctx context.Context, userID string) ([]application.SkillProgress, error) {
	userUUID, err := parseUUID(userID)
	if err != nil {
		return nil, application.ErrNotFound
	}
	rows, err := r.queries.GetStudentProgress(ctx, userUUID)
	if err != nil {
		return nil, fmt.Errorf("get student progress: %w", err)
	}
	progress := make([]application.SkillProgress, len(rows))
	for i, row := range rows {
		progress[i] = application.SkillProgress{
			Skill:             string(row.Skill),
			CompletedAttempts: row.CompletedAttempts,
			AverageBand:       numericFloat(row.AverageBand),
			BestBand:          numericFloat(row.BestBand),
			LatestBand:        numericFloat(row.LatestBand),
			CorrectAnswers:    numericFloat(row.CorrectAnswers),
			TotalQuestions:    row.TotalQuestions,
		}
	}
	return progress, nil
}

func (r *Repository) ListPracticeQuestions(ctx context.Context, skill, questionType string, limit int32) ([]application.PracticeQuestion, error) {
	rows, err := r.queries.ListPracticeQuestions(ctx, db.ListPracticeQuestionsParams{
		Skill: skill, QuestionType: questionType, ResultLimit: limit,
	})
	if err != nil {
		return nil, fmt.Errorf("list practice questions: %w", err)
	}
	questions := make([]application.PracticeQuestion, 0, len(rows))
	for _, row := range rows {
		optionRows, err := r.queries.ListQuestionOptions(ctx, row.ID)
		if err != nil {
			return nil, fmt.Errorf("list practice question options: %w", err)
		}
		options := make([]application.Option, len(optionRows))
		for i, option := range optionRows {
			options[i] = application.Option{ID: uuidString(option.ID), Position: option.Position, Label: option.Label, Value: option.Value}
		}
		questions = append(questions, application.PracticeQuestion{
			ID: uuidString(row.ID), Position: row.Position, Type: string(row.Type), Prompt: row.Prompt,
			Points: numericFloat(row.Points), Config: row.Config, Options: options,
			SectionTitle: row.SectionTitle, SectionInstructions: row.SectionInstructions,
			AudioURL: textString(row.AudioUrl), Passage: textString(row.Passage),
			ExamTitle: row.ExamTitle, Skill: string(row.Skill),
		})
	}
	return questions, nil
}

func (r *Repository) CheckPracticeAnswer(ctx context.Context, questionID string, values []string) (application.PracticeFeedback, error) {
	id, err := parseUUID(questionID)
	if err != nil {
		return application.PracticeFeedback{}, application.ErrNotFound
	}
	rows, err := r.queries.GetPracticeAnswerKey(ctx, id)
	if err != nil {
		return application.PracticeFeedback{}, fmt.Errorf("get practice answer key: %w", err)
	}
	if len(rows) == 0 {
		return application.PracticeFeedback{}, application.ErrNotFound
	}
	accepted := make([]acceptedAnswer, len(rows))
	answerValues := make([]string, len(rows))
	for i, row := range rows {
		accepted[i] = acceptedAnswer{value: row.Value, caseSensitive: row.IsCaseSensitive}
		answerValues[i] = row.Value
	}
	return application.PracticeFeedback{
		QuestionID:     questionID,
		Correct:        answerCorrect(rows[0].Type, values, accepted),
		AcceptedAnswer: answerValues,
		Explanation:    rows[0].Explanation,
	}, nil
}

func (r *Repository) GetAttemptReview(ctx context.Context, userID, attemptID string) (application.AttemptReview, error) {
	params, err := attemptParams(userID, attemptID)
	if err != nil {
		return application.AttemptReview{}, application.ErrNotFound
	}
	rows, err := r.queries.GetAttemptReview(ctx, db.GetAttemptReviewParams{ID: params.ID, UserID: params.UserID})
	if err != nil {
		return application.AttemptReview{}, fmt.Errorf("get attempt review: %w", err)
	}
	if len(rows) == 0 {
		return application.AttemptReview{}, application.ErrNotFound
	}
	review := application.AttemptReview{
		AttemptID: attemptID, ExamTitle: rows[0].ExamTitle, Skill: string(rows[0].Skill),
		BandScore: numericFloat(rows[0].BandScore), Questions: make([]application.ReviewQuestion, 0),
	}
	byQuestion := make(map[string]int)
	for _, row := range rows {
		key := uuidString(row.QuestionID)
		index, exists := byQuestion[key]
		if !exists {
			var submitted struct {
				Values []string `json:"values"`
			}
			if len(row.Answer) > 0 {
				_ = json.Unmarshal(row.Answer, &submitted)
			}
			index = len(review.Questions)
			byQuestion[key] = index
			review.Questions = append(review.Questions, application.ReviewQuestion{
				ID: key, Position: row.QuestionPosition, SectionTitle: row.SectionTitle,
				Type: string(row.QuestionType), Prompt: row.Prompt, SubmittedAnswer: submitted.Values,
				AcceptedAnswer: []string{}, Correct: row.IsCorrect.Valid && row.IsCorrect.Bool,
				Explanation: row.Explanation,
			})
		}
		review.Questions[index].AcceptedAnswer = append(review.Questions[index].AcceptedAnswer, row.AcceptedValue)
	}
	return review, nil
}

type scoreQuestion struct {
	id           pgtype.UUID
	questionType db.QuestionType
	points       float64
	accepted     []acceptedAnswer
	answer       []byte
}

type acceptedAnswer struct {
	value         string
	caseSensitive bool
}

func scoreRows(ctx context.Context, q *db.Queries, attemptID pgtype.UUID, rows []db.ListAttemptScoringRowsRow) (float64, error) {
	grouped := make([]scoreQuestion, 0)
	byID := make(map[string]int)
	for _, row := range rows {
		key := uuidString(row.QuestionID)
		index, exists := byID[key]
		if !exists {
			index = len(grouped)
			byID[key] = index
			grouped = append(grouped, scoreQuestion{id: row.QuestionID, questionType: row.QuestionType, points: numericFloat(row.Points), answer: row.Answer})
		}
		if row.AcceptedValue.Valid {
			grouped[index].accepted = append(grouped[index].accepted, acceptedAnswer{value: row.AcceptedValue.String, caseSensitive: row.IsCaseSensitive.Bool})
		}
	}
	total := 0.0
	for _, question := range grouped {
		if len(question.accepted) == 0 {
			return 0, fmt.Errorf("question %s has no answer key", uuidString(question.id))
		}
		var submitted struct {
			Values []string `json:"values"`
		}
		if len(question.answer) > 0 {
			_ = json.Unmarshal(question.answer, &submitted)
		}
		correct := answerCorrect(question.questionType, submitted.Values, question.accepted)
		awarded := 0.0
		if correct {
			awarded = question.points
			total += awarded
		}
		if len(question.answer) > 0 {
			if err := q.SetAttemptAnswerScore(ctx, db.SetAttemptAnswerScoreParams{
				AttemptID: attemptID, QuestionID: question.id,
				IsCorrect: pgtype.Bool{Bool: correct, Valid: true}, AwardedPoints: numeric(awarded),
			}); err != nil {
				return 0, fmt.Errorf("store answer score: %w", err)
			}
		}
	}
	return total, nil
}

func answerCorrect(questionType db.QuestionType, values []string, accepted []acceptedAnswer) bool {
	if questionType == db.QuestionTypeMultipleSelect {
		if len(values) != len(accepted) {
			return false
		}
		used := make([]bool, len(values))
		for _, expected := range accepted {
			matched := false
			for i, actual := range values {
				if !used[i] && normalize(actual, expected.caseSensitive) == normalize(expected.value, expected.caseSensitive) {
					used[i], matched = true, true
					break
				}
			}
			if !matched {
				return false
			}
		}
		return true
	}
	if len(values) != 1 {
		return false
	}
	for _, expected := range accepted {
		if normalize(values[0], expected.caseSensitive) == normalize(expected.value, expected.caseSensitive) {
			return true
		}
	}
	return false
}

func normalize(value string, caseSensitive bool) string {
	value = strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if !caseSensitive {
		return strings.ToLower(value)
	}
	return value
}

func (r *Repository) attemptWithAnswers(ctx context.Context, q *db.Queries, attempt db.Attempt) (application.Attempt, error) {
	answers, err := q.ListAttemptAnswers(ctx, attempt.ID)
	if err != nil {
		return application.Attempt{}, fmt.Errorf("list attempt answers: %w", err)
	}
	return toAttempt(attempt, answers), nil
}

func toAttempt(row db.Attempt, answers []db.AttemptAnswer) application.Attempt {
	result := application.Attempt{
		ID: uuidString(row.ID), ExamVersionID: uuidString(row.ExamVersionID), Status: string(row.Status),
		StartedAt: row.StartedAt.Time, DeadlineAt: row.DeadlineAt.Time, QuestionCount: row.QuestionCount,
		Answers: make([]application.SavedAnswer, len(answers)),
	}
	if row.SubmittedAt.Valid {
		value := row.SubmittedAt.Time
		result.SubmittedAt = &value
	}
	if row.RawScore.Valid {
		value := numericFloat(row.RawScore)
		result.RawScore = &value
	}
	if row.BandScore.Valid {
		value := numericFloat(row.BandScore)
		result.BandScore = &value
	}
	for i, answer := range answers {
		result.Answers[i] = toSavedAnswer(answer)
	}
	return result
}

func toSavedAnswer(row db.AttemptAnswer) application.SavedAnswer {
	var payload struct {
		Values []string `json:"values"`
	}
	_ = json.Unmarshal(row.Answer, &payload)
	result := application.SavedAnswer{QuestionID: uuidString(row.QuestionID), Values: payload.Values, Revision: row.Revision, SavedAt: row.SavedAt.Time}
	if row.IsCorrect.Valid {
		value := row.IsCorrect.Bool
		result.IsCorrect = &value
	}
	if row.AwardedPoints.Valid {
		value := numericFloat(row.AwardedPoints)
		result.Points = &value
	}
	return result
}

func attemptParams(userID, attemptID string) (db.GetAttemptParams, error) {
	userUUID, err := parseUUID(userID)
	if err != nil {
		return db.GetAttemptParams{}, err
	}
	attemptUUID, err := parseUUID(attemptID)
	if err != nil {
		return db.GetAttemptParams{}, err
	}
	return db.GetAttemptParams{ID: attemptUUID, UserID: userUUID}, nil
}

func parseUUID(value string) (pgtype.UUID, error) {
	parsed, err := uuid.Parse(value)
	if err != nil {
		return pgtype.UUID{}, err
	}
	return pgtype.UUID{Bytes: parsed, Valid: true}, nil
}

func uuidString(value pgtype.UUID) string {
	if !value.Valid {
		return ""
	}
	return uuid.UUID(value.Bytes).String()
}
func textString(value pgtype.Text) string {
	if !value.Valid {
		return ""
	}
	return value.String
}
func numericFloat(value pgtype.Numeric) float64 {
	parsed, _ := value.Float64Value()
	return parsed.Float64
}
func numeric(value float64) pgtype.Numeric {
	var result pgtype.Numeric
	_ = result.Scan(strconv.FormatFloat(value, 'f', 2, 64))
	return result
}
