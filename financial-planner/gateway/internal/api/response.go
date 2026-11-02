package api

import (
	"encoding/json"
	"log"
	"net/http"
)

type errorResponse struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	body, err := json.Marshal(value)
	if err != nil {
		log.Printf("failed to encode JSON response: %v", err)
		status = http.StatusInternalServerError
		body = []byte(`{"error":"internal server error"}`)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if _, err := w.Write(append(body, '\n')); err != nil {
		log.Printf("failed to write JSON response: %v", err)
	}
}
