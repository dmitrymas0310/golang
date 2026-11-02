package ledger_test

import (
	"errors"
	"testing"

	"ledger"
)

func resetLedger(t *testing.T) {
	t.Helper()
	ledger.Reset()
	t.Cleanup(ledger.Reset)
}

func TestValidation(t *testing.T) {
	t.Parallel()
	t.Run("transaction", testTransactionValidate)
	t.Run("budget", testBudgetValidate)
}

func TestLedgerStore(t *testing.T) {
	t.Run("reset", testReset)
	t.Run("budget_limit", testAddTransactionRespectsBudget)
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
		{"valid transaction", 450, "Food", "2026-10", false},
		{"positive fractional amount", 0.01, "Food", "2026-10", false},
		{"zero amount", 0, "Food", "2026-10", true},
		{"negative amount", -450, "Food", "2026-10", true},
		{"empty category", 450, "", "2026-10", true},
		{"whitespace category", 450, " \t\n", "2026-10", true},
		{"empty date", 450, "Food", "", true},
		{"whitespace date", 450, "Food", " \t\n", true},
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
	if err := ledger.AddTransaction(ledger.Transaction{Amount: 60, Category: "Food", Date: "2026-10"}); err != nil {
		t.Fatal(err)
	}

	ledger.Reset()
	if got := ledger.ListTransactions(); len(got) != 0 {
		t.Fatalf("transactions after Reset = %v, want empty", got)
	}
	if got := ledger.ListBudgets(); len(got) != 0 {
		t.Fatalf("budgets after Reset = %v, want empty", got)
	}
	// Старый лимит больше не влияет на добавление транзакций.
	if err := ledger.AddTransaction(ledger.Transaction{Amount: 200, Category: "Food", Date: "2026-10"}); err != nil {
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

	accepted := ledger.Transaction{ID: 1, Amount: 3000, Category: "еда", Date: "2026-10"}
	if err := ledger.AddTransaction(accepted); err != nil {
		t.Fatalf("transaction within budget failed: %v", err)
	}
	before := ledger.ListTransactions()
	if len(before) != 1 || before[0] != accepted {
		t.Fatalf("stored transactions = %v, want [%v]", before, accepted)
	}

	// Отдельно сумма 2500 допустима, но вместе с 3000 превышает лимит 5000.
	err := ledger.AddTransaction(ledger.Transaction{ID: 2, Amount: 2500, Category: "еда", Date: "2026-10"})
	if !errors.Is(err, ledger.ErrBudgetExceeded) {
		t.Fatalf("error = %v, want ErrBudgetExceeded", err)
	}
	after := ledger.ListTransactions()
	if len(after) != len(before) {
		t.Fatalf("rejected transaction changed count: before %d, after %d", len(before), len(after))
	}
	if after[0] != accepted {
		t.Fatalf("rejected transaction changed stored data: %v", after)
	}

	// Отказ не расходует бюджет: оставшиеся 2000 можно добавить ровно до лимита.
	if err := ledger.AddTransaction(ledger.Transaction{ID: 3, Amount: 2000, Category: "еда", Date: "2026-10"}); err != nil {
		t.Fatalf("transaction at exact budget limit failed: %v", err)
	}
	if got := len(ledger.ListTransactions()); got != 2 {
		t.Fatalf("stored transactions = %d, want 2", got)
	}
}
