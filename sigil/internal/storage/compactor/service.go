package compactor

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/services"
	"github.com/grafana/sigil/sigil/internal/config"
	sigilv1 "github.com/grafana/sigil/sigil/internal/gen/sigil/v1"
	"github.com/grafana/sigil/sigil/internal/storage"
)

const (
	compactPhase  = "compact"
	truncatePhase = "truncate"
)

type Service struct {
	cfg     config.CompactorConfig
	logger  log.Logger
	ownerID string

	discoverer TenantDiscoverer
	leaser     TenantLeaser
	claimer    TransactionalClaimer
	truncator  Truncator

	blockWriter   storage.BlockWriter
	metadataStore storage.BlockMetadataStore
}

func NewService(
	cfg config.CompactorConfig,
	logger log.Logger,
	ownerID string,
	discoverer TenantDiscoverer,
	leaser TenantLeaser,
	claimer TransactionalClaimer,
	truncator Truncator,
	blockWriter storage.BlockWriter,
	metadataStore storage.BlockMetadataStore,
) services.Service {
	module := &Service{
		cfg:           cfg,
		logger:        logger,
		ownerID:       strings.TrimSpace(ownerID),
		discoverer:    discoverer,
		leaser:        leaser,
		claimer:       claimer,
		truncator:     truncator,
		blockWriter:   blockWriter,
		metadataStore: metadataStore,
	}
	if module.logger == nil {
		module.logger = log.NewNopLogger()
	}
	if module.ownerID == "" {
		module.ownerID = defaultOwnerID()
	}

	return services.NewBasicService(module.start, module.run, module.stop).WithName(config.TargetCompactor)
}

func (s *Service) start(ctx context.Context) error {
	if err := s.validateDependencies(); err != nil {
		return err
	}

	s.runCompactCycle(ctx)
	s.runTruncateCycle(ctx)
	return nil
}

func (s *Service) run(ctx context.Context) error {
	compactTicker := time.NewTicker(s.cfg.CompactInterval)
	truncateTicker := time.NewTicker(s.cfg.TruncateInterval)
	defer compactTicker.Stop()
	defer truncateTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-compactTicker.C:
			s.runCompactCycle(ctx)
		case <-truncateTicker.C:
			s.runTruncateCycle(ctx)
		}
	}
}

func (s *Service) stop(_ error) error {
	return nil
}

func (s *Service) validateDependencies() error {
	switch {
	case s.discoverer == nil:
		return errors.New("discoverer dependency is required")
	case s.leaser == nil:
		return errors.New("leaser dependency is required")
	case s.claimer == nil:
		return errors.New("claimer dependency is required")
	case s.truncator == nil:
		return errors.New("truncator dependency is required")
	case s.blockWriter == nil:
		return errors.New("block writer dependency is required")
	case s.metadataStore == nil:
		return errors.New("metadata store dependency is required")
	}
	return nil
}

func (s *Service) runCompactCycle(ctx context.Context) {
	start := time.Now()
	status := "success"
	defer func() {
		observeRunMetrics(compactPhase, status, start)
	}()

	now := time.Now().UTC()
	tenants, err := s.discoverer.ListTenantsForCompaction(ctx, now, s.cfg.BatchSize)
	if err != nil {
		status = "error"
		_ = level.Error(s.logger).Log("msg", "compactor tenant discovery failed", "phase", compactPhase, "err", err)
		return
	}

	for _, tenantID := range tenants {
		held, currentOwnerID, expiresAt, err := s.leaser.AcquireLease(ctx, tenantID, s.ownerID, s.cfg.LeaseTTL)
		if err != nil {
			status = "error"
			_ = level.Error(s.logger).Log("msg", "compactor lease acquisition failed", "phase", compactPhase, "tenant_id", tenantID, "owner_id", s.ownerID, "err", err)
			continue
		}
		setLeaseMetric(tenantID, held)
		if !held {
			_ = level.Debug(s.logger).Log("msg", "compactor lease not held", "phase", compactPhase, "tenant_id", tenantID, "owner_id", s.ownerID, "current_owner_id", currentOwnerID, "lease_expires_at", expiresAt)
			continue
		}

		cycleStart := time.Now()
		claimedCount, err := s.claimer.WithClaimedUncompacted(ctx, tenantID, now, s.cfg.BatchSize, func(ctx context.Context, generations []*sigilv1.Generation) error {
			if len(generations) == 0 {
				return nil
			}
			built, err := BuildBlock(tenantID, generations)
			if err != nil {
				return fmt.Errorf("build compaction block: %w", err)
			}

			if err := s.blockWriter.WriteBlock(ctx, tenantID, built.Block); err != nil {
				return fmt.Errorf("write compaction block: %w", err)
			}
			if err := s.metadataStore.InsertBlock(ctx, built.Meta); err != nil {
				return fmt.Errorf("insert block metadata: %w", err)
			}

			observeCompacted(len(generations))
			_ = level.Info(s.logger).Log(
				"msg", "compactor block created",
				"phase", compactPhase,
				"tenant_id", tenantID,
				"owner_id", s.ownerID,
				"block_id", built.Block.ID,
				"compacted", len(generations),
				"size_bytes", built.Meta.SizeBytes,
				"duration", time.Since(cycleStart),
			)
			return nil
		})
		if err != nil {
			status = "error"
			_ = level.Error(s.logger).Log("msg", "compactor claim failed", "phase", compactPhase, "tenant_id", tenantID, "owner_id", s.ownerID, "err", err)
			continue
		}
		if claimedCount > 0 {
			_ = level.Info(s.logger).Log("msg", "compactor claim complete", "phase", compactPhase, "tenant_id", tenantID, "owner_id", s.ownerID, "claimed", claimedCount, "duration", time.Since(cycleStart))
		}
	}
}

func (s *Service) runTruncateCycle(ctx context.Context) {
	start := time.Now()
	status := "success"
	defer func() {
		observeRunMetrics(truncatePhase, status, start)
	}()

	olderThan := time.Now().UTC().Add(-s.cfg.Retention)
	tenants, err := s.discoverer.ListTenantsForTruncation(ctx, olderThan, s.cfg.BatchSize)
	if err != nil {
		status = "error"
		_ = level.Error(s.logger).Log("msg", "compactor tenant discovery failed", "phase", truncatePhase, "err", err)
		return
	}

	for _, tenantID := range tenants {
		held, _, _, err := s.leaser.AcquireLease(ctx, tenantID, s.ownerID, s.cfg.LeaseTTL)
		if err != nil {
			status = "error"
			_ = level.Error(s.logger).Log("msg", "compactor lease acquisition failed", "phase", truncatePhase, "tenant_id", tenantID, "owner_id", s.ownerID, "err", err)
			continue
		}
		setLeaseMetric(tenantID, held)
		if !held {
			continue
		}

		for {
			deleted, err := s.truncator.TruncateCompacted(ctx, tenantID, olderThan, s.cfg.BatchSize)
			if err != nil {
				status = "error"
				_ = level.Error(s.logger).Log("msg", "compactor truncate failed", "phase", truncatePhase, "tenant_id", tenantID, "owner_id", s.ownerID, "err", err)
				break
			}
			observeTruncated(deleted)
			_ = level.Info(s.logger).Log("msg", "compactor truncate pass complete", "phase", truncatePhase, "tenant_id", tenantID, "owner_id", s.ownerID, "truncated", deleted)
			if deleted < int64(s.cfg.BatchSize) {
				break
			}
		}
	}
}

func defaultOwnerID() string {
	hostname, err := os.Hostname()
	if err != nil || strings.TrimSpace(hostname) == "" {
		hostname = "unknown-host"
	}

	var suffix [4]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		return hostname
	}
	return fmt.Sprintf("%s-%s", hostname, hex.EncodeToString(suffix[:]))
}
