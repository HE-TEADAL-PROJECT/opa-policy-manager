package main

import (
	"dspn-regogenerator/cmd/web/handlers"
	"errors"
	"log/slog"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/policies", handlers.ListServicePolicies)
	mux.HandleFunc("PUT /api/policies", handlers.AddServicePolicies)
	mux.HandleFunc("DELETE /api/policies", handlers.DeleteServicePolicies)
	slog.Info("Starting server on :8080")
	err := http.ListenAndServe(":8080", mux)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("Server error", "error", err)
	} else {
		slog.Info("Server closed")
	}
}
