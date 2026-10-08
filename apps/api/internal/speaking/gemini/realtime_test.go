package gemini

import (
	"context"
	"encoding/json"
	"github.com/coder/websocket"
	d "github.com/own/mock-ai/apps/api/internal/speaking/domain"
	"github.com/own/mock-ai/apps/api/internal/speaking/live"
	"net/url"
	"os"
	"testing"
	"time"
)

// Opt-in contract test against the actual provider, using the production setup
// (tools, patient VAD, regional voice, transcription), not a mocked endpoint.
func TestRealtimeSetup(t *testing.T) {
	testRealtimeVoice(t, d.Countries[2].Voices[0])
}
func TestRealtimeVoiceCatalog(t *testing.T) {
	for _, country := range d.Countries {
		for _, voice := range country.Voices {
			t.Run(voice.Code+"_"+voice.Gender, func(t *testing.T) { t.Parallel(); testRealtimeVoice(t, voice) })
		}
	}
}
func testRealtimeVoice(t *testing.T, voice d.Examiner) {
	key := os.Getenv("SPEAKING_LIVE_KEY")
	if key == "" {
		t.Skip("requires explicit live key")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	conn, _, e := websocket.Dial(ctx, "wss://generativelanguage.googleapis.com/ws/google.ai.generativelanguage.v1beta.GenerativeService.BidiGenerateContent?key="+url.QueryEscape(key), nil)
	if e != nil {
		t.Fatal("live connection failed")
	}
	defer conn.CloseNow()
	conn.SetReadLimit(2 << 20)
	topic, turns, e := d.NewPlan()
	if e != nil {
		t.Fatal(e)
	}
	s := d.Session{LiveModel: "gemini-3.8-live", Examiner: voice, Topic: topic, Turns: turns}
	send := func(v any) {
		b, _ := json.Marshal(v)
		if conn.Write(ctx, websocket.MessageText, b) != nil {
			t.Fatal("write failed")
		}
	}
	send(live.Setup(s, 0, ""))
	audio := 0
	for {
		_, b, e := conn.Read(ctx)
		if e != nil {
			t.Fatalf("live setup/audio failed: %v", e)
		}
		var m map[string]json.RawMessage
		if json.Unmarshal(b, &m) != nil {
			t.Fatal("invalid JSON")
		}
		if _, ok := m["setupComplete"]; ok {
			send(live.Control("Begin now. Ask the first question aloud: " + turns[0].Prompt))
		}
		if raw, ok := m["serverContent"]; ok {
			var c struct {
				TurnComplete bool `json:"turnComplete"`
				ModelTurn    struct {
					Parts []struct {
						InlineData *struct {
							Data string `json:"data"`
						} `json:"inlineData"`
					}
				} `json:"modelTurn"`
			}
			_ = json.Unmarshal(raw, &c)
			for _, p := range c.ModelTurn.Parts {
				if p.InlineData != nil {
					audio += len(p.InlineData.Data)
				}
			}
			if c.TurnComplete {
				if audio == 0 {
					t.Fatal("no streamed audio")
				}
				t.Logf("Live setup accepted; streamed %d base64 audio bytes", audio)
				return
			}
		}
	}
}
