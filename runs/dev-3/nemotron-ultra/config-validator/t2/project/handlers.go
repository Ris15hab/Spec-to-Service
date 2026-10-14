package main

import (
	"encoding/json"
	"net/http"
)

type Config struct {
	Name     string `json:"name"`
	Replicas int    `json:"replicas"`
	Port     int    `json:"port"`
}

type ValidationResponse struct {
	Valid  bool     `json:"valid"`
	Errors []string `json:"errors"`
}

func registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /validate", validateHandler)
}

func validateHandler(w http.ResponseWriter, r *http.Request) {
	configValidationsTotal.Inc()

	var config Config
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&config); err != nil {
		http.Error(w, "malformed JSON", http.StatusBadRequest)
		return
	}

	var errors []string

	if config.Name == "" {
		errors = append(errors, "name must be non-empty")
	}

	if config.Replicas < 1 || config.Replicas > 100 {
		errors = append(errors, "replicas must be between 1 and 100")
	}

	if config.Port < 1 || config.Port > 65535 {
		errors = append(errors, "port must be between 1 and 65535")
	}

	if len(errors) > 0 {
		configValidationFailuresTotal.Inc()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(ValidationResponse{
			Valid:  false,
			Errors: errors,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(ValidationResponse{
		Valid:  true,
		Errors: []string{},
	})
}
