package main

import (
	"context"
	"dspn-regogenerator/cmd/web/handlers"
	"dspn-regogenerator/internal/usecases"
	"errors"
	"log/slog"
	"net/http"
)

func main() {
	slog.Info("Starting OPA policy manager")

	// Perform initial tests
	if err := usecases.InitialTest(context.TODO()); err != nil {
		slog.Error("Initial test failed", "error", err)
		return
	}
	slog.Info("Initial test passed")

	// Configure and start the HTTP server
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
