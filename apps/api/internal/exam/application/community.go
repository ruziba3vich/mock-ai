package application

import (
	"context"
	"strings"
	"unicode"
	"unicode/utf8"
)

type CommunityProfile struct {
	Nickname string `json:"nickname"`
	Enabled  bool   `json:"enabled"`
}
type CommunityStats struct {
	Questions      int32 `json:"questions"`
	Mocks          int32 `json:"mocks"`
	Days           int32 `json:"days"`
	TodayQuestions int32 `json:"today_questions"`
	WeekMocks      int32 `json:"week_mocks"`
	WeekDays       int32 `json:"week_days"`
	WeekPoints     int32 `json:"week_points"`
}
type CommunityEntry struct {
	Nickname string `json:"nickname"`
	Points   int32  `json:"points"`
	Position int32  `json:"position"`
	IsMe     bool   `json:"is_me"`
}
type Community struct {
	Profile CommunityProfile `json:"profile"`
	Stats   CommunityStats   `json:"stats"`
	Board   []CommunityEntry `json:"board"`
}

func (s *Service) GetCommunity(ctx context.Context, u string) (Community, error) {
	return s.repository.GetCommunity(ctx, u)
}
func (s *Service) SaveCommunityProfile(ctx context.Context, u string, p CommunityProfile) error {
	p.Nickname = strings.TrimSpace(p.Nickname)
	if utf8.RuneCountInString(p.Nickname) < 2 || utf8.RuneCountInString(p.Nickname) > 24 {
		return ErrInvalidProfile
	}
	for _, r := range p.Nickname {
		if !unicode.IsLetter(r) && !unicode.IsNumber(r) && r != ' ' && r != '-' && r != '_' {
			return ErrInvalidProfile
		}
	}
	return s.repository.SaveCommunityProfile(ctx, u, p)
}
