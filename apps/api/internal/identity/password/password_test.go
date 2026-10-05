package password

import "testing"

func TestHashAndVerify(t *testing.T) {
	encoded, err := Hash("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	if encoded == "correct horse battery staple" {
		t.Fatal("password was not hashed")
	}
	valid, err := Verify("correct horse battery staple", encoded)
	if err != nil || !valid {
		t.Fatalf("expected password to verify, valid=%v err=%v", valid, err)
	}
	valid, err = Verify("wrong password", encoded)
	if err != nil {
		t.Fatal(err)
	}
	if valid {
		t.Fatal("wrong password verified")
	}
}
