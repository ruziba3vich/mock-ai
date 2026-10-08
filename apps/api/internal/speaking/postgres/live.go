package postgres

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/own/mock-ai/apps/api/internal/platform/keyvault"
	db "github.com/own/mock-ai/apps/api/internal/platform/postgres/sqlc"
	d "github.com/own/mock-ai/apps/api/internal/speaking/domain"
)

func (r *Repository) ClaimLive(ctx context.Context, u, id string) (d.Session, error) {
	v, e := r.Q.ClaimSpeakingLive(ctx, db.ClaimSpeakingLiveParams{ID: uuid(id), UserID: uuid(u)})
	if errors.Is(e, pgx.ErrNoRows) {
		return d.Session{}, d.ErrConflict
	}
	if e != nil {
		return d.Session{}, e
	}
	s, e := r.Get(ctx, u, id)
	s.LiveRunID = str(v.LiveRunID)
	return s, e
}
func (r *Repository) RenewLive(ctx context.Context, s d.Session) error {
	n, e := r.Q.RenewSpeakingLive(ctx, db.RenewSpeakingLiveParams{ID: uuid(s.ID), LiveRunID: uuid(s.LiveRunID)})
	if e != nil {
		return e
	}
	if n != 1 {
		return d.ErrConflict
	}
	return nil
}
func (r *Repository) ReleaseLive(ctx context.Context, s d.Session) error {
	return r.Q.ReleaseSpeakingLive(ctx, db.ReleaseSpeakingLiveParams{ID: uuid(s.ID), LiveRunID: uuid(s.LiveRunID)})
}
func (r *Repository) SaveLive(ctx context.Context, s d.Session, pos int32, wav []byte, transcript string, done bool, next string) error {
	if pos < 0 || int(pos) >= len(s.Turns) || len(transcript) > 16000 {
		return d.ErrInvalid
	}
	duration, e := d.ValidateAudio(wav, s.Turns[pos].Seconds)
	if e != nil {
		return e
	}
	sealed, e := keyvault.SealBound(r.Master, string(wav), recordingBinding(s.ID, pos))
	if e != nil {
		return e
	}
	tx, e := r.Pool.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	q := r.Q.WithTx(tx)
	n, e := q.SaveSpeakingLive(ctx, db.SaveSpeakingLiveParams{ID: uuid(s.ID), LiveRunID: uuid(s.LiveRunID), Position: pos, Recording: sealed, DurationMs: duration, LiveTranscript: transcript, Answered: done})
	if e != nil {
		return e
	}
	if n != 1 {
		return d.ErrConflict
	}
	if next != "" {
		n, e = q.SetSpeakingLiveQuestion(ctx, db.SetSpeakingLiveQuestionParams{ID: uuid(s.ID), LiveRunID: uuid(s.LiveRunID), Position: pos + 1, Prompt: next})
		if e != nil {
			return e
		}
		if n != 1 {
			return d.ErrConflict
		}
	}
	return tx.Commit(ctx)
}

func (r *Repository) SkipLive(ctx context.Context, s d.Session, pos int32, next string) error {
	if pos < 0 || int(pos) >= len(s.Turns) {
		return d.ErrInvalid
	}
	tx, e := r.Pool.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	q := r.Q.WithTx(tx)
	n, e := q.SaveSpeakingLive(ctx, db.SaveSpeakingLiveParams{ID: uuid(s.ID), LiveRunID: uuid(s.LiveRunID), Position: pos, Answered: true, Skipped: true})
	if e != nil {
		return e
	}
	if n != 1 {
		return d.ErrConflict
	}
	if next != "" {
		n, e = q.SetSpeakingLiveQuestion(ctx, db.SetSpeakingLiveQuestionParams{ID: uuid(s.ID), LiveRunID: uuid(s.LiveRunID), Position: pos + 1, Prompt: next})
		if e != nil {
			return e
		}
		if n != 1 {
			return d.ErrConflict
		}
	}
	return tx.Commit(ctx)
}
