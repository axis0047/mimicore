package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Counter: Total requests received
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mockinggod_http_requests_total",
			Help: "Total number of HTTP requests processed",
		},
		[]string{"api", "method", "status"},
	)

	// Histogram: Total Request Duration (Client perspective)
	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "mockinggod_http_request_duration_seconds",
			Help:    "Latency of HTTP requests",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"api", "method"},
	)

	// Histogram: WASM Execution Time (Internal engine perspective)
	WasmDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "mockinggod_wasm_execution_seconds",
			Help:    "Time spent executing user WASM code",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5},
		},
		[]string{"api", "function"},
	)
)

// ResponseWriter wrapper to capture status code
type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (rec *statusRecorder) WriteHeader(code int) {
	rec.statusCode = code
	rec.ResponseWriter.WriteHeader(code)
}

// Middleware function to wrap the Gateway
func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Extract API name from Host (reuse logic or pass via context)
		// For now, simple extraction matching gateway.go logic:
		host := r.Host
		// (Simplified host parsing for metric label)

		rec := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(rec, r)

		duration := time.Since(start).Seconds()
		status := strconv.Itoa(rec.statusCode)

		httpRequestsTotal.WithLabelValues(host, r.Method, status).Inc()
		httpRequestDuration.WithLabelValues(host, r.Method).Observe(duration)
	})
}
