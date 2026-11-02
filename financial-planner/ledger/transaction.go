package ledger

import (
	"errors"
	"strings"
	"sync"
)

type Transaction struct {
	ID          int
	Amount      float64
	Category    string
	Description string
	Date        string
}

var transactions = make([]Transaction, 0)

// stateMu защищает транзакции и бюджеты, включая проверку лимита при добавлении.
var stateMu sync.RWMutex

var ErrBudgetExceeded = errors.New("transaction amount exceeds budget limit")

func (tx Transaction) Validate() error {
	if tx.Amount <= 0 {
		return errors.New("transaction amount must be greater than zero")
	}

	if strings.TrimSpace(tx.Category) == "" {
		return errors.New("transaction category must not be empty")
	}

	if strings.TrimSpace(tx.Date) == "" {
		return errors.New("transaction date must not be empty")
	}

	return nil
}

func AddTransaction(tx Transaction) error {
	if err := tx.Validate(); err != nil {
		return err
	}

	stateMu.Lock()
	defer stateMu.Unlock()

	budget, exists := budgets[tx.Category]

	if exists {
		var currentTotal float64

		for _, transaction := range transactions {
			if transaction.Category == tx.Category {
				currentTotal += transaction.Amount
			}
		}
		if currentTotal+tx.Amount > budget.Limit {
			return ErrBudgetExceeded
		}
	}

	transactions = append(transactions, tx)
	return nil
}

func ListTransactions() []Transaction {
	stateMu.RLock()
	defer stateMu.RUnlock()

	result := make([]Transaction, len(transactions))
	copy(result, transactions)
	return result
}
