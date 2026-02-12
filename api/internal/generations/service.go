package generations

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	sigilv1 "github.com/grafana/sigil/api/internal/gen/sigil/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	defaultOperationNameSync   = "generateText"
	defaultOperationNameStream = "streamText"
)

type ExportResult struct {
	GenerationID string
	Accepted     bool
	Error        string
}

type Store interface {
	SaveBatch(ctx context.Context, generations []*sigilv1.Generation) []error
}

type Exporter interface {
	Export(ctx context.Context, req *sigilv1.ExportGenerationsRequest) *sigilv1.ExportGenerationsResponse
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) Export(ctx context.Context, req *sigilv1.ExportGenerationsRequest) *sigilv1.ExportGenerationsResponse {
	if req == nil || len(req.Generations) == 0 {
		return &sigilv1.ExportGenerationsResponse{}
	}

	results := make([]ExportResult, len(req.Generations))
	accepted := make([]*sigilv1.Generation, 0, len(req.Generations))
	acceptedIdx := make([]int, 0, len(req.Generations))

	for i := range req.Generations {
		generation := cloneGeneration(req.Generations[i])
		if generation == nil {
			results[i] = ExportResult{Accepted: false, Error: "generation is required"}
			continue
		}

		normalizeGeneration(generation)
		if err := validateGeneration(generation); err != nil {
			results[i] = ExportResult{GenerationID: generation.Id, Accepted: false, Error: err.Error()}
			continue
		}

		results[i] = ExportResult{GenerationID: generation.Id, Accepted: true}
		accepted = append(accepted, generation)
		acceptedIdx = append(acceptedIdx, i)
	}

	if len(accepted) > 0 {
		errs := s.store.SaveBatch(ctx, accepted)
		for i := range acceptedIdx {
			idx := acceptedIdx[i]
			if i < len(errs) && errs[i] != nil {
				results[idx].Accepted = false
				results[idx].Error = errs[i].Error()
			}
		}
	}

	response := &sigilv1.ExportGenerationsResponse{Results: make([]*sigilv1.ExportGenerationResult, len(results))}
	for i := range results {
		response.Results[i] = &sigilv1.ExportGenerationResult{
			GenerationId: results[i].GenerationID,
			Accepted:     results[i].Accepted,
			Error:        results[i].Error,
		}
	}

	return response
}

func normalizeGeneration(g *sigilv1.Generation) {
	if g.Id == "" {
		g.Id = newGenerationID()
	}
	if g.Mode == sigilv1.GenerationMode_GENERATION_MODE_UNSPECIFIED {
		g.Mode = sigilv1.GenerationMode_GENERATION_MODE_SYNC
	}
	if g.OperationName == "" {
		g.OperationName = defaultOperationNameForMode(g.Mode)
	}
	if g.StartedAt == nil {
		g.StartedAt = timestamppb.New(time.Now().UTC())
	}
	if g.CompletedAt == nil {
		g.CompletedAt = timestamppb.New(time.Now().UTC())
	}
}

func validateGeneration(g *sigilv1.Generation) error {
	if g.Model == nil {
		return fmt.Errorf("generation.model is required")
	}
	if g.Model.Provider == "" {
		return fmt.Errorf("generation.model.provider is required")
	}
	if g.Model.Name == "" {
		return fmt.Errorf("generation.model.name is required")
	}
	if g.Mode != sigilv1.GenerationMode_GENERATION_MODE_SYNC && g.Mode != sigilv1.GenerationMode_GENERATION_MODE_STREAM {
		return fmt.Errorf("generation.mode is invalid")
	}
	return nil
}

func defaultOperationNameForMode(mode sigilv1.GenerationMode) string {
	if mode == sigilv1.GenerationMode_GENERATION_MODE_STREAM {
		return defaultOperationNameStream
	}
	return defaultOperationNameSync
}

var generationCounter atomic.Uint64

func newGenerationID() string {
	counter := generationCounter.Add(1)
	return fmt.Sprintf("gen-%d", counter)
}

type MemoryStore struct {
	mu          sync.RWMutex
	generations map[string]*sigilv1.Generation
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{generations: map[string]*sigilv1.Generation{}}
}

func (s *MemoryStore) SaveBatch(_ context.Context, generations []*sigilv1.Generation) []error {
	s.mu.Lock()
	defer s.mu.Unlock()

	errResults := make([]error, len(generations))
	for i := range generations {
		s.generations[generations[i].Id] = cloneGeneration(generations[i])
	}
	return errResults
}

func (s *MemoryStore) Get(id string) (*sigilv1.Generation, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	generation, ok := s.generations[id]
	if !ok {
		return nil, false
	}
	return cloneGeneration(generation), true
}

func cloneGeneration(in *sigilv1.Generation) *sigilv1.Generation {
	if in == nil {
		return nil
	}

	out := *in
	if in.Model != nil {
		model := *in.Model
		out.Model = &model
	}

	if len(in.Input) > 0 {
		out.Input = make([]*sigilv1.Message, len(in.Input))
		for i := range in.Input {
			out.Input[i] = cloneMessage(in.Input[i])
		}
	}
	if len(in.Output) > 0 {
		out.Output = make([]*sigilv1.Message, len(in.Output))
		for i := range in.Output {
			out.Output[i] = cloneMessage(in.Output[i])
		}
	}
	if len(in.Tools) > 0 {
		out.Tools = make([]*sigilv1.ToolDefinition, len(in.Tools))
		for i := range in.Tools {
			tool := *in.Tools[i]
			tool.InputSchemaJson = append([]byte(nil), in.Tools[i].InputSchemaJson...)
			out.Tools[i] = &tool
		}
	}
	if in.Usage != nil {
		usage := *in.Usage
		out.Usage = &usage
	}
	if len(in.Tags) > 0 {
		out.Tags = make(map[string]string, len(in.Tags))
		for key, value := range in.Tags {
			out.Tags[key] = value
		}
	}
	if len(in.RawArtifacts) > 0 {
		out.RawArtifacts = make([]*sigilv1.Artifact, len(in.RawArtifacts))
		for i := range in.RawArtifacts {
			artifact := *in.RawArtifacts[i]
			artifact.Payload = append([]byte(nil), in.RawArtifacts[i].Payload...)
			out.RawArtifacts[i] = &artifact
		}
	}

	return &out
}

func cloneMessage(in *sigilv1.Message) *sigilv1.Message {
	if in == nil {
		return nil
	}
	out := *in
	if len(in.Parts) > 0 {
		out.Parts = make([]*sigilv1.Part, len(in.Parts))
		for i := range in.Parts {
			out.Parts[i] = clonePart(in.Parts[i])
		}
	}
	return &out
}

func clonePart(in *sigilv1.Part) *sigilv1.Part {
	if in == nil {
		return nil
	}
	out := *in
	if in.Metadata != nil {
		metadata := *in.Metadata
		out.Metadata = &metadata
	}

	switch payload := in.Payload.(type) {
	case *sigilv1.Part_Text:
		out.Payload = &sigilv1.Part_Text{Text: payload.Text}
	case *sigilv1.Part_Thinking:
		out.Payload = &sigilv1.Part_Thinking{Thinking: payload.Thinking}
	case *sigilv1.Part_ToolCall:
		if payload.ToolCall != nil {
			call := *payload.ToolCall
			call.InputJson = append([]byte(nil), payload.ToolCall.InputJson...)
			out.Payload = &sigilv1.Part_ToolCall{ToolCall: &call}
		}
	case *sigilv1.Part_ToolResult:
		if payload.ToolResult != nil {
			result := *payload.ToolResult
			result.ContentJson = append([]byte(nil), payload.ToolResult.ContentJson...)
			out.Payload = &sigilv1.Part_ToolResult{ToolResult: &result}
		}
	}

	return &out
}
