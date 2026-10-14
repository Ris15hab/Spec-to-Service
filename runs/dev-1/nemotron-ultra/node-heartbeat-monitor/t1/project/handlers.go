package main

import (
	"encoding/json"
	"net/http"
)

var store = NewStore()

func registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /heartbeat", handleHeartbeat)
	mux.HandleFunc("GET /nodes", handleListNodes)
	mux.HandleFunc("GET /nodes/{node}", handleGetNode)
}

func handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Node string `json:"node"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "malformed JSON", http.StatusBadRequest)
		return
	}
	if req.Node == "" {
		http.Error(w, "node is required", http.StatusBadRequest)
		return
	}
	store.Heartbeat(req.Node)
	w.WriteHeader(http.StatusNoContent)
}

func handleListNodes(w http.ResponseWriter, r *http.Request) {
	nodes := store.GetAllNodes()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(nodes)
}

func handleGetNode(w http.ResponseWriter, r *http.Request) {
	node := r.PathValue("node")
	info, ok := store.GetNode(node)
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}
