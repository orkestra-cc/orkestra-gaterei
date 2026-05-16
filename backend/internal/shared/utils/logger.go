package utils

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync/atomic"
)

// globalPerModule remembers the PerModuleLevelHandler instance built
// by the most recent SetupLogger so SwapLevelResolver can hot-swap
// its resolver from main.go after the logging core module wires up.
// nil before first SetupLogger call. Atomic pointer so concurrent
// SetupLogger + SwapLevelResolver calls (rare, but possible during
// tests) are safe.
var globalPerModule atomic.Pointer[PerModuleLevelHandler]

// SwapLevelResolver replaces the resolver behind the most recently
// constructed PerModuleLevelHandler. ADR-0005 Phase F — called from
// main.go once the logging core module's LogLevelService is
// available, replacing the boot-time StaticLevelResolver with the
// DB-backed live resolver. Safe to call multiple times; no-op when
// no handler has been built yet (e.g. tests that bypass SetupLogger).
func SwapLevelResolver(r LevelResolver) {
	if h := globalPerModule.Load(); h != nil && r != nil {
		h.SetResolver(r)
	}
}

// SetupLogger creates and configures the application logger based on environment variables.
// LOG_LEVEL controls verbosity (debug, info, warn, error). Default: info.
// LOG_LEVEL_<MODULE> (e.g. LOG_LEVEL_RAG=debug) overrides the level for a
// single module without changing the global threshold (ADR-0005 §1.4).
// ENV controls format: JSON for production/staging, text for development.
//
// extras is the ADR-0005 Phase E hook: additional slog.Handler fanout
// targets (today: the OTLP-bridge handler from telemetry.InitLogs).
// Pass none for the stdout-only path used during early boot before
// telemetry is initialized; pass the OTLP handler in a second call
// after InitLogs to upgrade. Nil entries are filtered by
// NewFanoutHandler, so callers don't have to guard.
func SetupLogger(extras ...slog.Handler) *slog.Logger {
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

	var stdoutHandler slog.Handler
	env := os.Getenv("ENV")

	if env == "production" || env == "staging" {
		stdoutHandler = slog.NewJSONHandler(os.Stdout, &opts)
	} else {
		stdoutHandler = slog.NewTextHandler(os.Stdout, &opts)
	}

	// ADR-0005 Phase E — fan out to stdout + any extra handlers
	// (today: OTLP-bridge from telemetry.InitLogs). When no extras
	// are supplied the FanoutHandler is constructed with a single
	// member — equivalent to just using stdoutHandler directly, at
	// the cost of one extra Handle dispatch per record. Negligible.
	var handler slog.Handler
	if len(extras) > 0 {
		all := append([]slog.Handler{stdoutHandler}, extras...)
		handler = NewFanoutHandler(all...)
	} else {
		handler = stdoutHandler
	}

	// ADR-0005 §1.4 — per-module level overrides. Sits between the base
	// formatter and the trace handler so it can intercept "module"
	// attributes stamped by the module registry's per-module
	// .With(...) before they reach the base.
	//
	// Boot uses StaticLevelResolver (env-driven snapshot); ADR-0005
	// Phase F swaps in core/logging.LogLevelService at runtime via
	// SwapResolver below so admin mutations take effect without a
	// restart. The handler instance does not change — only the
	// resolver behind it.
	resolver := NewStaticLevelResolver(level, loadPerModuleLevels())
	perModule := NewPerModuleLevelHandler(handler, resolver)
	globalPerModule.Store(perModule)
	handler = perModule

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
