package application

import (
	"context"
	"encoding/json"
	"errors"
	d "github.com/own/mock-ai/apps/api/internal/speaking/domain"
	"time"
)

type EvaluationRepository interface {
	ClaimEvaluation(context.Context) (d.EvaluationJob, error)
	LoadEvaluation(context.Context, d.EvaluationJob) (d.Session, []d.SectionEvidence, error)
	FinishEvaluation(context.Context, d.EvaluationJob, []byte, string) error
	RecoverEvaluation(context.Context) error
}
type AggregateProvider interface {
	Aggregate(context.Context, d.Session, []d.SectionEvidence) ([]byte, error)
}

func (s *Service) evaluationWorker(ctx context.Context, repo EvaluationRepository, provider AggregateProvider) {
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	lastCleanup := time.Time{}
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
		if time.Since(lastCleanup) > time.Minute {
			if e := repo.RecoverEvaluation(ctx); e != nil {
				s.Log.Warn("speaking evaluation recovery failed")
			}
			if e := s.Repo.Cleanup(ctx); e != nil {
				s.Log.Warn("speaking cleanup failed")
			}
			lastCleanup = time.Now()
		}
		j, e := repo.ClaimEvaluation(ctx)
		if e != nil {
			continue
		}
		s.processEvaluation(ctx, repo, provider, j)
	}
}

func (s *Service) processEvaluation(ctx context.Context, repo EvaluationRepository, provider AggregateProvider, j d.EvaluationJob) {
	started := time.Now()
	s.Log.Info("speaking evaluation started", "session_id", j.SessionID, "part", j.Part, "attempt", j.Attempts)
	work, cancel := context.WithTimeout(ctx, 180*time.Second)
	defer cancel()
	v, evidence, e := repo.LoadEvaluation(work, j)
	stage := "load"
	var raw []byte
	if e == nil {
		stage = "section"
		if j.Part == 0 {
			stage = "aggregate"
			raw, e = provider.Aggregate(work, v, evidence)
		} else {
			raw, e = s.Provider.Evaluate(work, v)
		}
	}
	code := ""
	if e != nil {
		code = "provider_unavailable"
		reason := "request_failed"
		if stage == "load" {
			code = "load_failed"
		}
		if errors.Is(work.Err(), context.DeadlineExceeded) {
			reason = "timeout"
		}
		// Only explicitly safe provider errors may contribute details.
		var safe interface{ SafeReason() string }
		if errors.As(e, &safe) {
			reason = safe.SafeReason()
		}
		s.Log.Warn("speaking evaluation failed", "session_id", j.SessionID, "part", j.Part, "stage", stage, "reason", reason, "attempt", j.Attempts)
	} else {
		parsed, err := d.ParseReport(raw, v)
		if err != nil {
			code = "invalid_feedback"
			s.Log.Warn("speaking report validation failed", "session_id", j.SessionID, "part", j.Part, "reason", err.Error(), "attempt", j.Attempts)
		} else {
			raw, _ = json.Marshal(parsed)
		}
	}
	finish, done := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer done()
	if e = repo.FinishEvaluation(finish, j, raw, code); e != nil {
		s.Log.Warn("speaking evaluation persistence failed", "session_id", j.SessionID, "part", j.Part)
		return
	}
	s.Log.Info("speaking evaluation finished", "session_id", j.SessionID, "part", j.Part, "attempt", j.Attempts, "error_code", code, "duration_ms", time.Since(started).Milliseconds())
}
