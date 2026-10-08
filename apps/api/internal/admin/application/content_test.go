package application

import (
	"encoding/json"
	"testing"
)

func TestManagedContentValidation(t *testing.T) {
	valid := map[string]json.RawMessage{
		"daily_challenge": json.RawMessage(`{"metric":"today_questions","target":5,"action":"practice"}`),
		"achievement":     json.RawMessage(`{"metric":"mocks","target":1,"icon":"M"}`),
		"vocabulary":      json.RawMessage(`{"meaning":"in short supply","distractors":["common","infinite"]}`),
		"writing_prompt":  json.RawMessage(`{"task":"task_1","prompt":"Describe the chart and compare its main features for a university lecturer.","source_image_urls":["/api/v1/private-media/00000000-0000-0000-0000-000000000000"]}`),
		"speaking_plan":   speakingPlanForTest(),
	}
	for kind, data := range valid {
		if !validContentData(kind, data) {
			t.Errorf("expected %s data to be valid", kind)
		}
	}
	if validContentData("daily_challenge", json.RawMessage(`{"metric":"users","target":1,"action":"practice"}`)) {
		t.Fatal("unknown challenge metric was accepted")
	}
	if validContentData("speaking_plan", json.RawMessage(`{"turns":[]}`)) {
		t.Fatal("incomplete speaking plan was accepted")
	}
}

func TestWritingPromptRejectsUnknownTaskAndMedia(t *testing.T) {
	if validContentData("writing_prompt", json.RawMessage(`{"task":"task_3","prompt":"This prompt is long enough to pass the text length check."}`)) {
		t.Fatal("unknown writing task accepted")
	}
	if validContentData("writing_prompt", json.RawMessage(`{"task":"task_1","prompt":"This prompt is long enough to pass the text length check.","source_image_urls":["javascript:alert(1)"]}`)) {
		t.Fatal("unsafe writing source URL accepted")
	}
}

func speakingPlanForTest() json.RawMessage {
	turns := make([]map[string]any, 11)
	for i := range turns {
		part, seconds, prep := 1, 40, 0
		if i == 6 {
			part, seconds, prep = 2, 120, 60
		}
		if i > 6 {
			part, seconds = 3, 60
		}
		turns[i] = map[string]any{"part": part, "seconds": seconds, "prep_seconds": prep, "prompt": "A sufficiently clear interview question?"}
	}
	raw, _ := json.Marshal(map[string]any{"turns": turns})
	return raw
}
