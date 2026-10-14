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

	var cfg Config
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cfg); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ValidationResponse{
			Valid:  false,
			Errors: []string{"malformed JSON"},
		})
		return
	}

	var errors []string
	if cfg.Name == "" {
		errors = append(errors, "name must be non-empty")
	}
	if cfg.Replicas < 1 || cfg.Replicas > 100 {
		errors = append(errors, "replicas must be between 1 and 100")
	}
	if cfg.Port < 1 || cfg.Port > 65535 {
		errors = append(errors, "port must be between 1 and 65535")
	}

	w.Header().Set("Content-Type", "application/json")
	if len(errors) > 0 {
		configValidationFailuresTotal.Inc()
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(ValidationResponse{
			Valid:  false,
			Errors: errors,
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(ValidationResponse{
		Valid:  true,
		Errors: []string{},
	})
}
