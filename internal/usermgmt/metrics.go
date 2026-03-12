package usermgmt

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// UserUpsertsTotal tracks the total number of ES user upserts by status
	UserUpsertsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "keyline_user_upserts_total",
			Help: "Total number of ES user upserts",
		},
		[]string{"status"}, // "success", "failure"
	)

	// UserUpsertDuration tracks the duration of ES user upsert operations
	UserUpsertDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "keyline_user_upsert_duration_seconds",
			Help:    "Duration of ES user upsert operations",
			Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1.0, 5.0},
		},
		[]string{"cache_status"}, // "hit", "miss"
	)

	// CredCacheHits tracks the total number of credential cache hits
	CredCacheHits = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "keyline_cred_cache_hits_total",
			Help: "Total number of credential cache hits",
		},
	)

	// CredCacheMisses tracks the total number of credential cache misses
	CredCacheMisses = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "keyline_cred_cache_misses_total",
			Help: "Total number of credential cache misses",
		},
	)

	// RoleMappingMatches tracks the total number of role mapping matches by pattern
	RoleMappingMatches = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "keyline_role_mapping_matches_total",
			Help: "Total number of role mapping matches",
		},
		[]string{"pattern"},
	)

	// ESAPICallsTotal tracks the total number of ES API calls by operation and status
	ESAPICallsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "keyline_es_api_calls_total",
			Help: "Total number of ES API calls",
		},
		[]string{"operation", "status"}, // operation: "create_user", "get_user", "delete_user"; status: "success", "failure"
	)
)
