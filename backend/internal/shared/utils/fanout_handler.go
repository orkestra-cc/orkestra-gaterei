package utils

import (
	"context"
	"errors"
	"log/slog"
)

// FanoutHandler delivers every slog record to multiple wrapped
// handlers. ADR-0005 Phase E (Tier 2) — when OTEL_LOGS_ENABLED=true the
// SetupLogger pipeline tees records to both the stdout handler and the
// OTLP-bridge handler, so vendor backends receive the same line that
// docker logs / Promtail / a log aggregator captures from stdout.
//
// Why fan out instead of replace stdout:
//
//	"Stdout remains the source of truth: OTLP logs are an additional
//	 fanout, not a replacement. If the collector or vendor backend goes
//	 down, stdout-captured logs are still recoverable."
//	                                       — ADR-0005 §4
//
// Behavior contracts:
//
//   - Enabled returns true when ANY wrapped handler is enabled.
//     Suppression must be opt-in per wrapped handler; the fanout
//     never gates a record because of one strict downstream.
//
//   - Handle calls each wrapped handler's Enabled before dispatching
//     so a verbose stdout config can keep emitting debug while a
//     billed-per-byte OTLP backend only sees warn+ (operators
//     configure the per-handler levels at construction time).
//
//   - Record.Clone is used on every fanned dispatch — the slog
//     contract is that handlers may mutate the record they receive,
//     and skipping the clone would let stdout's processing corrupt
//     OTLP's view.
//
//   - WithAttrs / WithGroup propagate to every wrapped handler so
//     .With chains compose naturally across the fanout. Producing a
//     new FanoutHandler each time preserves immutability.
//
//   - Errors are aggregated with errors.Join — a flaky OTLP push must
//     not silently swallow a stdout write error and vice versa.
type FanoutHandler struct {
	handlers []slog.Handler
}

// NewFanoutHandler returns a handler that fans every record to each
// supplied handler. Nil entries are dropped so callers can pass an
// optional handler (e.g. the result of telemetry.InitLogs.Handler,
// which is nil when OTLP logs are disabled) without guarding.
//
// Constructing with zero handlers returns a no-op handler — Enabled
// reports false for every level so slog skips record construction
// entirely. This is intentional: a misconfigured pipeline that
// produces an empty fanout is loud (no logs anywhere) rather than
// silently dropping records into a closed channel.
func NewFanoutHandler(handlers ...slog.Handler) FanoutHandler {
	filtered := make([]slog.Handler, 0, len(handlers))
	for _, h := range handlers {
		if h != nil {
			filtered = append(filtered, h)
		}
	}
	return FanoutHandler{handlers: filtered}
}

// Enabled returns true when any wrapped handler is enabled at the
// supplied level. Slog calls Enabled before constructing the record;
// returning true here means at least one downstream will accept it.
func (h FanoutHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, hh := range h.handlers {
		if hh.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

// Handle dispatches a clone of the record to every wrapped handler
// that reports Enabled at the record's level. Errors are aggregated
// — a slow OTLP exporter that times out must not mask a stdout
// write failure, and a broken stdout pipe (k8s pod log rotation gone
// wrong) must not silence the OTLP fanout.
func (h FanoutHandler) Handle(ctx context.Context, r slog.Record) error {
	var errs []error
	for _, hh := range h.handlers {
		if !hh.Enabled(ctx, r.Level) {
			continue
		}
		if err := hh.Handle(ctx, r.Clone()); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// WithAttrs returns a new FanoutHandler whose every wrapped handler
// has the attributes applied. Preserves immutability so .With chains
// don't mutate the parent.
func (h FanoutHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	out := make([]slog.Handler, len(h.handlers))
	for i, hh := range h.handlers {
		out[i] = hh.WithAttrs(attrs)
	}
	return FanoutHandler{handlers: out}
}

// WithGroup returns a new FanoutHandler whose every wrapped handler
// has the group applied.
func (h FanoutHandler) WithGroup(name string) slog.Handler {
	out := make([]slog.Handler, len(h.handlers))
	for i, hh := range h.handlers {
		out[i] = hh.WithGroup(name)
	}
	return FanoutHandler{handlers: out}
}
