package domain

import "encoding/json"

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
