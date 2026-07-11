// Package httpx is the shared HTTP foundation for the services: a chi router
// with health probes, structured request logging, panic recovery, Prometheus
// metrics, RFC 7807 problem+json errors, and JSON read/write helpers.
package httpx

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Pinger is the readiness dependency the /readyz probe checks. *pgxpool.Pool
// satisfies it; tests can pass nil for a health-only router.
type Pinger interface {
	Ping(ctx context.Context) error
}

// Registrar is implemented by feature modules that mount their own routes.
type Registrar interface {
	Routes(r chi.Router)
}

// Options tune NewRouter. The zero value is production-sensible.
type Options struct {
	// Metrics, when true, mounts GET /metrics (Prometheus) and records a
	// request counter + latency histogram for every route.
	Metrics bool
	// RequestTimeout bounds each request; 0 means 30s.
	RequestTimeout time.Duration
}

// NewRouter builds the base router (metrics on) with shared middleware and
// health probes, then mounts every feature module. db may be nil (health-only).
func NewRouter(service string, db Pinger, mods ...Registrar) http.Handler {
	return NewRouterWithOptions(service, db, Options{Metrics: true}, mods...)
}

// NewRouterWithOptions is NewRouter with explicit Options.
func NewRouterWithOptions(service string, db Pinger, opts Options, mods ...Registrar) http.Handler {
	timeout := opts.RequestTimeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	// Note: chi's RealIP is intentionally NOT used — it trusts client-supplied
	// X-Forwarded-For/X-Real-IP headers and is vulnerable to IP spoofing.
	r.Use(RequestLogger(service))
	r.Use(Recoverer)
	r.Use(middleware.Timeout(timeout))

	if opts.Metrics {
		r.Use(metricsMiddleware(service))
		r.Handle("/metrics", metricsHandler())
	}

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		WriteJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": service})
	})
	r.Get("/readyz", func(w http.ResponseWriter, req *http.Request) {
		if db == nil {
			WriteJSON(w, http.StatusOK, map[string]string{"status": "ready"})
			return
		}
		ctx, cancel := context.WithTimeout(req.Context(), 2*time.Second)
		defer cancel()
		if err := db.Ping(ctx); err != nil {
			WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "down"})
			return
		}
		WriteJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})

	for _, m := range mods {
		m.Routes(r)
	}
	return r
}

// WriteJSON encodes v as an application/json response with the given status.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// ReadJSON decodes the request body into v, capping the body at 1 MiB and
// rejecting unknown fields.
func ReadJSON(r *http.Request, v any) error {
	dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}
