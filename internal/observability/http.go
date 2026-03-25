// Package observability provides Prometheus metrics and HTTP access logging for the API.
package observability

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequests = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total HTTP requests by method, path pattern, and status code.",
		},
		[]string{"method", "path", "code"},
	)
	httpDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)
)

// MetricsHandler serves Prometheus scrape data at GET /metrics.
func MetricsHandler() http.Handler {
	return promhttp.Handler()
}

// HTTPMiddleware records request counts, latency (histogram), and structured access logs.
func HTTPMiddleware(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		path := r.URL.Path
		if path == "" {
			path = "/"
		}

		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)

		code := strconv.Itoa(rec.status)
		httpRequests.WithLabelValues(r.Method, path, code).Inc()
		httpDuration.WithLabelValues(r.Method, path).Observe(time.Since(start).Seconds())

		log.Info("http_request",
			"method", r.Method,
			"path", path,
			"status", rec.status,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
	wrote  bool
}

func (r *statusRecorder) WriteHeader(code int) {
	if !r.wrote {
		r.status = code
		r.wrote = true
	}
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if !r.wrote {
		r.wrote = true
	}
	return r.ResponseWriter.Write(b)
}
