package domain

import (
	"errors"
	"time"
)

type Skill string

const (
	SkillListening Skill = "listening"
	SkillReading   Skill = "reading"
)

type Status string

const (
	StatusDraft     Status = "draft"
	StatusPublished Status = "published"
	StatusArchived  Status = "archived"
)

type Exam struct {
	ID              string
	Title           string
	Skill           Skill
	Status          Status
	Duration        time.Duration
	Version         int
	QuestionCount   int
	BandConversions []BandConversion
}

type BandConversion struct {
	MinimumRawScore int
	Band            float32
}

func (e Exam) ValidateForPublishing() error {
	if e.Title == "" {
		return errors.New("exam title is required")
	}
	if e.Skill != SkillListening && e.Skill != SkillReading {
		return errors.New("only listening and reading exams can be published")
	}
	if e.Duration <= 0 || e.QuestionCount <= 0 {
		return errors.New("exam duration and questions are required")
	}
	if len(e.BandConversions) == 0 {
		return errors.New("band conversion table is required")
	}
	return nil
}
