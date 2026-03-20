package telemetry

import (
	"context"
	"os"
)

// Span is a minimal no-op unless OTEL is wired later.
func Span(ctx context.Context, name string) (context.Context, func()) {
	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") == "" {
		return ctx, func() {}
	}
	return ctx, func() {}
}
