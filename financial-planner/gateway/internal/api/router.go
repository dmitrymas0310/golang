package api

import "net/http"

// RegisterRoutes подключает HTTP API с общим префиксом /api.
func RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/ping", ping)

	routes := http.NewServeMux()
	routes.HandleFunc("GET /transactions", listTransactions)
	routes.HandleFunc("POST /transactions", createTransaction)
	routes.HandleFunc("GET /budgets", listBudgets)
	routes.HandleFunc("POST /budgets", createBudget)
	routes.HandleFunc("PUT /budgets", notImplemented)

	mux.Handle("/api/", http.StripPrefix("/api", routes))
}

func notImplemented(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusNotImplemented, errorResponse{Error: "handler not implemented"})
}

func ping(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "pong"})
}
