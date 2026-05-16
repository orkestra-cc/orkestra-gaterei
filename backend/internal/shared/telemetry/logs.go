// Package telemetry's logs.go mirrors tracer.go for the OTLP logs
// signal (ADR-0005 Phase E). When OTEL_LOGS_ENABLED=true and
// OTEL_EXPORTER_OTLP_ENDPOINT is set, InitLogs returns an slog.Handler
// that ships every record to an OTLP backend (Tempo+Loki via the
// otel-collector, Grafana Cloud, Honeycomb, Datadog, Axiom — whatever
// speaks OTLP HTTP). The returned handler is meant to live ALONGSIDE
// the stdout JSON/Text handler under shared/utils.FanoutHandler so
// stdout remains the source of truth — losing the collector never
// means losing logs.

package telemetry

import (
	"context"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
)

// LogShutdownFunc graceful-shutdowns the log provider, flushing any
// buffered records. main.go defers it so log loss on SIGTERM is bounded.
type LogShutdownFunc func(ctx context.Context) error

// LogResult bundles the slog.Handler that ships records to OTLP and the
// shutdown function. When OTLP logs are disabled (default), Handler is
// nil — callers fall back to stdout only, no fanout needed.
type LogResult struct {
	Handler  slog.Handler
	Shutdown LogShutdownFunc
}

// InitLogs reads OTEL_LOGS_ENABLED + OTEL_EXPORTER_OTLP_ENDPOINT and,
// when enabled, builds an OTLP HTTP exporter wired to a LoggerProvider
// with a BatchProcessor, then returns an otelslog handler bound to that
// provider. Logger.Info / .Warn / etc. emitted through this handler are
// batched and shipped to the configured backend on a background goroutine.
//
// OTEL_LOGS_ENABLED is the Orkestra-named knob from ADR-0005 §1.5 /
// Phase E so operators can keep tracing on (cheap on the wire) while
// leaving logs off (potentially expensive at vendor backends that bill
// per ingested byte). Both signals share the same OTEL_EXPORTER_OTLP_*
// env vars — there is only one collector to point at.
//
// Unset / "false" / missing endpoint → returns LogResult{Handler: nil,
// Shutdown: noop}. The caller's SetupLogger sees nil and skips the
// fanout — stdout still works, costing zero overhead.
//
// On exporter failure, InitLogs warns and returns the nil-handler shape
// rather than killing boot. Operators get a degraded-but-running
// service; the warn line in the boot log surfaces the misconfig.
func InitLogs(serviceName, environment string, logger *slog.Logger) LogResult {
	if !logsEnabled() {
		return LogResult{Shutdown: noopShutdown}
	}

	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		logger.Warn("telemetry: OTEL_LOGS_ENABLED=true but OTEL_EXPORTER_OTLP_ENDPOINT unset — OTLP logs disabled")
		return LogResult{Shutdown: noopShutdown}
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewSchemaless(
			attribute.String("service.name", serviceName),
			attribute.String("deployment.environment", environment),
		),
	)
	if err != nil {
		logger.Warn("telemetry: building log resource failed; using defaults",
			slog.String("error", err.Error()))
		res = resource.Default()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Same scheme-handling story as tracer.go: WithEndpoint expects
	// bare host:port; WithEndpointURL expects a full URL with scheme.
	// Operators may set either form per the OTEL semantic-conventions
	// env-var spec, so accept both.
	//
	// Asymmetry with otlptracehttp: otlploghttp.WithEndpointURL preserves
	// the URL path verbatim and does NOT default to /v1/logs when the
	// path is empty — so an input like "http://collector:4318" makes the
	// exporter POST to "/" and every batch 404s. Append the OTLP-standard
	// path ourselves when the operator hasn't supplied one. (The sibling
	// otlptracehttp package does auto-append /v1/traces, which is why
	// tracer.go can hand WithEndpointURL the raw env var unchanged.)
	var endpointOpt otlploghttp.Option
	if strings.Contains(endpoint, "://") {
		u, parseErr := url.Parse(endpoint)
		if parseErr != nil {
			logger.Warn("telemetry: OTEL_EXPORTER_OTLP_ENDPOINT parse failed — OTLP logs disabled",
				slog.String("endpoint", endpoint),
				slog.String("error", parseErr.Error()))
			return LogResult{Shutdown: noopShutdown}
		}
		if u.Path == "" || u.Path == "/" {
			u.Path = "/v1/logs"
		}
		endpointOpt = otlploghttp.WithEndpointURL(u.String())
	} else {
		endpointOpt = otlploghttp.WithEndpoint(endpoint)
	}
	exporter, err := otlploghttp.New(ctx, endpointOpt)
	if err != nil {
		logger.Warn("telemetry: OTLP log exporter init failed — OTLP logs disabled",
			slog.String("endpoint", endpoint),
			slog.String("error", err.Error()))
		return LogResult{Shutdown: noopShutdown}
	}

	processor := log.NewBatchProcessor(exporter)
	provider := log.NewLoggerProvider(
		log.WithProcessor(processor),
		log.WithResource(res),
	)
	logger.Info("telemetry: OTLP log exporter ready", slog.String("endpoint", endpoint))

	// otelslog.NewHandler bridges slog records → OTEL log records using
	// the named scope. Keeping the scope name matched to the service
	// name means vendor backends (Honeycomb, Datadog, etc.) get a
	// scope.name attribute that already says "orkestra-backend" without
	// extra per-line configuration.
	handler := otelslog.NewHandler(
		serviceName,
		otelslog.WithLoggerProvider(provider),
	)

	shutdown := func(ctx context.Context) error {
		fctx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		if err := provider.ForceFlush(fctx); err != nil {
			logger.Warn("telemetry: log flush failed", slog.String("error", err.Error()))
		}
		return provider.Shutdown(ctx)
	}

	return LogResult{Handler: handler, Shutdown: shutdown}
}

// logsEnabled honors the same truthy-string parsing as other Orkestra
// boolean env vars. Empty / "false" / "0" / "no" all disable; "true" /
// "1" / "yes" enable.
func logsEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("OTEL_LOGS_ENABLED"))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func noopShutdown(_ context.Context) error { return nil }
