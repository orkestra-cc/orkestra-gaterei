package utils

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

// PerModuleLevelHandler wraps an slog.Handler with per-module level
// overrides. ADR-0005 §1.4 — operators can debug a single module
// (LOG_LEVEL_RAG=debug) without flooding the rest of the backend's
// output. The wrapped handler keeps doing its job (formatting,
// trace-id stamping, etc.); this layer only gates Enabled and stamps
// the "module" attribute back onto the record on the way out.
//
// How the gate works:
//
//  1. Module code obtains a logger via deps.Logger which the registry
//     pre-decorates with .With(slog.String("module", "<name>")). Slog
//     calls our WithAttrs, which detects the "module" key, remembers
//     it on a new handler instance, and drops the attr from the
//     pass-through list (we re-stamp it in Handle).
//  2. When the logger emits a record, slog first calls Enabled on the
//     module-decorated handler. Enabled looks up the remembered module
//     in the levels map; if present, it gates by that threshold,
//     otherwise it falls back to the global threshold.
//  3. If Enabled returns true, Handle re-stamps the "module" attribute
//     on the record so consumers (Loki, Tempo, etc.) still see it in
//     the JSON output.
//
// Logger.Info / slog.Info without a preceding .With("module", ...) hits
// the bare handler — h.module is the empty string, only the global
// level applies, and no "module" attribute appears on the record.
type PerModuleLevelHandler struct {
	base   slog.Handler
	levels map[string]slog.Level
	global slog.Level
	module string
}

// NewPerModuleLevelHandler reads LOG_LEVEL_<MODULE> env vars at boot
// and wraps the supplied base handler. Pass the global level
// separately rather than reading it from the env again — SetupLogger
// already resolves LOG_LEVEL once and the global threshold must be
// consistent across all wrapped layers.
//
// Module names are lowercased to match the conventional
// internal/addons/<name> and internal/core/<name> directory names. An
// env var like LOG_LEVEL_RAG=debug therefore matches the rag module's
// .With("module", "rag") logger.
//
// Unparseable level strings (LOG_LEVEL_FOO=babble) are silently
// ignored — we'd rather a typo not kill boot.
func NewPerModuleLevelHandler(base slog.Handler, global slog.Level) *PerModuleLevelHandler {
	return &PerModuleLevelHandler{
		base:   base,
		levels: loadPerModuleLevels(),
		global: global,
	}
}

// loadPerModuleLevels walks os.Environ and extracts every
// LOG_LEVEL_<MODULE>=<level> pair, skipping the bare LOG_LEVEL var
// (which is the global threshold). Returns an empty map if none are
// set.
func loadPerModuleLevels() map[string]slog.Level {
	const prefix = "LOG_LEVEL_"
	out := map[string]slog.Level{}
	for _, kv := range os.Environ() {
		sep := strings.IndexByte(kv, '=')
		if sep <= 0 {
			continue
		}
		key, value := kv[:sep], kv[sep+1:]
		if !strings.HasPrefix(key, prefix) || key == "LOG_LEVEL" {
			continue
		}
		module := strings.ToLower(key[len(prefix):])
		if module == "" {
			continue
		}
		if lvl, ok := parseSlogLevel(value); ok {
			out[module] = lvl
		}
	}
	return out
}

// parseSlogLevel mirrors SetupLogger's level parser — same accepted
// strings, same case-insensitive matching. Kept here so the per-module
// path doesn't drift from the global one.
func parseSlogLevel(s string) (slog.Level, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug, true
	case "info":
		return slog.LevelInfo, true
	case "warn", "warning":
		return slog.LevelWarn, true
	case "error":
		return slog.LevelError, true
	default:
		return slog.LevelInfo, false
	}
}

// Enabled returns true when the record's level passes the
// per-module threshold (if any) or the global threshold otherwise. The
// per-module threshold is selected by the module attribute remembered
// from a prior WithAttrs call — bare loggers (h.module=="") only see
// the global threshold.
func (h *PerModuleLevelHandler) Enabled(_ context.Context, level slog.Level) bool {
	threshold := h.global
	if h.module != "" {
		if l, ok := h.levels[h.module]; ok {
			threshold = l
		}
	}
	return level >= threshold
}

// Handle re-stamps the remembered module name on the record so Loki /
// vendor backends can filter by it, then delegates to the base
// handler. Bare handlers (no .With("module", ...) ancestry) leave the
// record unchanged.
func (h *PerModuleLevelHandler) Handle(ctx context.Context, r slog.Record) error {
	if h.module != "" {
		r.AddAttrs(slog.String("module", h.module))
	}
	return h.base.Handle(ctx, r)
}

// WithAttrs intercepts the "module" attribute (storing it on a new
// handler so Enabled and Handle can use it) and passes the remaining
// attributes through to the base handler. If no "module" attr is
// present the returned handler inherits the current module name —
// nesting .With() calls compose cleanly.
func (h *PerModuleLevelHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	moduleName := h.module
	remaining := attrs[:0:0] // new backing slice; never mutate caller's
	for _, a := range attrs {
		if a.Key == "module" {
			moduleName = a.Value.String()
			continue
		}
		remaining = append(remaining, a)
	}
	return &PerModuleLevelHandler{
		base:   h.base.WithAttrs(remaining),
		levels: h.levels,
		global: h.global,
		module: moduleName,
	}
}

// WithGroup wraps the base group, preserving the module name and
// level configuration.
func (h *PerModuleLevelHandler) WithGroup(name string) slog.Handler {
	return &PerModuleLevelHandler{
		base:   h.base.WithGroup(name),
		levels: h.levels,
		global: h.global,
		module: h.module,
	}
}
