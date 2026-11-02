package api

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"

	"ledger"
)

func listTransactions(w http.ResponseWriter, r *http.Request) {
	transactions, err := ledger.ListTransactions()
	if err != nil {
		log.Printf("failed to list transactions: %v", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
		return
	}
	response := make([]TransactionResponse, 0, len(transactions))
	for _, tx := range transactions {
		response = append(response, NewTransactionResponse(tx))
	}
	writeJSON(w, http.StatusOK, response)
}

func createTransaction(w http.ResponseWriter, r *http.Request) {
	var request CreateTransactionRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON body"})
		return
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "body must contain a single JSON object"})
		return
	}

	tx := request.ToDomain()
	if err := tx.Validate(); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	created, err := ledger.CreateTransaction(tx)
	if err != nil {
		if errors.Is(err, ledger.ErrBudgetExceeded) {
			writeJSON(w, http.StatusConflict, errorResponse{Error: "budget exceeded"})
			return
		}
		log.Printf("failed to add transaction: %v", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
		return
	}

	writeJSON(w, http.StatusCreated, NewTransactionResponse(created))
}
