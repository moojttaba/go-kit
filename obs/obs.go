// Package obs wires optional OpenTelemetry tracing. Tracing stays a no-op
// unless OTEL_EXPORTER_OTLP_ENDPOINT is set, so local and CI runs need no
// collector. Handler adds server-span instrumentation either way (cheap when
// no exporter is installed).
package obs

import (
	"context"
	"net/http"
	"os"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// SetupTracing installs a global OTLP/HTTP tracer provider when
// OTEL_EXPORTER_OTLP_ENDPOINT is set; otherwise it does nothing. It always
// returns a shutdown func that is safe to call (defer it in main).
func SetupTracing(ctx context.Context, service string) (func(context.Context) error, error) {
	noop := func(context.Context) error { return nil }
	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") == "" {
		return noop, nil
	}

	exp, err := otlptracehttp.New(ctx)
	if err != nil {
		return noop, err
	}
	res, err := resource.Merge(resource.Default(),
		resource.NewSchemaless(attribute.String("service.name", service)))
	if err != nil {
		return noop, err
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	return tp.Shutdown, nil
}

// Handler wraps h with OpenTelemetry HTTP server instrumentation, naming spans
// after the service. With no tracer provider installed the overhead is minimal.
func Handler(h http.Handler, service string) http.Handler {
	return otelhttp.NewHandler(h, service)
}
