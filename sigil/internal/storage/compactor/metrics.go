package compactor

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	compactorRunsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "sigil_compactor_runs_total",
		Help: "Total number of compactor loop runs partitioned by phase and status.",
	}, []string{"phase", "status"})
	compactorBlocksCreatedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "sigil_compactor_blocks_created_total",
		Help: "Total number of compacted blocks created by the compactor.",
	})
	compactorGenerationsCompactedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "sigil_compactor_generations_compacted_total",
		Help: "Total number of generations compacted into object storage blocks.",
	})
	compactorTruncatedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "sigil_compactor_truncated_total",
		Help: "Total number of compacted rows truncated from hot storage.",
	})
	compactorDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "sigil_compactor_duration_seconds",
		Help:    "Compactor phase duration in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"phase"})
	compactorLeaseHeld = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "sigil_compactor_lease_held",
		Help: "Whether the compactor currently holds a lease for the tenant (1=true, 0=false).",
	}, []string{"tenant_id"})
)

func observeRunMetrics(phase, status string, start time.Time) {
	compactorRunsTotal.WithLabelValues(phase, status).Inc()
	compactorDuration.WithLabelValues(phase).Observe(time.Since(start).Seconds())
}

func observeCompacted(count int) {
	if count <= 0 {
		return
	}
	compactorBlocksCreatedTotal.Inc()
	compactorGenerationsCompactedTotal.Add(float64(count))
}

func observeTruncated(count int64) {
	if count <= 0 {
		return
	}
	compactorTruncatedTotal.Add(float64(count))
}

func setLeaseMetric(tenantID string, held bool) {
	value := 0.0
	if held {
		value = 1
	}
	compactorLeaseHeld.WithLabelValues(tenantID).Set(value)
}
