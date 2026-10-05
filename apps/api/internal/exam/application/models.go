package application

import (
	"encoding/json"
	"time"
)

type ExamSummary struct {
	ID              string    `json:"id"`
	Slug            string    `json:"slug"`
	Title           string    `json:"title"`
	Description     string    `json:"description"`
	Skill           string    `json:"skill"`
	DurationSeconds int32     `json:"duration_seconds"`
	QuestionCount   int32     `json:"question_count"`
	PublishedAt     time.Time `json:"published_at"`
}

type Exam struct {
	ExamSummary
	VersionID    string    `json:"version_id"`
	Version      int32     `json:"version"`
	Instructions string    `json:"instructions"`
	Sections     []Section `json:"sections"`
}

type Section struct {
	ID           string     `json:"id"`
	Position     int32      `json:"position"`
	Title        string     `json:"title"`
	Instructions string     `json:"instructions"`
	AudioURL     string     `json:"audio_url,omitempty"`
	Passage      string     `json:"passage,omitempty"`
	Questions    []Question `json:"questions"`
}

type Question struct {
	ID       string          `json:"id"`
	Position int32           `json:"position"`
	Type     string          `json:"type"`
	Prompt   string          `json:"prompt"`
	Points   float64         `json:"points"`
	Config   json.RawMessage `json:"config"`
	Options  []Option        `json:"options"`
}

type Option struct {
	ID       string `json:"id"`
	Position int32  `json:"position"`
	Label    string `json:"label"`
	Value    string `json:"value"`
}

type Attempt struct {
	ID            string        `json:"id"`
	ExamVersionID string        `json:"exam_version_id"`
	Status        string        `json:"status"`
	StartedAt     time.Time     `json:"started_at"`
	DeadlineAt    time.Time     `json:"deadline_at"`
	SubmittedAt   *time.Time    `json:"submitted_at,omitempty"`
	RawScore      *float64      `json:"raw_score,omitempty"`
	BandScore     *float64      `json:"band_score,omitempty"`
	QuestionCount int32         `json:"question_count"`
	Answers       []SavedAnswer `json:"answers"`
}

type SavedAnswer struct {
	QuestionID string          `json:"question_id"`
	Values     []string        `json:"values"`
	Revision   int32           `json:"revision"`
	SavedAt    time.Time       `json:"saved_at"`
	IsCorrect  *bool           `json:"is_correct,omitempty"`
	Points     *float64        `json:"awarded_points,omitempty"`
	Raw        json.RawMessage `json:"-"`
}

type HistoryItem struct {
	Attempt
	Title string `json:"title"`
	Skill string `json:"skill"`
}

type SkillProgress struct {
	Skill             string  `json:"skill"`
	CompletedAttempts int32   `json:"completed_attempts"`
	AverageBand       float64 `json:"average_band"`
	BestBand          float64 `json:"best_band"`
	LatestBand        float64 `json:"latest_band"`
	CorrectAnswers    float64 `json:"correct_answers"`
	TotalQuestions    int32   `json:"total_questions"`
}

// PracticeQuestion is a published question with the source material needed to
// answer it. Answer keys are deliberately returned only after an answer check.
type PracticeQuestion struct {
	ID                  string          `json:"id"`
	Position            int32           `json:"position"`
	Type                string          `json:"type"`
	Prompt              string          `json:"prompt"`
	Points              float64         `json:"points"`
	Config              json.RawMessage `json:"config"`
	Options             []Option        `json:"options"`
	SectionTitle        string          `json:"section_title"`
	SectionInstructions string          `json:"section_instructions"`
	AudioURL            string          `json:"audio_url,omitempty"`
	Passage             string          `json:"passage,omitempty"`
	ExamTitle           string          `json:"exam_title"`
	Skill               string          `json:"skill"`
}

type PracticeFeedback struct {
	QuestionID     string   `json:"question_id"`
	Correct        bool     `json:"correct"`
	AcceptedAnswer []string `json:"accepted_answers"`
	Explanation    string   `json:"explanation"`
}

type AttemptReview struct {
	AttemptID string           `json:"attempt_id"`
	ExamTitle string           `json:"exam_title"`
	Skill     string           `json:"skill"`
	BandScore float64          `json:"band_score"`
	Questions []ReviewQuestion `json:"questions"`
}

type ReviewQuestion struct {
	ID              string   `json:"id"`
	Position        int32    `json:"position"`
	SectionTitle    string   `json:"section_title"`
	Type            string   `json:"type"`
	Prompt          string   `json:"prompt"`
	SubmittedAnswer []string `json:"submitted_answers"`
	AcceptedAnswer  []string `json:"accepted_answers"`
	Correct         bool     `json:"correct"`
	Explanation     string   `json:"explanation"`
}

type StudyProfile struct {
	TargetBand       float64 `json:"target_band"`
	ExamDate         string  `json:"exam_date,omitempty"`
	Track            string  `json:"track"`
	DailyGoalMinutes int32   `json:"daily_goal_minutes"`
}

type StudyProfileInput struct {
	TargetBand       float64 `json:"target_band"`
	ExamDate         string  `json:"exam_date"`
	Track            string  `json:"track"`
	DailyGoalMinutes int32   `json:"daily_goal_minutes"`
}

type StudyRecommendation struct {
	Skill        string `json:"skill"`
	QuestionType string `json:"question_type"`
	Reason       string `json:"reason"`
}

type DailyStudyActivity struct {
	Date            string `json:"date"`
	PracticeAnswers int32  `json:"practice_answers"`
	MocksCompleted  int32  `json:"mocks_completed"`
}

type StudyPlan struct {
	Profile          StudyProfile         `json:"profile"`
	StreakDays       int32                `json:"streak_days"`
	PracticeToday    int32                `json:"practice_today"`
	MocksToday       int32                `json:"mocks_today"`
	MocksThisWeek    int32                `json:"mocks_this_week"`
	ReviewsToday     int32                `json:"reviews_today"`
	EstimatedMinutes int32                `json:"estimated_minutes_today"`
	Recommendation   StudyRecommendation  `json:"recommendation"`
	Week             []DailyStudyActivity `json:"week"`
}
