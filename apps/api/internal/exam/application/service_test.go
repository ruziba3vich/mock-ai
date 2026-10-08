package application

import (
	"context"
	"errors"
	"testing"
)

func TestCleanAnswers(t *testing.T) {
	tests := []struct {
		name   string
		values []string
		want   []string
		fail   bool
	}{
		{name: "trims values", values: []string{"  King Street  "}, want: []string{"King Street"}},
		{name: "supports multiple select", values: []string{"A", " C "}, want: []string{"A", "C"}},
		{name: "rejects no values", values: nil, fail: true},
		{name: "rejects blank values", values: []string{"  "}, fail: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := cleanAnswers(test.values)
			if test.fail {
				if err == nil {
					t.Fatal("expected an error")
				}
				return
			}
			if err != nil {
				t.Fatalf("cleanAnswers() error = %v", err)
			}
			if len(got) != len(test.want) {
				t.Fatalf("cleanAnswers() = %v, want %v", got, test.want)
			}
			for i := range got {
				if got[i] != test.want[i] {
					t.Fatalf("cleanAnswers() = %v, want %v", got, test.want)
				}
			}
		})
	}
}

func TestUpdateStudyProfileRejectsInvalidValues(t *testing.T) {
	service := &Service{}
	tests := []StudyProfileInput{
		{TargetBand: 9.5, Track: "academic", DailyGoalMinutes: 20},
		{TargetBand: 7.2, Track: "academic", DailyGoalMinutes: 20},
		{TargetBand: 7, Track: "unsupported", DailyGoalMinutes: 20},
		{TargetBand: 7, Track: "academic", DailyGoalMinutes: 5},
		{TargetBand: 7, Track: "academic", DailyGoalMinutes: 20, ExamDate: "not-a-date"},
	}
	for _, input := range tests {
		if _, err := service.UpdateStudyProfile(context.Background(), "user", input); !errors.Is(err, ErrInvalidProfile) {
			t.Fatalf("UpdateStudyProfile(%+v) error = %v, want ErrInvalidProfile", input, err)
		}
	}
}

func TestManagedMediaPathSupportsPrivateAndPublicContent(t *testing.T) {
	content := Content{
		Title: "Private listening", Skill: "listening", Purpose: "mock", Duration: 60,
		AccessScope: "private", OwnerUserID: "a6b92277-5f5d-48bf-a5d2-ef004f75015d",
		Sections: []ContentSection{{
			Title: "Section 1", AudioURL: "/api/v1/private-media/11111111-1111-1111-1111-111111111111",
			Questions: []ContentQuestion{{Type: "short_answer", Prompt: "Answer", Answers: []string{"answer"}, Explanation: "Explanation"}},
		}},
		Bands: []ContentBand{{Minimum: 0, Band: 0}},
	}
	if err := ValidateContent(content, true); err != nil {
		t.Fatalf("private owned media should validate: %v", err)
	}
	content.AccessScope, content.OwnerUserID = "public", ""
	if err := ValidateContent(content, true); err != nil {
		t.Fatalf("published content may reference access-controlled managed media: %v", err)
	}
	content.AccessScope = "private"
	if err := ValidateContent(content, false); !errors.Is(err, ErrInvalidProfile) {
		t.Fatalf("private content without an owner must fail: %v", err)
	}
}
