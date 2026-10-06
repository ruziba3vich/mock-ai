package gemini

import (
	"context"
	"encoding/base64"
	"encoding/json"
	d "github.com/own/mock-ai/apps/api/internal/speaking/domain"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestVoiceUsesAssignedCatalogVoiceAndDoesNotStoreInteraction(t *testing.T) {
	wav := wave(make([]byte, 32000))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1beta/interactions" || r.Header.Get("x-goog-api-key") != "test-key" {
			t.Error("wrong route or auth")
		}
		var input map[string]any
		_ = json.NewDecoder(r.Body).Decode(&input)
		if input["store"] != false {
			t.Error("interaction retention enabled")
		}
		b, _ := json.Marshal(input)
		if !strings.Contains(string(b), "en-ie-advisor-2") {
			t.Error("lost assigned voice")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"steps": []any{map[string]any{"content": []any{map[string]any{"type": "audio", "mime_type": "audio/wav", "data": base64.StdEncoding.EncodeToString(wav)}}}}})
	}))
	defer server.Close()
	c := New(nil, "", "test-tts")
	c.Base = server.URL
	audio, e := c.speak(context.Background(), "test-key", d.VoiceJob{Prompt: "What do you enjoy?", Examiner: d.Countries[4].Voices[0]})
	if e != nil || len(audio) != len(wav) {
		t.Fatal(e, len(audio))
	}
}
func TestInlineEvaluationCarriesActualAudioAndOffsets(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var v struct {
			Contents []struct {
				Parts []json.RawMessage `json:"parts"`
			} `json:"contents"`
		}
		if e := json.NewDecoder(r.Body).Decode(&v); e != nil {
			t.Fatal(e)
		}
		b := string(v.Contents[0].Parts[1])
		if !strings.Contains(b, "inlineData") || !strings.Contains(b, "audio/wav") {
			t.Error("missing actual audio")
		}
		if !strings.Contains(string(v.Contents[0].Parts[0]), "offset_seconds") {
			t.Error("missing timing metadata")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"modelVersion": "verified-test-version", "candidates": []any{map[string]any{"finishReason": "STOP", "content": map[string]any{"parts": []any{map[string]any{"text": `{"assessable":false}`}}}}}})
	}))
	defer server.Close()
	c := New(nil, "", "test-tts")
	c.Base = server.URL
	raw, e := c.evaluate(context.Background(), "test-key", d.Session{Model: "test-model", Turns: []d.Turn{{Position: 0, Recording: wave(make([]byte, 32000)), DurationMS: 1000}}})
	if e != nil || !strings.Contains(string(raw), "verified-test-version") {
		t.Fatal(string(raw), e)
	}
}
func TestLongRecordingUploadsAndDeletesProviderFile(t *testing.T) {
	uploaded, deleted := false, false
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/upload/v1beta/files":
			w.Header().Set("X-Goog-Upload-URL", server.URL+"/upload-bytes")
			_, _ = w.Write([]byte(`{}`))
		case "/upload-bytes":
			uploaded = true
			_, _ = w.Write([]byte(`{"file":{"name":"files/test","uri":"test-file-uri","state":"ACTIVE"}}`))
		case "/v1beta/files/test":
			if r.Method != "DELETE" {
				t.Error("expected deletion")
			}
			deleted = true
			_, _ = w.Write([]byte(`{}`))
		default:
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			b, _ := json.Marshal(body)
			if !strings.Contains(string(b), "fileData") || strings.Contains(string(b), "inlineData") {
				t.Error("wrong audio transport")
			}
			_, _ = w.Write([]byte(`{"modelVersion":"test","candidates":[{"finishReason":"STOP","content":{"parts":[{"text":"{\"assessable\":false}"}]}}]}`))
		}
	}))
	defer server.Close()
	c := New(nil, "", "test")
	c.Base = server.URL
	_, e := c.evaluate(context.Background(), "key", d.Session{Model: "test", Turns: []d.Turn{{Recording: wave(make([]byte, 12_100_000))}}})
	if e != nil || !uploaded || !deleted {
		t.Fatal(e, uploaded, deleted)
	}
}
func TestAllSkippedDoesNotCallProvider(t *testing.T) {
	c := New(nil, "", "test")
	c.Base = "http://127.0.0.1:1"
	_, turns, _ := d.NewPlan()
	for i := range turns {
		turns[i].Skipped = true
	}
	raw, e := c.evaluate(context.Background(), "key", d.Session{Turns: turns})
	if e != nil {
		t.Fatal(e)
	}
	report, e := d.ParseReport(raw, d.Session{Turns: turns})
	if e != nil || report.Overall != nil || report.Assessable {
		t.Fatal(report, e)
	}
}
