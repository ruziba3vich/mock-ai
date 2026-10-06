package keyvault

import (
	"bytes"
	"strings"
	"testing"
)

func TestEncryptedKeys(t *testing.T) {
	master := strings.Repeat("ab", 32)
	secret := "test-provider-secret-not-real"
	sealed, e := Seal(master, secret)
	if e != nil {
		t.Fatal(e)
	}
	if bytes.Contains(sealed, []byte(secret)) {
		t.Fatal("plaintext stored")
	}
	value, e := Open(master, sealed)
	if e != nil || value != secret {
		t.Fatal("roundtrip failed", e)
	}
	if _, e = Open(strings.Repeat("cd", 32), sealed); e == nil {
		t.Fatal("wrong master accepted")
	}
	sealed[len(sealed)-1] ^= 1
	if _, e = Open(master, sealed); e == nil {
		t.Fatal("tampering accepted")
	}
	if _, e = Seal("", secret); e == nil {
		t.Fatal("empty master accepted")
	}
}
