package live

// Operator-only live replay. Production access is read-only; all session writes
// go to an in-memory repository. Decrypted human audio never touches disk.
import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	d "github.com/own/mock-ai/apps/api/internal/speaking/domain"
)

type replayRepo struct{ *testRepo }

func (r replayRepo) RenewLive(context.Context, d.Session) error { return nil }

func TestAuthorizedHumanLiveReplay(t *testing.T) {
	source := os.Getenv("SPEAKING_REPLAY_SOURCE")
	if source == "" {
		t.Skip("explicit authorization and source required; incurs live provider usage")
	}
	if !regexp.MustCompile(`^[0-9a-f-]{36}$`).MatchString(source) {
		t.Fatal("invalid source")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	ssh := func(command, input string) []byte {
		t.Helper()
		readCtx, stop := context.WithTimeout(context.Background(), 30*time.Second)
		defer stop()
		home, _ := os.UserHomeDir()
		cmd := exec.CommandContext(readCtx, "ssh", "-i", filepath.Join(home, ".ssh/arch"), "-o", "BatchMode=yes", "statistics@46.8.176.85", command)
		cmd.Stdin = strings.NewReader(input)
		b, err := cmd.Output()
		if err != nil {
			t.Fatal("read-only source retrieval failed")
		}
		return b
	}
	sql := func(q string) []byte {
		return ssh("docker exec -i mock_ai_postgres psql -X -qAt -U mock_ai -d mock_ai -v ON_ERROR_STOP=1", q)
	}
	fingerprint := func() string {
		return strings.TrimSpace(string(sql(fmt.Sprintf("SELECT md5(row_to_json(s)::text || (SELECT json_agg(row_to_json(t) ORDER BY position)::text FROM speaking_turns t WHERE session_id=s.id)) FROM speaking_sessions s WHERE id='%s';", source))))
	}
	before := fingerprint()
	defer func() {
		if fingerprint() != before {
			t.Error("original source changed during replay")
		}
	}()
	var env []string
	if json.Unmarshal(ssh("docker inspect --format '{{json .Config.Env}}' mock_ai_api", ""), &env) != nil {
		t.Fatal("config unavailable")
	}
	config := map[string]string{}
	for _, v := range env {
		k, val, ok := strings.Cut(v, "=")
		if ok {
			config[k] = val
		}
	}
	master, err := hex.DecodeString(config["AI_KEY_ENCRYPTION_KEY"])
	if err != nil {
		t.Fatal("invalid encryption config")
	}
	defer clear(master)
	block, err := aes.NewCipher(master)
	if err != nil {
		t.Fatal("invalid encryption config")
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatal(err)
	}
	var session d.Session
	raw := sql(fmt.Sprintf("SELECT json_build_object('id',id,'examiner',examiner,'topic',topic,'turns',(SELECT json_agg(json_build_object('position',position,'part',part,'prompt',prompt,'seconds',seconds,'prep_seconds',prep_seconds) ORDER BY position) FROM speaking_turns WHERE session_id=s.id)) FROM speaking_sessions s WHERE id='%s' AND status='completed' AND audio_expires_at>now();", source))
	if json.Unmarshal(raw, &session) != nil || len(session.Turns) < 3 {
		t.Fatal("completed unexpired source unavailable")
	}
	session.ID = "temporary-in-memory-replay"
	session.LiveModel = config["GEMINI_SPEAKING_LIVE_MODEL"]
	if session.LiveModel == "" {
		session.LiveModel = "gemini-3.8-live"
	}
	// Use the new local key without exposing it in command arguments or output.
	dotenv, err := os.ReadFile("../../../../../.env")
	if err != nil {
		t.Fatal("local credential unavailable")
	}
	defer clear(dotenv)
	key := ""
	for _, line := range strings.Split(string(dotenv), "\n") {
		k, v, ok := strings.Cut(line, "=")
		if ok && strings.TrimSpace(k) == "GEMINI_API_KEY" {
			key = strings.Trim(strings.TrimSpace(v), "\"'")
		}
	}
	if key == "" {
		t.Fatal("local credential unavailable")
	}
	answers := make([][]byte, 2)
	for i := range answers {
		encoded := strings.TrimSpace(string(sql(fmt.Sprintf("SELECT encode(recording,'base64') FROM speaking_turns WHERE session_id='%s' AND position=%d AND answered AND NOT skipped;", source, i))))
		sealed, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil || len(sealed) < 28 {
			t.Fatal("recording unavailable")
		}
		wav, err := gcm.Open(nil, sealed[:12], sealed[12:], []byte(fmt.Sprintf("mockai:speaking:recording:v1:%s:%d", source, i)))
		if err != nil {
			t.Fatal("cannot decrypt source")
		}
		defer clear(wav)
		if len(wav) < 44 || string(wav[:4]) != "RIFF" || string(wav[36:40]) != "data" {
			t.Fatal("unsupported replay audio")
		}
		answers[i] = wav[44:]
	}
	var mu sync.Mutex
	var spoken strings.Builder
	// Transparent provider relay records examiner transcription only, never keys
	// or the student's transcript, and exercises the real local controller.
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		downstream, e := websocket.Accept(w, r, nil)
		if e != nil {
			return
		}
		defer downstream.CloseNow()
		upstream, _, e := websocket.Dial(ctx, "wss://generativelanguage.googleapis.com/ws/google.ai.generativelanguage.v1beta.GenerativeService.BidiGenerateContent?key="+url.QueryEscape(key), nil)
		if e != nil {
			return
		}
		defer upstream.CloseNow()
		upstream.SetReadLimit(2 << 20)
		downstream.SetReadLimit(2 << 20)
		go func() {
			for {
				kind, b, e := downstream.Read(ctx)
				if e != nil {
					return
				}
				if upstream.Write(ctx, kind, b) != nil {
					return
				}
			}
		}()
		for {
			kind, b, e := upstream.Read(ctx)
			if e != nil {
				return
			}
			var m struct {
				ToolCall      struct{ FunctionCalls []struct{ Name string } }
				ServerContent struct {
					OutputTranscription struct{ Text string }
					TurnComplete        bool
				}
			}
			_ = json.Unmarshal(b, &m)
			for _, call := range m.ToolCall.FunctionCalls {
				t.Logf("Provider decision: %s", call.Name)
			}
			mu.Lock()
			spoken.WriteString(m.ServerContent.OutputTranscription.Text)
			mu.Unlock()
			if downstream.Write(ctx, kind, b) != nil {
				return
			}
		}
	}))
	defer proxy.Close()
	repo := replayRepo{&testRepo{saves: make(chan saved, 4), released: make(chan bool, 1)}}
	service := Service{Repo: repo, Provider: testProvider{strings.Replace(proxy.URL, "http", "ws", 1)}}
	app := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, e := websocket.Accept(w, r, nil)
		if e != nil {
			return
		}
		defer c.CloseNow()
		_ = service.Run(ctx, c, session)
	}))
	defer app.Close()
	client, _, err := websocket.Dial(ctx, strings.Replace(app.URL, "http", "ws", 1), nil)
	if err != nil {
		t.Fatal("local controller connection failed")
	}
	defer client.CloseNow()
	client.SetReadLimit(2 << 20)
	position := 0
	started := map[int]bool{}
	var stopStream context.CancelFunc
	defer func() {
		if stopStream != nil {
			stopStream()
		}
	}()
	audioBytes := 0
	savedCount := 0
	for {
		kind, b, e := client.Read(ctx)
		if e != nil {
			t.Fatal("live replay ended before two answer transitions")
		}
		if kind == websocket.MessageBinary {
			audioBytes += len(b)
			continue
		}
		var event struct {
			Type, Phase string
			Position    int
		}
		_ = json.Unmarshal(b, &event)
		if event.Type == "state" {
			position = event.Position
			if event.Phase != "listening" && stopStream != nil {
				stopStream()
				stopStream = nil
			}
		}
		if event.Type == "audio_end" {
			mu.Lock()
			text := spoken.String()
			spoken.Reset()
			mu.Unlock()
			t.Logf("Examiner turn %d (%d audio bytes): %s", position, audioBytes, text)
			if audioBytes == 0 {
				t.Fatal("missing examiner audio")
			}
			audioBytes = 0
			if position == 2 {
				for savedCount < 2 {
					select {
					case result := <-repo.saves:
						if result.pos != int32(savedCount) {
							t.Fatal("unexpected answer progression")
						}
						savedCount++
					case <-ctx.Done():
						t.Fatal("missing saved replay answer")
					}
				}
				t.Log("Two human answers replayed at real-time speed; controller saved both and delivered subsequent examiner turns. Production source unchanged.")
				return
			}
			if write(ctx, client, map[string]string{"type": "playback_done"}) != nil {
				t.Fatal("playback confirmation failed")
			}
		}
		if event.Type == "state" && event.Phase == "listening" && position < 2 && !started[position] {
			started[position] = true
			streamCtx, stop := context.WithCancel(ctx)
			stopStream = stop
			go func(pcm []byte) {
				ticker := time.NewTicker(100 * time.Millisecond)
				defer ticker.Stop()
				for offset := 0; offset < len(pcm); offset += 3200 {
					select {
					case <-streamCtx.Done():
						return
					case <-ticker.C:
					}
					end := min(offset+3200, len(pcm))
					if client.Write(ctx, websocket.MessageBinary, pcm[offset:end]) != nil {
						return
					}
				}
				// Keep the mic stream alive with silence, letting real VAD decide the end.
				for {
					select {
					case <-streamCtx.Done():
						return
					case <-ticker.C:
					}
					if client.Write(ctx, websocket.MessageBinary, make([]byte, 3200)) != nil {
						return
					}
				}
			}(answers[position])
		}
	}
}
