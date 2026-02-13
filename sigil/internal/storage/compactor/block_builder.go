package compactor

import (
	"crypto/sha1"
	"encoding/binary"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	sigilv1 "github.com/grafana/sigil/sigil/internal/gen/sigil/v1"
	"github.com/grafana/sigil/sigil/internal/storage"
	"github.com/grafana/sigil/sigil/internal/storage/object"
	"google.golang.org/protobuf/proto"
)

type BuiltBlock struct {
	Block *storage.Block
	Meta  storage.BlockMeta
}

func BuildBlock(tenantID string, generations []*sigilv1.Generation) (*BuiltBlock, error) {
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return nil, errors.New("tenant id is required")
	}
	if len(generations) == 0 {
		return nil, errors.New("at least one generation is required")
	}

	records := make([]storage.GenerationRecord, 0, len(generations))
	for _, generation := range generations {
		if generation == nil {
			return nil, errors.New("generation is required")
		}
		if strings.TrimSpace(generation.GetId()) == "" {
			return nil, errors.New("generation.id is required")
		}
		payload, err := proto.Marshal(generation)
		if err != nil {
			return nil, fmt.Errorf("marshal generation %q: %w", generation.GetId(), err)
		}
		records = append(records, storage.GenerationRecord{
			GenerationID:   generation.GetId(),
			ConversationID: generation.GetConversationId(),
			CreatedAt:      generationCreatedAt(generation),
			Payload:        payload,
		})
	}

	sort.SliceStable(records, func(i, j int) bool {
		if records[i].CreatedAt.Equal(records[j].CreatedAt) {
			return records[i].GenerationID < records[j].GenerationID
		}
		return records[i].CreatedAt.Before(records[j].CreatedAt)
	})

	minTime := records[0].CreatedAt.UTC()
	maxTime := records[len(records)-1].CreatedAt.UTC()
	blockID := deterministicBlockID(records)
	block := &storage.Block{
		ID:          blockID,
		Generations: records,
	}

	dataBytes, indexBytes, _, err := object.EncodeBlock(block)
	if err != nil {
		return nil, fmt.Errorf("encode block: %w", err)
	}
	objectPath, indexPath := object.BlockObjectPaths(tenantID, blockID)

	return &BuiltBlock{
		Block: block,
		Meta: storage.BlockMeta{
			TenantID:        tenantID,
			BlockID:         blockID,
			MinTime:         minTime,
			MaxTime:         maxTime,
			GenerationCount: len(records),
			SizeBytes:       int64(len(dataBytes) + len(indexBytes)),
			ObjectPath:      objectPath,
			IndexPath:       indexPath,
			CreatedAt:       time.Now().UTC(),
		},
	}, nil
}

func deterministicBlockID(records []storage.GenerationRecord) string {
	hasher := sha1.New()
	for _, record := range records {
		_, _ = hasher.Write([]byte(record.GenerationID))
		_ = binary.Write(hasher, binary.LittleEndian, record.CreatedAt.UTC().UnixNano())
	}
	sum := hasher.Sum(nil)

	minTs := records[0].CreatedAt.UTC().UnixNano()
	maxTs := records[len(records)-1].CreatedAt.UTC().UnixNano()
	return fmt.Sprintf("block-%d-%d-%x", minTs, maxTs, sum[:6])
}

func generationCreatedAt(generation *sigilv1.Generation) time.Time {
	if generation.GetCompletedAt() != nil && generation.GetCompletedAt().AsTime().UnixNano() > 0 {
		return generation.GetCompletedAt().AsTime().UTC()
	}
	if generation.GetStartedAt() != nil && generation.GetStartedAt().AsTime().UnixNano() > 0 {
		return generation.GetStartedAt().AsTime().UTC()
	}
	return time.Now().UTC()
}
