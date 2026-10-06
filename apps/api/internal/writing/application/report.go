package application

import (
	"bytes"
	"encoding/json"
	"errors"
	"math"
	"strings"
)

const RubricVersion = "task2-v1"

type Criterion struct {
	Key    string  `json:"key"`
	Score  float64 `json:"score"`
	Reason string  `json:"reason"`
}
type Issue struct {
	Category    string `json:"category"`
	Quote       string `json:"quote"`
	Explanation string `json:"explanation"`
	Suggestion  string `json:"suggestion"`
	Example     string `json:"example"`
}
type Rewrite struct {
	Original    string `json:"original"`
	Improved    string `json:"improved"`
	Explanation string `json:"explanation"`
}
type Report struct {
	ModelVersion string      `json:"model_version"`
	Overall      float64     `json:"overall"`
	Summary      string      `json:"summary"`
	Criteria     []Criterion `json:"criteria"`
	Strengths    []string    `json:"strengths"`
	Issues       []Issue     `json:"issues"`
	NextSteps    []string    `json:"next_steps"`
	Rewrite      Rewrite     `json:"rewrite"`
}

// The model supplies criterion estimates, never the authoritative aggregate.
func ParseReport(raw []byte, essay string) (Report, error) {
	var r Report
	if len(raw) > 64000 || json.Unmarshal(raw, &r) != nil {
		return r, errors.New("invalid report")
	}
	var fields struct {
		Criteria []map[string]json.RawMessage `json:"criteria"`
	}
	if json.Unmarshal(raw, &fields) != nil {
		return r, errors.New("invalid report")
	}
	for _, c := range fields.Criteria {
		score, ok := c["score"]
		if !ok || bytes.Equal(bytes.TrimSpace(score), []byte("null")) {
			return r, errors.New("criterion score missing")
		}
	}
	required := map[string]bool{"task_response": false, "coherence_cohesion": false, "lexical_resource": false, "grammar": false}
	sum := 0.0
	if len(r.Criteria) != 4 || strings.TrimSpace(r.Summary) == "" || len(r.Summary) > 4000 {
		return r, errors.New("incomplete report")
	}
	for _, c := range r.Criteria {
		seen, ok := required[c.Key]
		if !ok || seen || math.IsNaN(c.Score) || c.Score < 0 || c.Score > 9 || c.Score*2 != math.Trunc(c.Score*2) || strings.TrimSpace(c.Reason) == "" || len(c.Reason) > 4000 {
			return r, errors.New("invalid criteria")
		}
		required[c.Key] = true
		sum += c.Score
	}
	if len(r.Issues) > 15 || len(r.NextSteps) < 1 || len(r.NextSteps) > 6 || len(r.Strengths) > 5 {
		return r, errors.New("invalid feedback length")
	}
	categories := map[string]bool{"task_response": true, "coherence": true, "vocabulary": true, "grammar": true, "argument": true}
	for _, i := range r.Issues {
		if !categories[i.Category] || strings.TrimSpace(i.Quote) == "" || !strings.Contains(essay, i.Quote) || len(i.Quote) > 2000 || i.Explanation == "" || i.Suggestion == "" {
			return r, errors.New("feedback does not match essay")
		}
	}
	if r.Rewrite.Original == "" || !strings.Contains(essay, r.Rewrite.Original) || r.Rewrite.Improved == "" || r.Rewrite.Explanation == "" {
		return r, errors.New("invalid rewrite evidence")
	}
	for _, v := range append(append([]string{}, r.Strengths...), r.NextSteps...) {
		if strings.TrimSpace(v) == "" || len(v) > 2000 {
			return r, errors.New("invalid guidance")
		}
	}
	r.Overall = math.Round(sum/4*2) / 2
	return r, nil
}
