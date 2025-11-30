package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/lumeweb/lbry-cli-demo/shared"
	"go.lumeweb.com/liblbry/server"
	"go.uber.org/zap"
)

// Config holds the CLI configuration
type Config struct {
	Port     int
	PeerPort int
	LogLevel string
}

// parseFlags parses command-line flags and returns configuration
func parseFlags() *Config {
	config := &Config{}
	flag.IntVar(&config.Port, "port", 5669, "Port for the reflector server")
	flag.IntVar(&config.PeerPort, "peer-port", 5570, "Port for the peer server")
	flag.StringVar(&config.LogLevel, "log-level", "info", "Log level (debug, info, warn, error)")
	flag.Parse()
	return config
}

// setupLogger creates and returns a logger with proper error handling
func setupLogger(logLevel string) (*zap.Logger, error) {
	parsedLogLevel := shared.ParseLogLevel(logLevel)
	logger, err := shared.CreateZapLogger(parsedLogLevel)
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}
	return logger, nil
}

// createServer creates and starts the reflector server with peer support
func createServer(port int, peerPort int, logger *zap.Logger) (server.Server, error) {
	logger.Info("Creating LBRY server components (reflector + peer)",
		zap.Int("reflector_port", port),
		zap.Int("peer_port", peerPort))

	builder := server.ReflectorOnlyBuilder(port)
	if builder == nil {
		return nil, fmt.Errorf("failed to create reflector builder")
	}

	builder = builder.WithPeer(peerPort)

	srv, err := builder.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build reflector server: %w", err)
	}
	if srv == nil {
		return nil, fmt.Errorf("failed to build reflector server: nil server")
	}

	// Start the server
	logger.Info("Starting LBRY servers...")
	err = srv.Start(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to start reflector server: %w", err)
	}

	logger.Info("LBRY servers started successfully")
	return srv, nil
}

// setupSignalHandling creates a context and signal channel for graceful shutdown
func setupSignalHandling() (context.Context, context.CancelFunc, chan os.Signal) {
	ctx, cancel := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	return ctx, cancel, sigChan
}

// waitForShutdown waits for shutdown signal and cancels the context
func waitForShutdown(ctx context.Context, cancel context.CancelFunc, sigChan chan os.Signal, logger *zap.Logger) {
	go func() {
		sig := <-sigChan
		logger.Info("Received shutdown signal", zap.String("signal", sig.String()))
		cancel()
	}()

	// Wait for context cancellation
	<-ctx.Done()
}

// performGracefulShutdown handles the graceful shutdown process
func performGracefulShutdown(srv server.Server, logger *zap.Logger) {
	logger.Info("Shutting down reflector server...")

	// Stop the server
	err := srv.Stop(context.Background())
	if err != nil {
		logger.Warn("Error during server shutdown", zap.Error(err))
	} else {
		logger.Info("Reflector server shutdown completed successfully")
	}
}

func main() {
	// Parse command-line flags
	config := parseFlags()

	// Setup logger with proper error handling
	logger, err := setupLogger(config.LogLevel)
	if err != nil {
		log.Fatalf("Failed to setup logger: %v", err)
	}

	logger.Info("Starting LBRY reflector and peer servers",
		zap.Int("reflector_port", config.Port),
		zap.Int("peer_port", config.PeerPort),
		zap.String("log_level", config.LogLevel),
	)

	// Setup signal handling for graceful shutdown
	ctx, cancel, sigChan := setupSignalHandling()
	defer cancel()

	// Create and start the reflector server
	srv, err := createServer(config.Port, config.PeerPort, logger)
	if err != nil {
		logger.Fatal("Failed to create server", zap.Error(err))
	}

	logger.Info("LBRY servers started successfully (reflector + peer)")

	// Wait for shutdown signal
	waitForShutdown(ctx, cancel, sigChan, logger)

	// Perform graceful shutdown
	performGracefulShutdown(srv, logger)
}
