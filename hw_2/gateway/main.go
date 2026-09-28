package main

import (
	"fmt"
	"log"
	"net/http"
)

func pingHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.WriteHeader(http.StatusOK)

	if _, err := fmt.Fprint(w, "pong"); err != nil {
		log.Printf("failed to write response: %v", err)
	}
}

func main() {
	http.HandleFunc("/ping", pingHandler)

	addr := ":8080"

	log.Printf("Gateway server started on http://localhost%s", addr)

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}