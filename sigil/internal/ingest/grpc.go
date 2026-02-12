package ingest

import (
	"context"

	collecttracev1 "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	"google.golang.org/protobuf/proto"
)

type GRPCServer struct {
	collecttracev1.UnimplementedTraceServiceServer

	service *Service
}

func NewGRPCServer(service *Service) *GRPCServer {
	return &GRPCServer{service: service}
}

func (s *GRPCServer) Export(ctx context.Context, request *collecttracev1.ExportTraceServiceRequest) (*collecttracev1.ExportTraceServiceResponse, error) {
	if request != nil {
		if payload, err := proto.Marshal(request); err == nil {
			_ = s.service.ForwardToTempo(ctx, payload)
		}
	}

	return &collecttracev1.ExportTraceServiceResponse{}, nil
}
