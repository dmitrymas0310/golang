package ledger

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"ledger/internal/cache"
)

const ReportCacheTTL = 30 * time.Second

var ErrInvalidReportRange = errors.New("invalid report date range")

type ReportSummary struct {
	Category string  `json:"category"`
	Total    float64 `json:"total"`
}

// GetReportSummary агрегирует расходы по категориям за включённый интервал дат.
func GetReportSummary(ctx context.Context, from, to string) ([]ReportSummary, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	fromDate, err := time.Parse(time.DateOnly, from)
	if err != nil {
		return nil, fmt.Errorf("%w: from must have format YYYY-MM-DD", ErrInvalidReportRange)
	}
	toDate, err := time.Parse(time.DateOnly, to)
	if err != nil {
		return nil, fmt.Errorf("%w: to must have format YYYY-MM-DD", ErrInvalidReportRange)
	}
	if fromDate.After(toDate) {
		return nil, fmt.Errorf("%w: from must not be after to", ErrInvalidReportRange)
	}
	key := "report:summary:" + fromDate.Format(time.DateOnly) + ":" + toDate.Format(time.DateOnly)
	value, found, cacheErr := cache.Get(ctx, key)
	if cacheErr != nil {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		log.Printf("cannot read report cache; using PostgreSQL: %v", cacheErr)
	}
	if found {
		var summary []ReportSummary
		if err := json.Unmarshal(value, &summary); err == nil {
			if summary == nil {
				summary = make([]ReportSummary, 0)
			}
			log.Printf("report summary cache hit: key=%s", key)
			return summary, nil
		} else {
			log.Printf("invalid report cache JSON; using PostgreSQL: %v", err)
		}
	}
	log.Printf("report summary cache miss: key=%s", key)
	connection, err := database()
	if err != nil {
		return nil, err
	}
	queryCtx, cancel := context.WithTimeout(ctx, databaseTimeout)
	defer cancel()
	rows, err := connection.QueryContext(queryCtx, `
		SELECT category, SUM(amount) FROM expenses
		WHERE date >= $1 AND date <= $2
		GROUP BY category ORDER BY category`, from, to)
	if err != nil {
		return nil, fmt.Errorf("query report summary: %w", err)
	}
	defer rows.Close()
	summary := make([]ReportSummary, 0)
	for rows.Next() {
		var row ReportSummary
		if err := rows.Scan(&row.Category, &row.Total); err != nil {
			return nil, fmt.Errorf("read report summary: %w", err)
		}
		summary = append(summary, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate report summary: %w", err)
	}
	// Ошибка кеша не меняет успешный результат SQL-запроса.
	if cacheErr == nil {
		encoded, err := json.Marshal(summary)
		if err != nil {
			log.Printf("cannot encode report cache: %v", err)
		} else if err := cache.Set(ctx, key, encoded, ReportCacheTTL); err != nil {
			log.Printf("cannot write report cache: %v", err)
		}
	}
	return summary, nil
}
