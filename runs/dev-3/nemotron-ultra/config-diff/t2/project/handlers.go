package main

import (
	"encoding/json"
	"net/http"
)

type diffRequest struct {
	Old map[string]string `json:"old"`
	New map[string]string `json:"new"`
}

type diffResponse struct {
	Added   map[string]string       `json:"added"`
	Removed map[string]string       `json:"removed"`
	Changed map[string]changedValue `json:"changed"`
}

type changedValue struct {
	Old string `json:"old"`
	New string `json:"new"`
}

func registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /diff", func(w http.ResponseWriter, r *http.Request) {
		var req diffRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}

		if req.Old == nil || req.New == nil {
			http.Error(w, "old and new are required", http.StatusBadRequest)
			return
		}

		added := make(map[string]string)
		removed := make(map[string]string)
		changed := make(map[string]changedValue)

		for k, v := range req.New {
			if oldVal, exists := req.Old[k]; exists {
				if oldVal != v {
					changed[k] = changedValue{Old: oldVal, New: v}
				}
			} else {
				added[k] = v
			}
		}

		for k, v := range req.Old {
			if _, exists := req.New[k]; !exists {
				removed[k] = v
			}
		}

		resp := diffResponse{
			Added:   added,
			Removed: removed,
			Changed: changed,
		}

		totalChanges := len(added) + len(removed) + len(changed)
		configDiffsTotal.Inc()
		configDiffChangesLast.Set(float64(totalChanges))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
}
