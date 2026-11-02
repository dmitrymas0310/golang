package api

import "ledger"

type CreateBudgetRequest struct {
	Category string  `json:"category"`
	Limit    float64 `json:"limit"`
}

type BudgetResponse struct {
	Category string  `json:"category"`
	Limit    float64 `json:"limit"`
	Period   string  `json:"period"`
}

func (request CreateBudgetRequest) ToDomain() ledger.Budget {
	return ledger.Budget{
		Category: request.Category,
		Limit:    request.Limit,
	}
}

func NewBudgetResponse(budget ledger.Budget) BudgetResponse {
	return BudgetResponse{
		Category: budget.Category,
		Limit:    budget.Limit,
		Period:   budget.Period,
	}
}
