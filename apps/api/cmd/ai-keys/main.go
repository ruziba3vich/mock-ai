// Import Gemini API keys from a JSON array on stdin. Never prints credentials.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/own/mock-ai/apps/api/internal/platform/keyvault"
	db "github.com/own/mock-ai/apps/api/internal/platform/postgres/sqlc"
	"io"
	"os"
	"strings"
	"time"
)

func run() error {
	var keys []string
	if e := json.NewDecoder(io.LimitReader(os.Stdin, 64000)).Decode(&keys); e != nil || len(keys) == 0 || len(keys) > 100 {
		return fmt.Errorf("expected 1–100 keys as a JSON array on stdin")
	}
	master := os.Getenv("AI_KEY_ENCRYPTION_KEY")
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	pool, e := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if e != nil {
		return fmt.Errorf("database unavailable")
	}
	defer pool.Close()
	tx, e := pool.Begin(ctx)
	if e != nil {
		return fmt.Errorf("database unavailable")
	}
	defer tx.Rollback(ctx)
	q := db.New(tx)
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if len(key) < 20 || len(key) > 256 {
			return fmt.Errorf("invalid key length")
		}
		sealed, e := keyvault.Seal(master, key)
		if e != nil {
			return e
		}
		if e = q.ImportAIKey(ctx, db.ImportAIKeyParams{Fingerprint: keyvault.Fingerprint(key), Ciphertext: sealed}); e != nil {
			return fmt.Errorf("key import failed")
		}
	}
	if e = tx.Commit(ctx); e != nil {
		return fmt.Errorf("key import failed")
	}
	fmt.Printf("Processed %d Gemini keys; duplicates retained without overwriting. Values encrypted, never printed.\n", len(keys))
	return nil
}
func main() {
	if e := run(); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
