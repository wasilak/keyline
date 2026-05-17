package auth

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// LDAPBindAttempts tracks the total number of LDAP bind attempts by result.
	LDAPBindAttempts = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "keyline_ldap_bind_attempts_total",
			Help: "Total number of LDAP bind attempts",
		},
		[]string{"result"}, // "success", "failure"
	)

	// LDAPSearchDuration tracks the duration of LDAP user search operations.
	LDAPSearchDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "keyline_ldap_search_duration_seconds",
			Help:    "Duration of LDAP user search operations",
			Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1.0, 5.0},
		},
	)

	// LDAPConnectionErrors tracks the total number of LDAP connection errors.
	LDAPConnectionErrors = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "keyline_ldap_connection_errors_total",
			Help: "Total number of LDAP connection errors",
		},
	)
)
