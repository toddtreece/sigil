package records

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

type Record struct {
	ID        string         `json:"id"`
	URI       string         `json:"uri"`
	Kind      string         `json:"kind"`
	Payload   map[string]any `json:"payload"`
	CreatedAt time.Time      `json:"createdAt"`
}

type CreateRecordRequest struct {
	Kind    string         `json:"kind"`
	Payload map[string]any `json:"payload"`
}

type Store interface {
	Save(ctx context.Context, record Record) error
	Get(ctx context.Context, id string) (Record, error)
}

type Service struct {
	store Store
	next  uint64
}

func NewService(store Store) *Service {
	return &Service{store: store, next: 1}
}

func (s *Service) Create(ctx context.Context, request CreateRecordRequest) (Record, error) {
	if request.Kind == "" {
		request.Kind = "observation"
	}

	record := Record{
		ID:        fmt.Sprintf("r-%d", s.next),
		URI:       fmt.Sprintf("sigil://record/r-%d", s.next),
		Kind:      request.Kind,
		Payload:   request.Payload,
		CreatedAt: time.Now().UTC(),
	}
	s.next++

	if err := s.store.Save(ctx, record); err != nil {
		return Record{}, err
	}
	return record, nil
}

func (s *Service) Get(ctx context.Context, id string) (Record, error) {
	if id == "" {
		return Record{}, errors.New("id is required")
	}
	return s.store.Get(ctx, id)
}

type MemoryStore struct {
	mu      sync.RWMutex
	records map[string]Record
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{records: map[string]Record{}}
}

func (s *MemoryStore) Save(_ context.Context, record Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records[record.ID] = record
	return nil
}

func (s *MemoryStore) Get(_ context.Context, id string) (Record, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	record, ok := s.records[id]
	if !ok {
		return Record{}, errors.New("record not found")
	}
	return record, nil
}
