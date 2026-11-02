package api_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"

	"gateway/internal/api"
	"ledger"
)

func testReportHandlers(t *testing.T) {
	handler := newTestHandler(t)
	server := miniredis.RunT(t)
	t.Setenv("REDIS_ADDR", server.Addr())
	t.Setenv("REDIS_DB", "0")
	t.Setenv("REDIS_PASSWORD", "")
	if err := ledger.InitCache(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ledger.CloseCache() })
	assertHTTPJSON(t, handler, http.MethodPost, "/api/transactions",
		`{"amount":450,"category":"еда","date":"2025-09-10"}`, http.StatusCreated,
		`{"id":1,"amount":450,"category":"еда","description":"","date":"2025-09-10"}`)
	const path = "/api/reports/summary?from=2025-09-01&to=2025-09-30"
	const key = "report:summary:2025-09-01:2025-09-30"
	const initial = `[{"category":"еда","total":450}]`
	assertHTTPJSON(t, handler, http.MethodGet, path, "", http.StatusOK, initial)
	if !server.Exists(key) || server.TTL(key) != ledger.ReportCacheTTL {
		t.Fatal("report was not cached with the expected TTL")
	}
	assertHTTPJSON(t, handler, http.MethodGet, path, "", http.StatusOK, initial)
	assertHTTPJSON(t, handler, http.MethodPost, "/api/transactions",
		`{"amount":50,"category":"еда","date":"2025-09-11"}`, http.StatusCreated,
		`{"id":2,"amount":50,"category":"еда","description":"","date":"2025-09-11"}`)
	assertHTTPJSON(t, handler, http.MethodGet, path, "", http.StatusOK, initial)
	server.FastForward(31 * time.Second)
	assertHTTPJSON(t, handler, http.MethodGet, path, "", http.StatusOK, `[{"category":"еда","total":500}]`)
}

func TestReportHandlerInvalidRange(t *testing.T) {
	// Диапазон проверяется до обращения к общим БД и Redis.
	t.Parallel()

	for _, tt := range []struct{ name, path, want string }{
		{"missing from", "/api/reports/summary?to=2025-09-30",
			`{"error":"invalid report date range: from must have format YYYY-MM-DD"}`},
		{"invalid to", "/api/reports/summary?from=2025-09-01&to=bad",
			`{"error":"invalid report date range: to must have format YYYY-MM-DD"}`},
		{"reversed", "/api/reports/summary?from=2025-10-01&to=2025-09-30",
			`{"error":"invalid report date range: from must not be after to"}`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mux := http.NewServeMux()
			api.RegisterRoutes(mux)
			assertHTTPJSON(t, api.JSONMiddleware(mux), http.MethodGet, tt.path, "", http.StatusBadRequest, tt.want)
		})
	}
}
