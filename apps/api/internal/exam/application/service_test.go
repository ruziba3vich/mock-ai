package application

import "testing"

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
