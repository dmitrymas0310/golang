package api

import (
	"errors"
	"log"
	"net/http"

	"ledger"
)

func reportSummary(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	summary, err := ledger.GetReportSummary(r.Context(), query.Get("from"), query.Get("to"))
	if err != nil {
		if errors.Is(err, ledger.ErrInvalidReportRange) {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
			return
		}
		log.Printf("failed to get report summary: %v", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
		return
	}
	writeJSON(w, http.StatusOK, summary)
}
