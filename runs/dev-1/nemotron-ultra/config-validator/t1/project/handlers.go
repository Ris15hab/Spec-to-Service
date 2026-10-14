package main

import (
	"encoding/json"
	"net/http"
	"sync"
)

type ValidateRequest struct {
	Name     string `json:"name"`
	Replicas int    `json:"replicas"`
	Port     int    `json:"port"`
}

type ValidateResponse struct {
	Valid  bool     `json:"valid"`
	Errors []string `json:"errors"`
}

var (
	mu sync.Mutex
)

func registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /validate", handleValidate)
}

func handleValidate(w http.ResponseWriter, r *http.Request) {
	configValidationsTotal.Inc()

	var req ValidateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ValidateResponse{
			Valid:  false,
			Errors: []string{"malformed JSON"},
		})
		return
	}

	var errors []string

	if req.Name == "" {
		errors = append(errors, "name must be non-empty")
	}

	if req.Replicas < 1 || req.Replicas > 100 {
		errors = append(errors, "replicas must be between 1 and 100")
	}

	if req.Port < 1 || req.Port > 65535 {
		errors = append(errors, "port must be between 1 and 65535")
	}

	w.Header().Set("Content-Type", "application/json")

	if len(errors) > 0 {
		configValidationFailuresTotal.Inc()
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(ValidateResponse{
			Valid:  false,
			Errors: errors,
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(ValidateResponse{
		Valid:  true,
		Errors: []string{},
	})
}
