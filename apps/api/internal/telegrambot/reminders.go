package telegrambot

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	db "github.com/own/mock-ai/apps/api/internal/platform/postgres/sqlc"
	"strconv"
	"time"
)

// RunReminders claims one delivery per student's local day. Preferences are
// disabled by default and are enabled explicitly in the student profile.
func (b *Bot) RunReminders(ctx context.Context, pool *pgxpool.Pool) {
	q := db.New(pool)
	for ctx.Err() == nil {
		rows, err := q.ClaimReminders(ctx)
		if err != nil && ctx.Err() == nil {
			b.logger.Error("claim study reminders", "error", err)
		}
		for _, row := range rows {
			chatID, err := strconv.ParseInt(row.ProviderSubject, 10, 64)
			if err != nil {
				continue
			}
			err = b.call(ctx, "sendMessage", map[string]any{"chat_id": chatID, "text": "Your study time is here 📚\nA short practice session is a good next step. Open your study plan to continue.\n\nYou can change or disable reminders in your profile.", "reply_markup": map[string]any{"inline_keyboard": [][]any{{map[string]any{"text": "Open my study plan", "web_app": map[string]string{"url": b.miniAppURL + "/#plan"}}}}}}, nil)
			if err != nil && ctx.Err() == nil {
				b.logger.Warn("study reminder delivery failed", "error", err)
			}
		}
		if !wait(ctx, time.Minute) {
			return
		}
	}
}
