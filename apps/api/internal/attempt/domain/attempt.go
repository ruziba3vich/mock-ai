package domain

import (
	"errors"
	"time"
)

type Status string

const (
	StatusInProgress Status = "in_progress"
	StatusSubmitted  Status = "submitted"
	StatusExpired    Status = "expired"
)

var (
	ErrAttemptClosed = errors.New("attempt is already closed")
	ErrTimeExpired   = errors.New("attempt time has expired")
)

type Attempt struct {
	ID          string
	StudentID   string
	ExamID      string
	Status      Status
	StartedAt   time.Time
	DeadlineAt  time.Time
	SubmittedAt *time.Time
}

func (a Attempt) CanSaveAnswer(now time.Time) error {
	if a.Status != StatusInProgress {
		return ErrAttemptClosed
	}
	if !now.Before(a.DeadlineAt) {
		return ErrTimeExpired
	}
	return nil
}

func (a *Attempt) Submit(now time.Time) error {
	if a.Status != StatusInProgress {
		return ErrAttemptClosed
	}
	if now.Before(a.DeadlineAt) {
		a.Status = StatusSubmitted
	} else {
		a.Status = StatusExpired
	}
	a.SubmittedAt = &now
	return nil
}
