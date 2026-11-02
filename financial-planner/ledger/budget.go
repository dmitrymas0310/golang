package ledger

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type Budget struct {
	Category string
	Limit    float64
	Period   string
}

func (budget Budget) Validate() error {
	if budget.Limit <= 0 {
		return errors.New("budget limit must be greater than zero")
	}

	if strings.TrimSpace(budget.Category) == "" {
		return errors.New("budget category must not be empty")
	}

	return nil
}

func SetBudget(budget Budget) error {
	if err := budget.Validate(); err != nil {
		return err
	}

	connection, err := database()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), databaseTimeout)
	defer cancel()
	_, err = connection.ExecContext(ctx, `
		INSERT INTO budgets(category, limit_amount) VALUES($1, $2)
		ON CONFLICT(category) DO UPDATE SET limit_amount = EXCLUDED.limit_amount`,
		budget.Category, strconv.FormatFloat(budget.Limit, 'f', -1, 64))
	if err != nil {
		return fmt.Errorf("set budget for category %q: %w", budget.Category, err)
	}
	return nil
}

func ListBudgets() ([]Budget, error) {
	connection, err := database()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), databaseTimeout)
	defer cancel()
	rows, err := connection.QueryContext(ctx, `SELECT category, limit_amount FROM budgets ORDER BY category`)
	if err != nil {
		return nil, fmt.Errorf("list budgets: %w", err)
	}
	defer rows.Close()

	result := make([]Budget, 0)
	for rows.Next() {
		var budget Budget
		if err := rows.Scan(&budget.Category, &budget.Limit); err != nil {
			return nil, fmt.Errorf("read budget: %w", err)
		}
		result = append(result, budget)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate budgets: %w", err)
	}
	return result, nil
}

func LoadBudgets(r io.Reader) error {
	var loadedBudgets []Budget

	decoder := json.NewDecoder(r)
	if err := decoder.Decode(&loadedBudgets); err != nil {
		return fmt.Errorf("failed to read or parse budgets: %w", err)
	}

	for _, budget := range loadedBudgets {
		if err := SetBudget(budget); err != nil {
			return fmt.Errorf("failed to set budget for category %q: %w", budget.Category, err)
		}
	}

	return nil
}
