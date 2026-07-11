package httpx

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

// panicMod mounts a route that panics, to exercise Recoverer.
type panicMod struct{}

func (panicMod) Routes(r chi.Router) {
	r.Get("/boom", func(http.ResponseWriter, *http.Request) { panic("kaboom") })
}

func newTestRouter() http.Handler {
	// nil db -> health-only; still exercises middleware + metrics.
	return NewRouter("svc-test", nil, panicMod{})
}

func TestHealthz(t *testing.T) {
	rec := httptest.NewRecorder()
	newTestRouter().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("healthz: want 200, got %d", rec.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["service"] != "svc-test" {
		t.Fatalf("want service svc-test, got %q", body["service"])
	}
}

func TestReadyzNilDB(t *testing.T) {
	rec := httptest.NewRecorder()
	newTestRouter().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("readyz(nil db): want 200, got %d", rec.Code)
	}
}

func TestMetricsEndpoint(t *testing.T) {
	rec := httptest.NewRecorder()
	newTestRouter().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("metrics: want 200, got %d", rec.Code)
	}
}

func TestRecovererReturnsProblemJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	newTestRouter().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/boom", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("panic route: want 500, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Fatalf("want problem+json, got %q", ct)
	}
	var p Problem
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatalf("decode problem: %v", err)
	}
	if p.Status != http.StatusInternalServerError {
		t.Fatalf("problem.status: want 500, got %d", p.Status)
	}
}

func TestErrorWritesProblem(t *testing.T) {
	rec := httptest.NewRecorder()
	Error(rec, http.StatusBadRequest, "bad thing")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rec.Code)
	}
	var p Problem
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if p.Title != "Bad Request" || p.Detail != "bad thing" {
		t.Fatalf("unexpected problem: %+v", p)
	}
}

func TestReadJSONRejectsUnknownFields(t *testing.T) {
	body := bytes.NewBufferString(`{"nope": 1}`)
	r := httptest.NewRequest(http.MethodPost, "/", body)
	var dst struct {
		Known string `json:"known"`
	}
	if err := ReadJSON(r, &dst); err == nil {
		t.Fatal("want error on unknown field, got nil")
	}
}

func TestReadJSONOK(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"known":"x"}`))
	var dst struct {
		Known string `json:"known"`
	}
	if err := ReadJSON(r, &dst); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if dst.Known != "x" {
		t.Fatalf("want x, got %q", dst.Known)
	}
}

func TestNewServerHasHardenedTimeouts(t *testing.T) {
	srv := NewServer(":8080", newTestRouter())
	if srv.Addr != ":8080" {
		t.Fatalf("addr = %q, want :8080", srv.Addr)
	}
	if srv.ReadTimeout == 0 || srv.ReadHeaderTimeout == 0 ||
		srv.WriteTimeout == 0 || srv.IdleTimeout == 0 || srv.MaxHeaderBytes == 0 {
		t.Fatalf("expected all timeouts + MaxHeaderBytes set, got %+v", srv)
	}
}
