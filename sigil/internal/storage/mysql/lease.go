package mysql

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

func (s *WALStore) ListTenantsForCompaction(ctx context.Context, olderThan time.Time, limit int) ([]string, error) {
	start := time.Now()
	tenants, err := s.listDistinctTenantIDs(ctx, `
SELECT DISTINCT tenant_id
FROM generations
WHERE compacted = FALSE
  AND created_at <= ?
ORDER BY tenant_id ASC
LIMIT ?`, olderThan.UTC(), limit)
	if err != nil {
		observeWALMetrics("list_tenants_compaction", "error", start, 0)
		return nil, err
	}
	observeWALMetrics("list_tenants_compaction", "success", start, len(tenants))
	return tenants, nil
}

func (s *WALStore) ListTenantsForTruncation(ctx context.Context, olderThan time.Time, limit int) ([]string, error) {
	start := time.Now()
	tenants, err := s.listDistinctTenantIDs(ctx, `
SELECT DISTINCT tenant_id
FROM generations
WHERE compacted = TRUE
  AND compacted_at IS NOT NULL
  AND compacted_at <= ?
ORDER BY tenant_id ASC
LIMIT ?`, olderThan.UTC(), limit)
	if err != nil {
		observeWALMetrics("list_tenants_truncation", "error", start, 0)
		return nil, err
	}
	observeWALMetrics("list_tenants_truncation", "success", start, len(tenants))
	return tenants, nil
}

func (s *WALStore) AcquireLease(ctx context.Context, tenantID, ownerID string, ttl time.Duration) (bool, string, time.Time, error) {
	start := time.Now()
	if strings.TrimSpace(tenantID) == "" {
		observeWALMetrics("acquire_lease", "error", start, 0)
		return false, "", time.Time{}, errors.New("tenant id is required")
	}
	if strings.TrimSpace(ownerID) == "" {
		observeWALMetrics("acquire_lease", "error", start, 0)
		return false, "", time.Time{}, errors.New("owner id is required")
	}
	if ttl <= 0 {
		observeWALMetrics("acquire_lease", "error", start, 0)
		return false, "", time.Time{}, errors.New("lease ttl must be > 0")
	}

	now := time.Now().UTC()
	expiresAt := now.Add(ttl)
	if err := s.db.WithContext(ctx).Exec(`
INSERT INTO compactor_leases (tenant_id, owner_id, leased_at, expires_at)
VALUES (?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
  owner_id = IF(owner_id = VALUES(owner_id) OR expires_at < VALUES(leased_at), VALUES(owner_id), owner_id),
  leased_at = IF(owner_id = VALUES(owner_id) OR expires_at < VALUES(leased_at), VALUES(leased_at), leased_at),
  expires_at = IF(owner_id = VALUES(owner_id) OR expires_at < VALUES(leased_at), VALUES(expires_at), expires_at)`,
		tenantID, ownerID, now, expiresAt,
	).Error; err != nil {
		observeWALMetrics("acquire_lease", "error", start, 0)
		return false, "", time.Time{}, fmt.Errorf("upsert lease: %w", err)
	}

	var lease CompactorLeaseModel
	if err := s.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		First(&lease).Error; err != nil {
		observeWALMetrics("acquire_lease", "error", start, 0)
		return false, "", time.Time{}, fmt.Errorf("load lease state: %w", err)
	}

	held := lease.OwnerID == ownerID && lease.ExpiresAt.After(now)
	observeWALMetrics("acquire_lease", "success", start, 1)
	return held, lease.OwnerID, lease.ExpiresAt.UTC(), nil
}

func (s *WALStore) listDistinctTenantIDs(ctx context.Context, query string, olderThan time.Time, limit int) ([]string, error) {
	if limit <= 0 {
		return []string{}, nil
	}

	rows, err := s.db.WithContext(ctx).Raw(query, olderThan, limit).Rows()
	if err != nil {
		return nil, fmt.Errorf("query tenant ids: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	tenants := make([]string, 0, limit)
	for rows.Next() {
		var tenantID string
		if err := rows.Scan(&tenantID); err != nil {
			return nil, fmt.Errorf("scan tenant id: %w", err)
		}
		tenantID = strings.TrimSpace(tenantID)
		if tenantID == "" {
			continue
		}
		tenants = append(tenants, tenantID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tenant ids: %w", err)
	}

	return tenants, nil
}
