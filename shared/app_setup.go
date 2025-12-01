package shared

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"go.uber.org/zap"
)

// AppConfig holds common application configuration
type AppConfig struct {
	PortalDomain string
	LogLevel     string
	Logger       *zap.Logger
	StdLogger    *log.Logger
}

// PortalURLs holds the constructed portal URLs
type PortalURLs struct {
	AccountBaseURL string
	LBRYBaseURL    string
}

// BuildSubdomainURL creates a full URL for a subdomain of the given base domain
func BuildSubdomainURL(baseDomain, subdomain string) string {
	// Ensure baseDomain doesn't have protocol prefix
	baseDomain = strings.TrimPrefix(baseDomain, "http://")
	baseDomain = strings.TrimPrefix(baseDomain, "https://")

	// Remove trailing slash
	baseDomain = strings.TrimSuffix(baseDomain, "/")

	return fmt.Sprintf("https://%s.%s", subdomain, baseDomain)
}

// BuildPortalURLs constructs the account and LBRY portal URLs from a base domain
func BuildPortalURLs(portalDomain string) PortalURLs {
	return PortalURLs{
		AccountBaseURL: BuildSubdomainURL(portalDomain, "account"),
		LBRYBaseURL:    BuildSubdomainURL(portalDomain, "lbry"),
	}
}

// ParseCommonFlags parses common command-line flags used across demos
func ParseCommonFlags() (portalDomain *string, logLevel *string) {
	// Get values from environment variables or use defaults
	portalDefault := os.Getenv("PORTAL")
	if portalDefault == "" {
		portalDefault = "pinner.xyz"
	}

	logLevelDefault := os.Getenv("LOG_LEVEL")
	if logLevelDefault == "" {
		logLevelDefault = "info"
	}

	portalDomain = flag.String("portal", portalDefault, "Base portal domain (e.g., pinner.xyz)")
	logLevel = flag.String("log-level", logLevelDefault, "Log level (debug, info, warn, error)")
	return portalDomain, logLevel
}

// ParseAccountFlags parses account-related command-line flags used across demos
func ParseAccountFlags() (portalDomain *string, logLevel *string) {
	return ParseCommonFlags()
}

// SetupApp initializes common application components (logging, URLs, etc.)
func SetupApp(portalDomain, logLevelStr, appName string) (*AppConfig, error) {
	// Parse log level and create zap logger
	parsedLogLevel := ParseLogLevel(logLevelStr)
	zapLogger, err := CreateZapLogger(parsedLogLevel)
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	// Create a standard logger for backward compatibility
	stdLogger := log.New(os.Stdout, fmt.Sprintf("[%s] ", strings.ToUpper(appName)), log.LstdFlags)

	config := &AppConfig{
		PortalDomain: portalDomain,
		LogLevel:     logLevelStr,
		Logger:       zapLogger,
		StdLogger:    stdLogger,
	}

	zapLogger.Info("Starting demo",
		zap.String("app_name", appName),
		zap.String("portal_domain", portalDomain),
		zap.String("log_level", parsedLogLevel.String()))

	return config, nil
}

// CleanupApp performs cleanup of application resources
func CleanupApp(config *AppConfig) {
	if config != nil && config.Logger != nil {
		config.Logger.Sync()
	}
}
