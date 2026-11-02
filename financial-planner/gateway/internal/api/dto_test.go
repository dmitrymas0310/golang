package api_test

import (
	"encoding/json"
	"ledger"
	"reflect"
	"testing"

	"gateway/internal/api"
)

func TestTransactionJSONConversion(t *testing.T) {
	var request api.CreateTransactionRequest
	if err := json.Unmarshal([]byte(`{"id":999,"amount":125.5,"category":"Food","description":"Lunch","date":"2026-10"}`), &request); err != nil {
		t.Fatal(err)
	}

	transaction := request.ToDomain()
	want := ledger.Transaction{
		Amount:      125.5,
		Category:    "Food",
		Description: "Lunch",
		Date:        "2026-10",
	}
	if transaction != want {
		t.Fatalf("domain transaction = %+v, want %+v", transaction, want)
	}

	// Идентификатор задаётся доменом, а не запросом клиента.
	transaction.ID = 7
	assertJSON(t, api.NewTransactionResponse(transaction),
		`{"id":7,"amount":125.5,"category":"Food","description":"Lunch","date":"2026-10"}`)
}

func TestBudgetJSONConversion(t *testing.T) {
	var request api.CreateBudgetRequest
	if err := json.Unmarshal([]byte(`{"category":"Food","limit":2000,"period":"2026-10"}`), &request); err != nil {
		t.Fatal(err)
	}

	budget := request.ToDomain()
	want := ledger.Budget{Category: "Food", Limit: 2000}
	if budget != want {
		t.Fatalf("domain budget = %+v, want %+v", budget, want)
	}

	// Период не входит в запрос, но публикуется в ответе из доменной модели.
	budget.Period = "2026-11"
	assertJSON(t, api.NewBudgetResponse(budget),
		`{"category":"Food","limit":2000,"period":"2026-11"}`)
}

func assertJSON(t *testing.T, value any, want string) {
	t.Helper()

	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}

	var gotJSON, wantJSON any
	if err := json.Unmarshal(encoded, &gotJSON); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(want), &wantJSON); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotJSON, wantJSON) {
		t.Fatalf("response JSON = %s, want %s", encoded, want)
	}
}
