package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	// Create and start server
	srv, err := server.New(cfg, version)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
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
		log.Fatalf("Server error: %v", err)
	case sig := <-sigChan:
		log.Printf("Received signal %v, shutting down gracefully...", sig)
	}

	// Graceful shutdown with 30-second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}

	log.Println("Server stopped")
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
