package postgres

import (
	"context"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	app "github.com/own/mock-ai/apps/api/internal/admin/application"
	db "github.com/own/mock-ai/apps/api/internal/platform/postgres/sqlc"
	"time"
)

type Repository struct{ q *db.Queries }

func New(p *pgxpool.Pool) *Repository  { return &Repository{db.New(p)} }
func id(s string) (pgtype.UUID, error) { var v pgtype.UUID; e := v.Scan(s); return v, e }
func str(v pgtype.UUID) string         { s, _ := v.Value(); return s.(string) }
func (r *Repository) Create(ctx context.Context, email, name, hash string) (string, error) {
	v, e := r.q.CreateAdmin(ctx, db.CreateAdminParams{Email: email, DisplayName: name, PasswordHash: hash})
	if e != nil {
		return "", e
	}
	return str(v), nil
}
func (r *Repository) ReserveLogin(ctx context.Context, email string) (app.Credential, error) {
	v, e := r.q.ReserveAdminLogin(ctx, email)
	if e != nil {
		return app.Credential{}, e
	}
	return app.Credential{Admin: app.Admin{ID: str(v.ID), Email: v.Email, DisplayName: v.DisplayName}, Hash: v.PasswordHash}, nil
}
func (r *Repository) ResetLogin(ctx context.Context, s string) error {
	v, e := id(s)
	if e != nil {
		return e
	}
	return r.q.ResetAdminLogin(ctx, v)
}
func (r *Repository) SaveSession(ctx context.Context, s, hash string, expires time.Time) error {
	v, e := id(s)
	if e != nil {
		return e
	}
	return r.q.CreateAdminSession(ctx, db.CreateAdminSessionParams{TokenHash: hash, AdminID: v, ExpiresAt: pgtype.Timestamptz{Time: expires, Valid: true}})
}
func (r *Repository) Session(ctx context.Context, hash string) (app.Admin, error) {
	v, e := r.q.GetAdminSession(ctx, hash)
	if e != nil {
		return app.Admin{}, e
	}
	return app.Admin{ID: str(v.ID), Email: v.Email, DisplayName: v.DisplayName}, nil
}
func (r *Repository) DeleteSession(ctx context.Context, hash string) error {
	return r.q.DeleteAdminSession(ctx, hash)
}
