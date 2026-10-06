package postgres

import (
	"context"
	app "github.com/own/mock-ai/apps/api/internal/exam/application"
	db "github.com/own/mock-ai/apps/api/internal/platform/postgres/sqlc"
)

func (r *Repository) GetCommunity(ctx context.Context, u string) (app.Community, error) {
	var out app.Community
	uid, e := parseUUID(u)
	if e != nil {
		return out, app.ErrNotFound
	}
	p, e := r.queries.GetCommunityProfile(ctx, uid)
	if e != nil {
		return out, e
	}
	s, e := r.queries.GetCommunityStats(ctx, uid)
	if e != nil {
		return out, e
	}
	rows, e := r.queries.GetCommunityBoard(ctx, uid)
	if e != nil {
		return out, e
	}
	out.Profile = app.CommunityProfile{Nickname: p.Nickname, Enabled: p.Enabled}
	out.Stats = app.CommunityStats{Questions: s.Questions, Mocks: s.Mocks, Days: s.Days, TodayQuestions: s.TodayQuestions, WeekMocks: s.WeekMocks, WeekDays: s.WeekDays, WeekPoints: s.WeekPoints}
	out.Board = make([]app.CommunityEntry, 0, len(rows))
	for _, v := range rows {
		out.Board = append(out.Board, app.CommunityEntry{Nickname: v.Nickname, Points: v.Points, Position: v.Position, IsMe: v.IsMe})
	}
	return out, nil
}
func (r *Repository) SaveCommunityProfile(ctx context.Context, u string, p app.CommunityProfile) error {
	uid, e := parseUUID(u)
	if e != nil {
		return app.ErrNotFound
	}
	return r.queries.SaveCommunityProfile(ctx, db.SaveCommunityProfileParams{UserID: uid, Nickname: p.Nickname, Enabled: p.Enabled})
}
