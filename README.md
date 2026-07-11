# go-kit

Shared Go building blocks for the **new-code** microservices
(`moojttaba/svc-auth`, `moojttaba/svc-blog`). Public so `go get` works in CI
without credentials — it contains **generic infrastructure only, no secrets and
no business logic**.

```bash
go get github.com/moojttaba/go-kit@latest
```

## Packages
| Package | What it gives you |
|---------|-------------------|
| `httpx` | chi router with `/healthz` + `/readyz`, request-ID, structured slog request logging, panic→`problem+json` recovery, Prometheus `/metrics`, and `WriteJSON`/`ReadJSON`/`Error` helpers |
| `config` | dependency-free `Getenv` / `GetenvInt` / `GetenvDuration` env readers |
| `store` | pgx pool `Open` with startup retry/backoff + forward-only `Migrate` driven by an embedded `fs.FS` |
| `obs` | optional OpenTelemetry tracing (`SetupTracing`) + `Handler` HTTP instrumentation — a no-op unless `OTEL_EXPORTER_OTLP_ENDPOINT` is set |

## Typical wiring (in a service `main.go`)
```go
//go:embed migrations/*.sql
var migrationsFS embed.FS

db, _ := store.Open(ctx, cfg.DatabaseURL)
sub, _ := fs.Sub(migrationsFS, "migrations")
_ = store.Migrate(ctx, db, sub)

shutdown, _ := obs.SetupTracing(ctx, cfg.ServiceName)
defer shutdown(context.Background())

handler := obs.Handler(
    httpx.NewRouter(cfg.ServiceName, db, users.NewHandler(repo)),
    cfg.ServiceName,
)
```

## Errors — RFC 7807
Every failure is a `application/problem+json` body:
```json
{ "type": "about:blank", "title": "Bad Request", "status": 400, "detail": "email is required" }
```
Emit one with `httpx.Error(w, http.StatusBadRequest, "email is required")`.

## Metrics
`GET /metrics` exposes Prometheus counters/histograms:
`http_requests_total{service,method,status}` and
`http_request_duration_seconds{service,method,route}` (route = chi pattern, so
cardinality stays bounded).

## Versioning
Tagged `vMAJOR.MINOR.PATCH`. Services pin an exact version in `go.mod`;
Dependabot proposes bumps. Breaking changes get a new major.

---
Part of the [`moojttaba/new-code`](https://github.com/moojttaba/new-code)
workspace — see its `RULES.md` and `docs/` vault.
