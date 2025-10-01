// Copyright 2025 Matteo Brambilla - TEADAL
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
