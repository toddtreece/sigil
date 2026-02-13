package mysql

import (
	"context"
	"errors"
	"testing"
	"time"

	sigilv1 "github.com/grafana/sigil/sigil/internal/gen/sigil/v1"
)

func TestAcquireLeaseLifecycle(t *testing.T) {
	store, cleanup := newTestWALStore(t)
	defer cleanup()

	if err := store.AutoMigrate(context.Background()); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	const (
		tenant = "tenant-lease"
		ttl    = 30 * time.Second
	)

	firstHeld, firstOwner, _, err := store.AcquireLease(context.Background(), tenant, "owner-a", ttl)
	if err != nil {
		t.Fatalf("acquire lease first: %v", err)
	}
	if !firstHeld || firstOwner != "owner-a" {
		t.Fatalf("expected owner-a lease held, got held=%v owner=%q", firstHeld, firstOwner)
	}

	renewedHeld, renewedOwner, _, err := store.AcquireLease(context.Background(), tenant, "owner-a", ttl)
	if err != nil {
		t.Fatalf("renew lease: %v", err)
	}
	if !renewedHeld || renewedOwner != "owner-a" {
		t.Fatalf("expected owner-a renew to succeed, got held=%v owner=%q", renewedHeld, renewedOwner)
	}

	deniedHeld, deniedOwner, _, err := store.AcquireLease(context.Background(), tenant, "owner-b", ttl)
	if err != nil {
		t.Fatalf("competing lease acquire: %v", err)
	}
	if deniedHeld || deniedOwner != "owner-a" {
		t.Fatalf("expected owner-b lease deny while active, got held=%v owner=%q", deniedHeld, deniedOwner)
	}

	if err := store.DB().Model(&CompactorLeaseModel{}).
		Where("tenant_id = ?", tenant).
		Update("expires_at", time.Now().UTC().Add(-time.Second)).Error; err != nil {
		t.Fatalf("expire lease: %v", err)
	}

	stolenHeld, stolenOwner, _, err := store.AcquireLease(context.Background(), tenant, "owner-b", ttl)
	if err != nil {
		t.Fatalf("steal expired lease: %v", err)
	}
	if !stolenHeld || stolenOwner != "owner-b" {
		t.Fatalf("expected owner-b to steal expired lease, got held=%v owner=%q", stolenHeld, stolenOwner)
	}
}

func TestListTenantsForCompactionAndTruncation(t *testing.T) {
	store, cleanup := newTestWALStore(t)
	defer cleanup()

	if err := store.AutoMigrate(context.Background()); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	base := time.Date(2026, 2, 12, 18, 0, 0, 0, time.UTC)
	requireNoBatchErrors(t, store.SaveBatch(context.Background(), "tenant-a", []*sigilv1.Generation{
		testGeneration("gen-tenant-a-1", "conv-a", base),
	}))
	requireNoBatchErrors(t, store.SaveBatch(context.Background(), "tenant-b", []*sigilv1.Generation{
		testGeneration("gen-tenant-b-1", "conv-b", base),
	}))

	compactionTenants, err := store.ListTenantsForCompaction(context.Background(), base.Add(time.Hour), 10)
	if err != nil {
		t.Fatalf("list tenants for compaction: %v", err)
	}
	if len(compactionTenants) != 2 || compactionTenants[0] != "tenant-a" || compactionTenants[1] != "tenant-b" {
		t.Fatalf("unexpected compaction tenants: %#v", compactionTenants)
	}

	if err := store.DB().Model(&GenerationModel{}).
		Where("tenant_id = ? AND generation_id = ?", "tenant-a", "gen-tenant-a-1").
		Updates(map[string]any{
			"compacted":    true,
			"compacted_at": time.Now().UTC().Add(-2 * time.Hour),
		}).Error; err != nil {
		t.Fatalf("set compaction state: %v", err)
	}

	truncationTenants, err := store.ListTenantsForTruncation(context.Background(), time.Now().UTC().Add(-time.Hour), 10)
	if err != nil {
		t.Fatalf("list tenants for truncation: %v", err)
	}
	if len(truncationTenants) != 1 || truncationTenants[0] != "tenant-a" {
		t.Fatalf("unexpected truncation tenants: %#v", truncationTenants)
	}
}

func TestWithClaimedUncompactedCommitsOnlyOnCallbackSuccess(t *testing.T) {
	store, cleanup := newTestWALStore(t)
	defer cleanup()

	if err := store.AutoMigrate(context.Background()); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	base := time.Date(2026, 2, 12, 18, 0, 0, 0, time.UTC)
	requireNoBatchErrors(t, store.SaveBatch(context.Background(), "tenant-a", []*sigilv1.Generation{
		testGeneration("gen-claim-1", "conv-claim", base),
		testGeneration("gen-claim-2", "conv-claim", base.Add(time.Minute)),
		testGeneration("gen-claim-3", "conv-claim", base.Add(2*time.Minute)),
	}))

	claimed, err := store.WithClaimedUncompacted(context.Background(), "tenant-a", base.Add(10*time.Minute), 2,
		func(_ context.Context, generations []*sigilv1.Generation) error {
			if len(generations) != 2 {
				t.Fatalf("expected 2 claimed generations, got %d", len(generations))
			}
			return nil
		},
	)
	if err != nil {
		t.Fatalf("with claimed uncompacted success: %v", err)
	}
	if claimed != 2 {
		t.Fatalf("expected 2 claimed rows, got %d", claimed)
	}

	var compactedCount int64
	if err := store.DB().Model(&GenerationModel{}).
		Where("tenant_id = ? AND compacted = ?", "tenant-a", true).
		Count(&compactedCount).Error; err != nil {
		t.Fatalf("count compacted rows: %v", err)
	}
	if compactedCount != 2 {
		t.Fatalf("expected 2 compacted rows, got %d", compactedCount)
	}

	expectedErr := errors.New("upload failed")
	_, err = store.WithClaimedUncompacted(context.Background(), "tenant-a", base.Add(10*time.Minute), 2,
		func(_ context.Context, generations []*sigilv1.Generation) error {
			if len(generations) != 1 {
				t.Fatalf("expected 1 claimed generation, got %d", len(generations))
			}
			return expectedErr
		},
	)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected callback error %v, got %v", expectedErr, err)
	}

	var remainingUncompacted int64
	if err := store.DB().Model(&GenerationModel{}).
		Where("tenant_id = ? AND compacted = ?", "tenant-a", false).
		Count(&remainingUncompacted).Error; err != nil {
		t.Fatalf("count uncompacted rows: %v", err)
	}
	if remainingUncompacted != 1 {
		t.Fatalf("expected 1 uncompacted row after callback failure, got %d", remainingUncompacted)
	}
}
