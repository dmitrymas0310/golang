package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"ledger"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	err := ledger.InitDB(ctx)
	cancel()
	if err != nil {
		log.Fatalf("Ledger startup failed: %v", err)
	}
	defer ledger.DB.Close()
	defer ledger.CloseCache()
	log.Println("Ledger connected to PostgreSQL")

	budgets, err := ledger.ListBudgets()
	if err != nil {
		log.Printf("Cannot list budgets: %v", err)
		return
	}
	transactions, err := ledger.ListTransactions()
	if err != nil {
		log.Printf("Cannot list transactions: %v", err)
		return
	}
	fmt.Println("Budgets:", budgets)
	fmt.Println("Transactions:", transactions)
}
