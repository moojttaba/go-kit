package httpx

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	reqTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total HTTP requests by service, method and status code.",
	}, []string{"service", "method", "status"})

	reqDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request latency in seconds by service, method and route.",
		Buckets: prometheus.DefBuckets,
	}, []string{"service", "method", "route"})
)

func metricsHandler() http.Handler { return promhttp.Handler() }

// metricsMiddleware records a counter + latency histogram per request, labelled
// by the chi route pattern (not the raw path) to keep cardinality bounded.
func metricsMiddleware(service string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			start := time.Now()
			next.ServeHTTP(ww, r)

			route := chi.RouteContext(r.Context()).RoutePattern()
			if route == "" {
				route = "unmatched"
			}
			reqTotal.WithLabelValues(service, r.Method, strconv.Itoa(ww.Status())).Inc()
			reqDuration.WithLabelValues(service, r.Method, route).Observe(time.Since(start).Seconds())
		})
	}
}
