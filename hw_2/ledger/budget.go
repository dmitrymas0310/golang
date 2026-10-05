package main

import (
	"encoding/json"
	"fmt"
	"io"
)

type Budget struct {
	Category string
	Limit    float64
	Period   string
}

var budgets = make(map[string]Budget)

func SetBudget(budget Budget) {
	budgets[budget.Category] = budget
}

func LoadBudgets(r io.Reader) error {
	var loadedBudgets []Budget

	decoder := json.NewDecoder(r)
	if err := decoder.Decode(&loadedBudgets); err != nil {
		return fmt.Errorf("failed to read or parse budgets: %w", err)
	}

	for _, budget := range loadedBudgets {
		SetBudget(budget)
	}

	return nil
}
