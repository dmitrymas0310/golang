package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func Open(ctx context.Context) (*sql.DB, error) {
	connection, err := sql.Open("pgx", dsnFromEnv())
	if err != nil {
		return nil, connectionError("cannot initialize PostgreSQL", err)
	}
	connection.SetMaxOpenConns(10)
	connection.SetMaxIdleConns(5)
	connection.SetConnMaxLifetime(30 * time.Minute)

	if err := connection.PingContext(ctx); err != nil {
		connection.Close()
		return nil, connectionError("cannot connect to PostgreSQL", err)
	}

	return connection, nil
}

func dsnFromEnv() string {
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		return dsn
	}

	password, exists := os.LookupEnv("DB_PASS")
	if !exists {
		password = "postgres"
	}
	dsn := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(envOrDefault("DB_USER", "postgres"), password),
		Host:     net.JoinHostPort(envOrDefault("DB_HOST", "localhost"), envOrDefault("DB_PORT", "5432")),
		Path:     "/" + envOrDefault("DB_NAME", "cashapp"),
		RawQuery: "sslmode=disable",
	}
	return dsn.String()
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func connectionError(message string, err error) error {
	var parseError *pgconn.ParseConfigError
	if errors.As(err, &parseError) {
		// Ошибочная строка подключения может содержать пароль.
		return fmt.Errorf("%s: invalid DSN; check DATABASE_URL or DB_* settings", message)
	}
	return fmt.Errorf("%s: %w", message, err)
}
