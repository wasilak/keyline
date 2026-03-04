package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/wasilak/loggergo"
	"github.com/wasilak/otelgo/tracing"
	"github.com/yourusername/keyline/internal/auth"
	"github.com/yourusername/keyline/internal/cache"
	"github.com/yourusername/keyline/internal/config"
	"github.com/yourusername/keyline/internal/server"
)

const version = "0.1.0"

func main() {
	// Parse command-line flags
	validateOnly := false
	configFile := ""

	for i, arg := range os.Args[1:] {
		switch arg {
		case "--validate-config":
			validateOnly = true
		case "--config":
			if i+1 < len(os.Args[1:]) {
				configFile = os.Args[i+2]
			}
		case "--version":
			fmt.Printf("Keyline v%s\n", version)
			os.Exit(0)
		case "--help", "-h":
			printHelp()
			os.Exit(0)
		}
	}

	// Load configuration
	cfg, err := config.Load(configFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Validate configuration
	if err := config.Validate(cfg); err != nil {
		log.Fatalf("Configuration validation failed: %v", err)
	}

	// If validate-only mode, exit successfully
	if validateOnly {
		fmt.Println("Configuration valid")
		os.Exit(0)
	}

	// Initialize global logger with loggergo
	ctx := context.Background()

	logConfig := loggergo.Config{
		Level: parseLogLevel(cfg.Observability.LogLevel),
	}

	// Set format based on config - use loggergo types
	if cfg.Observability.LogFormat == "json" {
		logConfig.Format = loggergo.Types.LogFormatJSON
	} else {
		logConfig.Format = loggergo.Types.LogFormatText
	}

	ctx, logger, err := loggergo.Init(ctx, logConfig)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	// Set global slog logger for use throughout application
	slog.SetDefault(logger)

	logger.Info("Keyline starting",
		slog.String("version", version),
		slog.String("log_level", cfg.Observability.LogLevel),
		slog.String("log_format", cfg.Observability.LogFormat),
	)

	// Initialize OpenTelemetry tracing with otelgo
	var traceProvider interface{ Shutdown(context.Context) error }
	if cfg.Observability.OTelEnabled {
		// Set environment variables for otelgo
		os.Setenv("OTEL_SERVICE_NAME", cfg.Observability.OTelServiceName)
		os.Setenv("OTEL_SERVICE_VERSION", cfg.Observability.OTelServiceVersion)
		os.Setenv("OTEL_DEPLOYMENT_ENVIRONMENT", cfg.Observability.OTelEnvironment)
		os.Setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", cfg.Observability.OTelEndpoint)
		os.Setenv("OTEL_EXPORTER_OTLP_INSECURE", "true") // TODO: Make configurable

		_, traceProvider, err = tracing.NewBuilder().
			WithTLSInsecure().
			Build(ctx)
		if err != nil {
			logger.Warn("Failed to initialize OpenTelemetry tracing, continuing without tracing",
				slog.String("error", err.Error()),
			)
		} else {
			logger.Info("OpenTelemetry tracing initialized",
				slog.String("endpoint", cfg.Observability.OTelEndpoint),
				slog.String("service_name", cfg.Observability.OTelServiceName),
				slog.String("service_version", cfg.Observability.OTelServiceVersion),
				slog.String("environment", cfg.Observability.OTelEnvironment),
			)
			defer func() {
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := traceProvider.Shutdown(shutdownCtx); err != nil {
					logger.Error("Failed to shutdown OpenTelemetry tracing", slog.String("error", err.Error()))
				}
			}()
		}
	}

	// Initialize cache backend
	cache, err := cache.InitCache(ctx, &cfg.Cache)
	if err != nil {
		logger.Error("Failed to initialize cache", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Initialize OIDC provider if enabled
	var oidcProvider *auth.OIDCProvider
	if cfg.OIDC.Enabled {
		oidcProvider, err = auth.NewOIDCProvider(&cfg.OIDC)
		if err != nil {
			logger.Error("Failed to initialize OIDC provider", slog.String("error", err.Error()))
			os.Exit(1)
		}
		logger.Info("OIDC provider initialized",
			slog.String("issuer", cfg.OIDC.IssuerURL),
			slog.String("client_id", cfg.OIDC.ClientID),
		)
	}

	// Create and start server
	srv, err := server.New(cfg, version, cache, oidcProvider)
	if err != nil {
		logger.Error("Failed to create server", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := srv.Start(); err != nil {
			errChan <- err
		}
	}()

	// Wait for interrupt signal or error
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	select {
	case err := <-errChan:
		logger.Error("Server error", slog.String("error", err.Error()))
		os.Exit(1)
	case sig := <-sigChan:
		logger.Info("Received signal, shutting down gracefully", slog.String("signal", sig.String()))
	}

	// Graceful shutdown with 30-second timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server shutdown failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

	logger.Info("Server stopped")
}

func parseLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func printHelp() {
	fmt.Println("Keyline - Authentication Proxy for Elasticsearch")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  keyline [flags]")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  --config FILE          Path to configuration file (default: config.yaml)")
	fmt.Println("  --validate-config      Validate configuration and exit")
	fmt.Println("  --version              Print version and exit")
	fmt.Println("  --help, -h             Print this help message")
	fmt.Println()
	fmt.Println("Environment Variables:")
	fmt.Println("  CONFIG_FILE            Path to configuration file")
	fmt.Println()
}
