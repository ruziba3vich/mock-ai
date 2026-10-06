package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	migrations "github.com/own/mock-ai/apps/api/db/migrations"
	app "github.com/own/mock-ai/apps/api/internal/exam/application"
	examhttp "github.com/own/mock-ai/apps/api/internal/exam/httpapi"
	exampg "github.com/own/mock-ai/apps/api/internal/exam/postgres"
	"github.com/own/mock-ai/apps/api/internal/identity"
	"github.com/own/mock-ai/apps/api/internal/identity/googleauth"
	authhttp "github.com/own/mock-ai/apps/api/internal/identity/httpapi"
	authpg "github.com/own/mock-ai/apps/api/internal/identity/postgres"
	"github.com/own/mock-ai/apps/api/internal/platform/httpserver"
	db "github.com/own/mock-ai/apps/api/internal/platform/postgres/sqlc"
)

// Run only against a newly created mock_ai_learning_test database. The URL's
// database is overridden so this test cannot migrate the production database.
func TestLearningIntegration(t *testing.T) {
	if os.Getenv("MOCKAI_INTEGRATION") != "1" {
		t.Skip("requires isolated mock_ai_learning_test database")
	}
	address, e := url.Parse(os.Getenv("DATABASE_URL"))
	if e != nil {
		t.Fatal(e)
	}
	address.Path = "/mock_ai_learning_test"
	source, e := iofs.New(migrations.Files, ".")
	if e != nil {
		t.Fatal(e)
	}
	migration, e := migrate.NewWithSourceInstance("iofs", source, address.String())
	if e != nil {
		t.Fatal("connect test migrations:", e)
	}
	defer migration.Close()
	if e = migration.Up(); e != nil {
		t.Fatal(e)
	}
	pool, e := pgxpool.New(context.Background(), address.String())
	if e != nil {
		t.Fatal(e)
	}
	defer pool.Close()
	repo := exampg.New(pool)
	service := app.NewService(repo)
	auth, e := identity.NewService(authpg.New(pool), "test-token", 3600000000000)
	if e != nil {
		t.Fatal(e)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	authHandler := authhttp.New(auth, googleauth.New("", "", "", ""), logger, "http://localhost", false)
	handler := httpserver.New(logger, "http://localhost", authHandler, examhttp.New(service, authHandler, logger))
	request := func(method, path string, payload any, cookie *http.Cookie, status int) map[string]json.RawMessage {
		t.Helper()
		b, _ := json.Marshal(payload)
		req := httptest.NewRequest(method, path, bytes.NewReader(b))
		if cookie != nil {
			req.AddCookie(cookie)
		}
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != status {
			t.Fatalf("%s %s: got %d want %d: %s", method, path, rr.Code, status, rr.Body.String())
		}
		var out map[string]json.RawMessage
		if status != 204 {
			if e := json.Unmarshal(rr.Body.Bytes(), &out); e != nil {
				t.Fatal(e)
			}
		}
		return out
	}
	student, session, e := auth.Register(context.Background(), "Learning QA", "learning@example.com", "long-test-password", "", "")
	if e != nil {
		t.Fatal(e)
	}
	_, otherSession, e := auth.Register(context.Background(), "Other QA", "other@example.com", "long-test-password", "", "")
	if e != nil {
		t.Fatal(e)
	}
	cookie := &http.Cookie{Name: "mock_ai_session", Value: session.Token}
	other := &http.Cookie{Name: "mock_ai_session", Value: otherSession.Token}
	request("GET", "/api/v1/me/mistakes", nil, nil, 401)
	request("GET", "/api/v1/admin/content", nil, cookie, 403)
	exams, e := service.ListExams(context.Background())
	if e != nil || len(exams) == 0 {
		t.Fatal("missing exams", e)
	}
	attempt, e := service.StartAttempt(context.Background(), student.ID, exams[0].ID)
	if e != nil {
		t.Fatal(e)
	}
	path := "/api/v1/attempts/" + attempt.ID + "/workspace"
	workspace := app.Workspace{Flags: []string{"question-1"}, Notes: map[string]string{"question-1": "Recheck the date"}, Highlights: map[string][]string{"section-1": {"important"}}}
	request("PUT", path, workspace, cookie, 204)
	request("GET", path, nil, other, 404)
	request("PUT", path, workspace, other, 404)
	saved := request("GET", path, nil, cookie, 200)
	if !strings.Contains(string(saved["notes"]), "Recheck the date") {
		t.Fatal("notes not persisted")
	}
	practice, e := service.ListPracticeQuestions(context.Background(), "reading", "", 5)
	if e != nil || len(practice) == 0 {
		t.Fatal("no practice", e)
	}
	qid := practice[0].ID
	request("POST", "/api/v1/me/mistakes/"+qid+"/retry", map[string]any{"values": []string{"wrong"}}, other, 404)
	bad := request("POST", "/api/v1/practice/questions/"+qid+"/check", map[string]any{"values": []string{"deliberately wrong"}}, cookie, 200)
	var feedback app.PracticeFeedback
	json.Unmarshal(bad["feedback"], &feedback)
	notebook := request("GET", "/api/v1/me/mistakes", nil, cookie, 200)
	var mistakes []app.Mistake
	json.Unmarshal(notebook["mistakes"], &mistakes)
	if len(mistakes) != 1 || mistakes[0].Resolved {
		t.Fatal("missing unresolved mistake")
	}
	request("POST", "/api/v1/me/mistakes/"+qid+"/retry", map[string]any{"values": feedback.AcceptedAnswer[:1]}, cookie, 200)
	notebook = request("GET", "/api/v1/me/mistakes", nil, cookie, 200)
	json.Unmarshal(notebook["mistakes"], &mistakes)
	if !mistakes[0].Resolved {
		t.Fatal("mistake did not resolve")
	}
	request("PUT", "/api/v1/me/reminder", app.Reminder{Enabled: true, LocalTime: "19:00", Timezone: "Asia/Tashkent"}, cookie, 422)
	request("PUT", "/api/v1/me/reminder", app.Reminder{Enabled: false, LocalTime: "19:00", Timezone: "Invalid/Zone"}, cookie, 422)
	request("PUT", "/api/v1/me/reminder", app.Reminder{Enabled: false, LocalTime: "19:00", Timezone: "Asia/Tashkent"}, cookie, 204)
	var uid pgtype.UUID
	uid.Scan(student.ID)
	queries := db.New(pool)
	if e = queries.AssignUserRole(context.Background(), db.AssignUserRoleParams{UserID: uid, Role: db.UserRoleAdmin}); e != nil {
		t.Fatal(e)
	}
	content := app.Content{Title: "Integration reading", Skill: "reading", Purpose: "practice", Duration: 600, Sections: []app.ContentSection{{Title: "Passage", Passage: "The library opens at nine.", Questions: []app.ContentQuestion{{Type: "short_answer", Prompt: "When does it open?", Answers: []string{"nine"}, Explanation: "The opening time is stated in the passage."}}}}}
	result := request("POST", "/api/v1/admin/content", content, cookie, 200)
	var draftID string
	json.Unmarshal(result["id"], &draftID)
	request("POST", "/api/v1/admin/content/"+draftID+"/publish", nil, other, 403)
	request("POST", "/api/v1/admin/content/"+draftID+"/publish", nil, cookie, 200)
	content.Sections[0].Questions[0].Explanation = "An improved explanation."
	request("PUT", "/api/v1/admin/content/"+draftID, content, cookie, 200)
	request("POST", "/api/v1/admin/content/"+draftID+"/publish", nil, cookie, 200)
	content.Skill = "listening"
	request("PUT", "/api/v1/admin/content/"+draftID, content, cookie, 200)
	request("POST", "/api/v1/admin/content/"+draftID+"/publish", nil, cookie, 422)
	if _, e = queries.CreateIdentity(context.Background(), db.CreateIdentityParams{UserID: uid, Provider: db.IdentityProviderTelegram, ProviderSubject: "999999999999", Profile: []byte(`{}`)}); e != nil {
		t.Fatal(e)
	}
	request("PUT", "/api/v1/me/reminder", app.Reminder{Enabled: true, LocalTime: "00:00", Timezone: "UTC"}, cookie, 204)
	first, e := queries.ClaimReminders(context.Background())
	if e != nil || len(first) != 1 {
		t.Fatal("expected one due reminder", len(first), e)
	}
	again, e := queries.ClaimReminders(context.Background())
	if e != nil || len(again) != 0 {
		t.Fatal("reminder claimed twice", e)
	}
	request("PUT", "/api/v1/me/reminder", app.Reminder{Enabled: false, LocalTime: "00:00", Timezone: "UTC"}, cookie, 204)
	t.Log("ownership, notebook resolution, annotations, admin publishing/versioning, reminder validation and single daily claim passed")
	if e = migration.Steps(-3); e != nil {
		t.Fatal("rollback new migrations:", e)
	}
	if e = migration.Up(); e != nil {
		t.Fatal("reapply new migrations:", e)
	}
	t.Log("new migrations rolled back and reapplied successfully")
}
