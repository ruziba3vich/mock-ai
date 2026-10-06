package keyvault

import (
	"strings"
	"testing"
)

func TestPurposeAndRecordBinding(t *testing.T) {
	master := strings.Repeat("ab", 32)
	audio, e := SealBound(master, "wave bytes", "mockai:speaking:recording:v1:session:0")
	if e != nil {
		t.Fatal(e)
	}
	plain, e := OpenBound(master, audio, "mockai:speaking:recording:v1:session:0")
	if e != nil || plain != "wave bytes" {
		t.Fatal(e)
	}
	for _, binding := range []string{"mockai:gemini:v1", "mockai:speaking:recording:v1:session:1", "mockai:speaking:recording:v1:other:0"} {
		if _, e = OpenBound(master, audio, binding); e == nil {
			t.Fatal("accepted transplanted ciphertext")
		}
	}
	key, e := Seal(master, "test-provider-key")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = OpenBound(master, key, "mockai:speaking:recording:v1:session:0"); e == nil {
		t.Fatal("provider credential decoded as student audio")
	}
}
