package domain

import (
	"encoding/json"
	"math"
	"strings"
)

// VerifiedIssues omits optional annotations that cannot be anchored to an
// answer. Mandatory scores, responses and structure still pass ParseReport.
// Never fabricate a quote or clamp a timestamp to make evidence appear valid.
func VerifiedIssues(raw []byte, s Session) ([]byte, int) {
	if len(raw) > 90000 {
		return raw, 0
	}
	var r Report
	if json.Unmarshal(raw, &r) != nil {
		return raw, 0
	}
	transcripts := map[int32]string{}
	for _, response := range r.Responses {
		transcripts[response.Position] = response.Transcript
	}
	valid := make([]Issue, 0, len(r.Issues))
	for _, issue := range r.Issues {
		if issue.Position < 0 || int(issue.Position) >= len(s.Turns) {
			continue
		}
		turn := s.Turns[issue.Position]
		if turn.Skipped || math.IsNaN(issue.Second) || math.IsInf(issue.Second, 0) || issue.Second < 0 || issue.Second > float64(turn.DurationMS)/1000 {
			continue
		}
		if strings.TrimSpace(issue.Quote) == "" || !strings.Contains(transcripts[issue.Position], issue.Quote) {
			continue
		}
		valid = append(valid, issue)
	}
	omitted := len(r.Issues) - len(valid)
	if omitted == 0 {
		return raw, 0
	}
	// Preserve the original JSON structure so this cannot repair invalid required fields.
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil || fields == nil {
		return raw, 0
	}
	fields["issues"], _ = json.Marshal(valid)
	cleaned, e := json.Marshal(fields)
	if e != nil {
		return raw, 0
	}
	return cleaned, omitted
}

type EvaluationJob struct {
	SessionID string
	Part      int32
	ClaimID   string
	Attempts  int32
}

type EvaluationProgress struct {
	Completed int `json:"completed"`
	Total     int `json:"total"`
}

// Section uses local positions for validation; the final report restores the
// original positions so playback and issue links still address the right answer.
func Section(s Session, part int32) Session {
	section := s
	section.Turns = nil
	for _, turn := range s.Turns {
		if turn.Part == part {
			turn.Position = int32(len(section.Turns))
			section.Turns = append(section.Turns, turn)
		}
	}
	return section
}

func RestorePositions(s Session, part int32, report Report) Report {
	positions := []int32{}
	for _, turn := range s.Turns {
		if turn.Part == part {
			positions = append(positions, turn.Position)
		}
	}
	for i := range report.Responses {
		report.Responses[i].Position = positions[report.Responses[i].Position]
	}
	for i := range report.Issues {
		report.Issues[i].Position = positions[report.Issues[i].Position]
	}
	return report
}

type SectionEvidence struct {
	Part   int32           `json:"part"`
	Report json.RawMessage `json:"report"`
}
