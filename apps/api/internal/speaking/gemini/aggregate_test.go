package gemini

import (
	"context"
	"encoding/json"
	d "github.com/own/mock-ai/apps/api/internal/speaking/domain"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAggregationPreservesEvidenceAndSendsOnlyText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if e := json.NewDecoder(r.Body).Decode(&body); e != nil {
			t.Error(e)
		}
		b, _ := json.Marshal(body)
		if strings.Contains(string(b), "inlineData") || strings.Contains(string(b), "fileData") {
			t.Error("aggregation sent audio")
		}
		if !strings.Contains(string(b), "verified quote") {
			t.Error("missing section evidence")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"candidates": []any{map[string]any{"finishReason": "STOP", "content": map[string]any{"parts": []any{map[string]string{"text": `{"assessable":false,"summary":"Combined","responses":[{"position":99,"transcript":"invented"}]}`}}}}}})
	}))
	defer server.Close()
	c := New(nil, "", "")
	c.Base = server.URL
	evidence := []d.SectionEvidence{}
	for i := int32(0); i < 3; i++ {
		raw, _ := json.Marshal(d.Report{Responses: []d.Response{{Position: i, Transcript: "verified quote", Feedback: "feedback"}}, Issues: []d.Issue{{Position: i, Quote: "verified quote", Second: 1}}})
		evidence = append(evidence, d.SectionEvidence{Part: i + 1, Report: raw})
	}
	raw, e := c.aggregate(context.Background(), "test-key", d.Session{Model: "test"}, evidence)
	if e != nil {
		t.Fatal(e)
	}
	var report d.Report
	_ = json.Unmarshal(raw, &report)
	if len(report.Responses) != 3 || len(report.Issues) != 3 {
		t.Fatal("lost evidence", report)
	}
	for i, response := range report.Responses {
		if response.Position != int32(i) || response.Transcript != "verified quote" {
			t.Fatal("changed evidence", response)
		}
	}
}

func TestProviderErrorsHidePayloadAndEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "secret-provider-payload", 429) }))
	defer server.Close()
	c := New(nil, "", "")
	c.Base = server.URL
	_, e := c.generateReport(context.Background(), "secret-key", "secret-model", "", nil, schema())
	if e == nil || safeReason(e) != "generate_http_429" {
		t.Fatalf("unsafe or unclassified error: %v", e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, e = c.generateReport(ctx, "secret-key", "secret-model", "", nil, schema())
	if e == nil || strings.Contains(e.Error(), "secret") {
		t.Fatal(e)
	}
}
