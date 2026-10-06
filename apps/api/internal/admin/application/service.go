package application

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"github.com/own/mock-ai/apps/api/internal/identity/password"
	"net/mail"
	"strings"
	"time"
)

var ErrUnauthorized = errors.New("invalid admin credentials or temporarily unavailable; try again later")
var ErrInvalid = errors.New("valid email, name and a password of 12–1024 bytes are required")

type Admin struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
}
type Credential struct {
	Admin
	Hash string
}
type Repository interface {
	Create(context.Context, string, string, string) (string, error)
	ReserveLogin(context.Context, string) (Credential, error)
	ResetLogin(context.Context, string) error
	SaveSession(context.Context, string, string, time.Time) error
	Session(context.Context, string) (Admin, error)
	DeleteSession(context.Context, string) error
}
type Service struct {
	repository Repository
	dummyHash  string
}

func New(r Repository) *Service {
	// A fixed dummy hash makes missing/locked accounts follow password verification too.
	hash, e := password.Hash("non-account-dummy-password")
	if e != nil {
		panic(e)
	}
	return &Service{repository: r, dummyHash: hash}
}
func (s *Service) Create(ctx context.Context, email, name, plain string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	name = strings.TrimSpace(name)
	parsed, e := mail.ParseAddress(email)
	if e != nil || parsed.Address != email || len(email) > 254 || name == "" || len(name) > 100 || len(plain) < 12 || len(plain) > 1024 {
		return "", ErrInvalid
	}
	hash, e := password.Hash(plain)
	if e != nil {
		return "", e
	}
	return s.repository.Create(ctx, email, name, hash)
}
func tokenHash(token string) string {
	v := sha256.Sum256([]byte(token))
	return hex.EncodeToString(v[:])
}
func (s *Service) Login(ctx context.Context, email, plain string) (Admin, string, error) {
	if len(email) > 254 || len(plain) > 1024 {
		return Admin{}, "", ErrUnauthorized
	}
	c, e := s.repository.ReserveLogin(ctx, strings.ToLower(strings.TrimSpace(email)))
	if e != nil {
		_, _ = password.Verify(plain, s.dummyHash)
		return Admin{}, "", ErrUnauthorized
	}
	ok, e := password.Verify(plain, c.Hash)
	if e != nil || !ok {
		return Admin{}, "", ErrUnauthorized
	}
	raw := make([]byte, 32)
	if _, e = rand.Read(raw); e != nil {
		return Admin{}, "", e
	}
	token := hex.EncodeToString(raw)
	if e = s.repository.SaveSession(ctx, c.ID, tokenHash(token), time.Now().Add(8*time.Hour)); e != nil {
		return Admin{}, "", e
	}
	if e = s.repository.ResetLogin(ctx, c.ID); e != nil {
		return Admin{}, "", e
	}
	return c.Admin, token, nil
}
func (s *Service) Session(ctx context.Context, token string) (Admin, error) {
	if len(token) != 64 {
		return Admin{}, ErrUnauthorized
	}
	a, e := s.repository.Session(ctx, tokenHash(token))
	if e != nil {
		return Admin{}, ErrUnauthorized
	}
	return a, nil
}
func (s *Service) Logout(ctx context.Context, token string) error {
	return s.repository.DeleteSession(ctx, tokenHash(token))
}
