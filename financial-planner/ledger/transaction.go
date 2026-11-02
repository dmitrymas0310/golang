package ledger

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Transaction struct {
	ID          int
	Amount      float64
	Category    string
	Description string
	Date        string
}

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
	if _, err := time.Parse(time.DateOnly, tx.Date); err != nil {
		return errors.New("transaction date must have format YYYY-MM-DD")
	}
	return nil
}

// AddTransaction сохраняет прежнюю сигнатуру; для получения ID используйте CreateTransaction.
func AddTransaction(tx Transaction) error {
	_, err := CreateTransaction(tx)
	return err
}

func CreateTransaction(tx Transaction) (Transaction, error) {
	if err := tx.Validate(); err != nil {
		return Transaction{}, err
	}
	connection, err := database()
	if err != nil {
		return Transaction{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), databaseTimeout)
	defer cancel()
	sqlTx, err := connection.BeginTx(ctx, nil)
	if err != nil {
		return Transaction{}, fmt.Errorf("begin expense transaction: %w", err)
	}
	defer sqlTx.Rollback()

	var limit string
	err = sqlTx.QueryRowContext(ctx,
		`SELECT limit_amount FROM budgets WHERE category = $1 FOR UPDATE`, tx.Category).Scan(&limit)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return Transaction{}, fmt.Errorf("find budget for category %q: %w", tx.Category, err)
	}
	amount := strconv.FormatFloat(tx.Amount, 'f', -1, 64)
	if err == nil {
		var exceeded bool
		// NUMERIC выполняет сложение и сравнение без погрешности float64.
		err = sqlTx.QueryRowContext(ctx, `
            SELECT COALESCE(SUM(amount), 0) + $2::numeric > $3::numeric
            FROM expenses WHERE category = $1`, tx.Category, amount, limit).Scan(&exceeded)
		if err != nil {
			return Transaction{}, fmt.Errorf("calculate category spending: %w", err)
		}
		if exceeded {
			return Transaction{}, ErrBudgetExceeded
		}
	}

	var date time.Time
	err = sqlTx.QueryRowContext(ctx, `
        INSERT INTO expenses(amount, category, description, date) VALUES($1, $2, $3, $4)
        RETURNING id, amount, date`, amount, tx.Category, tx.Description, tx.Date).
		Scan(&tx.ID, &tx.Amount, &date)
	if err != nil {
		return Transaction{}, fmt.Errorf("insert expense: %w", err)
	}
	tx.Date = date.Format(time.DateOnly)
	if err := sqlTx.Commit(); err != nil {
		return Transaction{}, fmt.Errorf("commit expense: %w", err)
	}
	return tx, nil
}

func ListTransactions() ([]Transaction, error) {
	connection, err := database()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), databaseTimeout)
	defer cancel()
	rows, err := connection.QueryContext(ctx,
		`SELECT id, amount, category, description, date FROM expenses ORDER BY date DESC, id DESC`)
	if err != nil {
		return nil, fmt.Errorf("list expenses: %w", err)
	}
	defer rows.Close()
	result := make([]Transaction, 0)
	for rows.Next() {
		var tx Transaction
		var description sql.NullString
		var date time.Time
		if err := rows.Scan(&tx.ID, &tx.Amount, &tx.Category, &description, &date); err != nil {
			return nil, fmt.Errorf("read expense: %w", err)
		}
		tx.Description = description.String
		tx.Date = date.Format(time.DateOnly)
		result = append(result, tx)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate expenses: %w", err)
	}
	return result, nil
}
