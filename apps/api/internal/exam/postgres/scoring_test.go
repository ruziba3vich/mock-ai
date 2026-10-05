package postgres

import (
	"testing"

	db "github.com/own/mock-ai/apps/api/internal/platform/postgres/sqlc"
)

func TestAnswerCorrect(t *testing.T) {
	tests := []struct {
		name     string
		kind     db.QuestionType
		values   []string
		accepted []acceptedAnswer
		want     bool
	}{
		{name: "case insensitive and whitespace", kind: db.QuestionTypeShortAnswer, values: []string{"  New   York "}, accepted: []acceptedAnswer{{value: "new york"}}, want: true},
		{name: "case sensitive", kind: db.QuestionTypeShortAnswer, values: []string{"b"}, accepted: []acceptedAnswer{{value: "B", caseSensitive: true}}, want: false},
		{name: "alternative answer", kind: db.QuestionTypeShortAnswer, values: []string{"UK"}, accepted: []acceptedAnswer{{value: "Britain"}, {value: "UK"}}, want: true},
		{name: "multiple select order independent", kind: db.QuestionTypeMultipleSelect, values: []string{"C", "A"}, accepted: []acceptedAnswer{{value: "A"}, {value: "C"}}, want: true},
		{name: "multiple select must be exact", kind: db.QuestionTypeMultipleSelect, values: []string{"A"}, accepted: []acceptedAnswer{{value: "A"}, {value: "C"}}, want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := answerCorrect(test.kind, test.values, test.accepted); got != test.want {
				t.Fatalf("answerCorrect() = %v, want %v", got, test.want)
			}
		})
	}
}
