package application

import (
	"encoding/json"
	"testing"
)

func TestReportValidationAndAggregate(t *testing.T) {
	essay := "Public transport reduces congestion. It also improves access to work."
	r := Report{Overall: 9, Summary: "Develop the explanation.", TopicAlignment: TopicAlignment{Status: "aligned", PromptTopic: "public transport investment", EssayTopic: "public transport benefits", Explanation: "The essay addresses the task."}, Criteria: []Criterion{{Key: "task_response", Score: 6, Reason: "Evidence is thin.", HowToImprove: "Develop one example."}, {Key: "coherence_cohesion", Score: 7, Reason: "Clear organisation.", HowToImprove: "Link the example back to the claim."}, {Key: "lexical_resource", Score: 6, Reason: "Limited variety.", HowToImprove: "Use more precise transport vocabulary."}, {Key: "grammar", Score: 7, Reason: "Mostly accurate.", HowToImprove: "Vary complex sentence structures."}}, Strengths: []string{"Clear position."}, Issues: []Issue{{"argument", "Public transport reduces congestion.", "Explain the mechanism.", "Explain how fewer cars help.", "Fewer car journeys free road space."}}, NextSteps: []string{"Add a causal explanation."}, Rewrite: Rewrite{essay, "Reliable buses let more workers travel without a car.", "Makes the causal link explicit."}}
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
	r.Criteria[1].Key = "coherence_cohesion"
	r.TopicAlignment.Status = "unknown"
	raw, _ = json.Marshal(r)
	if _, e = ParseReport(raw, essay); e == nil {
		t.Fatal("invalid topic alignment accepted")
	}
}
