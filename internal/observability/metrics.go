package observability

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// AuthAttempts tracks authentication attempts by method and result
	AuthAttempts = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "auth_attempts_total",
			Help: "Total number of authentication attempts",
		},
		[]string{"method", "result"},
	)

	// AuthDuration tracks authentication request duration by method
	AuthDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "auth_request_duration_seconds",
			Help:    "Authentication request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method"},
	)

	// ActiveSessions tracks the number of active sessions
	ActiveSessions = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "active_sessions",
			Help: "Number of active sessions",
		},
	)

	// SessionOperations tracks session operations by operation type
	SessionOperations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "session_operations_total",
			Help: "Total number of session operations",
		},
		[]string{"operation"},
	)

	// OIDCProviderRequests tracks OIDC provider requests by endpoint and result
	OIDCProviderRequests = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "oidc_provider_requests_total",
			Help: "Total number of OIDC provider requests",
		},
		[]string{"endpoint", "result"},
	)

	// UpstreamProxyDuration tracks upstream proxy request duration
	UpstreamProxyDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "upstream_proxy_duration_seconds",
			Help:    "Upstream proxy request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
	)

	// ConcurrentRequests tracks the current number of concurrent requests
	ConcurrentRequests = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "concurrent_requests",
			Help: "Current number of concurrent requests",
		},
	)

	// Errors tracks errors by error type
	Errors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "errors_total",
			Help: "Total number of errors",
		},
		[]string{"error_type"},
	)
)
