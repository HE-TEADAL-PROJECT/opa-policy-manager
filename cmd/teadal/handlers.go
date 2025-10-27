package main

import (
	"dspn-regogenerator/internal/bundle"
	"dspn-regogenerator/internal/config"
	"dspn-regogenerator/internal/policy/parser"
	"encoding/json"
	"net/http"
	"sync"
)

const (
	serviceNameFieldName = "serviceName"
	specFileFieldName    = "openAPISpec"
)

func describeBundleHandler(repo bundle.Repository, m *sync.RWMutex) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m.RLock()
		defer m.RUnlock()

		b, err := repo.Get(config.LatestBundleName)
		if err != nil {
			http.Error(w, "Failed to get bundle: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err = json.NewEncoder(w).Encode(b.Describe())
		if err != nil {
			http.Error(w, "Failed to encode bundle description: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func addServiceHandler(repo bundle.Repository, mutex *sync.RWMutex) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Validate input
		serviceName := r.FormValue(serviceNameFieldName)
		if serviceName == "" {
			http.Error(w, serviceNameFieldName+" is required", http.StatusBadRequest)
			return
		}

		file, _, err := r.FormFile(specFileFieldName)
		if err != nil {
			http.Error(w, "Failed to read OpenAPI spec file: "+err.Error(), http.StatusBadRequest)
			return
		}
		defer file.Close()

		parsedSpec, err := parser.ParseServiceSpec(file)
		if err != nil {
			http.Error(w, "Failed to parse OpenAPI spec: "+err.Error(), http.StatusBadRequest)
			return
		}

		newService := bundle.NewService(serviceName, parsedSpec)

		mutex.Lock()
		defer mutex.Unlock()

		b, err := repo.Get(config.LatestBundleName)
		if err != nil {
			http.Error(w, "Failed to get existing bundle: "+err.Error(), http.StatusInternalServerError)
			return
		}

		err = b.AddService(*newService)
		if err != nil {
			http.Error(w, "Failed to add service to bundle: "+err.Error(), http.StatusInternalServerError)
			return
		}

		err = repo.Save(config.LatestBundleName, b)
		if err != nil {
			http.Error(w, "Failed to save updated bundle: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
	}
}

func deleteServiceHandler(repo bundle.Repository, mutex *sync.RWMutex) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serviceName := r.URL.Query().Get(serviceNameFieldName)
		if serviceName == "" {
			serviceName = r.FormValue(serviceNameFieldName)
		}
		if serviceName == "" {
			http.Error(w, serviceNameFieldName+" is required", http.StatusBadRequest)
			return
		}

		mutex.Lock()
		defer mutex.Unlock()

		b, err := repo.Get(config.LatestBundleName)
		if err != nil {
			http.Error(w, "Failed to get existing bundle: "+err.Error(), http.StatusInternalServerError)
			return
		}

		err = b.RemoveService(serviceName)
		if err != nil {
			http.Error(w, "Failed to remove service from bundle: "+err.Error(), http.StatusInternalServerError)
			return
		}

		err = repo.Save(config.LatestBundleName, b)
		if err != nil {
			http.Error(w, "Failed to save updated bundle: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func createServer(repo bundle.Repository) *http.Server {
	mux := http.NewServeMux()
	mutex := &sync.RWMutex{}

	mux.HandleFunc("/bundle/describe", describeBundleHandler(repo, mutex))
	mux.HandleFunc("/service/add", addServiceHandler(repo, mutex))

	return &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
}
