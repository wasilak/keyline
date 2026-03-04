package observability

import (
	"log/slog"
	"os"
	"strings"
)

// NewLogger creates a new structured logger
func NewLogger(level, format string) *slog.Logger {
	// Parse log level
	var logLevel slog.Level
	switch strings.ToLower(level) {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn", "warning":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	// Create handler based on format
	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level: logLevel,
	}

	switch strings.ToLower(format) {
	case "json":
		handler = slog.NewJSONHandler(os.Stdout, opts)
	default:
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}
