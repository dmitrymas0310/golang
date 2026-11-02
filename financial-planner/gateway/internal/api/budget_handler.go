package api

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"ledger"
)

func listBudgets(w http.ResponseWriter, r *http.Request) {
	budgets := ledger.ListBudgets()
	response := make([]BudgetResponse, 0, len(budgets))
	for _, budget := range budgets {
		response = append(response, NewBudgetResponse(budget))
	}
	writeJSON(w, http.StatusOK, response)
}

func createBudget(w http.ResponseWriter, r *http.Request) {
	var request CreateBudgetRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON body"})
		return
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "body must contain a single JSON object"})
		return
	}

	budget := request.ToDomain()
	if err := budget.Validate(); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	if err := ledger.SetBudget(budget); err != nil {
		log.Printf("failed to set budget: %v", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
		return
	}

	writeJSON(w, http.StatusCreated, NewBudgetResponse(budget))
}
