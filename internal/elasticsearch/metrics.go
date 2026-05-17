package elasticsearch

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// ESAPICallsTotal tracks the total number of ES API calls by operation and status.
	ESAPICallsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "keyline_es_api_calls_total",
			Help: "Total number of ES API calls",
		},
		[]string{"operation", "status"}, // operation: "create_user", "get_user", "delete_user"; status: "success", "failure"
	)

	// CircuitBreakerState tracks the current circuit breaker state: 0=closed, 1=open, 2=half-open.
	CircuitBreakerState = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "keyline_circuit_breaker_state",
			Help: "Current state of the ES circuit breaker (0=closed, 1=open, 2=half-open)",
		},
	)
)
