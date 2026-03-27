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
	"github.com/yourusername/keyline/internal/elasticsearch"
	"github.com/yourusername/keyline/internal/server"
	"github.com/yourusername/keyline/internal/usermgmt"
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

	// Log configuration details
	localUsersCount := 0
	if cfg.LocalUsers.Enabled {
		localUsersCount = len(cfg.LocalUsers.Users)
	}
	logger.Info("Configuration loaded",
		slog.String("config_file", configFile),
		slog.Bool("oidc_enabled", cfg.OIDC.Enabled),
		slog.Int("local_users_count", localUsersCount),
		slog.Bool("ldap_enabled", cfg.LDAP.Enabled),
		slog.String("mode", cfg.Server.Mode),
		slog.String("cache_backend", cfg.Cache.Backend),
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
	cacheBackend, err := cache.InitCache(ctx, &cfg.Cache)
	if err != nil {
		logger.Error("Failed to initialize cache", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Initialize user manager (dynamic user management is always enabled)
	var userManager usermgmt.Manager
	logger.Info("Initializing dynamic user management")

	// Initialize ES API client with admin credentials
	esClient, err := elasticsearch.NewClient(elasticsearch.Config{
		URL:                cfg.Elasticsearch.URL,
		AdminUser:          cfg.Elasticsearch.AdminUser,
		AdminPassword:      cfg.Elasticsearch.AdminPassword,
		Timeout:            cfg.Elasticsearch.Timeout,
		InsecureSkipVerify: cfg.Elasticsearch.InsecureSkipVerify,
	})
	if err != nil {
		logger.Error("Failed to initialize Elasticsearch client", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Validate ES connection on startup
	if err := esClient.ValidateConnection(ctx); err != nil {
		logger.Error("Failed to validate Elasticsearch connection",
			slog.String("error", err.Error()),
			slog.String("url", cfg.Elasticsearch.URL),
			slog.String("admin_user", cfg.Elasticsearch.AdminUser),
		)
		logger.Error("Please verify:")
		logger.Error("  1. Elasticsearch is running and accessible")
		logger.Error("  2. Admin credentials are correct")
		logger.Error("  3. Admin user has 'manage_security' privilege")
		os.Exit(1)
	}
	logger.Info("Elasticsearch connection validated",
		slog.String("url", cfg.Elasticsearch.URL),
		slog.String("admin_user", cfg.Elasticsearch.AdminUser),
	)

	// Initialize password generator
	passwordLength := cfg.UserManagement.PasswordLength
	if passwordLength == 0 {
		passwordLength = 32 // Default
	}

	// Initialize credential encryptor
	// Load encryption key from config
	encryptionKey := cfg.Cache.EncryptionKey
	if encryptionKey == "" {
		logger.Error("Encryption key is required (must be 32 bytes for AES-256)")
		logger.Error("Please set cache.encryption_key in configuration")
		os.Exit(1)
	}

	// Validate key is 32 bytes
	keyBytes := []byte(encryptionKey)
	if len(keyBytes) != 32 {
		logger.Error("Encryption key must be exactly 32 bytes for AES-256",
			slog.Int("actual_length", len(keyBytes)),
			slog.Int("required_length", 32),
		)
		logger.Error("Please provide a 32-byte encryption key in configuration")
		os.Exit(1)
	}

	// Initialize user manager with encryptor
	userManager = usermgmt.NewManager(
		esClient,
		cacheBackend,
		cfg,
	)
	if err != nil {
		logger.Error("Failed to initialize user manager", slog.String("error", err.Error()))
		os.Exit(1)
	}

	credentialTTL := cfg.Cache.CredentialTTL
	if credentialTTL == 0 {
		credentialTTL = 1 * time.Hour // Default
	}
	logger.Info("User manager initialized",
		slog.Duration("credential_ttl", credentialTTL),
	)

	// Add startup logging for user management status
	logger.Info("Dynamic user management ready",
		slog.String("cache_backend", cfg.Cache.Backend),
		slog.Duration("credential_ttl", credentialTTL),
		slog.Int("password_length", passwordLength),
		slog.Int("role_mappings", len(cfg.RoleMappings)),
		slog.Int("default_roles", len(cfg.DefaultESRoles)),
	)

	// Initialize OIDC provider if enabled
	var oidcProvider *auth.OIDCProvider
	if cfg.OIDC.Enabled {
		oidcProvider, err = auth.NewOIDCProvider(&cfg.OIDC, cfg)
		if err != nil {
			logger.Error("Failed to initialize OIDC provider", slog.String("error", err.Error()))
			os.Exit(1)
		}
		logger.Info("OIDC provider initialized",
			slog.String("issuer", cfg.OIDC.IssuerURL),
			slog.String("client_id", cfg.OIDC.ClientID),
		)
	}

	// 14.7: Create and start server with user manager
	srv, err := server.New(cfg, version, cacheBackend, oidcProvider, userManager)
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
