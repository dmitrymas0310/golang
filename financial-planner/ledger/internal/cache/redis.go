package cache

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

const operationTimeout = 500 * time.Millisecond

var (
	mu     sync.RWMutex
	client *redis.Client
)

func optionsFromEnv() (*redis.Options, error) {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	database := 0
	if value := os.Getenv("REDIS_DB"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 0 {
			return nil, errors.New("REDIS_DB must be a non-negative integer")
		}
		database = parsed
	}
	return &redis.Options{
		Addr:                  addr,
		DB:                    database,
		Password:              os.Getenv("REDIS_PASSWORD"),
		MaxRetries:            -1,
		DialTimeout:           operationTimeout,
		ReadTimeout:           operationTimeout,
		WriteTimeout:          operationTimeout,
		ContextTimeoutEnabled: true,
	}, nil
}

func Init(ctx context.Context) error {
	options, err := optionsFromEnv()
	if err != nil {
		return err
	}
	connection := redis.NewClient(options)
	pingCtx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	if err := connection.Ping(pingCtx).Err(); err != nil {
		connection.Close()
		return fmt.Errorf("connect to Redis: %w", err)
	}
	mu.Lock()
	previous := client
	client = connection
	mu.Unlock()
	if previous != nil {
		previous.Close()
	}
	log.Printf("Redis connected: addr=%s db=%d", options.Addr, options.DB)
	return nil
}

func current() *redis.Client {
	mu.RLock()
	defer mu.RUnlock()
	return client
}

// Get возвращает found=false для отсутствующего ключа или неинициализированного кеша.
func Get(ctx context.Context, key string) ([]byte, bool, error) {
	connection := current()
	if connection == nil {
		return nil, false, nil
	}
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	value, err := connection.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return value, true, nil
}

func Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	connection := current()
	if connection == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	return connection.Set(ctx, key, value, ttl).Err()
}

func Close() error {
	mu.Lock()
	previous := client
	client = nil
	mu.Unlock()
	if previous != nil {
		return previous.Close()
	}
	return nil
}
