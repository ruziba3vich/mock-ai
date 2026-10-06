package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	Environment        string
	HTTPAddr           string
	DatabaseURL        string
	WebOrigin          string
	TelegramBotToken   string
	TelegramMiniAppURL string
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
	SessionSecret      string
	SessionTTL         time.Duration
	SecureCookie       bool
}

func Load() (Config, error) {
	cfg := Config{
		Environment:        envOr("APP_ENV", "development"),
		HTTPAddr:           envOr("HTTP_ADDR", ":8080"),
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		WebOrigin:          envOr("WEB_ORIGIN", "http://localhost:5173"),
		TelegramBotToken:   os.Getenv("TELEGRAM_BOT_TOKEN"),
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		GoogleRedirectURL:  envOr("GOOGLE_REDIRECT_URL", "http://localhost:8080/api/v1/auth/google/callback"),
		SessionSecret:      os.Getenv("SESSION_SECRET"),
		SessionTTL:         30 * 24 * time.Hour,
	}
	cfg.TelegramMiniAppURL = envOr("TELEGRAM_MINI_APP_URL", cfg.WebOrigin)
	cfg.SecureCookie = cfg.Environment != "development"
	if cfg.HTTPAddr == "" {
		return Config{}, fmt.Errorf("HTTP_ADDR must not be empty")
	}
	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL must not be empty")
	}
	if len(cfg.SessionSecret) < 32 || strings.HasPrefix(cfg.SessionSecret, "replace-") {
		return Config{}, fmt.Errorf("SESSION_SECRET must contain at least 32 random characters")
	}
	return cfg, nil
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
