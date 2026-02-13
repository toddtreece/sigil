package modelcards

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	modelCardRefreshRunsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "sigil_model_cards_refresh_runs_total",
		Help: "Total number of model-card refresh runs partitioned by source, mode, and status.",
	}, []string{"source", "mode", "status"})
	modelCardRefreshDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "sigil_model_cards_refresh_duration_seconds",
		Help:    "Model-card refresh run latency in seconds partitioned by source and mode.",
		Buckets: prometheus.DefBuckets,
	}, []string{"source", "mode"})
	modelCardCatalogRows = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "sigil_model_cards_catalog_rows",
		Help: "Current number of model-card rows in the active in-memory catalog per source.",
	}, []string{"source"})
	modelCardCatalogAgeSeconds = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "sigil_model_cards_catalog_age_seconds",
		Help: "Seconds since the catalog was last refreshed for each source; -1 when unknown.",
	}, []string{"source"})
	modelCardReadPathTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "sigil_model_cards_read_path_total",
		Help: "Total number of model-card read path selections partitioned by operation and source path.",
	}, []string{"operation", "source_path"})
)

func observeRefreshMetrics(run RefreshRun) {
	mode := run.RunMode
	if mode == "" {
		mode = "primary"
	}
	status := run.Status
	if status == "" {
		status = "unknown"
	}

	modelCardRefreshRunsTotal.WithLabelValues(run.Source, mode, status).Inc()

	if !run.StartedAt.IsZero() && !run.FinishedAt.IsZero() {
		duration := run.FinishedAt.Sub(run.StartedAt).Seconds()
		if duration < 0 {
			duration = 0
		}
		modelCardRefreshDuration.WithLabelValues(run.Source, mode).Observe(duration)
	}
}

func observeReadPath(operation string, sourcePath string) {
	if operation == "" {
		operation = "unknown"
	}
	if sourcePath == "" {
		sourcePath = "unknown"
	}
	modelCardReadPathTotal.WithLabelValues(operation, sourcePath).Inc()
}

func setCatalogState(source string, rows int64, latestRefreshedAt *time.Time, now time.Time) {
	if source == "" {
		source = SourceOpenRouter
	}

	if rows < 0 {
		rows = 0
	}
	modelCardCatalogRows.WithLabelValues(source).Set(float64(rows))

	ageSeconds := -1.0
	if latestRefreshedAt != nil {
		ageSeconds = now.Sub(*latestRefreshedAt).Seconds()
		if ageSeconds < 0 {
			ageSeconds = 0
		}
	}
	modelCardCatalogAgeSeconds.WithLabelValues(source).Set(ageSeconds)
}
