package utils

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"sync/atomic"
)

// LevelResolver is the contract for "what threshold applies to this
// module right now?". ADR-0005 Phase F splits the original
// boot-snapshot implementation into a stable handler + a swappable
// resolver so the per-module threshold can be mutated at runtime
// without restarting the process.
//
// Two implementations ship today:
//
//   - StaticLevelResolver (this file) — captures LOG_LEVEL_<MODULE>
//     env vars at boot. Default for the early-boot logger before any
//     DB-backed service is wired.
//   - core/logging/services.LogLevelService — DB-backed via an
//     atomic.Value snapshot, refreshed on admin mutation. Replaces
//     the static resolver after the logging core module initializes.
//
// Resolvers must be safe for concurrent read by an arbitrary number
// of goroutines (every log call hits Global/LevelFor on the hot
// path). Mutators run on the admin API path which is bounded.
type LevelResolver interface {
	// Global returns the threshold applied to records that have no
	// module attribute (slog.Info without a module-decorated logger).
	Global() slog.Level
	// LevelFor returns (threshold, true) when the module has an
	// explicit override, or (_, false) to indicate "fall back to
	// global". Returning the empty Level with false is the documented
	// way to express "no override".
	LevelFor(module string) (slog.Level, bool)
}

// StaticLevelResolver captures the env-driven configuration at
// construction time. Honors the same LOG_LEVEL / LOG_LEVEL_<MODULE>
// env var contract that v1 (Phase C) used. Safe for concurrent read
// because the maps are never mutated post-construction — operators
// who want runtime mutation use the DB-backed resolver instead.
type StaticLevelResolver struct {
	global slog.Level
	levels map[string]slog.Level
}

// NewStaticLevelResolver wraps a pre-resolved global level + the
// per-module map produced by loadPerModuleLevels. SetupLogger calls
// this with the env-driven snapshot during early boot.
func NewStaticLevelResolver(global slog.Level, levels map[string]slog.Level) StaticLevelResolver {
	if levels == nil {
		levels = map[string]slog.Level{}
	}
	return StaticLevelResolver{global: global, levels: levels}
}

func (r StaticLevelResolver) Global() slog.Level { return r.global }

func (r StaticLevelResolver) LevelFor(module string) (slog.Level, bool) {
	l, ok := r.levels[module]
	return l, ok
}

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
// resolverBox indirects the atomic pointer so every clone produced
// by WithAttrs / WithGroup shares the SAME pointer-to-atomic. Without
// this indirection, a clone would carry its own atomic.Pointer
// initialized once and never updated — SetResolver on the root would
// mutate only the root's pointer, and module-decorated loggers
// (which are clones produced during deps.Logger.With("module", ...))
// would observe stale resolvers forever.
type resolverBox struct {
	p atomic.Pointer[LevelResolver]
}

type PerModuleLevelHandler struct {
	base     slog.Handler
	resolver *resolverBox
	module   string
}

// NewPerModuleLevelHandler wraps the supplied base handler with the
// given level resolver. The resolver decides "what threshold applies
// right now" for each module; this handler only consumes its answer
// to gate Enabled.
//
// SetupLogger constructs the handler with a StaticLevelResolver at
// boot (env-driven). ADR-0005 Phase F swaps in a DB-backed resolver
// via SetResolver after the logging core module wires up — every
// existing clone (module loggers built via deps.Logger.With) picks
// up the new resolver immediately because they all share the same
// resolverBox.
//
// Module names follow the conventional internal/addons/<name> /
// internal/core/<name> shape. An env var like LOG_LEVEL_RAG=debug
// matches the rag module's .With("module", "rag") logger; the same
// "rag" key is what the admin endpoint accepts.
func NewPerModuleLevelHandler(base slog.Handler, resolver LevelResolver) *PerModuleLevelHandler {
	box := &resolverBox{}
	box.p.Store(&resolver)
	return &PerModuleLevelHandler{base: base, resolver: box}
}

// SetResolver atomically swaps the resolver. Subsequent Enabled
// calls — including those from clones produced by prior WithAttrs /
// WithGroup invocations — observe the new resolver immediately
// (atomic.Pointer.Load is the memory barrier). Used by main.go to
// upgrade from the boot StaticLevelResolver to the DB-backed live
// resolver once the logging core module is up.
func (h *PerModuleLevelHandler) SetResolver(r LevelResolver) {
	if r == nil || h.resolver == nil {
		return
	}
	h.resolver.p.Store(&r)
}

// currentResolver returns the active resolver. Internal helper that
// dereferences the atomic pointer; nil-safety belongs here so the
// handler methods stay readable.
func (h *PerModuleLevelHandler) currentResolver() LevelResolver {
	if h.resolver == nil {
		return nil
	}
	if p := h.resolver.p.Load(); p != nil {
		return *p
	}
	return nil
}

// LoadPerModuleLevels is the public form of loadPerModuleLevels.
// ADR-0005 Phase F's logging core module reuses the same env parse
// to seed its DB-backed snapshot on first boot.
func LoadPerModuleLevels() map[string]slog.Level {
	return loadPerModuleLevels()
}

// GlobalLevelFromEnv mirrors the LOG_LEVEL parse done in
// SetupLogger so the logging core module can seed its DB-backed
// snapshot with the same default. Returns slog.LevelInfo when
// LOG_LEVEL is unset or unrecognised — same fallback SetupLogger
// uses.
func GlobalLevelFromEnv() slog.Level {
	if l, ok := parseSlogLevel(os.Getenv("LOG_LEVEL")); ok {
		return l
	}
	return slog.LevelInfo
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
// the global threshold. The resolver is consulted on every call so a
// runtime mutation via the admin API takes effect immediately.
func (h *PerModuleLevelHandler) Enabled(_ context.Context, level slog.Level) bool {
	r := h.currentResolver()
	if r == nil {
		return true // defensive — never built one, accept everything
	}
	threshold := r.Global()
	if h.module != "" {
		if l, ok := r.LevelFor(h.module); ok {
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
		base:     h.base.WithAttrs(remaining),
		resolver: h.resolver, // shared box — SetResolver propagates
		module:   moduleName,
	}
}

// WithGroup wraps the base group, preserving the module name and
// resolver box pointer (so runtime resolver swaps reach this clone).
func (h *PerModuleLevelHandler) WithGroup(name string) slog.Handler {
	return &PerModuleLevelHandler{
		base:     h.base.WithGroup(name),
		resolver: h.resolver,
		module:   h.module,
	}
}
