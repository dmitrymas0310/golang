package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ledger"

	"gateway/internal/api"
)

func TestHandlers(t *testing.T) {
	t.Run("empty_collections", testHandlersEmptyCollections)
	t.Run("budgets", testBudgetHandlers)
	t.Run("transactions", testTransactionHandlers)
	t.Run("transaction_validation", testCreateTransaction)
	t.Run("concurrent_transactions", testConcurrentTransactionsRespectBudget)
}

// Сценарии с общим Ledger выполняются последовательно: Reset очищает всё хранилище.
func newTestHandler(t *testing.T) http.Handler {
	t.Helper()
	ledger.Reset()
	t.Cleanup(ledger.Reset)

	mux := http.NewServeMux()
	api.RegisterRoutes(mux)
	return api.LoggingMiddleware(api.JSONMiddleware(mux))
}

func testHandlersEmptyCollections(t *testing.T) {
	for _, path := range []string{"/api/transactions", "/api/budgets"} {
		t.Run(path, func(t *testing.T) {
			handler := newTestHandler(t)
			assertHTTPJSON(t, handler, http.MethodGet, path, "", http.StatusOK, `[]`)
		})
	}
}

func testBudgetHandlers(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		handler := newTestHandler(t)
		assertHTTPJSON(t, handler, http.MethodPost, "/api/budgets",
			`{"category":"еда","limit":5000}`, http.StatusCreated,
			`{"category":"еда","limit":5000,"period":""}`)
		assertHTTPJSON(t, handler, http.MethodGet, "/api/budgets", "", http.StatusOK,
			`[{"category":"еда","limit":5000,"period":""}]`)
	})

	tests := []struct {
		name string
		body string
		want string
	}{
		{"bad_json", `{"category":`, `{"error":"invalid JSON body"}`},
		{"zero_limit", `{"category":"еда","limit":0}`, `{"error":"budget limit must be greater than zero"}`},
		{"negative_limit", `{"category":"еда","limit":-100}`, `{"error":"budget limit must be greater than zero"}`},
		{"empty_category", `{"category":"","limit":5000}`, `{"error":"budget category must not be empty"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := newTestHandler(t)
			assertHTTPJSON(t, handler, http.MethodPost, "/api/budgets", tt.body, http.StatusBadRequest, tt.want)
			assertHTTPJSON(t, handler, http.MethodGet, "/api/budgets", "", http.StatusOK, `[]`)
		})
	}
}

func testTransactionHandlers(t *testing.T) {
	const validRequest = `{"amount":60,"category":"Food","description":"Lunch","date":"2026-11-02"}`
	const createdResponse = `{"id":0,"amount":60,"category":"Food","description":"Lunch","date":"2026-11-02"}`

	tests := []struct {
		name string
		body string
		seed bool
		code int
		want string
	}{
		{"ok", validRequest, false, http.StatusCreated, createdResponse},
		{"exceeded", `{"amount":50,"category":"Food","date":"2026-11-02"}`, true,
			http.StatusConflict, `{"error":"budget exceeded"}`},
		{"bad_json", `{"amount":`, true, http.StatusBadRequest, `{"error":"invalid JSON body"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := newTestHandler(t)
			assertHTTPJSON(t, handler, http.MethodPost, "/api/budgets",
				`{"category":"Food","limit":100}`, http.StatusCreated,
				`{"category":"Food","limit":100,"period":""}`)
			if tt.seed {
				assertHTTPJSON(t, handler, http.MethodPost, "/api/transactions",
					validRequest, http.StatusCreated, createdResponse)
			}

			assertHTTPJSON(t, handler, http.MethodPost, "/api/transactions", tt.body, tt.code, tt.want)
			// После успеха или отказа в списке должна быть только корректная транзакция.
			assertHTTPJSON(t, handler, http.MethodGet, "/api/transactions", "", http.StatusOK,
				"["+createdResponse+"]")
			assertHTTPJSON(t, handler, http.MethodGet, "/api/budgets", "", http.StatusOK,
				`[{"category":"Food","limit":100,"period":""}]`)
		})
	}
}

func assertHTTPJSON(t *testing.T, handler http.Handler, method, path, body string, status int, want string) {
	t.Helper()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != status {
		t.Fatalf("%s %s: status = %d, want %d: %s", method, path, response.Code, status, response.Body.String())
	}
	if got := response.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}
	var payload any
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	assertJSON(t, payload, want)
}
