package httpx

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// RequestLogger logs one structured line per request via slog: request id,
// method, path, status, bytes and duration.
func RequestLogger(service string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			start := time.Now()
			defer func() {
				slog.LogAttrs(
					r.Context(), slog.LevelInfo, "http_request",
					slog.String("service", service),
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
					slog.Int("status", ww.Status()),
					slog.Int("bytes", ww.BytesWritten()),
					slog.Duration("duration", time.Since(start)),
					slog.String("request_id", middleware.GetReqID(r.Context())),
				)
			}()
			next.ServeHTTP(ww, r)
		})
	}
}

// Recoverer converts a panic into a 500 problem+json response (and an error
// log) instead of dropping the connection.
func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}
			// Let a deliberate handler abort propagate to the server.
			if err, ok := rec.(error); ok && errors.Is(err, http.ErrAbortHandler) {
				panic(rec)
			}
			slog.LogAttrs(
				r.Context(), slog.LevelError, "panic",
				slog.Any("recover", rec),
				slog.String("request_id", middleware.GetReqID(r.Context())),
			)
			Error(w, http.StatusInternalServerError, "internal server error")
		}()
		next.ServeHTTP(w, r)
	})
}
