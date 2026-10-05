package config

import (
	"fmt"
	"os"
)

type Config struct {
	Environment string
	HTTPAddr    string
	DatabaseURL string
	WebOrigin   string
}

func Load() (Config, error) {
	cfg := Config{
		Environment: envOr("APP_ENV", "development"),
		HTTPAddr:    envOr("HTTP_ADDR", ":8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		WebOrigin:   envOr("WEB_ORIGIN", "http://localhost:5173"),
	}
	if cfg.HTTPAddr == "" {
		return Config{}, fmt.Errorf("HTTP_ADDR must not be empty")
	}
	return cfg, nil
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
