package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"gateway/internal/api"
	"ledger"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	err := ledger.InitDB(ctx)
	cancel()
	if err != nil {
		log.Fatalf("Gateway database connection failed: %v", err)
	}
	defer ledger.DB.Close()
	log.Println("Gateway connected to PostgreSQL")

	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	addr := ":8080"

	log.Printf("Gateway server started on http://localhost%s", addr)

	handler := api.LoggingMiddleware(api.JSONMiddleware(mux))
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
