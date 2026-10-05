package main

import (
	"bufio"
	"fmt"
	"os"
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
	if err := LoadBudgets(reader); err != nil {
		fmt.Println("Error loading budgets:", err)
		return
	}

	fmt.Println("Loaded budgets:", budgets)

	trans_1 := Transaction{
		ID:          1,
		Amount:      1050.0,
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
		Amount:      1075.0,
		Category:    "Restaurant",
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
