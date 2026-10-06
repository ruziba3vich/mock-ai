package domain

import (
	"encoding/binary"
	"encoding/json"
	"math"
	"testing"
)

func audio(seconds int, silent bool) []byte {
	b := make([]byte, 44+seconds*32000)
	copy(b, "RIFF")
	binary.LittleEndian.PutUint32(b[4:], uint32(len(b)-8))
	copy(b[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(b[16:], 16)
	binary.LittleEndian.PutUint16(b[20:], 1)
	binary.LittleEndian.PutUint16(b[22:], 1)
	binary.LittleEndian.PutUint32(b[24:], 16000)
	binary.LittleEndian.PutUint32(b[28:], 32000)
	binary.LittleEndian.PutUint16(b[32:], 2)
	binary.LittleEndian.PutUint16(b[34:], 16)
	copy(b[36:], "data")
	binary.LittleEndian.PutUint32(b[40:], uint32(len(b)-44))
	if !silent {
		for i := 44; i < len(b); i += 2 {
			binary.LittleEndian.PutUint16(b[i:], uint16(int16(math.Sin(float64(i)*.1)*10000)))
		}
	}
	return b
}
func TestAudioValidation(t *testing.T) {
	if n, e := ValidateAudio(audio(5, false), 40); e != nil || n != 5000 {
		t.Fatal(n, e)
	}
	for _, b := range [][]byte{nil, []byte("not audio"), audio(5, true), audio(43, false), audio(0, false)} {
		if _, e := ValidateAudio(b, 40); e == nil {
			t.Fatal("accepted invalid audio")
		}
	}
	v := audio(2, false)
	v[24] = 0
	if _, e := ValidateAudio(v, 40); e == nil {
		t.Fatal("accepted wrong sample rate")
	}
}
func fixture() (Session, Report) {
	_, turns, _ := NewPlan()
	r := Report{Assessable: true, Summary: "Clear answers.", NextSteps: []string{"Develop examples."}}
	for i := range turns {
		turns[i].DurationMS = 6000
		turns[i].Answered = true
		r.Responses = append(r.Responses, Response{Position: int32(i), Transcript: "I study science.", Feedback: "Explain your reason."})
	}
	for _, k := range []string{"fluency_coherence", "lexical_resource", "grammar", "pronunciation"} {
		v := 6.5
		r.Criteria = append(r.Criteria, Criterion{Key: k, Score: &v, Reason: "Evidence.", Improve: "Practise."})
	}
	r.Issues = []Issue{{Category: "grammar", Position: 0, Second: 1, Quote: "I study", Explanation: "Expand the clause.", Suggestion: "Add a reason."}}
	return Session{Turns: turns}, r
}
func TestReportComputedBandAndEvidence(t *testing.T) {
	s, r := fixture()
	b, _ := json.Marshal(r)
	got, e := ParseReport(b, s)
	if e != nil || got.Overall == nil || *got.Overall != 6.5 {
		t.Fatal(got, e)
	}
	for _, modify := range []func(*Report){func(v *Report) { v.Issues[0].Quote = "invented words" }, func(v *Report) { v.Issues[0].Second = 99 }, func(v *Report) { v.Criteria[0].Key = "unknown" }, func(v *Report) { v.Responses[0].Position = 1 }, func(v *Report) { v.Criteria[0].Score = nil }} {
		_, r = fixture()
		modify(&r)
		b, _ = json.Marshal(r)
		if _, e = ParseReport(b, s); e == nil {
			t.Fatal("accepted invalid evidence")
		}
	}
}
func TestIncompleteSpeechHasNoBand(t *testing.T) {
	s, r := fixture()
	s.Turns[6].Skipped = true
	s.Turns[6].DurationMS = 0
	r.Responses[6].Transcript = ""
	b, _ := json.Marshal(r)
	got, e := ParseReport(b, s)
	if e != nil || got.Assessable || got.Overall != nil {
		t.Fatal(got, e)
	}
	for _, v := range got.Criteria {
		if v.Score != nil {
			t.Fatal("invented score for incomplete interview")
		}
	}
}
func TestCatalogAndPlan(t *testing.T) {
	for _, country := range Countries {
		if len(country.Voices) != 2 || country.Voices[0].Gender != "female" || country.Voices[1].Gender != "male" {
			t.Fatal(country)
		}
		for _, v := range country.Voices {
			if v.Code != country.Code || v.Accent == "" || v.Voice == "" {
				t.Fatal(v)
			}
		}
	}
	for range 50 {
		ex, e := RandomExaminer()
		if e != nil || ex.Code == "" {
			t.Fatal(ex, e)
		}
		_, turns, e := NewPlan()
		if e != nil || len(turns) != 11 || turns[6].PrepSeconds != 60 || turns[6].Seconds != 120 || turns[10].Part != 3 {
			t.Fatal(turns, e)
		}
	}
}
