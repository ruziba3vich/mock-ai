package live

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"github.com/coder/websocket"
	d "github.com/own/mock-ai/apps/api/internal/speaking/domain"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type saved struct {
	pos        int32
	wav        []byte
	text, next string
}
type testRepo struct {
	Repository
	saves    chan saved
	released chan bool
}

func (r *testRepo) SaveLive(_ context.Context, _ d.Session, p int32, b []byte, text string, done bool, next string) error {
	r.saves <- saved{p, b, text, next}
	return nil
}
func (r *testRepo) ReleaseLive(context.Context, d.Session) error { r.released <- true; return nil }

type testProvider struct{ url string }

func (p testProvider) ConnectLive(ctx context.Context) (*websocket.Conn, error) {
	c, _, e := websocket.Dial(ctx, p.url, nil)
	return c, e
}
func TestLiveProgressionWaitAndOutputGate(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	upstream := make(chan *websocket.Conn, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, e := websocket.Accept(w, r, nil)
		if e != nil {
			return
		}
		upstream <- c
		<-ctx.Done()
		c.CloseNow()
	}))
	defer server.Close()
	topic, turns, _ := d.NewPlan()
	session := d.Session{ID: "session", UserID: "user", LiveModel: "test", Turns: turns, Topic: topic}
	repo := &testRepo{saves: make(chan saved, 2), released: make(chan bool, 1)}
	service := Service{Repo: repo, Provider: testProvider{strings.Replace(server.URL, "http", "ws", 1)}}
	app := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, e := websocket.Accept(w, r, nil)
		if e != nil {
			return
		}
		defer c.CloseNow()
		_ = service.Run(ctx, c, session)
	}))
	defer app.Close()
	client, _, e := websocket.Dial(ctx, strings.Replace(app.URL, "http", "ws", 1), nil)
	if e != nil {
		t.Fatal(e)
	}
	defer client.CloseNow()
	provider := <-upstream
	receive := func(c *websocket.Conn) map[string]any {
		t.Helper()
		kind, b, e := c.Read(ctx)
		if e != nil {
			t.Fatal(e)
		}
		if kind != websocket.MessageText {
			t.Fatal("unexpected audio while examiner must be silent")
		}
		var v map[string]any
		if json.Unmarshal(b, &v) != nil {
			t.Fatal("bad JSON")
		}
		return v
	}
	send := func(c *websocket.Conn, v any) {
		t.Helper()
		if e := write(ctx, c, v); e != nil {
			t.Fatal(e)
		}
	}
	tool := func(name string) {
		send(provider, map[string]any{"toolCall": map[string]any{"functionCalls": []any{map[string]any{"id": "call", "name": name, "args": map[string]string{"question": "What is your typical weekday like?"}}}}})
	}
	receive(provider)
	send(provider, map[string]any{"setupComplete": map[string]any{}})
	receive(client)
	receive(provider)
	send(provider, map[string]any{"serverContent": map[string]bool{"turnComplete": true}})
	// Empty provider/tool-only turns trigger a bounded retry, not listening.
	if receive(provider)["clientContent"] == nil {
		t.Fatal("missing empty-audio retry")
	}
	send(provider, map[string]any{"serverContent": map[string]any{"modelTurn": map[string]any{"parts": []any{map[string]any{"inlineData": map[string]string{"data": base64.StdEncoding.EncodeToString(make([]byte, 4800)), "mimeType": "audio/pcm;rate=24000"}}}}, "turnComplete": true}})
	if kind, _, err := client.Read(ctx); err != nil || kind != websocket.MessageBinary {
		t.Fatal("examiner audio missing", err)
	}
	receive(client)
	send(client, map[string]string{"type": "playback_done"})
	if receive(client)["phase"] != "listening" {
		t.Fatal("not listening")
	}
	// Noise/no transcript cannot advance, even if a provider proposes doing so.
	tool("advance_question")
	receive(provider)
	if receive(client)["position"] != float64(0) {
		t.Fatal("noise advanced question")
	}
	pcm := make([]byte, 8000)
	for i := 0; i < len(pcm); i += 2 {
		binary.LittleEndian.PutUint16(pcm[i:], 2000)
	}
	for i := 0; i < 4; i++ {
		if e = client.Write(ctx, websocket.MessageBinary, pcm); e != nil {
			t.Fatal(e)
		}
		receive(provider)
	}
	send(provider, map[string]any{"serverContent": map[string]any{"inputTranscription": map[string]string{"text": "I study computer science because"}}})
	tool("wait_for_student")
	receive(provider)
	if receive(client)["phase"] != "listening" {
		t.Fatal("thinking pause interrupted")
	}
	// Unapproved output after a pause must not reach the browser.
	send(provider, map[string]any{"serverContent": map[string]any{"modelTurn": map[string]any{"parts": []any{map[string]any{"inlineData": map[string]string{"data": "AAAAAA==", "mimeType": "audio/pcm;rate=24000"}}}}, "turnComplete": true}})
	send(provider, map[string]any{"serverContent": map[string]any{"inputTranscription": map[string]string{"text": " I enjoy solving practical problems."}}})
	tool("advance_question")
	receive(provider)
	state := receive(client)
	if state["position"] != float64(1) || state["phase"] != "examiner" {
		t.Fatalf("did not advance: %v", state)
	}
	select {
	case v := <-repo.saves:
		if v.pos != 0 || len(v.wav) != 32044 || !strings.Contains(v.text, "practical problems") {
			t.Fatal("answer not saved")
		}
	case <-ctx.Done():
		t.Fatal("no saved answer")
	}
	// A second call in the same provider turn cannot skip another question.
	tool("advance_question")
	receive(provider)
	if receive(client)["position"] != float64(1) {
		t.Fatal("double advancement")
	}
	client.CloseNow()
	select {
	case <-repo.released:
	case <-ctx.Done():
		t.Fatal("lease not released")
	}
	cancel()
}
func TestMeaningful(t *testing.T) {
	for _, s := range []string{"", "um uh hmm", "[background noise]"} {
		if Meaningful(s) {
			t.Fatalf("noise accepted: %s", s)
		}
	}
	if !Meaningful("I enjoy studying languages.") {
		t.Fatal("speech rejected")
	}
}
