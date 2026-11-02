package main

import (
	"log"
	"net/http"

	"gateway/internal/api"
)

func main() {
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	addr := ":8080"

	log.Printf("Gateway server started on http://localhost%s", addr)

	handler := api.LoggingMiddleware(api.JSONMiddleware(mux))
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
