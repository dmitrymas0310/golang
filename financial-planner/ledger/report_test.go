package ledger_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"

	"ledger"
	"ledger/testutil"
)

const reportKey = "report:summary:2026-01-01:2026-01-31"

func reportCache(t *testing.T) *miniredis.Miniredis {
	t.Helper()
	server := miniredis.RunT(t)
	t.Setenv("REDIS_ADDR", server.Addr())
	t.Setenv("REDIS_DB", "0")
	t.Setenv("REDIS_PASSWORD", "")
	if err := ledger.InitCache(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ledger.CloseCache() })
	return server
}

func TestReportInput(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct{ name, from, to string }{
		{"invalid from", "2026-01", "2026-01-31"},
		{"invalid to", "2026-01-01", "2026-02-30"},
		{"reversed interval", "2026-02-01", "2026-01-31"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := ledger.GetReportSummary(context.Background(), tt.from, tt.to); err == nil {
				t.Fatal("invalid report interval accepted")
			}
		})
	}
	t.Run("canceled context", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, err := ledger.GetReportSummary(ctx, "2026-01-01", "2026-01-31"); !errors.Is(err, context.Canceled) {
			t.Fatalf("error = %v, want context.Canceled", err)
		}
	})
}

func TestReportCacheHitWithoutDatabase(t *testing.T) {
	server := reportCache(t)
	previous := ledger.DB
	ledger.DB = nil
	t.Cleanup(func() { ledger.DB = previous })
	server.Set(reportKey, `[{"category":"Food","total":12.5}]`)
	got, err := ledger.GetReportSummary(context.Background(), "2026-01-01", "2026-01-31")
	if err != nil || !reflect.DeepEqual(got, []ledger.ReportSummary{{Category: "Food", Total: 12.5}}) {
		t.Fatalf("cached report = %v, err = %v", got, err)
	}
}

func TestReportPostgreSQL(t *testing.T) {
	for _, name := range []string{"miss_hit_expiry", "invalid_json", "redis_unavailable", "sql_error", "empty"} {
		t.Run(name, func(t *testing.T) {
			testutil.Database(t)
			server := reportCache(t)
			ctx := context.Background()
			if name != "empty" {
				for _, tx := range []ledger.Transaction{
					{Amount: 10, Category: "Food", Date: "2026-01-01"},
					{Amount: 20, Category: "Food", Date: "2026-01-31"},
					{Amount: 5, Category: "Transport", Date: "2026-01-15"},
					{Amount: 99, Category: "Food", Date: "2025-12-31"},
				} {
					if err := ledger.AddTransaction(tx); err != nil {
						t.Fatal(err)
					}
				}
			}
			switch name {
			case "invalid_json":
				server.Set(reportKey, "{")
			case "redis_unavailable":
				server.Close()
			case "sql_error":
				if _, err := ledger.DB.Exec(`DROP TABLE expenses`); err != nil {
					t.Fatal(err)
				}
			}
			got, err := ledger.GetReportSummary(ctx, "2026-01-01", "2026-01-31")
			if name == "sql_error" {
				if err == nil || server.Exists(reportKey) {
					t.Fatal("SQL error was swallowed or cached")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			want := []ledger.ReportSummary{{Category: "Food", Total: 30}, {Category: "Transport", Total: 5}}
			if name == "empty" {
				want = []ledger.ReportSummary{}
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("report = %v, want %v", got, want)
			}
			if name == "redis_unavailable" {
				return
			}
			cached, err := server.Get(reportKey)
			if err != nil || !json.Valid([]byte(cached)) || server.TTL(reportKey) != ledger.ReportCacheTTL {
				t.Fatalf("invalid cached JSON or TTL: value=%s err=%v", cached, err)
			}
			if name != "miss_hit_expiry" {
				return
			}
			if err := ledger.AddTransaction(ledger.Transaction{Amount: 7, Category: "Food", Date: "2026-01-10"}); err != nil {
				t.Fatal(err)
			}
			got, err = ledger.GetReportSummary(ctx, "2026-01-01", "2026-01-31")
			if err != nil || !reflect.DeepEqual(got, want) {
				t.Fatalf("cache hit did not reuse previous report: %v, %v", got, err)
			}
			server.FastForward(31 * time.Second)
			got, err = ledger.GetReportSummary(ctx, "2026-01-01", "2026-01-31")
			want[0].Total = 37
			if err != nil || !reflect.DeepEqual(got, want) {
				t.Fatalf("expired cache was not refreshed: %v, %v", got, err)
			}
		})
	}
}
