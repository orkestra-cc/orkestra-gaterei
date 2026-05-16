package utils

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// SetupLogger creates and configures the application logger based on environment variables.
// LOG_LEVEL controls verbosity (debug, info, warn, error). Default: info.
// LOG_LEVEL_<MODULE> (e.g. LOG_LEVEL_RAG=debug) overrides the level for a
// single module without changing the global threshold (ADR-0005 §1.4).
// ENV controls format: JSON for production/staging, text for development.
func SetupLogger() *slog.Logger {
	level := slog.LevelInfo
	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		switch strings.ToLower(logLevel) {
		case "debug":
			level = slog.LevelDebug
		case "info":
			level = slog.LevelInfo
		case "warn", "warning":
			level = slog.LevelWarn
		case "error":
			level = slog.LevelError
		}
	}

	// The base handler is forced to LevelDebug so it never gates a
	// record on its own. The actual gating happens in
	// PerModuleLevelHandler.Enabled, which has the module-aware
	// thresholds. If the base handler kept its own Level filter, a
	// per-module LOG_LEVEL_RAG=debug override would silently be lost
	// at the JSON encoder when the global LOG_LEVEL is info.
	opts := slog.HandlerOptions{
		Level: slog.LevelDebug,
	}

	var handler slog.Handler
	env := os.Getenv("ENV")

	if env == "production" || env == "staging" {
		handler = slog.NewJSONHandler(os.Stdout, &opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, &opts)
	}

	// ADR-0005 §1.4 — per-module level overrides. Sits between the base
	// formatter and the trace handler so it can intercept "module"
	// attributes stamped by the module registry's per-module
	// .With(...) before they reach the base.
	handler = NewPerModuleLevelHandler(handler, level)

	// ADR-0005 §1.1 — wrap with trace correlation so every log line
	// stamped via *Context variants carries trace_id / span_id.
	handler = NewTraceContextHandler(handler)

	logger := slog.New(handler)

	return logger.With(
		slog.String("service", "orkestra-backend"),
		slog.String("version", "1.0.0"),
		slog.String("environment", env),
	)
}

// PrintDevelopmentWarning prints a prominent warning when running in non-production mode.
func PrintDevelopmentWarning(environment string) {
	isStaging := environment == "staging"

	var hstsLine string
	if isStaging {
		hstsLine = "║   • HSTS header is ENABLED (production-like security)                        ║"
	} else {
		hstsLine = "║   • HSTS header is disabled                                                   ║"
	}

	warning := `
╔═══════════════════════════════════════════════════════════════════════════════╗
║                                                                               ║
║   ██████╗ ███████╗██╗   ██╗    ███╗   ███╗ ██████╗ ██████╗ ███████╗          ║
║   ██╔══██╗██╔════╝██║   ██║    ████╗ ████║██╔═══██╗██╔══██╗██╔════╝          ║
║   ██║  ██║█████╗  ██║   ██║    ██╔████╔██║██║   ██║██║  ██║█████╗            ║
║   ██║  ██║██╔══╝  ╚██╗ ██╔╝    ██║╚██╔╝██║██║   ██║██║  ██║██╔══╝            ║
║   ██████╔╝███████╗ ╚████╔╝     ██║ ╚═╝ ██║╚██████╔╝██████╔╝███████╗          ║
║   ╚═════╝ ╚══════╝  ╚═══╝      ╚═╝     ╚═╝ ╚═════╝ ╚═════╝ ╚══════╝          ║
║                                                                               ║
║   RUNNING IN DEVELOPMENT MODE - NOT FOR PRODUCTION USE                        ║
║                                                                               ║
║   Environment: %-12s                                                       ║
║                                                                               ║
║   The following security features are RELAXED:                                ║
║   • Dev token endpoints are enabled (/dev/token)                              ║
║   • Verbose error messages are shown                                          ║
║   • Localhost OAuth redirects are allowed                                     ║
%s
║                                                                               ║
║   DO NOT deploy to production with these settings!                            ║
║   Set APP_ENV=production for production deployments.                          ║
║                                                                               ║
╚═══════════════════════════════════════════════════════════════════════════════╝
`
	fmt.Printf(warning, environment, hstsLine)
}
