package postgres

import (
	"context"
	"encoding/json"
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

func (r *Repository) Dashboard(ctx context.Context) (app.Dashboard, error) {
	s, e := r.q.AdminDashboardSummary(ctx)
	if e != nil {
		return app.Dashboard{}, e
	}
	trend, e := r.q.AdminDashboardTrend(ctx)
	if e != nil {
		return app.Dashboard{}, e
	}
	counts, e := r.q.AdminContentBreakdown(ctx)
	if e != nil {
		return app.Dashboard{}, e
	}
	avg, _ := s.AverageBand.Float64Value()
	out := app.Dashboard{Summary: app.DashboardSummary{Users: s.Users, NewUsers7d: s.NewUsers7d, ActiveUsers7d: s.ActiveUsers7d, CompletedMocks: s.CompletedMocks, AverageBand: avg.Float64, WritingReports: s.WritingReports, SpeakingReports: s.SpeakingReports, PublishedExams: s.PublishedExams, ExamDrafts: s.ExamDrafts, PublishedContent: s.PublishedContent, ManagedDrafts: s.ManagedDrafts}, Trend: make([]app.DashboardDay, 0, len(trend)), Content: make([]app.ContentCount, 0, len(counts))}
	for _, v := range trend {
		out.Trend = append(out.Trend, app.DashboardDay{Date: v.ActivityDate.Time.Format("2006-01-02"), Signups: v.Signups, ActiveStudents: v.ActiveStudents, Attempts: v.Attempts})
	}
	for _, v := range counts {
		out.Content = append(out.Content, app.ContentCount{Kind: v.Kind, Status: v.Status, Count: v.Count})
	}
	return out, nil
}
func (r *Repository) ListStudents(ctx context.Context) ([]app.Student, error) {
	rows, e := r.q.ListAdminStudents(ctx)
	if e != nil {
		return nil, e
	}
	out := make([]app.Student, 0, len(rows))
	for _, v := range rows {
		out = append(out, app.Student{ID: str(v.ID), DisplayName: v.DisplayName, Email: v.Email, TelegramUsername: v.TelegramUsername, TelegramID: v.TelegramID})
	}
	return out, nil
}
func content(v db.ManagedContent) app.ManagedContent {
	out := app.ManagedContent{ID: str(v.ID), Kind: v.Kind, Slug: v.Slug, Title: v.Title, Summary: v.Summary, Status: v.Status, SortOrder: v.SortOrder, Data: json.RawMessage(v.Data), CreatedAt: v.CreatedAt.Time, UpdatedAt: v.UpdatedAt.Time}
	if v.StartsAt.Valid {
		t := v.StartsAt.Time
		out.StartsAt = &t
	}
	if v.EndsAt.Valid {
		t := v.EndsAt.Time
		out.EndsAt = &t
	}
	return out
}
func nullableTime(v *time.Time) pgtype.Timestamptz {
	if v == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *v, Valid: true}
}
func (r *Repository) ListContent(ctx context.Context, kind string) ([]app.ManagedContent, error) {
	rows, e := r.q.ListManagedContent(ctx, kind)
	if e != nil {
		return nil, e
	}
	out := make([]app.ManagedContent, 0, len(rows))
	for _, v := range rows {
		out = append(out, content(v))
	}
	return out, nil
}
func (r *Repository) CreateContent(ctx context.Context, adminID string, v app.ManagedContent) (app.ManagedContent, error) {
	a, e := id(adminID)
	if e != nil {
		return app.ManagedContent{}, e
	}
	row, e := r.q.CreateManagedContent(ctx, db.CreateManagedContentParams{Kind: v.Kind, Slug: v.Slug, Title: v.Title, Summary: v.Summary, Status: v.Status, SortOrder: v.SortOrder, StartsAt: nullableTime(v.StartsAt), EndsAt: nullableTime(v.EndsAt), Data: v.Data, CreatedBy: a})
	if e != nil {
		return app.ManagedContent{}, e
	}
	return content(row), nil
}
func (r *Repository) UpdateContent(ctx context.Context, adminID string, v app.ManagedContent) (app.ManagedContent, error) {
	a, e := id(adminID)
	if e != nil {
		return app.ManagedContent{}, e
	}
	contentID, e := id(v.ID)
	if e != nil {
		return app.ManagedContent{}, app.ErrContentNotFound
	}
	row, e := r.q.UpdateManagedContent(ctx, db.UpdateManagedContentParams{ID: contentID, Kind: v.Kind, Slug: v.Slug, Title: v.Title, Summary: v.Summary, Status: v.Status, SortOrder: v.SortOrder, StartsAt: nullableTime(v.StartsAt), EndsAt: nullableTime(v.EndsAt), Data: v.Data, UpdatedBy: a})
	if e != nil {
		return app.ManagedContent{}, e
	}
	return content(row), nil
}
func (r *Repository) DeleteContent(ctx context.Context, value string) error {
	v, e := id(value)
	if e != nil {
		return app.ErrContentNotFound
	}
	n, e := r.q.DeleteManagedContent(ctx, v)
	if e == nil && n == 0 {
		return app.ErrContentNotFound
	}
	return e
}
