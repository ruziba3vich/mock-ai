package telegram

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestValidate(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	values := url.Values{
		"auth_date": {"1800000000"},
		"query_id":  {"query"},
		"user":      {`{"id":123,"first_name":"Ada","last_name":"Lovelace","photo_url":"https://example.com/a.jpg"}`},
	}
	values.Set("hash", sign(values, "123:token"))

	validator := NewValidator("123:token", 10*time.Minute)
	validator.now = func() time.Time { return now }
	result, err := validator.Validate(values.Encode())
	if err != nil {
		t.Fatal(err)
	}
	if result.User.ID != 123 || result.User.DisplayName() != "Ada Lovelace" {
		t.Fatalf("unexpected user: %#v", result.User)
	}
}

func TestRejectsTamperedData(t *testing.T) {
	values := url.Values{
		"auth_date": {"1800000000"},
		"user":      {`{"id":123,"first_name":"Ada"}`},
	}
	values.Set("hash", sign(values, "123:token"))
	values.Set("user", `{"id":456,"first_name":"Mallory"}`)

	validator := NewValidator("123:token", 10*time.Minute)
	validator.now = func() time.Time { return time.Unix(1_800_000_000, 0) }
	if _, err := validator.Validate(values.Encode()); !errorsIs(err, ErrInvalidData) {
		t.Fatalf("expected invalid data, got %v", err)
	}
}

func sign(values url.Values, token string) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		if key != "hash" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+values.Get(key))
	}
	secret := hmac.New(sha256.New, []byte("WebAppData"))
	_, _ = secret.Write([]byte(token))
	signature := hmac.New(sha256.New, secret.Sum(nil))
	_, _ = signature.Write([]byte(strings.Join(parts, "\n")))
	return hex.EncodeToString(signature.Sum(nil))
}

func errorsIs(err, target error) bool {
	return err == target || strings.Contains(err.Error(), target.Error())
}
