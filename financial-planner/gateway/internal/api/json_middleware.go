package api

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

// JSONMiddleware приводит ответы, включая ошибки ServeMux, к единому JSON-формату.
func JSONMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := &bufferedResponse{header: make(http.Header)}
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Printf("panic handling %s %s: %v", r.Method, r.URL.Path, recovered)
				writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
			}
		}()
		next.ServeHTTP(response, r)

		for key, values := range response.header {
			if key != "Content-Type" && key != "Content-Length" {
				w.Header()[key] = values
			}
		}
		status := response.status
		if status == 0 {
			status = http.StatusOK
		}
		if status >= http.StatusBadRequest {
			var payload errorResponse
			if err := json.Unmarshal(response.body.Bytes(), &payload); err != nil || payload.Error == "" {
				payload.Error = strings.TrimSpace(response.body.String())
				if payload.Error == "" {
					payload.Error = strings.ToLower(http.StatusText(status))
				}
			}
			writeJSON(w, status, payload)
			return
		}
		if status >= http.StatusMultipleChoices && status < http.StatusBadRequest {
			writeJSON(w, status, map[string]string{"message": http.StatusText(status)})
			return
		}
		if json.Valid(response.body.Bytes()) {
			writeJSON(w, status, json.RawMessage(response.body.Bytes()))
			return
		}
		writeJSON(w, status, map[string]string{"message": strings.TrimSpace(response.body.String())})
	})
}

type bufferedResponse struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func (response *bufferedResponse) Header() http.Header {
	return response.header
}

func (response *bufferedResponse) WriteHeader(status int) {
	if response.status == 0 {
		response.status = status
	}
}

func (response *bufferedResponse) Write(body []byte) (int, error) {
	if response.status == 0 {
		response.WriteHeader(http.StatusOK)
	}
	return response.body.Write(body)
}
