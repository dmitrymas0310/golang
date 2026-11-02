package api

import (
	"log"
	"net/http"
	"time"
)

// LoggingMiddleware логирует метод, исходный путь и длительность запроса.
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		method, path := r.Method, r.URL.Path
		defer func() {
			log.Printf("method=%s path=%s duration=%s", method, path, time.Since(started))
		}()

		next.ServeHTTP(w, r)
	})
}
