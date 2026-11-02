package api

import "ledger"

type CreateTransactionRequest struct {
	Amount      float64 `json:"amount"`
	Category    string  `json:"category"`
	Description string  `json:"description"`
	Date        string  `json:"date"`
}

type TransactionResponse struct {
	ID          int     `json:"id"`
	Amount      float64 `json:"amount"`
	Category    string  `json:"category"`
	Description string  `json:"description"`
	Date        string  `json:"date"`
}

func (request CreateTransactionRequest) ToDomain() ledger.Transaction {
	return ledger.Transaction{
		Amount:      request.Amount,
		Category:    request.Category,
		Description: request.Description,
		Date:        request.Date,
	}
}

func NewTransactionResponse(transaction ledger.Transaction) TransactionResponse {
	return TransactionResponse{
		ID:          transaction.ID,
		Amount:      transaction.Amount,
		Category:    transaction.Category,
		Description: transaction.Description,
		Date:        transaction.Date,
	}
}
