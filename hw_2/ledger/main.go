package main

import "fmt"

func main() {
	fmt.Println("Ledger service started")

	budget := Budget{
		Category: "Food",
		Limit:    1000.0,
		Period:   "2026-10",
	}

	fmt.Println(budget.Category)
	fmt.Println(budget.Limit)
	fmt.Println(budget.Period)
	fmt.Println(budgets["Food"])

	trans_1 := Transaction{
		ID:          1,
		Amount:      50.0,
		Category:    "Food",
		Description: "Groceries",
		Date:        "2026-10",
	}

	trans_2 := Transaction{
		ID:          2,
		Amount:      55.0,
		Category:    "Transport",
		Description: "Taxi",
		Date:        "2026-10",
	}

	trans_3 := Transaction{
		ID:          3,
		Amount:      75.0,
		Category:    "Transport",
		Description: "Cinema",
		Date:        "2026-10",
	}

	if err := AddTransaction(trans_1); err != nil {
		fmt.Println("Error adding transaction:", err)
	}
	if err := AddTransaction(trans_2); err != nil {
		fmt.Println("Error adding transaction:", err)
	}
	if err := AddTransaction(trans_3); err != nil {
		fmt.Println("Error adding transaction:", err)
	}

	fmt.Println(ListTransactions())

}
