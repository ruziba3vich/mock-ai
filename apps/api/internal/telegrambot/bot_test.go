package telegrambot

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/own/mock-ai/apps/api/internal/identity/domain"
	"github.com/own/mock-ai/apps/api/internal/identity/telegram"
)

type userStoreStub struct {
	created bool
	calls   int
}

func (s *userStoreStub) EnsureTelegramUser(_ context.Context, user telegram.User, raw []byte) (domain.User, bool, error) {
	s.calls++
	if user.ID != 42 || !json.Valid(raw) {
		return domain.User{}, false, io.ErrUnexpectedEOF
	}
	return domain.User{DisplayName: user.DisplayName()}, s.created, nil
}

func TestHandleStartCreatesProfileAndSendsMiniAppButtons(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name      string
		created   bool
		wantWords string
	}{
		{name: "first visit", created: true, wantWords: "Your student profile is ready"},
		{name: "returning", created: false, wantWords: "Welcome back to MockAI"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			var requests []map[string]any
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatalf("decode request: %v", err)
				}
				body["method"] = strings.TrimPrefix(r.URL.Path, "/bottest-token/")
				requests = append(requests, body)
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `{"ok":true,"result":true}`)
			}))
			defer server.Close()

			store := &userStoreStub{created: test.created}
			bot, err := New("test-token", "https://mock.web.chessinsight.uz", store, slog.New(slog.NewTextHandler(io.Discard, nil)))
			if err != nil {
				t.Fatalf("new bot: %v", err)
			}
			bot.apiBase = server.URL
			bot.client = server.Client()
			err = bot.handle(context.Background(), update{ID: 7, Message: &message{
				Text: "/start campaign",
				Chat: chat{ID: 99, Type: "private"},
				From: telegram.User{ID: 42, FirstName: "Donik", LastName: "Student"},
			}})
			if err != nil {
				t.Fatalf("handle: %v", err)
			}
			if store.calls != 1 {
				t.Fatalf("profile calls = %d, want 1", store.calls)
			}
			if len(requests) != 2 {
				t.Fatalf("Telegram calls = %d, want 2", len(requests))
			}
			if requests[0]["method"] != "setChatMenuButton" {
				t.Fatalf("first method = %v", requests[0]["method"])
			}
			menu := requests[0]["menu_button"].(map[string]any)
			if menu["text"] != "Open" || menu["type"] != "web_app" {
				t.Fatalf("unexpected menu button: %#v", menu)
			}
			if requests[1]["method"] != "sendMessage" {
				t.Fatalf("second method = %v", requests[1]["method"])
			}
			text, _ := requests[1]["text"].(string)
			if !strings.Contains(text, "Donik") || !strings.Contains(text, test.wantWords) {
				t.Fatalf("unexpected greeting: %q", text)
			}
			markup := requests[1]["reply_markup"].(map[string]any)
			keyboard := markup["inline_keyboard"].([]any)
			row := keyboard[0].([]any)
			button := row[0].(map[string]any)
			if button["text"] != "Open MockAI" {
				t.Fatalf("unexpected inline button: %#v", button)
			}
			webApp := button["web_app"].(map[string]any)
			if webApp["url"] != "https://mock.web.chessinsight.uz" {
				t.Fatalf("unexpected Mini App URL: %v", webApp["url"])
			}
		})
	}
}

func TestHandleIgnoresNonStartAndNonPrivateMessages(t *testing.T) {
	t.Parallel()
	store := &userStoreStub{}
	bot, err := New("test-token", "https://mock.web.chessinsight.uz", store, slog.Default())
	if err != nil {
		t.Fatalf("new bot: %v", err)
	}
	for _, item := range []update{
		{Message: &message{Text: "hello", Chat: chat{Type: "private"}}},
		{Message: &message{Text: "/start", Chat: chat{Type: "group"}}},
		{},
	} {
		if err := bot.handle(context.Background(), item); err != nil {
			t.Fatalf("handle ignored update: %v", err)
		}
	}
	if store.calls != 0 {
		t.Fatalf("profile calls = %d, want 0", store.calls)
	}
}

func TestStartCommandRecognition(t *testing.T) {
	t.Parallel()
	for input, want := range map[string]bool{
		"/start": true, "/start referral": true, "/START@mockai_bot": true,
		"start": false, "/starter": false, "": false,
	} {
		if got := isStartCommand(input); got != want {
			t.Errorf("isStartCommand(%q) = %v, want %v", input, got, want)
		}
	}
}
