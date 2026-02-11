package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/grafana/sigil/api/internal/query"
	"github.com/grafana/sigil/api/internal/records"
)

func RegisterRoutes(mux *http.ServeMux, querySvc *query.Service, recordsSvc *records.Service) {
	mux.HandleFunc("/healthz", health)
	mux.HandleFunc("/api/v1/records", createRecord(recordsSvc))
	mux.HandleFunc("/api/v1/records/", getRecord(recordsSvc))
	mux.HandleFunc("/api/v1/conversations", listConversations(querySvc))
	mux.HandleFunc("/api/v1/conversations/", getConversation(querySvc))
	mux.HandleFunc("/api/v1/completions", listCompletions(querySvc))
	mux.HandleFunc("/api/v1/traces/", getTrace(querySvc))
}

func health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func createRecord(recordsSvc *records.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var body records.CreateRecordRequest
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		record, err := recordsSvc.Create(req.Context(), body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusCreated, record)
	}
}

func getRecord(recordsSvc *records.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		id := strings.TrimPrefix(req.URL.Path, "/api/v1/records/")
		record, err := recordsSvc.Get(req.Context(), id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		writeJSON(w, http.StatusOK, record)
	}
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
