package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/grafana/sigil/api/internal/generations"
	"github.com/grafana/sigil/api/internal/query"
)

func RegisterRoutes(mux *http.ServeMux, querySvc *query.Service, generationsSvc *generations.Service) {
	mux.HandleFunc("/healthz", health)
	mux.HandleFunc("/api/v1/generations:export", generations.NewHTTPHandler(generationsSvc))
	mux.HandleFunc("/api/v1/conversations", listConversations(querySvc))
	mux.HandleFunc("/api/v1/conversations/", getConversation(querySvc))
	mux.HandleFunc("/api/v1/completions", listCompletions(querySvc))
	mux.HandleFunc("/api/v1/traces/", getTrace(querySvc))
}

func health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func listConversations(querySvc *query.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": querySvc.ListConversations()})
	}
}

func getConversation(querySvc *query.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		id := strings.TrimPrefix(req.URL.Path, "/api/v1/conversations/")
		if id == "" || strings.Contains(id, "/") {
			http.Error(w, "invalid conversation id", http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, querySvc.GetConversation(id))
	}
}

func listCompletions(querySvc *query.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": querySvc.ListCompletions()})
	}
}

func getTrace(querySvc *query.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		id := strings.TrimPrefix(req.URL.Path, "/api/v1/traces/")
		if id == "" || strings.Contains(id, "/") {
			http.Error(w, "invalid trace id", http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, querySvc.GetTrace(id))
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
