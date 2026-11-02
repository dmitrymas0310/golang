package db

import (
	"context"
	"errors"
	"net"
	"net/url"
	"os"
	"strings"
	"testing"
)

func clearEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{"DATABASE_URL", "DB_HOST", "DB_PORT", "DB_USER", "DB_PASS", "DB_NAME"} {
		t.Setenv(key, "")
		if err := os.Unsetenv(key); err != nil {
			t.Fatal(err)
		}
	}
}

func TestDSNFromEnv(t *testing.T) {
	t.Run("DATABASE_URL takes priority", func(t *testing.T) {
		clearEnv(t)
		const want = "postgres://custom:secret@db.example:5433/custom?sslmode=require"
		t.Setenv("DATABASE_URL", want)
		t.Setenv("DB_HOST", "ignored")
		t.Setenv("DB_PASS", "ignored")
		if got := dsnFromEnv(); got != want {
			t.Fatal("DATABASE_URL was not used unchanged")
		}
	})

	tests := []struct {
		name string
		env  map[string]string
		host string
		port string
		user string
		pass string
		db   string
	}{
		{"defaults", nil, "localhost", "5432", "postgres", "postgres", "cashapp"},
		{"DB variables", map[string]string{
			"DB_HOST": "db.example", "DB_PORT": "5433", "DB_USER": "planner", "DB_PASS": "secret", "DB_NAME": "custom",
		}, "db.example", "5433", "planner", "secret", "custom"},
		{"escaped credentials and IPv6", map[string]string{
			"DB_HOST": "::1", "DB_PORT": "5433", "DB_USER": "user@dev", "DB_PASS": "p@ss:/?# %", "DB_NAME": "cash app",
		}, "::1", "5433", "user@dev", "p@ss:/?# %", "cash app"},
		{"explicit empty password", map[string]string{"DB_PASS": ""}, "localhost", "5432", "postgres", "", "cashapp"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnv(t)
			for name, value := range tt.env {
				t.Setenv(name, value)
			}
			dsn, err := url.Parse(dsnFromEnv())
			if err != nil {
				t.Fatal(err)
			}
			password, _ := dsn.User.Password()
			if dsn.Scheme != "postgres" || dsn.Host != net.JoinHostPort(tt.host, tt.port) ||
				dsn.User.Username() != tt.user || password != tt.pass || dsn.Path != "/"+tt.db ||
				dsn.Query().Get("sslmode") != "disable" {
				t.Fatal("DSN does not match configured database settings")
			}
		})
	}
}

func TestOpen(t *testing.T) {
	t.Run("invalid DSN", func(t *testing.T) {
		clearEnv(t)
		t.Setenv("DATABASE_URL", "postgres://user:private-password@localhost:invalid/cashapp")
		connection, err := Open(context.Background())
		if err == nil || !strings.Contains(err.Error(), "invalid DSN") {
			t.Fatalf("error = %v, want invalid DSN", err)
		}
		if strings.Contains(err.Error(), "private-password") {
			t.Fatal("error exposes database password")
		}
		if connection != nil {
			t.Fatal("failed initialization published a database handle")
		}
	})
	t.Run("ping is canceled", func(t *testing.T) {
		clearEnv(t)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		connection, err := Open(ctx)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error = %v, want context.Canceled", err)
		}
		if connection != nil {
			t.Fatal("failed ping published a database handle")
		}
	})
}
