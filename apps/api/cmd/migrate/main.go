package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/lib/pq"

	migrationfiles "github.com/own/mock-ai/apps/api/db/migrations"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		logger.Error("DATABASE_URL must not be empty")
		os.Exit(1)
	}

	source, err := iofs.New(migrationfiles.Files, ".")
	if err != nil {
		fail(logger, "open embedded migrations", err)
	}
	runner, err := migrate.NewWithSourceInstance("iofs", source, databaseURL)
	if err != nil {
		fail(logger, "initialize migrations", err)
	}
	defer func() {
		_, _ = runner.Close()
	}()
	if err := runner.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		fail(logger, "apply migrations", err)
	}
	version, dirty, err := runner.Version()
	if err != nil {
		fail(logger, "read migration version", err)
	}
	if dirty {
		fail(logger, "migration state is dirty", fmt.Errorf("version %d", version))
	}
	logger.Info("migrations ready", "version", version)
}

func fail(logger *slog.Logger, message string, err error) {
	logger.Error(message, "error", err)
	os.Exit(1)
}
