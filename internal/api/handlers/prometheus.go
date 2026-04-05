package handlers

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// httpRequestsTotal tracks the total number of HTTP requests
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	// httpRequestDuration tracks HTTP request duration
	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_milliseconds",
			Help:    "HTTP request duration in milliseconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	// dbConnectionsActive tracks active database connections
	dbConnectionsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_connections_active",
			Help: "Number of active database connections",
		},
	)
)

// RecordRequest records an HTTP request with method, path, and status
func RecordRequest(method, path string, status int) {
	httpRequestsTotal.WithLabelValues(method, path, http.StatusText(status)).Inc()
}

// RecordRequestDuration records HTTP request duration
func RecordRequestDuration(method, path string, duration float64) {
	httpRequestDuration.WithLabelValues(method, path).Observe(duration)
}

// SetDBConnections sets the active database connection count
func SetDBConnections(count float64) {
	dbConnectionsActive.Set(count)
}

// PrometheusHandler returns the Prometheus metrics HTTP handler
func PrometheusHandler() http.Handler {
	return promhttp.Handler()
}
