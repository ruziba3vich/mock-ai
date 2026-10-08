package httpapi_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"github.com/coder/websocket"
	"github.com/jackc/pgx/v5/pgxpool"
	app "github.com/own/mock-ai/apps/api/internal/speaking/application"
	d "github.com/own/mock-ai/apps/api/internal/speaking/domain"
	speakingpg "github.com/own/mock-ai/apps/api/internal/speaking/postgres"
	"math"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

type speakingTestProvider struct{}

func (p speakingTestProvider) Aggregate(ctx context.Context, s d.Session, _ []d.SectionEvidence) ([]byte, error) {
	return p.Evaluate(ctx, s)
}

func (speakingTestProvider) ConnectLive(context.Context) (*websocket.Conn, error) {
	return nil, d.ErrUnavailable
}

func testLiveSpeakingHTTP(t *testing.T, h http.Handler, owner, other *http.Cookie) {
	t.Helper()
	request := func(method, path, origin string, cookie *http.Cookie, body string, code int) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		if origin != "" {
			r.Header.Set("Origin", origin)
		}
		if cookie != nil {
			r.AddCookie(cookie)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != code {
			t.Fatalf("live %s: got %d want %d: %s", path, w.Code, code, w.Body.String())
		}
		return w
	}
	base := "/api/v1/speaking/live/sessions"
	request("POST", base, "http://localhost", nil, `{"consent":true}`, 401)
	request("POST", base, "http://localhost", owner, `{"consent":false}`, 422)
	var v d.Session
	_ = json.Unmarshal(request("POST", base, "http://localhost", owner, `{"consent":true}`, 201).Body.Bytes(), &v)
	stream := base + "/" + v.ID + "/stream"
	request("GET", stream, "http://localhost", nil, "", 401)
	request("GET", stream, "https://evil.example", owner, "", 403)
	request("GET", stream, "", owner, "", 403)
	request("GET", stream, "http://localhost", other, "", 409)
	// Failed upgrade must release its claimed lease, so a later retry can claim.
	request("GET", stream, "http://localhost", owner, "", 426)
	request("GET", stream, "http://localhost", owner, "", 426)
	request("DELETE", "/api/v1/speaking/sessions/"+v.ID, "http://localhost", owner, "", 204)
	t.Log("live HTTP: consent, authentication, exact WebSocket Origin, ownership and failed-upgrade lease cleanup passed")
}

func (speakingTestProvider) Speak(context.Context, d.VoiceJob) ([]byte, error) {
	return speakingWav(), nil
}

func testLiveSpeakingStorage(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	repo := speakingpg.New(pool, strings.Repeat("ab", 32))
	var uid, other string
	if e := pool.QueryRow(ctx, "INSERT INTO users(display_name) VALUES('Live storage QA') RETURNING id").Scan(&uid); e != nil {
		t.Fatal(e)
	}
	if e := pool.QueryRow(ctx, "INSERT INTO users(display_name) VALUES('Other live QA') RETURNING id").Scan(&other); e != nil {
		t.Fatal(e)
	}
	topic, turns, _ := d.NewPlan()
	v, e := repo.CreateLive(ctx, uid, "test-report", "test-live", d.Countries[0].Voices[0], topic, turns)
	if e != nil {
		t.Fatal(e)
	}
	if v.Mode != "live" || v.LiveModel != "test-live" {
		t.Fatal("live mode not persisted")
	}
	if _, e = repo.ClaimVoice(ctx); e == nil {
		t.Fatal("live question incorrectly queued for TTS")
	}
	if _, e = repo.ClaimLive(ctx, other, v.ID); e == nil {
		t.Fatal("foreign live session claimed")
	}
	claimed, e := repo.ClaimLive(ctx, uid, v.ID)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = repo.ClaimLive(ctx, uid, v.ID); e == nil {
		t.Fatal("concurrent live connection allowed")
	}
	if _, e = repo.Answer(ctx, uid, v.ID, 0, speakingWav(), false); e == nil {
		t.Fatal("recorded endpoint mutated live interview")
	}
	if e = repo.SaveLive(ctx, claimed, 0, speakingWav(), "I enjoy learning languages.", true, "What do you enjoy about your city?"); e != nil {
		t.Fatal(e)
	}
	if e = repo.SaveLive(ctx, claimed, 0, speakingWav(), "overwrite", true, ""); e == nil {
		t.Fatal("completed answer overwritten")
	}
	got, e := repo.Get(ctx, uid, v.ID)
	if e != nil || !got.Turns[0].Answered || got.Turns[1].Prompt != "What do you enjoy about your city?" {
		t.Fatal("progression not persisted", e)
	}
	audio, e := repo.Audio(ctx, uid, v.ID, 0, false)
	if e != nil || !bytes.Equal(audio, speakingWav()) {
		t.Fatal("live recording encryption roundtrip", e)
	}
	if e = repo.ReleaseLive(ctx, claimed); e != nil {
		t.Fatal(e)
	}
	resumed, e := repo.ClaimLive(ctx, uid, v.ID)
	if e != nil || resumed.LiveRunID == claimed.LiveRunID || resumed.Examiner != claimed.Examiner {
		t.Fatal("resume identity", e)
	}
	if e = repo.SaveLive(ctx, claimed, 1, speakingWav(), "stale connection", true, ""); e == nil {
		t.Fatal("stale connection wrote audio")
	}
	if e = repo.ReleaseLive(ctx, claimed); e != nil {
		t.Fatal(e)
	}
	if e = repo.RenewLive(ctx, resumed); e != nil {
		t.Fatal("stale release invalidated current lease", e)
	}
	if e = repo.Delete(ctx, uid, v.ID); e != nil {
		t.Fatal(e)
	}
	if e = repo.SaveLive(ctx, resumed, 1, speakingWav(), "deleted session", true, ""); e == nil {
		t.Fatal("deleted session resurrected")
	}
	t.Log("live storage: ownership, exclusive lease, stale writes, stable examiner, encrypted audio, atomic progression and deletion passed")
}
func (speakingTestProvider) Evaluate(_ context.Context, s d.Session) ([]byte, error) {
	r := d.Report{Assessable: true, Summary: "Synthetic integration feedback, not a real evaluation.", NextSteps: []string{"Add examples."}, Issues: []d.Issue{}, Strengths: []string{"Clear ideas."}}
	for _, k := range []string{"fluency_coherence", "lexical_resource", "grammar", "pronunciation"} {
		n := 6.5
		r.Criteria = append(r.Criteria, d.Criterion{Key: k, Score: &n, Reason: "Test evidence.", Improve: "Test exercise."})
	}
	for _, turn := range s.Turns {
		transcript := "I enjoy learning."
		if turn.Skipped {
			transcript = ""
		}
		r.Responses = append(r.Responses, d.Response{Position: turn.Position, Transcript: transcript, Feedback: "Give a reason."})
	}
	return json.Marshal(r)
}
func speakingWav() []byte {
	b := make([]byte, 44+32000*7)
	copy(b, "RIFF")
	binary.LittleEndian.PutUint32(b[4:], uint32(len(b)-8))
	copy(b[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(b[16:], 16)
	binary.LittleEndian.PutUint16(b[20:], 1)
	binary.LittleEndian.PutUint16(b[22:], 1)
	binary.LittleEndian.PutUint32(b[24:], 16000)
	binary.LittleEndian.PutUint32(b[28:], 32000)
	binary.LittleEndian.PutUint16(b[32:], 2)
	binary.LittleEndian.PutUint16(b[34:], 16)
	copy(b[36:], "data")
	binary.LittleEndian.PutUint32(b[40:], uint32(len(b)-44))
	for i := 44; i < len(b); i += 2 {
		binary.LittleEndian.PutUint16(b[i:], uint16(int16(math.Sin(float64(i)*.1)*10000)))
	}
	return b
}
func testSpeaking(t *testing.T, h http.Handler, s *app.Service, pool *pgxpool.Pool, cookie, other *http.Cookie) {
	t.Helper()
	ctx := context.Background()
	request := func(method, path string, body []byte, c *http.Cookie, status int) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, path, bytes.NewReader(body))
		req.Header.Set("Origin", "http://localhost")
		if c != nil {
			req.AddCookie(c)
		}
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != status {
			t.Fatalf("speaking %s %s: %d want %d: %s", method, path, rr.Code, status, rr.Body.String())
		}
		return rr
	}
	request("GET", "/api/v1/speaking/settings", nil, nil, 401)
	request("POST", "/api/v1/speaking/sessions", []byte(`{"consent":false}`), cookie, 422)
	created := request("POST", "/api/v1/speaking/sessions", []byte(`{"consent":true}`), cookie, 201)
	var v d.Session
	if e := json.Unmarshal(created.Body.Bytes(), &v); e != nil {
		t.Fatal(e)
	}
	path := "/api/v1/speaking/sessions/" + v.ID
	if len(v.Turns) != 11 || v.Examiner.Voice == "" {
		t.Fatal("invalid interview plan")
	}
	request("POST", "/api/v1/speaking/sessions", []byte(`{"consent":true}`), cookie, 409)
	request("GET", path, nil, other, 404)
	request("DELETE", path, nil, other, 404)
	request("POST", path+"/submit", nil, other, 404)
	request("POST", path+"/submit", nil, cookie, 409)
	req := httptest.NewRequest("DELETE", path, nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != 403 {
		t.Fatal("missing origin accepted")
	}
	wav := speakingWav()
	for i := 0; i < 11; i++ {
		j, e := s.Repo.ClaimVoice(ctx)
		if e != nil || j.Position != int32(i) {
			t.Fatal("claim voice", i, j, e)
		}
		if j.Examiner != v.Examiner {
			t.Fatal("examiner changed")
		}
		if i == 0 {
			stale := j
			if _, e = pool.Exec(ctx, "UPDATE speaking_turns SET voice_lease_until=now()-interval '1 second' WHERE session_id=$1 AND position=0", v.ID); e != nil {
				t.Fatal(e)
			}
			if e = s.Repo.Cleanup(ctx); e != nil {
				t.Fatal(e)
			}
			request("POST", path+"/turns/0/voice/retry", nil, other, 404)
			request("POST", path+"/turns/0/voice/retry", nil, cookie, 202)
			j, e = s.Repo.ClaimVoice(ctx)
			if e != nil || j.RunID == stale.RunID {
				t.Fatal("voice retry did not change run", e)
			}
			if e = s.Repo.FinishVoice(ctx, stale, wav); e != nil {
				t.Fatal(e)
			}
			var state string
			_ = pool.QueryRow(ctx, "SELECT voice_status FROM speaking_turns WHERE session_id=$1 AND position=0", v.ID).Scan(&state)
			if state != "processing" {
				t.Fatal("stale voice overwrote retry", state)
			}
		}
		if e = s.Repo.FinishVoice(ctx, j, wav); e != nil {
			t.Fatal(e)
		}
		turn := path + "/turns/" + strconv.Itoa(i)
		response := request("GET", turn+"/examiner", nil, cookie, 200)
		if response.Header().Get("Cache-Control") != "private, no-store" || !bytes.Equal(response.Body.Bytes(), wav) {
			t.Fatal("audio bytes or private cache")
		}
		request("GET", turn+"/examiner", nil, other, 404)
		if i == 0 {
			request("PUT", turn+"/answer", []byte("invalid"), cookie, 422)
			request("PUT", turn+"/answer", wav, other, 404)
			request("PUT", path+"/turns/1/answer", wav, cookie, 409)
		}
		if i == 1 {
			request("POST", turn+"/skip", nil, cookie, 200)
		} else {
			request("PUT", turn+"/answer", wav, cookie, 200)
			request("PUT", turn+"/answer", wav, cookie, 409)
			got := request("GET", turn+"/recording", nil, cookie, 200)
			if !bytes.Equal(got.Body.Bytes(), wav) {
				t.Fatal("recording roundtrip")
			}
		}
	}
	var encrypted []byte
	if e := pool.QueryRow(ctx, "SELECT recording FROM speaking_turns WHERE session_id=$1 AND position=0", v.ID).Scan(&encrypted); e != nil {
		t.Fatal(e)
	}
	if bytes.Equal(encrypted, wav) || bytes.HasPrefix(encrypted, []byte("RIFF")) {
		t.Fatal("recording not encrypted")
	}
	request("POST", path+"/submit", nil, cookie, 202)
	request("POST", path+"/submit", nil, cookie, 202)
	var n int
	if e := pool.QueryRow(ctx, "SELECT count(*) FROM speaking_requests WHERE kind='evaluate'").Scan(&n); e != nil || n != 1 {
		t.Fatal("non-idempotent submit", n, e)
	}
	// Three concurrent claims must select different section jobs. Aggregation
	// must wait, and an expired worker cannot overwrite its replacement.
	repo := s.Repo.(app.EvaluationRepository)
	jobs := make([]d.EvaluationJob, 3)
	errs := make([]error, 3)
	var group sync.WaitGroup
	for i := range jobs {
		group.Add(1)
		go func(i int) { defer group.Done(); jobs[i], errs[i] = repo.ClaimEvaluation(ctx) }(i)
	}
	group.Wait()
	seen := map[int32]bool{}
	for i, j := range jobs {
		if errs[i] != nil || j.Part == 0 || seen[j.Part] {
			t.Fatal("concurrent section claim", j, errs[i])
		}
		seen[j.Part] = true
	}
	if _, e := repo.ClaimEvaluation(ctx); e == nil {
		t.Fatal("aggregation claimed before sections completed")
	}
	stale := jobs[0]
	if _, e := pool.Exec(ctx, "UPDATE speaking_evaluation_jobs SET lease_until=now()-interval '1 second' WHERE session_id=$1 AND part=$2", v.ID, stale.Part); e != nil {
		t.Fatal(e)
	}
	if e := repo.RecoverEvaluation(ctx); e != nil {
		t.Fatal(e)
	}
	replacement, e := repo.ClaimEvaluation(ctx)
	if e != nil || replacement.Part != stale.Part || replacement.ClaimID == stale.ClaimID {
		t.Fatal("recovery", replacement, e)
	}
	if e := repo.FinishEvaluation(ctx, stale, nil, "provider_unavailable"); e == nil {
		t.Fatal("stale evaluation accepted")
	}
	jobs[0] = replacement
	for _, j := range jobs {
		section, _, e := repo.LoadEvaluation(ctx, j)
		if e != nil {
			t.Fatal(e)
		}
		for i, turn := range section.Turns {
			if turn.Part != j.Part || turn.Position != int32(i) || (!turn.Skipped && len(turn.Recording) == 0) {
				t.Fatal("section mapping/audio", j.Part, i)
			}
		}
		// Release jobs for the actual worker pool below, preserving attempt counts.
		if e = repo.FinishEvaluation(ctx, j, nil, "test_retry"); e != nil {
			t.Fatal(e)
		}
	}
	if _, e := pool.Exec(ctx, "UPDATE speaking_evaluation_jobs SET available_at=now() WHERE session_id=$1", v.ID); e != nil {
		t.Fatal(e)
	}
	work, cancel := context.WithCancel(ctx)
	s.Run(work)
	deadline := time.Now().Add(10 * time.Second)
	for {
		got := request("GET", path, nil, cookie, 200)
		_ = json.Unmarshal(got.Body.Bytes(), &v)
		if v.Status == "completed" {
			break
		}
		if time.Now().After(deadline) {
			cancel()
			t.Fatal("report worker timed out", v.Status, v.ErrorCode)
		}
		time.Sleep(50 * time.Millisecond)
	}
	cancel()
	if e := pool.QueryRow(ctx, "SELECT count(*) FROM speaking_evaluation_jobs WHERE session_id=$1 AND status='completed'", v.ID).Scan(&n); e != nil || n != 4 {
		t.Fatal("durable section and aggregate results", n, e)
	}
	var report d.Report
	if e := json.Unmarshal(v.Report, &report); e != nil || report.Overall == nil || *report.Overall != 6.5 {
		t.Fatal("report", report, e)
	}
	// A failed aggregation retry must reuse the three completed audio jobs.
	if _, e := pool.Exec(ctx, "UPDATE speaking_sessions SET status='failed' WHERE id=$1", v.ID); e != nil {
		t.Fatal(e)
	}
	request("POST", path+"/submit", nil, cookie, 202)
	aggregate, e := repo.ClaimEvaluation(ctx)
	if e != nil || aggregate.Part != 0 {
		t.Fatal("retry repeated completed audio", aggregate, e)
	}
	_, evidence, e := repo.LoadEvaluation(ctx, aggregate)
	if e != nil || len(evidence) != 3 {
		t.Fatal("stored evidence", e)
	}
	for _, section := range evidence {
		var partReport d.Report
		_ = json.Unmarshal(section.Report, &partReport)
		for _, response := range partReport.Responses {
			if v.Turns[response.Position].Part != section.Part {
				t.Fatal("lost original answer positions")
			}
		}
	}
	if e = repo.FinishEvaluation(ctx, aggregate, v.Report, ""); e != nil {
		t.Fatal(e)
	}
	if _, e := pool.Exec(ctx, "UPDATE speaking_sessions SET audio_expires_at=now()-interval '1 second' WHERE id=$1", v.ID); e != nil {
		t.Fatal(e)
	}
	if e := s.Repo.Cleanup(ctx); e != nil {
		t.Fatal(e)
	}
	request("GET", path+"/turns/0/recording", nil, cookie, 404)
	request("GET", path, nil, cookie, 200)
	request("DELETE", path, nil, cookie, 204)
	request("GET", path, nil, cookie, 404)
	// Deletion must not erase daily usage records or allow unlimited TTS spending.
	for range 2 {
		res := request("POST", "/api/v1/speaking/sessions", []byte(`{"consent":true}`), cookie, 201)
		_ = json.Unmarshal(res.Body.Bytes(), &v)
		job, e := s.Repo.ClaimVoice(ctx)
		if e != nil {
			t.Fatal(e)
		}
		request("DELETE", "/api/v1/speaking/sessions/"+v.ID, nil, cookie, 204)
		if e = s.Repo.FinishVoice(ctx, job, wav); e != nil {
			t.Fatal(e)
		}
		request("GET", "/api/v1/speaking/sessions/"+v.ID, nil, cookie, 404)
	}
	request("POST", "/api/v1/speaking/sessions", []byte(`{"consent":true}`), cookie, 429)
	t.Log("speaking: consent, ownership, CSRF, stable examiner, encrypted WAV roundtrip, turn ordering, idempotent submit, worker report, expiry and deletion-resistant quotas passed")
}
