package application

import (
	"encoding/json"
	"testing"
)

func TestReportValidationAndAggregate(t *testing.T) {
	essay := "Public transport reduces congestion. It also improves access to work."
	r := Report{Overall: 9, Summary: "Develop the explanation.", Criteria: []Criterion{{"task_response", 6, "Evidence is thin."}, {"coherence_cohesion", 7, "Clear organisation."}, {"lexical_resource", 6, "Limited variety."}, {"grammar", 7, "Mostly accurate."}}, Strengths: []string{"Clear position."}, Issues: []Issue{{"argument", "Public transport reduces congestion.", "Explain the mechanism.", "Explain how fewer cars help.", "Fewer car journeys free road space."}}, NextSteps: []string{"Add a causal explanation."}, Rewrite: Rewrite{essay, "Reliable buses let more workers travel without a car.", "Makes the causal link explicit."}}
	raw, _ := json.Marshal(r)
	got, e := ParseReport(raw, essay)
	if e != nil || got.Overall != 6.5 {
		t.Fatal("overall must be server calculated", got, e)
	}
	r.Issues[0].Quote = "This sentence is not in the essay."
	raw, _ = json.Marshal(r)
	if _, e = ParseReport(raw, essay); e == nil {
		t.Fatal("invented evidence accepted")
	}
	r.Issues = nil
	r.Criteria[0].Score = 9.5
	raw, _ = json.Marshal(r)
	if _, e = ParseReport(raw, essay); e == nil {
		t.Fatal("invalid band accepted")
	}
	r.Criteria[0].Score = 6
	r.Criteria[1].Key = "task_response"
	raw, _ = json.Marshal(r)
	if _, e = ParseReport(raw, essay); e == nil {
		t.Fatal("duplicate criteria accepted")
	}
}
