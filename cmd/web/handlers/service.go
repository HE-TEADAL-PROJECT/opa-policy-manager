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
