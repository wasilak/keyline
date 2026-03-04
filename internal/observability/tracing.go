package observability

import (
	"context"
	"log/slog"
	"os"

	"github.com/wasilak/otelgo/tracing"
	"github.com/yourusername/keyline/internal/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// TracerShutdown is a function that shuts down the tracer
type TracerShutdown func(context.Context) error

// InitTracer initializes OpenTelemetry tracing with OTLP exporter
// Returns a shutdown function that should be called on application exit
// If initialization fails, returns a no-op tracer and logs a warning
func InitTracer(ctx context.Context, cfg *config.ObservabilityConfig) (trace.Tracer, TracerShutdown, error) {
	if !cfg.OTelEnabled {
		slog.InfoContext(ctx, "OpenTelemetry tracing disabled")
		// Return no-op tracer
		noopProvider := noop.NewTracerProvider()
		return noopProvider.Tracer("keyline"), func(context.Context) error { return nil }, nil
	}

	// otelgo uses environment variables for configuration
	// Set them before calling Build (these will be read by otelgo)
	// Note: In production, these should be set externally, but we set them here for convenience
	if cfg.OTelServiceName != "" {
		// Only set if not already set to allow external override
		if os.Getenv("OTEL_SERVICE_NAME") == "" {
			os.Setenv("OTEL_SERVICE_NAME", cfg.OTelServiceName)
		}
	}
	if cfg.OTelServiceVersion != "" {
		if os.Getenv("OTEL_SERVICE_VERSION") == "" {
			os.Setenv("OTEL_SERVICE_VERSION", cfg.OTelServiceVersion)
		}
	}
	if cfg.OTelEnvironment != "" {
		if os.Getenv("OTEL_DEPLOYMENT_ENVIRONMENT") == "" {
			os.Setenv("OTEL_DEPLOYMENT_ENVIRONMENT", cfg.OTelEnvironment)
		}
	}
	if cfg.OTelEndpoint != "" {
		if os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT") == "" {
			os.Setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", cfg.OTelEndpoint)
		}
	}

	// Use otelgo to initialize tracing
	// otelgo uses environment variables for configuration
	// Set them before calling Build
	_, traceProvider, err := tracing.NewBuilder().
		WithTLSInsecure(). // TODO: Make configurable based on endpoint scheme
		Build(ctx)

	if err != nil {
		slog.WarnContext(ctx, "Failed to initialize OpenTelemetry tracing, using no-op tracer",
			slog.String("error", err.Error()),
		)
		// Return no-op tracer on failure
		noopProvider := noop.NewTracerProvider()
		return noopProvider.Tracer("keyline"), func(context.Context) error { return nil }, nil
	}

	// Configure W3C Trace Context propagation
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	slog.InfoContext(ctx, "OpenTelemetry tracing initialized",
		slog.String("endpoint", cfg.OTelEndpoint),
		slog.String("service_name", cfg.OTelServiceName),
		slog.String("service_version", cfg.OTelServiceVersion),
		slog.String("environment", cfg.OTelEnvironment),
		slog.Float64("trace_ratio", cfg.OTelTraceRatio),
	)

	// Get tracer from the global provider
	tracer := otel.Tracer("keyline")

	// Return tracer and shutdown function
	shutdown := func(ctx context.Context) error {
		if traceProvider != nil {
			return traceProvider.Shutdown(ctx)
		}
		return nil
	}

	return tracer, shutdown, nil
}
