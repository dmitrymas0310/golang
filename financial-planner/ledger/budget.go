package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

type Budget struct {
	Category string
	Limit    float64
	Period   string
}

var budgets = make(map[string]Budget)

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

	budgets[budget.Category] = budget
	return nil
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
