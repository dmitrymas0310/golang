package ledger

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"time"

	"ledger/internal/db"
)

// DB инициализируется один раз при старте каждого сервиса через InitDB.
var DB *sql.DB

var ErrDatabaseNotInitialized = errors.New("ledger database is not initialized")

const databaseTimeout = 5 * time.Second

func InitDB(ctx context.Context) error {
	connection, err := db.Open(ctx)
	if err != nil {
		return err
	}
	DB = connection
	// Кеш инициализируется прозрачно и для существующего Gateway.
	if err := InitCache(ctx); err != nil {
		log.Printf("Redis cache unavailable; reports will use PostgreSQL: %v", err)
	}
	return nil
}

func database() (*sql.DB, error) {
	if DB == nil {
		return nil, ErrDatabaseNotInitialized
	}
	return DB, nil
}
