package ledger

import (
	"context"
	"fmt"
)

// Reset удаляет данные из текущей схемы БД. Предназначен только для тестов.
func Reset() error {
	connection, err := database()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), databaseTimeout)
	defer cancel()
	if _, err := connection.ExecContext(ctx, `TRUNCATE TABLE expenses, budgets RESTART IDENTITY`); err != nil {
		return fmt.Errorf("reset test database: %w", err)
	}
	return nil
}
