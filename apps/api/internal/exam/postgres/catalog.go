package postgres

import (
	"context"
	app "github.com/own/mock-ai/apps/api/internal/exam/application"
)

func (r *Repository) ListPublishedContent(ctx context.Context, kind string) ([]app.ManagedContent, error) {
	rows, e := r.queries.ListPublishedContent(ctx, kind)
	if e != nil {
		return nil, e
	}
	out := make([]app.ManagedContent, 0, len(rows))
	for _, v := range rows {
		item := app.ManagedContent{ID: uuidString(v.ID), Kind: v.Kind, Slug: v.Slug, Title: v.Title, Summary: v.Summary, SortOrder: v.SortOrder, Data: v.Data}
		if v.StartsAt.Valid {
			t := v.StartsAt.Time
			item.StartsAt = &t
		}
		if v.EndsAt.Valid {
			t := v.EndsAt.Time
			item.EndsAt = &t
		}
		out = append(out, item)
	}
	return out, nil
}
