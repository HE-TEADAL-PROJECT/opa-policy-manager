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

package handlers

import (
	"dspn-regogenerator/internal/usecases"
	"encoding/json"
	"io"
	"net/http"
)

func ListServicePolicies(w http.ResponseWriter, r *http.Request) {
	bundleStructure, err := usecases.GetBundleStructure()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json, err := json.Marshal(struct {
		Services []string `json:"services"`
	}{
		Services: bundleStructure.Services,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(json)
}

func AddServicePolicies(w http.ResponseWriter, r *http.Request) {
	serviceName := r.FormValue("serviceName")
	if serviceName == "" {
		http.Error(w, "serviceName is required", http.StatusBadRequest)
		return
	}
	file, _, err := r.FormFile("openAPISpec")
	if err != nil {
		http.Error(w, "Failed to get file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Read the file content
	specData, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusInternalServerError)
		return
	}

	err = usecases.AddService(serviceName, specData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func DeleteServicePolicies(w http.ResponseWriter, r *http.Request) {
	serviceName := r.FormValue("serviceName")
	if serviceName == "" {
		http.Error(w, "serviceName is required", http.StatusBadRequest)
		return
	}

	err := usecases.DeleteService(serviceName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
