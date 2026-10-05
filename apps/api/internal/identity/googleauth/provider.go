package googleauth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/own/mock-ai/apps/api/internal/identity/domain"
)

var (
	ErrDisabled    = errors.New("Google authentication is not configured")
	ErrInvalidFlow = errors.New("invalid or expired OAuth flow")
	ErrInvalidUser = errors.New("Google did not return a verified user")
)

const userInfoURL = "https://openidconnect.googleapis.com/v1/userinfo"

type Provider struct {
	config     oauth2.Config
	signingKey []byte
	enabled    bool
	now        func() time.Time
}

func New(clientID, clientSecret, redirectURL, sessionSecret string) *Provider {
	return &Provider{
		config: oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Endpoint:     google.Endpoint,
			Scopes:       []string{"openid", "email", "profile"},
		},
		signingKey: []byte(sessionSecret),
		enabled:    clientID != "" && clientSecret != "" && redirectURL != "",
		now:        time.Now,
	}
}

func (p *Provider) Start() (authorizationURL, flowCookie string, err error) {
	if !p.enabled {
		return "", "", ErrDisabled
	}
	state, err := randomString(32)
	if err != nil {
		return "", "", err
	}
	verifier := oauth2.GenerateVerifier()
	expires := p.now().Add(10 * time.Minute).Unix()
	payload := strings.Join([]string{state, verifier, strconv.FormatInt(expires, 10)}, ".")
	flowCookie = payload + "." + p.sign(payload)
	authorizationURL = p.config.AuthCodeURL(
		state,
		oauth2.AccessTypeOnline,
		oauth2.S256ChallengeOption(verifier),
	)
	return authorizationURL, flowCookie, nil
}

func (p *Provider) Complete(ctx context.Context, code, state, flowCookie string) (domain.ExternalProfile, error) {
	if !p.enabled {
		return domain.ExternalProfile{}, ErrDisabled
	}
	verifier, err := p.validateFlow(state, flowCookie)
	if err != nil || code == "" {
		return domain.ExternalProfile{}, ErrInvalidFlow
	}
	token, err := p.config.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return domain.ExternalProfile{}, fmt.Errorf("exchange Google authorization code: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, userInfoURL, nil)
	if err != nil {
		return domain.ExternalProfile{}, err
	}
	request.Header.Set("Authorization", "Bearer "+token.AccessToken)
	response, err := p.config.Client(ctx, token).Do(request)
	if err != nil {
		return domain.ExternalProfile{}, fmt.Errorf("fetch Google user: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		return domain.ExternalProfile{}, fmt.Errorf("fetch Google user: unexpected status %d", response.StatusCode)
	}
	var user struct {
		Subject       string `json:"sub"`
		Name          string `json:"name"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Picture       string `json:"picture"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&user); err != nil {
		return domain.ExternalProfile{}, fmt.Errorf("decode Google user: %w", err)
	}
	if user.Subject == "" || user.Name == "" || user.Email == "" || !user.EmailVerified {
		return domain.ExternalProfile{}, ErrInvalidUser
	}
	rawProfile, _ := json.Marshal(user)
	return domain.ExternalProfile{
		Provider:      "google",
		Subject:       user.Subject,
		DisplayName:   user.Name,
		AvatarURL:     user.Picture,
		Email:         strings.ToLower(user.Email),
		EmailVerified: true,
		RawProfile:    rawProfile,
	}, nil
}

func (p *Provider) validateFlow(state, cookie string) (string, error) {
	parts := strings.Split(cookie, ".")
	if len(parts) != 4 {
		return "", ErrInvalidFlow
	}
	payload := strings.Join(parts[:3], ".")
	expected := p.sign(payload)
	if subtle.ConstantTimeCompare([]byte(parts[3]), []byte(expected)) != 1 || subtle.ConstantTimeCompare([]byte(parts[0]), []byte(state)) != 1 {
		return "", ErrInvalidFlow
	}
	expires, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil || p.now().After(time.Unix(expires, 0)) {
		return "", ErrInvalidFlow
	}
	return parts[1], nil
}

func (p *Provider) sign(payload string) string {
	mac := hmac.New(sha256.New, p.signingKey)
	_, _ = mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func randomString(size int) (string, error) {
	buffer := make([]byte, size)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}
