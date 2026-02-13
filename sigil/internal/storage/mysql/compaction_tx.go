package mysql

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	sigilv1 "github.com/grafana/sigil/sigil/internal/gen/sigil/v1"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (s *WALStore) WithClaimedUncompacted(
	ctx context.Context,
	tenantID string,
	olderThan time.Time,
	limit int,
	fn func(context.Context, []*sigilv1.Generation) error,
) (int, error) {
	start := time.Now()
	if strings.TrimSpace(tenantID) == "" {
		observeWALMetrics("with_claimed_uncompacted", "error", start, 0)
		return 0, errors.New("tenant id is required")
	}
	if limit <= 0 {
		observeWALMetrics("with_claimed_uncompacted", "success", start, 0)
		return 0, nil
	}
	if fn == nil {
		observeWALMetrics("with_claimed_uncompacted", "error", start, 0)
		return 0, errors.New("claim callback is required")
	}

	claimedRows := 0
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var rows []GenerationModel
		if err := tx.
			Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("tenant_id = ? AND compacted = ? AND created_at <= ?", tenantID, false, olderThan.UTC()).
			Order("created_at ASC, id ASC").
			Limit(limit).
			Find(&rows).Error; err != nil {
			return fmt.Errorf("claim uncompacted rows for update: %w", err)
		}

		if len(rows) == 0 {
			return nil
		}

		generations := make([]*sigilv1.Generation, 0, len(rows))
		ids := make([]uint64, 0, len(rows))
		for _, row := range rows {
			generation, err := decodeGenerationPayload(row.Payload)
			if err != nil {
				return fmt.Errorf("decode generation payload %q: %w", row.GenerationID, err)
			}
			generations = append(generations, generation)
			ids = append(ids, row.ID)
		}

		if err := fn(ctx, generations); err != nil {
			return err
		}

		now := time.Now().UTC()
		result := tx.Model(&GenerationModel{}).
			Where("tenant_id = ? AND id IN ?", tenantID, ids).
			Where("compacted = ?", false).
			Updates(map[string]any{
				"compacted":    true,
				"compacted_at": now,
			})
		if result.Error != nil {
			return fmt.Errorf("mark claimed rows compacted: %w", result.Error)
		}
		claimedRows = int(result.RowsAffected)
		return nil
	})
	if err != nil {
		observeWALMetrics("with_claimed_uncompacted", "error", start, claimedRows)
		return 0, err
	}

	observeWALMetrics("with_claimed_uncompacted", "success", start, claimedRows)
	return claimedRows, nil
}
