package ingest

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/grafana/sigil/api/internal/records"
	"github.com/grafana/sigil/api/internal/tempo"
)

type Service struct {
	recordsStore    *records.Service
	tempoClient     *tempo.Client
	payloadMaxBytes int
}

type ingestResponse struct {
	Status              string         `json:"status"`
	PayloadExternalized bool           `json:"payloadExternalized"`
	RecordIDs           []string       `json:"recordIds,omitempty"`
	PayloadRef          string         `json:"payloadRef,omitempty"`
	SpanAttributes      map[string]any `json:"spanAttributes,omitempty"`
}

func NewService(recordsStore *records.Service, tempoClient *tempo.Client, payloadMaxBytes int) *Service {
	return &Service{
		recordsStore:    recordsStore,
		tempoClient:     tempoClient,
		payloadMaxBytes: payloadMaxBytes,
	}
}

func RegisterHTTPRoutes(mux *http.ServeMux, service *Service) {
	mux.HandleFunc("/v1/traces", service.HandleOTLPHTTP)
	mux.HandleFunc("/healthz", service.HandleHealth)
}

func (s *Service) HandleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Service) HandleOTLPHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	payload, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(w, "read body", http.StatusBadRequest)
		return
	}

	_ = s.tempoClient.ForwardTrace(req.Context(), payload)

	response := ingestResponse{Status: "accepted"}
	if len(payload) > s.payloadMaxBytes {
		record, createErr := s.recordsStore.Create(req.Context(), records.CreateRecordRequest{
			Kind: "otlp-payload",
			Payload: map[string]any{
				"size": len(payload),
			},
		})
		if createErr == nil {
			response.PayloadExternalized = true
			response.RecordIDs = []string{record.ID}
			response.PayloadRef = record.URI
			response.SpanAttributes = map[string]any{
				"sigil.payload_externalized": true,
				"sigil.record_ids":           []string{record.ID},
			}
		}
	}

	writeJSON(w, http.StatusAccepted, response)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func (s *Service) ForwardToTempo(ctx context.Context, payload []byte) error {
	return s.tempoClient.ForwardTrace(ctx, payload)
}
