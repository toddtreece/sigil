package compactor

import (
	"context"
	"errors"
	"testing"
	"time"

	sigilv1 "github.com/grafana/sigil/sigil/internal/gen/sigil/v1"
	"github.com/grafana/sigil/sigil/internal/storage/mysql"
	"github.com/grafana/sigil/sigil/internal/storage/object"
	"github.com/thanos-io/objstore"
)

func TestRunCompactCycleSuccessWritesBlockAndMarksCompacted(t *testing.T) {
	store, cleanup := newTestWALStore(t)
	defer cleanup()

	if err := store.AutoMigrate(context.Background()); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	base := time.Date(2026, 2, 12, 20, 0, 0, 0, time.UTC)
	mustSaveGenerations(t, store, "tenant-a", []*sigilv1.Generation{
		testGeneration("gen-c-1", "conv-c", base),
		testGeneration("gen-c-2", "conv-c", base.Add(time.Minute)),
		testGeneration("gen-c-3", "conv-c", base.Add(2*time.Minute)),
	})

	bucket := objstore.NewInMemBucket()
	blockStore := object.NewStoreWithBucket("sigil", bucket)
	service := newTestService(store, "owner-a", blockStore, nil)
	service.runCompactCycle(context.Background())

	var compactedCount int64
	if err := store.DB().Model(&mysql.GenerationModel{}).
		Where("tenant_id = ? AND compacted = ?", "tenant-a", true).
		Count(&compactedCount).Error; err != nil {
		t.Fatalf("count compacted rows: %v", err)
	}
	if compactedCount != 3 {
		t.Fatalf("expected 3 compacted rows, got %d", compactedCount)
	}

	var block mysql.CompactionBlockModel
	if err := store.DB().Where("tenant_id = ?", "tenant-a").First(&block).Error; err != nil {
		t.Fatalf("load compaction block: %v", err)
	}

	dataExists, err := bucket.Exists(context.Background(), block.ObjectPath)
	if err != nil {
		t.Fatalf("check data object existence: %v", err)
	}
	if !dataExists {
		t.Fatalf("expected data object to exist at %q", block.ObjectPath)
	}

	indexExists, err := bucket.Exists(context.Background(), block.IndexPath)
	if err != nil {
		t.Fatalf("check index object existence: %v", err)
	}
	if !indexExists {
		t.Fatalf("expected index object to exist at %q", block.IndexPath)
	}
}

func TestRunCompactCycleUploadFailureDoesNotMarkCompacted(t *testing.T) {
	store, cleanup := newTestWALStore(t)
	defer cleanup()

	if err := store.AutoMigrate(context.Background()); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	base := time.Date(2026, 2, 12, 20, 0, 0, 0, time.UTC)
	mustSaveGenerations(t, store, "tenant-a", []*sigilv1.Generation{
		testGeneration("gen-u-1", "conv-u", base),
	})

	service := newTestService(store, "owner-a", failingBlockWriter{err: errors.New("upload failed")}, nil)
	service.runCompactCycle(context.Background())

	var compactedCount int64
	if err := store.DB().Model(&mysql.GenerationModel{}).
		Where("tenant_id = ? AND compacted = ?", "tenant-a", true).
		Count(&compactedCount).Error; err != nil {
		t.Fatalf("count compacted rows: %v", err)
	}
	if compactedCount != 0 {
		t.Fatalf("expected no compacted rows after upload failure, got %d", compactedCount)
	}

	var blockCount int64
	if err := store.DB().Model(&mysql.CompactionBlockModel{}).
		Where("tenant_id = ?", "tenant-a").
		Count(&blockCount).Error; err != nil {
		t.Fatalf("count compaction blocks: %v", err)
	}
	if blockCount != 0 {
		t.Fatalf("expected no block metadata rows after upload failure, got %d", blockCount)
	}
}

func TestRunCompactCycleMetadataFailureDoesNotMarkCompacted(t *testing.T) {
	store, cleanup := newTestWALStore(t)
	defer cleanup()

	if err := store.AutoMigrate(context.Background()); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	base := time.Date(2026, 2, 12, 20, 0, 0, 0, time.UTC)
	mustSaveGenerations(t, store, "tenant-a", []*sigilv1.Generation{
		testGeneration("gen-m-1", "conv-m", base),
	})

	blockStore := object.NewStoreWithBucket("sigil", objstore.NewInMemBucket())
	service := newTestService(store, "owner-a", blockStore, failingMetadataStore{err: errors.New("metadata insert failed")})
	service.runCompactCycle(context.Background())

	var compactedCount int64
	if err := store.DB().Model(&mysql.GenerationModel{}).
		Where("tenant_id = ? AND compacted = ?", "tenant-a", true).
		Count(&compactedCount).Error; err != nil {
		t.Fatalf("count compacted rows: %v", err)
	}
	if compactedCount != 0 {
		t.Fatalf("expected no compacted rows after metadata failure, got %d", compactedCount)
	}
}

func TestRunTruncateCycleDeletesOnlyCompactedRowsOlderThanRetention(t *testing.T) {
	store, cleanup := newTestWALStore(t)
	defer cleanup()

	if err := store.AutoMigrate(context.Background()); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	base := time.Date(2026, 2, 12, 20, 0, 0, 0, time.UTC)
	mustSaveGenerations(t, store, "tenant-a", []*sigilv1.Generation{
		testGeneration("gen-old", "conv-t", base),
		testGeneration("gen-recent", "conv-t", base.Add(time.Minute)),
		testGeneration("gen-hot", "conv-t", base.Add(2*time.Minute)),
	})

	if err := store.DB().Model(&mysql.GenerationModel{}).
		Where("tenant_id = ? AND generation_id = ?", "tenant-a", "gen-old").
		Updates(map[string]any{
			"compacted":    true,
			"compacted_at": time.Now().UTC().Add(-2 * time.Hour),
		}).Error; err != nil {
		t.Fatalf("set old compaction state: %v", err)
	}
	if err := store.DB().Model(&mysql.GenerationModel{}).
		Where("tenant_id = ? AND generation_id = ?", "tenant-a", "gen-recent").
		Updates(map[string]any{
			"compacted":    true,
			"compacted_at": time.Now().UTC().Add(-10 * time.Minute),
		}).Error; err != nil {
		t.Fatalf("set recent compaction state: %v", err)
	}

	service := newTestService(store, "owner-a", object.NewStoreWithBucket("sigil", objstore.NewInMemBucket()), nil)
	service.runTruncateCycle(context.Background())

	assertGenerationExists(t, store, "tenant-a", "gen-old", false)
	assertGenerationExists(t, store, "tenant-a", "gen-recent", true)
	assertGenerationExists(t, store, "tenant-a", "gen-hot", true)
}

func assertGenerationExists(t *testing.T, store *mysql.WALStore, tenantID, generationID string, expected bool) {
	t.Helper()

	var count int64
	if err := store.DB().Model(&mysql.GenerationModel{}).
		Where("tenant_id = ? AND generation_id = ?", tenantID, generationID).
		Count(&count).Error; err != nil {
		t.Fatalf("count generation %q: %v", generationID, err)
	}
	if (count > 0) != expected {
		t.Fatalf("expected generation %q existence=%v, got count=%d", generationID, expected, count)
	}
}
