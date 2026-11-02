package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
)

func TestOptionsFromEnv(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		t.Setenv("REDIS_ADDR", "")
		t.Setenv("REDIS_DB", "")
		t.Setenv("REDIS_PASSWORD", "")
		options, err := optionsFromEnv()
		if err != nil || options.Addr != "localhost:6379" || options.DB != 0 || options.Password != "" {
			t.Fatalf("unexpected defaults: %v", err)
		}
	})
	for _, value := range []string{"-1", "abc"} {
		t.Run("invalid database "+value, func(t *testing.T) {
			t.Setenv("REDIS_DB", value)
			if _, err := optionsFromEnv(); err == nil {
				t.Fatal("invalid REDIS_DB accepted")
			}
		})
	}
}

func TestRedisCache(t *testing.T) {
	server := miniredis.RunT(t)
	server.RequireAuth("test-password")
	t.Setenv("REDIS_ADDR", server.Addr())
	t.Setenv("REDIS_PASSWORD", "test-password")
	t.Setenv("REDIS_DB", "3")
	if err := Init(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { Close() })
	ctx := context.Background()
	if _, found, err := Get(ctx, "missing"); err != nil || found {
		t.Fatalf("missing key: found=%t err=%v", found, err)
	}
	if err := Set(ctx, "report", []byte(`[{"total":10}]`), 30*time.Second); err != nil {
		t.Fatal(err)
	}
	if value, found, err := Get(ctx, "report"); err != nil || !found || string(value) != `[{"total":10}]` {
		t.Fatalf("unexpected cache hit: value=%s found=%t err=%v", value, found, err)
	}
	server.Select(3)
	if got := server.TTL("report"); got != 30*time.Second {
		t.Fatalf("TTL = %v, want 30s", got)
	}
	server.FastForward(31 * time.Second)
	if _, found, err := Get(ctx, "report"); err != nil || found {
		t.Fatalf("expired key: found=%t err=%v", found, err)
	}
}

func TestRedisAuthenticationFailure(t *testing.T) {
	server := miniredis.RunT(t)
	server.RequireAuth("secret")
	t.Setenv("REDIS_ADDR", server.Addr())
	t.Setenv("REDIS_PASSWORD", "wrong")
	t.Setenv("REDIS_DB", "0")
	if err := Init(context.Background()); err == nil {
		Close()
		t.Fatal("wrong Redis password accepted")
	}
}
