package application

import (
	"context"
	"encoding/json"
	"time"
)

type ManagedContent struct {
	ID        string          `json:"id"`
	Kind      string          `json:"kind"`
	Slug      string          `json:"slug"`
	Title     string          `json:"title"`
	Summary   string          `json:"summary"`
	SortOrder int32           `json:"sort_order"`
	StartsAt  *time.Time      `json:"starts_at,omitempty"`
	EndsAt    *time.Time      `json:"ends_at,omitempty"`
	Data      json.RawMessage `json:"data"`
}

func (s *Service) ListPublishedContent(ctx context.Context, kind string) ([]ManagedContent, error) {
	valid := map[string]bool{"daily_challenge": true, "achievement": true, "vocabulary": true, "writing_prompt": true}
	if !valid[kind] {
		return nil, ErrNotFound
	}
	return s.repository.ListPublishedContent(ctx, kind)
}
