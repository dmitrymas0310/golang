// Package testutil содержит настройку изолированной PostgreSQL для тестов.
package testutil

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"

	"ledger"
	"ledger/migrations"
)

var schemaNumber atomic.Uint64

// Database создаёт отдельную схему и удаляет её в Cleanup.
// Тесты остаются последовательными, так как ledger.DB — общий указатель.
func Database(t testing.TB) {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL to run PostgreSQL integration tests")
	}
	config, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatal("invalid TEST_DATABASE_URL")
	}
	admin := stdlib.OpenDB(*config)
	t.Cleanup(func() { admin.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := admin.PingContext(ctx); err != nil {
		t.Fatalf("connect to test PostgreSQL: %v", err)
	}
	schema := fmt.Sprintf("fp_test_%d_%d", os.Getpid(), schemaNumber.Add(1))
	if _, err := admin.ExecContext(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatalf("create test schema: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if _, err := admin.ExecContext(ctx, "DROP SCHEMA "+schema+" CASCADE"); err != nil {
			t.Errorf("drop test schema: %v", err)
		}
	})
	config.RuntimeParams["search_path"] = schema
	connection := stdlib.OpenDB(*config)
	previous := ledger.DB
	ledger.DB = connection
	t.Cleanup(func() {
		ledger.DB = previous
		connection.Close()
	})
	entries, err := migrations.Files.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		content, err := migrations.Files.ReadFile(entry.Name())
		if err != nil {
			t.Fatal(err)
		}
		up, _, _ := strings.Cut(string(content), "-- +goose Down")
		if _, err := connection.ExecContext(ctx, up); err != nil {
			t.Fatalf("apply %s: %v", entry.Name(), err)
		}
	}
	if err := ledger.Reset(); err != nil {
		t.Fatal(err)
	}
}
