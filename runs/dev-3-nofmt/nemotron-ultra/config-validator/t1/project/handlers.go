package main

import (
	"encoding/json"
	"net/http"
)

type validationRequest struct {
	Name     string `json:"name"`
	Replicas int    `json:"replicas"`
	Port     int    `json:"port"`
}

type validationResponse struct {
	Valid  bool     `json:"valid"`
	Errors []string `json:"errors"`
}

func registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /validate", validateHandler)
}

func validateHandler(w http.ResponseWriter, r *http.Request) {
	configValidationsTotal.Inc()

	var req validationRequest
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

	resp := validationResponse{
		Valid:  len(errors) == 0,
		Errors: errors,
	}

	if len(errors) > 0 {
		configValidationFailuresTotal.Inc()
		w.WriteHeader(http.StatusUnprocessableEntity)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
