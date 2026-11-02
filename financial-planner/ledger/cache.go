package ledger

import (
	"context"

	"ledger/internal/cache"
)

func InitCache(ctx context.Context) error {
	return cache.Init(ctx)
}

func CloseCache() error {
	return cache.Close()
}
