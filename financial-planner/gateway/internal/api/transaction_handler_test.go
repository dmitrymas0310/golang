package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"ledger"

	"gateway/internal/api"
)

func testCreateTransaction(t *testing.T) {
	const category = "Food"

	tests := []struct {
		name   string
		body   string
		status int
		want   string
	}{
		{"malformed JSON", `{`, 400, `{"error":"invalid JSON body"}`},
		{"empty body", ``, 400, `{"error":"invalid JSON body"}`},
		{"wrong amount type", `{"amount":"10"}`, 400, `{"error":"invalid JSON body"}`},
		{"invalid amount", `{"amount":0,"category":"Food","date":"2026-10-01"}`, 400, `{"error":"transaction amount must be greater than zero"}`},
		{"empty category", `{"amount":10,"category":" ","date":"2026-10-01"}`, 400, `{"error":"transaction category must not be empty"}`},
		{"empty date", `{"amount":10,"category":"Food"}`, 400, `{"error":"transaction date must not be empty"}`},
		{"multiple objects", `{"amount":10} {}`, 400, `{"error":"body must contain a single JSON object"}`},
		{"trailing garbage", `{"amount":10} garbage`, 400, `{"error":"body must contain a single JSON object"}`},
		{
			"exact budget limit",
			fmt.Sprintf(`{"id":999,"amount":100,"category":%q,"description":"Lunch","date":"2026-10-01"}`, category),
			201,
			fmt.Sprintf(`{"id":1,"amount":100,"category":%q,"description":"Lunch","date":"2026-10-01"}`, category),
		},
		{
			"budget exceeded",
			fmt.Sprintf(`{"amount":1,"category":%q,"date":"2026-10-01"}`, category),
			409,
			`{"error":"budget exceeded"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := newTestHandler(t)
			if err := ledger.SetBudget(ledger.Budget{Category: category, Limit: 100}); err != nil {
				t.Fatal(err)
			}
			if tt.status == http.StatusConflict {
				if err := ledger.AddTransaction(ledger.Transaction{Amount: 100, Category: category, Date: "2026-10-01"}); err != nil {
					t.Fatal(err)
				}
			}
			before, err := ledger.ListTransactions()
			if err != nil {
				t.Fatal(err)
			}
			request := httptest.NewRequest(http.MethodPost, "/api/transactions", strings.NewReader(tt.body))
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)

			if response.Code != tt.status {
				t.Fatalf("status = %d, want %d: %s", response.Code, tt.status, response.Body.String())
			}
			if got := response.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
				t.Fatalf("Content-Type = %q, want application/json; charset=utf-8", got)
			}
			var body any
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			assertJSON(t, body, tt.want)

			stored, err := ledger.ListTransactions()
			if err != nil {
				t.Fatal(err)
			}
			if tt.status == http.StatusCreated {
				if len(stored) != len(before)+1 {
					t.Fatal("created transaction was not saved")
				}
				assertJSON(t, api.NewTransactionResponse(stored[len(stored)-1]), tt.want)
			} else if len(stored) != len(before) {
				t.Fatal("rejected request changed stored transactions")
			}
		})
	}
}

func testConcurrentTransactionsRespectBudget(t *testing.T) {
	handler := newTestHandler(t)
	const category = "Food"
	if err := ledger.SetBudget(ledger.Budget{Category: category, Limit: 100}); err != nil {
		t.Fatal(err)
	}
	body := fmt.Sprintf(`{"amount":10,"category":%q,"date":"2026-10-01"}`, category)
	statuses := make(chan int, 20)
	var requests sync.WaitGroup
	for i := 0; i < 20; i++ {
		requests.Add(1)
		go func() {
			defer requests.Done()
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/transactions", strings.NewReader(body)))
			statuses <- response.Code
		}()
	}
	requests.Wait()
	close(statuses)

	counts := make(map[int]int)
	for status := range statuses {
		counts[status]++
	}
	if counts[http.StatusCreated] != 10 || counts[http.StatusConflict] != 10 {
		t.Fatalf("concurrent responses = %v, want ten 201 and ten 409", counts)
	}
}
