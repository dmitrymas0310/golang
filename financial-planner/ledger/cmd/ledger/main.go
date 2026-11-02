package main

import (
	"bufio"
	"fmt"
	"os"

	"ledger"
)

func main() {
	fmt.Println("Ledger service started")

	file, err := os.Open("budgets.json")
	if err != nil {
		fmt.Println("Error opening budgets file:", err)
		return
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	if err := ledger.LoadBudgets(reader); err != nil {
		fmt.Println("Error loading budgets:", err)
		return
	}

	fmt.Println("Budgets loaded")

	trans_1 := ledger.Transaction{
		ID:          1,
		Amount:      1050.0,
		Category:    "Food",
		Description: "Groceries",
		Date:        "2026-10",
	}

	trans_2 := ledger.Transaction{
		ID:          2,
		Amount:      55.0,
		Category:    "Transport",
		Description: "Taxi",
		Date:        "2026-10",
	}

	trans_3 := ledger.Transaction{
		ID:          3,
		Amount:      1075.0,
		Category:    "Restaurant",
		Description: "Cinema",
		Date:        "2026-10",
	}

	// Проверяем разные типы через один интерфейс: корректные и ошибочные данные.
	validationExamples := []ledger.Validatable{
		trans_1,
		ledger.Budget{Category: "Food", Limit: 2000, Period: "2026-10"},
		ledger.Transaction{Amount: -10, Category: "Food", Date: "2026-10"},
		ledger.Budget{Category: "Food", Limit: 0},
	}
	for _, example := range validationExamples {
		if err := ledger.CheckValid(example); err != nil {
			fmt.Printf("%T validation failed: %v\n", example, err)
			continue
		}
		fmt.Printf("%T validation passed\n", example)
	}

	if err := ledger.AddTransaction(trans_1); err != nil {
		fmt.Println("Error adding transaction:", err)
	}
	if err := ledger.AddTransaction(trans_2); err != nil {
		fmt.Println("Error adding transaction:", err)
	}
	if err := ledger.AddTransaction(trans_3); err != nil {
		fmt.Println("Error adding transaction:", err)
	}

	fmt.Println(ledger.ListTransactions())
}
