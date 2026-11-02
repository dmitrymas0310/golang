package main

import "errors"

type Transaction struct {
	ID          int
	Amount      float64
	Category    string
	Description string
	Date        string
}

var transactions = make([]Transaction, 0)

func AddTransaction(tx Transaction) error {
	if tx.Amount <= 0 {
		return errors.New("transaction amount must be greater than zero")
	}

	budget, exists := budgets[tx.Category]

	if exists {
		var currentTotal float64

		for _, transaction := range transactions {
			if transaction.Category == tx.Category {
				currentTotal += transaction.Amount
			}
		}
		if currentTotal+tx.Amount > budget.Limit {
			return errors.New("transaction amount exceeds budget limit")
		}
	}

	transactions = append(transactions, tx)
	return nil
}

func ListTransactions() []Transaction {
	return transactions
}
