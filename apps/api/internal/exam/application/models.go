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
