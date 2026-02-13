package compactor

import (
	"context"
	"time"

	sigilv1 "github.com/grafana/sigil/sigil/internal/gen/sigil/v1"
)

type TenantDiscoverer interface {
	ListTenantsForCompaction(ctx context.Context, olderThan time.Time, limit int) ([]string, error)
	ListTenantsForTruncation(ctx context.Context, olderThan time.Time, limit int) ([]string, error)
}

type TenantLeaser interface {
	AcquireLease(ctx context.Context, tenantID, ownerID string, ttl time.Duration) (held bool, currentOwnerID string, expiresAt time.Time, err error)
}

type TransactionalClaimer interface {
	WithClaimedUncompacted(
		ctx context.Context,
		tenantID string,
		olderThan time.Time,
		limit int,
		fn func(context.Context, []*sigilv1.Generation) error,
	) (int, error)
}

type Truncator interface {
	TruncateCompacted(ctx context.Context, tenantID string, olderThan time.Time, limit int) (int64, error)
}
