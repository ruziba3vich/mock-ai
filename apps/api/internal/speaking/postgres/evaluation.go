package postgres

import (
	"context"
	"encoding/json"
	db "github.com/own/mock-ai/apps/api/internal/platform/postgres/sqlc"
	d "github.com/own/mock-ai/apps/api/internal/speaking/domain"
)

func (r *Repository) ClaimEvaluation(ctx context.Context) (d.EvaluationJob, error) {
	j, e := r.Q.ClaimSpeakingEvaluation(ctx)
	if e != nil {
		return d.EvaluationJob{}, e
	}
	return d.EvaluationJob{SessionID: str(j.SessionID), Part: j.Part, ClaimID: str(j.ClaimID), Attempts: j.Attempts}, nil
}

func (r *Repository) LoadEvaluation(ctx context.Context, j d.EvaluationJob) (d.Session, []d.SectionEvidence, error) {
	uid, e := r.Q.SpeakingEvaluationOwner(ctx, uuid(j.SessionID))
	if e != nil {
		return d.Session{}, nil, e
	}
	s, e := r.Get(ctx, str(uid), j.SessionID)
	if e != nil {
		return s, nil, e
	}
	if e = r.Q.MarkSpeakingEvaluationProcessing(ctx, uuid(s.ID)); e != nil {
		return s, nil, e
	}
	if j.Part > 0 {
		for i := range s.Turns {
			t := &s.Turns[i]
			if t.Part == j.Part && t.HasRecording {
				t.Recording, e = r.Audio(ctx, s.UserID, s.ID, t.Position, false)
				if e != nil {
					return s, nil, e
				}
			}
		}
		return d.Section(s, j.Part), nil, nil
	}
	rows, e := r.Q.ListSpeakingEvaluation(ctx, uuid(s.ID))
	if e != nil {
		return s, nil, e
	}
	evidence := []d.SectionEvidence{}
	for _, row := range rows {
		if row.Part == 0 {
			continue
		}
		var report d.Report
		if e = json.Unmarshal(row.Result, &report); e != nil {
			return s, nil, e
		}
		report = d.RestorePositions(s, row.Part, report)
		raw, _ := json.Marshal(report)
		evidence = append(evidence, d.SectionEvidence{Part: row.Part, Report: raw})
	}
	return s, evidence, nil
}

func (r *Repository) FinishEvaluation(ctx context.Context, j d.EvaluationJob, raw []byte, code string) error {
	tx, e := r.Pool.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	q := r.Q.WithTx(tx)
	state := "completed"
	if code != "" {
		state = "queued"
		if j.Attempts >= 3 {
			state = "failed"
		}
		raw = nil
	}
	n, e := q.FinishSpeakingEvaluation(ctx, db.FinishSpeakingEvaluationParams{SessionID: uuid(j.SessionID), Part: j.Part, ClaimID: uuid(j.ClaimID), Status: state, Result: raw, ErrorCode: code})
	if e != nil {
		return e
	}
	if n != 1 {
		return d.ErrConflict
	}
	if j.Part == 0 && state == "completed" {
		if _, e = q.CompleteSpeakingEvaluation(ctx, db.CompleteSpeakingEvaluationParams{ID: uuid(j.SessionID), Report: raw}); e != nil {
			return e
		}
	}
	if state == "failed" {
		if e = q.FailExhaustedSpeakingEvaluation(ctx); e != nil {
			return e
		}
	}
	return tx.Commit(ctx)
}

func (r *Repository) RecoverEvaluation(ctx context.Context) error {
	if e := r.Q.RecoverSpeakingEvaluation(ctx); e != nil {
		return e
	}
	return r.Q.FailExhaustedSpeakingEvaluation(ctx)
}
