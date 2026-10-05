package telegram

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	ErrInvalidData = errors.New("invalid Telegram init data")
	ErrExpiredData = errors.New("expired Telegram init data")
)

type User struct {
	ID           int64  `json:"id"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Username     string `json:"username"`
	LanguageCode string `json:"language_code"`
	PhotoURL     string `json:"photo_url"`
}

type Result struct {
	User       User
	AuthDate   time.Time
	RawProfile []byte
}

type Validator struct {
	botToken string
	maxAge   time.Duration
	now      func() time.Time
}

func NewValidator(botToken string, maxAge time.Duration) *Validator {
	return &Validator{botToken: botToken, maxAge: maxAge, now: time.Now}
}

func (v *Validator) Validate(raw string) (Result, error) {
	if v.botToken == "" || raw == "" {
		return Result{}, ErrInvalidData
	}
	values, err := url.ParseQuery(raw)
	if err != nil {
		return Result{}, ErrInvalidData
	}
	receivedHash, err := hex.DecodeString(values.Get("hash"))
	if err != nil || len(receivedHash) != sha256.Size {
		return Result{}, ErrInvalidData
	}

	keys := make([]string, 0, len(values)-1)
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
	checkString := strings.Join(parts, "\n")

	secretMAC := hmac.New(sha256.New, []byte("WebAppData"))
	_, _ = secretMAC.Write([]byte(v.botToken))
	dataMAC := hmac.New(sha256.New, secretMAC.Sum(nil))
	_, _ = dataMAC.Write([]byte(checkString))
	if !hmac.Equal(receivedHash, dataMAC.Sum(nil)) {
		return Result{}, ErrInvalidData
	}

	authUnix, err := strconv.ParseInt(values.Get("auth_date"), 10, 64)
	if err != nil || authUnix <= 0 {
		return Result{}, ErrInvalidData
	}
	authDate := time.Unix(authUnix, 0)
	now := v.now()
	if authDate.After(now.Add(time.Minute)) || now.Sub(authDate) > v.maxAge {
		return Result{}, ErrExpiredData
	}

	rawUser := []byte(values.Get("user"))
	var user User
	if err := json.Unmarshal(rawUser, &user); err != nil || user.ID == 0 || strings.TrimSpace(user.FirstName) == "" {
		return Result{}, fmt.Errorf("%w: missing user", ErrInvalidData)
	}
	return Result{User: user, AuthDate: authDate, RawProfile: rawUser}, nil
}

func (u User) DisplayName() string {
	name := strings.TrimSpace(strings.Join([]string{u.FirstName, u.LastName}, " "))
	if name != "" {
		return name
	}
	if u.Username != "" {
		return u.Username
	}
	return "Telegram user"
}
