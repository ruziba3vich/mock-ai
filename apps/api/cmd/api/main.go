package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	examapp "github.com/own/mock-ai/apps/api/internal/exam/application"
	examhttp "github.com/own/mock-ai/apps/api/internal/exam/httpapi"
	exampostgres "github.com/own/mock-ai/apps/api/internal/exam/postgres"
	"github.com/own/mock-ai/apps/api/internal/identity"
	"github.com/own/mock-ai/apps/api/internal/identity/googleauth"
	identityhttp "github.com/own/mock-ai/apps/api/internal/identity/httpapi"
	identitypostgres "github.com/own/mock-ai/apps/api/internal/identity/postgres"
	"github.com/own/mock-ai/apps/api/internal/platform/config"
	"github.com/own/mock-ai/apps/api/internal/platform/httpserver"
	"github.com/own/mock-ai/apps/api/internal/telegrambot"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := config.Load()
	if err != nil {
		logger.Error("invalid configuration", "error", err)
		os.Exit(1)
	}
	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		logger.Error("open database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	connectCtx, cancelConnect := context.WithTimeout(context.Background(), 10*time.Second)
	if err := pool.Ping(connectCtx); err != nil {
		cancelConnect()
		logger.Error("connect to database", "error", err)
		os.Exit(1)
	}
	cancelConnect()

	identityRepository := identitypostgres.New(pool)
	identityService, err := identity.NewService(identityRepository, cfg.TelegramBotToken, cfg.SessionTTL)
	if err != nil {
		logger.Error("initialize identity service", "error", err)
		os.Exit(1)
	}
	googleProvider := googleauth.New(cfg.GoogleClientID, cfg.GoogleClientSecret, cfg.GoogleRedirectURL, cfg.SessionSecret)
	identityHandler := identityhttp.New(identityService, googleProvider, logger, cfg.WebOrigin, cfg.SecureCookie)
	examRepository := exampostgres.New(pool)
	examService := examapp.NewService(examRepository)
	examHandler := examhttp.New(examService, identityHandler, logger)
	bot, err := telegrambot.New(cfg.TelegramBotToken, cfg.TelegramMiniAppURL, identityService, logger)
	if err != nil {
		logger.Error("initialize Telegram bot", "error", err)
		os.Exit(1)
	}

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpserver.New(logger, cfg.WebOrigin, identityHandler, examHandler),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if bot != nil {
		go bot.Run(ctx)
	} else {
		logger.Info("Telegram bot disabled")
	}

	go func() {
		logger.Info("api listening", "address", cfg.HTTPAddr, "environment", cfg.Environment)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("api stopped unexpectedly", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
	}
}
