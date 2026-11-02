package ledger_test

import (
	"errors"
	"reflect"
	"testing"

	"ledger"
	"ledger/testutil"
)

func resetLedger(t *testing.T) {
	t.Helper()
	testutil.Database(t)
}

func TestValidation(t *testing.T) {
	t.Parallel()
	t.Run("transaction", testTransactionValidate)
	t.Run("budget", testBudgetValidate)
}

func TestLedgerStore(t *testing.T) {
	t.Run("reset", testReset)
	t.Run("budget_limit", testAddTransactionRespectsBudget)
	t.Run("budget_upsert", testBudgetUpsert)
	t.Run("list_order_and_null_description", testTransactionListOrder)
	t.Run("decimal_limit", testDecimalBudget)
	t.Run("query_errors", testQueryErrors)
}

func testTransactionValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		amount   float64
		category string
		date     string
		wantErr  bool
	}{
		{"valid transaction", 450, "Food", "2026-10-01", false},
		{"positive fractional amount", 0.01, "Food", "2026-10-01", false},
		{"zero amount", 0, "Food", "2026-10-01", true},
		{"negative amount", -450, "Food", "2026-10-01", true},
		{"empty category", 450, "", "2026-10-01", true},
		{"whitespace category", 450, " \t\n", "2026-10-01", true},
		{"empty date", 450, "Food", "", true},
		{"whitespace date", 450, "Food", " \t\n", true},
		{"invalid date format", 450, "Food", "2026-10", true},
		{"invalid calendar date", 450, "Food", "2026-02-30", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tx := ledger.Transaction{
				Amount:   tt.amount,
				Category: tt.category,
				Date:     tt.date,
			}
			err := tx.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, want error = %t", err, tt.wantErr)
			}
		})
	}
}

func testBudgetUpsert(t *testing.T) {
	resetLedger(t)
	for _, budget := range []ledger.Budget{
		{Category: "Transport", Limit: 500},
		{Category: "Food", Limit: 100},
		{Category: "Food", Limit: 250},
	} {
		if err := ledger.SetBudget(budget); err != nil {
			t.Fatal(err)
		}
	}
	got, err := ledger.ListBudgets()
	if err != nil {
		t.Fatal(err)
	}
	want := []ledger.Budget{{Category: "Food", Limit: 250}, {Category: "Transport", Limit: 500}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("budgets = %v, want %v", got, want)
	}
}

func testTransactionListOrder(t *testing.T) {
	resetLedger(t)
	for _, date := range []string{"2026-10-01", "2026-11-02", "2026-11-02"} {
		created, err := ledger.CreateTransaction(ledger.Transaction{Amount: 10, Category: "NoBudget", Date: date})
		if err != nil {
			t.Fatal(err)
		}
		if created.ID <= 0 {
			t.Fatal("database did not generate an ID")
		}
	}
	// NULL может присутствовать в строках, добавленных напрямую через SQL.
	if _, err := ledger.DB.Exec(`UPDATE expenses SET description = NULL WHERE id = 2`); err != nil {
		t.Fatal(err)
	}
	got, err := ledger.ListTransactions()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[0].ID != 3 || got[1].ID != 2 || got[2].ID != 1 || got[1].Description != "" {
		t.Fatalf("unexpected expense order or NULL handling: %v", got)
	}
}

func testDecimalBudget(t *testing.T) {
	resetLedger(t)
	if err := ledger.SetBudget(ledger.Budget{Category: "Food", Limit: 0.3}); err != nil {
		t.Fatal(err)
	}
	for _, amount := range []float64{0.1, 0.2} {
		if err := ledger.AddTransaction(ledger.Transaction{Amount: amount, Category: "Food", Date: "2026-11-02"}); err != nil {
			t.Fatalf("exact decimal limit rejected: %v", err)
		}
	}
	if err := ledger.AddTransaction(ledger.Transaction{Amount: 0.01, Category: "Food", Date: "2026-11-02"}); !errors.Is(err, ledger.ErrBudgetExceeded) {
		t.Fatalf("error = %v, want ErrBudgetExceeded", err)
	}
}

func testQueryErrors(t *testing.T) {
	resetLedger(t)
	if _, err := ledger.DB.Exec(`DROP TABLE expenses, budgets`); err != nil {
		t.Fatal(err)
	}
	if _, err := ledger.ListTransactions(); err == nil {
		t.Fatal("ListTransactions swallowed a database error")
	}
	if _, err := ledger.ListBudgets(); err == nil {
		t.Fatal("ListBudgets swallowed a database error")
	}
	if err := ledger.SetBudget(ledger.Budget{Category: "Food", Limit: 100}); err == nil {
		t.Fatal("SetBudget swallowed a database error")
	}
	if err := ledger.AddTransaction(ledger.Transaction{Amount: 10, Category: "Food", Date: "2026-11-02"}); err == nil {
		t.Fatal("AddTransaction swallowed a database error")
	}
}

func testBudgetValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		limit    float64
		category string
		wantErr  bool
	}{
		{"positive limit", 1000, "Food", false},
		{"positive fractional limit", 0.01, "Food", false},
		{"zero limit", 0, "Food", true},
		{"negative limit", -1000, "Food", true},
		{"empty category", 1000, "", true},
		{"whitespace category", 1000, " \t\n", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			budget := ledger.Budget{
				Limit:    tt.limit,
				Category: tt.category,
			}
			err := budget.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, want error = %t", err, tt.wantErr)
			}
		})
	}
}

func testReset(t *testing.T) {
	resetLedger(t)
	if err := ledger.SetBudget(ledger.Budget{Category: "Food", Limit: 100}); err != nil {
		t.Fatal(err)
	}
	if err := ledger.AddTransaction(ledger.Transaction{Amount: 60, Category: "Food", Date: "2026-10-01"}); err != nil {
		t.Fatal(err)
	}

	if err := ledger.Reset(); err != nil {
		t.Fatal(err)
	}
	if got, err := ledger.ListTransactions(); err != nil || len(got) != 0 {
		t.Fatalf("transactions after Reset = %v, want empty", got)
	}
	if got, err := ledger.ListBudgets(); err != nil || len(got) != 0 {
		t.Fatalf("budgets after Reset = %v, want empty", got)
	}
	// Старый лимит больше не влияет на добавление транзакций.
	if err := ledger.AddTransaction(ledger.Transaction{Amount: 200, Category: "Food", Date: "2026-10-01"}); err != nil {
		t.Fatalf("old budget survived Reset: %v", err)
	}
	if err := ledger.SetBudget(ledger.Budget{Category: "Transport", Limit: 500}); err != nil {
		t.Fatalf("cannot set budget after Reset: %v", err)
	}
}

func testAddTransactionRespectsBudget(t *testing.T) {
	resetLedger(t)
	if err := ledger.SetBudget(ledger.Budget{Category: "еда", Limit: 5000}); err != nil {
		t.Fatal(err)
	}

	accepted := ledger.Transaction{ID: 1, Amount: 3000, Category: "еда", Date: "2026-10-01"}
	accepted, err := ledger.CreateTransaction(accepted)
	if err != nil {
		t.Fatalf("transaction within budget failed: %v", err)
	}
	before, err := ledger.ListTransactions()
	if err != nil {
		t.Fatal(err)
	}
	if len(before) != 1 || before[0] != accepted {
		t.Fatalf("stored transactions = %v, want [%v]", before, accepted)
	}

	// Отдельно сумма 2500 допустима, но вместе с 3000 превышает лимит 5000.
	err = ledger.AddTransaction(ledger.Transaction{ID: 2, Amount: 2500, Category: "еда", Date: "2026-10-01"})
	if !errors.Is(err, ledger.ErrBudgetExceeded) {
		t.Fatalf("error = %v, want ErrBudgetExceeded", err)
	}
	after, err := ledger.ListTransactions()
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != len(before) {
		t.Fatalf("rejected transaction changed count: before %d, after %d", len(before), len(after))
	}
	if after[0] != accepted {
		t.Fatalf("rejected transaction changed stored data: %v", after)
	}

	// Отказ не расходует бюджет: оставшиеся 2000 можно добавить ровно до лимита.
	if err := ledger.AddTransaction(ledger.Transaction{ID: 3, Amount: 2000, Category: "еда", Date: "2026-10-01"}); err != nil {
		t.Fatalf("transaction at exact budget limit failed: %v", err)
	}
	if got, err := ledger.ListTransactions(); err != nil || len(got) != 2 {
		t.Fatalf("stored transactions = %v, error = %v, want 2 records", got, err)
	}
}
