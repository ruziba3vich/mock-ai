package identity

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/own/mock-ai/apps/api/internal/identity/domain"
	"github.com/own/mock-ai/apps/api/internal/identity/password"
	identitypostgres "github.com/own/mock-ai/apps/api/internal/identity/postgres"
	"github.com/own/mock-ai/apps/api/internal/identity/telegram"
)

var (
	ErrInvalidInput       = errors.New("invalid input")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrEmailTaken         = errors.New("email is already registered")
	ErrInvalidSession     = errors.New("invalid or expired session")
	ErrTelegramDisabled   = errors.New("Telegram authentication is not configured")
)

type Repository interface {
	RegisterEmail(ctx context.Context, name, email, passwordHash string) (domain.User, error)
	PasswordLogin(ctx context.Context, email string) (domain.User, string, error)
	FindOrCreateExternal(ctx context.Context, profile domain.ExternalProfile) (domain.User, bool, error)
	CreateSession(ctx context.Context, userID string, tokenHash []byte, expiresAt time.Time, userAgent, ipAddress string) error
	UserBySession(ctx context.Context, tokenHash []byte) (domain.User, error)
	RevokeSession(ctx context.Context, tokenHash []byte) error
}

type Session struct {
	Token     string
	ExpiresAt time.Time
}

type Service struct {
	repository        Repository
	telegramValidator *telegram.Validator
	telegramEnabled   bool
	sessionTTL        time.Duration
	dummyPasswordHash string
}

func NewService(repository Repository, telegramBotToken string, sessionTTL time.Duration) (*Service, error) {
	dummyHash, err := password.Hash("not-a-real-user-password")
	if err != nil {
		return nil, err
	}
	return &Service{
		repository:        repository,
		telegramValidator: telegram.NewValidator(telegramBotToken, 10*time.Minute),
		telegramEnabled:   telegramBotToken != "",
		sessionTTL:        sessionTTL,
		dummyPasswordHash: dummyHash,
	}, nil
}

func (s *Service) Register(ctx context.Context, name, email, plainPassword, userAgent, ipAddress string) (domain.User, Session, error) {
	name = strings.TrimSpace(name)
	email, err := normalizeEmail(email)
	if err != nil || utf8.RuneCountInString(name) < 2 || utf8.RuneCountInString(name) > 100 {
		return domain.User{}, Session{}, fmt.Errorf("%w: provide a valid name and email", ErrInvalidInput)
	}
	if len(plainPassword) < 10 || len(plainPassword) > 128 {
		return domain.User{}, Session{}, fmt.Errorf("%w: password must be between 10 and 128 characters", ErrInvalidInput)
	}
	passwordHash, err := password.Hash(plainPassword)
	if err != nil {
		return domain.User{}, Session{}, fmt.Errorf("hash password: %w", err)
	}
	user, err := s.repository.RegisterEmail(ctx, name, email, passwordHash)
	if errors.Is(err, identitypostgres.ErrConflict) {
		return domain.User{}, Session{}, ErrEmailTaken
	}
	if err != nil {
		return domain.User{}, Session{}, err
	}
	session, err := s.newSession(ctx, user.ID, userAgent, ipAddress)
	return user, session, err
}

func (s *Service) Login(ctx context.Context, email, plainPassword, userAgent, ipAddress string) (domain.User, Session, error) {
	email, err := normalizeEmail(email)
	if err != nil || plainPassword == "" {
		return domain.User{}, Session{}, ErrInvalidCredentials
	}
	user, encodedHash, err := s.repository.PasswordLogin(ctx, email)
	if errors.Is(err, identitypostgres.ErrNotFound) {
		_, _ = password.Verify(plainPassword, s.dummyPasswordHash)
		return domain.User{}, Session{}, ErrInvalidCredentials
	}
	if err != nil {
		return domain.User{}, Session{}, err
	}
	valid, err := password.Verify(plainPassword, encodedHash)
	if err != nil {
		return domain.User{}, Session{}, fmt.Errorf("verify password: %w", err)
	}
	if !valid {
		return domain.User{}, Session{}, ErrInvalidCredentials
	}
	session, err := s.newSession(ctx, user.ID, userAgent, ipAddress)
	return user, session, err
}

func (s *Service) LoginTelegram(ctx context.Context, initData, userAgent, ipAddress string) (domain.User, Session, error) {
	if !s.telegramEnabled {
		return domain.User{}, Session{}, ErrTelegramDisabled
	}
	result, err := s.telegramValidator.Validate(initData)
	if err != nil {
		return domain.User{}, Session{}, err
	}
	user, _, err := s.repository.FindOrCreateExternal(ctx, domain.ExternalProfile{
		Provider:    "telegram",
		Subject:     fmt.Sprintf("%d", result.User.ID),
		DisplayName: result.User.DisplayName(),
		AvatarURL:   result.User.PhotoURL,
		RawProfile:  result.RawProfile,
	})
	if err != nil {
		return domain.User{}, Session{}, err
	}
	session, err := s.newSession(ctx, user.ID, userAgent, ipAddress)
	return user, session, err
}

func (s *Service) LoginExternal(ctx context.Context, profile domain.ExternalProfile, userAgent, ipAddress string) (domain.User, Session, error) {
	if profile.Provider == "" || profile.Subject == "" || profile.DisplayName == "" {
		return domain.User{}, Session{}, ErrInvalidInput
	}
	user, _, err := s.repository.FindOrCreateExternal(ctx, profile)
	if err != nil {
		return domain.User{}, Session{}, err
	}
	session, err := s.newSession(ctx, user.ID, userAgent, ipAddress)
	return user, session, err
}

// EnsureTelegramUser creates or refreshes a Telegram-backed student without
// creating a browser session. It is used by the bot when a student sends
// /start; the Mini App still establishes its own session from signed initData.
func (s *Service) EnsureTelegramUser(ctx context.Context, user telegram.User, rawProfile []byte) (domain.User, bool, error) {
	if !s.telegramEnabled {
		return domain.User{}, false, ErrTelegramDisabled
	}
	if user.ID == 0 || strings.TrimSpace(user.DisplayName()) == "" {
		return domain.User{}, false, ErrInvalidInput
	}
	return s.repository.FindOrCreateExternal(ctx, domain.ExternalProfile{
		Provider:    "telegram",
		Subject:     fmt.Sprintf("%d", user.ID),
		DisplayName: user.DisplayName(),
		AvatarURL:   user.PhotoURL,
		RawProfile:  rawProfile,
	})
}

func (s *Service) Authenticate(ctx context.Context, token string) (domain.User, error) {
	hash, err := tokenHash(token)
	if err != nil {
		return domain.User{}, ErrInvalidSession
	}
	user, err := s.repository.UserBySession(ctx, hash)
	if errors.Is(err, identitypostgres.ErrNotFound) {
		return domain.User{}, ErrInvalidSession
	}
	return user, err
}

func (s *Service) Logout(ctx context.Context, token string) error {
	hash, err := tokenHash(token)
	if err != nil {
		return nil
	}
	return s.repository.RevokeSession(ctx, hash)
}

func (s *Service) newSession(ctx context.Context, userID, userAgent, ipAddress string) (Session, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return Session{}, fmt.Errorf("generate session token: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	expiresAt := time.Now().Add(s.sessionTTL)
	hash := sha256.Sum256([]byte(token))
	if err := s.repository.CreateSession(ctx, userID, hash[:], expiresAt, userAgent, ipAddress); err != nil {
		return Session{}, err
	}
	return Session{Token: token, ExpiresAt: expiresAt}, nil
}

func tokenHash(token string) ([]byte, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(decoded) != 32 {
		return nil, ErrInvalidSession
	}
	hash := sha256.Sum256([]byte(token))
	return hash[:], nil
}

func normalizeEmail(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if len(value) > 254 {
		return "", ErrInvalidInput
	}
	parsed, err := mail.ParseAddress(value)
	if err != nil || parsed.Address != value || !strings.Contains(value, "@") {
		return "", ErrInvalidInput
	}
	return value, nil
}
