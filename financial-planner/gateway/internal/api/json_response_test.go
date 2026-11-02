package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestJSONRoutingResponses(t *testing.T) {
	mux := http.NewServeMux()
	RegisterRoutes(mux)
	handler := JSONMiddleware(mux)

	tests := []struct {
		method string
		path   string
		status int
	}{
		{http.MethodGet, "/ping", 200},
		{http.MethodPost, "/ping", 405},
		{http.MethodGet, "/missing", 404},
		{http.MethodGet, "/api/missing", 404},
		{http.MethodDelete, "/api/transactions", 405},
		{http.MethodPut, "/api/budgets", 501},
		{http.MethodGet, "/api", http.StatusTemporaryRedirect},
		{http.MethodGet, "/api//budgets", http.StatusTemporaryRedirect},
	}
	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(tt.method, tt.path, nil))
			if response.Code != tt.status {
				t.Fatalf("status = %d, want %d", response.Code, tt.status)
			}
			if got := response.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
				t.Fatalf("Content-Type = %q", got)
			}
			var payload map[string]any
			if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
				t.Fatalf("invalid JSON: %s", response.Body.String())
			}
			if tt.status >= 400 {
				message, ok := payload["error"].(string)
				if !ok || message == "" || len(payload) != 1 {
					t.Fatalf("invalid error response: %v", payload)
				}
			}
			if tt.method == http.MethodDelete && response.Header().Get("Allow") == "" {
				t.Fatal("405 response lost the Allow header")
			}
			if tt.status >= 300 && tt.status < 400 && response.Header().Get("Location") == "" {
				t.Fatal("redirect lost the Location header")
			}
		})
	}
}

func TestJSONMiddlewareDiscardsPartialResponseOnPanic(t *testing.T) {
	handler := JSONMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("partial response"))
		panic("private error details")
	}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	assertInternalError(t, response)
}

func TestWriteJSONEncodingFailure(t *testing.T) {
	response := httptest.NewRecorder()
	writeJSON(response, http.StatusOK, make(chan int))
	assertInternalError(t, response)
}

func assertInternalError(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", response.Code)
	}
	if got := response.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}
	if got := strings.TrimSpace(response.Body.String()); got != `{"error":"internal server error"}` {
		t.Fatalf("unexpected body: %s", got)
	}
}
