package main

import (
	"encoding/json"
	"net/http"
)

type validateRequest struct {
	Name     string `json:"name"`
	Replicas int    `json:"replicas"`
	Port     int    `json:"port"`
}

type validateResponse struct {
	Valid  bool     `json:"valid"`
	Errors []string `json:"errors"`
}

func registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /validate", handleValidate)
}

func handleValidate(w http.ResponseWriter, r *http.Request) {
	configValidationsTotal.Inc()

	var req validateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "malformed JSON", http.StatusBadRequest)
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

	if len(errors) > 0 {
		configValidationFailuresTotal.Inc()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(validateResponse{
			Valid:  false,
			Errors: errors,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(validateResponse{
		Valid:  true,
		Errors: []string{},
	})
}
