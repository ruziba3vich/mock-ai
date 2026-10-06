package gemini

import (
	"context"
	"encoding/binary"
	"encoding/json"
	d "github.com/own/mock-ai/apps/api/internal/speaking/domain"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func TestLiveInterviewReport(t *testing.T) {
	key, dir := os.Getenv("SPEAKING_LIVE_KEY"), os.Getenv("SPEAKING_LIVE_FIXTURES")
	if key == "" || dir == "" {
		t.Skip("requires explicit live key and 11 distinct synthetic WAV answers")
	}
	topic, turns, e := d.NewPlan()
	for topic != "Learning something new" && e == nil {
		topic, turns, e = d.NewPlan()
	}
	if e != nil {
		t.Fatal(e)
	}
	for i := range turns {
		b, e := os.ReadFile(filepath.Join(dir, strconv.Itoa(i)+".wav"))
		if e != nil {
			t.Fatal(e)
		}
		var pcm []byte
		for p := 12; p+8 <= len(b); {
			size := int(binary.LittleEndian.Uint32(b[p+4:]))
			if p+8+size > len(b) {
				t.Fatal("invalid fixture WAV")
			}
			if string(b[p:p+4]) == "data" {
				pcm = b[p+8 : p+8+size]
				break
			}
			p += 8 + size + size%2
		}
		turns[i].Recording = wave(pcm)
		turns[i].DurationMS, e = d.ValidateAudio(turns[i].Recording, turns[i].Seconds)
		if e != nil {
			t.Fatal("fixture duration or format", i, e)
		}
		turns[i].Answered = true
	}
	s := d.Session{Model: "gemini-3.7-flash", Topic: topic, Turns: turns}
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
	c := New(nil, "", "gemini-3.8-flash-tts")
	raw, e := c.evaluate(ctx, key, s)
	if e != nil {
		t.Fatal(e)
	}
	report, e := d.ParseReport(raw, s)
	if e != nil {
		t.Fatal("live report failed validation", e)
	}
	if !report.Assessable || report.Overall == nil {
		t.Fatal("distinct synthetic answers were not assessable:", report.Summary)
	}
	if len(report.Responses) != 11 || len(report.Criteria) != 4 {
		t.Fatal("incomplete report")
	}
	t.Logf("live distinct-answer report: band %.1f, %d criteria, %d transcripts, %d timestamped issues", *report.Overall, len(report.Criteria), len(report.Responses), len(report.Issues))
	artifact, _ := json.MarshalIndent(report, "", "  ")
	if e = os.WriteFile(filepath.Join(dir, "report.json"), artifact, 0600); e != nil {
		t.Fatal(e)
	}
}

// Explicit opt-in only: incurs provider usage. Never print the key or raw audio.
func TestLiveRegionalVoices(t *testing.T) {
	key := os.Getenv("SPEAKING_LIVE_KEY")
	if key == "" {
		t.Skip("requires explicit SPEAKING_LIVE_KEY")
	}
	for _, country := range d.Countries {
		for _, voice := range country.Voices {
			t.Run(voice.Code+"-"+voice.Gender, func(t *testing.T) {
				t.Parallel()
				ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
				defer cancel()
				c := New(nil, "", "gemini-3.8-flash-tts")
				audio, e := c.speak(ctx, key, d.VoiceJob{Examiner: voice, Prompt: "What do you enjoy about the place where you live?"})
				if e != nil || len(audio) < 1000 {
					t.Fatal("live voice generation failed", e)
				}
				t.Logf("catalog voice %s returned %d WAV bytes", voice.Voice, len(audio))
			})
		}
	}
}
func TestLiveLargeAudioUploadAndDelete(t *testing.T) {
	key := os.Getenv("SPEAKING_LIVE_KEY")
	if key == "" {
		t.Skip("requires explicit SPEAKING_LIVE_KEY")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()
	c := New(nil, "", "gemini-3.8-flash-tts")
	uri, name, e := c.upload(ctx, key, wave(make([]byte, 12_100_000)))
	if e != nil || uri == "" {
		t.Fatal("large WAV upload failed", e)
	}
	req, _ := http.NewRequestWithContext(ctx, "DELETE", c.Base+"/v1beta/"+name, nil)
	req.Header.Set("x-goog-api-key", key)
	if _, _, e = c.do(req, 10000); e != nil {
		t.Fatal("provider file deletion failed", e)
	}
	t.Log("12.1 MB WAV uploaded, ready, and deleted")
}
