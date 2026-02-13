package compactor

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	sigilv1 "github.com/grafana/sigil/sigil/internal/gen/sigil/v1"
	"github.com/grafana/sigil/sigil/internal/storage/mysql"
	"github.com/grafana/sigil/sigil/internal/storage/object"
	"github.com/thanos-io/objstore"
)

func TestConcurrentCompactorsCompactWithoutDoubleCompaction(t *testing.T) {
	store, cleanup := newTestWALStore(t)
	defer cleanup()

	if err := store.AutoMigrate(context.Background()); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	base := time.Date(2026, 2, 12, 21, 0, 0, 0, time.UTC)
	batch := make([]*sigilv1.Generation, 0, 20)
	for i := 0; i < 20; i++ {
		batch = append(batch, testGeneration(
			fmt.Sprintf("gen-concurrent-%d", i),
			"conv-concurrent",
			base.Add(time.Duration(i)*time.Second),
		))
	}
	mustSaveGenerations(t, store, "tenant-concurrent", batch)

	blockStore := object.NewStoreWithBucket("sigil", objstore.NewInMemBucket())
	serviceA := newTestService(store, "owner-a", blockStore, nil)
	serviceB := newTestService(store, "owner-b", blockStore, nil)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		serviceA.runCompactCycle(context.Background())
	}()
	go func() {
		defer wg.Done()
		serviceB.runCompactCycle(context.Background())
	}()
	wg.Wait()

	var compactedCount int64
	if err := store.DB().Model(&mysql.GenerationModel{}).
		Where("tenant_id = ? AND compacted = ?", "tenant-concurrent", true).
		Count(&compactedCount).Error; err != nil {
		t.Fatalf("count compacted rows: %v", err)
	}
	if compactedCount != int64(len(batch)) {
		t.Fatalf("expected %d compacted rows, got %d", len(batch), compactedCount)
	}

	var blockCount int64
	if err := store.DB().Model(&mysql.CompactionBlockModel{}).
		Where("tenant_id = ?", "tenant-concurrent").
		Count(&blockCount).Error; err != nil {
		t.Fatalf("count compaction blocks: %v", err)
	}
	if blockCount != 1 {
		t.Fatalf("expected 1 compaction block for single tenant batch, got %d", blockCount)
	}
}

func TestConcurrentCompactorsTruncateSafely(t *testing.T) {
	store, cleanup := newTestWALStore(t)
	defer cleanup()

	if err := store.AutoMigrate(context.Background()); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	base := time.Date(2026, 2, 12, 21, 0, 0, 0, time.UTC)
	mustSaveGenerations(t, store, "tenant-truncate", []*sigilv1.Generation{
		testGeneration("gen-old-1", "conv-t", base),
		testGeneration("gen-old-2", "conv-t", base.Add(time.Second)),
		testGeneration("gen-recent", "conv-t", base.Add(2*time.Second)),
		testGeneration("gen-hot", "conv-t", base.Add(3*time.Second)),
	})

	if err := store.DB().Model(&mysql.GenerationModel{}).
		Where("tenant_id = ? AND generation_id IN ?", "tenant-truncate", []string{"gen-old-1", "gen-old-2"}).
		Updates(map[string]any{
			"compacted":    true,
			"compacted_at": time.Now().UTC().Add(-3 * time.Hour),
		}).Error; err != nil {
		t.Fatalf("set old compaction state values: %v", err)
	}
	if err := store.DB().Model(&mysql.GenerationModel{}).
		Where("tenant_id = ? AND generation_id = ?", "tenant-truncate", "gen-recent").
		Updates(map[string]any{
			"compacted":    true,
			"compacted_at": time.Now().UTC().Add(-5 * time.Minute),
		}).Error; err != nil {
		t.Fatalf("set recent compaction state value: %v", err)
	}

	blockStore := object.NewStoreWithBucket("sigil", objstore.NewInMemBucket())
	serviceA := newTestService(store, "owner-a", blockStore, nil)
	serviceB := newTestService(store, "owner-b", blockStore, nil)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		serviceA.runTruncateCycle(context.Background())
	}()
	go func() {
		defer wg.Done()
		serviceB.runTruncateCycle(context.Background())
	}()
	wg.Wait()

	assertGenerationExists(t, store, "tenant-truncate", "gen-old-1", false)
	assertGenerationExists(t, store, "tenant-truncate", "gen-old-2", false)
	assertGenerationExists(t, store, "tenant-truncate", "gen-recent", true)
	assertGenerationExists(t, store, "tenant-truncate", "gen-hot", true)
}
