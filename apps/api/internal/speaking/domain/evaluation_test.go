package domain

import (
	"encoding/json"
	"testing"
)

func TestUnverifiedAnnotationsDoNotDiscardValidAssessment(t *testing.T) {
	s, r := fixture()
	r.Issues = append(r.Issues, Issue{Position: 0, Second: 1, Quote: "invented quotation"}, Issue{Position: 0, Second: 999, Quote: "I study"})
	raw, _ := json.Marshal(r)
	clean, n := VerifiedIssues(raw, s)
	got, e := ParseReport(clean, s)
	if n != 2 || e != nil || len(got.Issues) != 1 || got.Overall == nil || *got.Overall != 6.5 {
		t.Fatal(n, got, e)
	}
	if got.Issues[0].Quote != r.Issues[0].Quote {
		t.Fatal("changed valid evidence")
	}
	r.Criteria = nil
	raw, _ = json.Marshal(r)
	clean, _ = VerifiedIssues(raw, s)
	if _, e = ParseReport(clean, s); e == nil {
		t.Fatal("accepted invalid mandatory criteria")
	}
}
