// Package services holds the LogLevelService — the DB-backed
// LevelResolver that backs ADR-0005 Phase F. The service owns an
// atomic snapshot of (global, perModule) refreshed on every admin
// mutation; the slog handler reads it lock-free on every Enabled
// call.
package services

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/orkestra/backend/internal/core/logging/models"
	"github.com/orkestra/backend/internal/core/logging/repository"
)

// LevelResolver mirrors utils.LevelResolver so consumers can depend
// on this package without pulling shared/utils. Both interfaces have
// the same shape — Go's structural typing lets a *LogLevelService
// satisfy both without any glue.
type LevelResolver interface {
	Global() slog.Level
	LevelFor(module string) (slog.Level, bool)
}

// snapshot is the immutable value stored under the atomic.Pointer.
// Replaced wholesale on every mutation so readers never see a
// partially-updated state.
type snapshot struct {
	global    slog.Level
	perModule map[string]slog.Level
	updatedAt time.Time
	updatedBy string
}

// LogLevelService owns the persisted log-level configuration and
// serves the LevelResolver contract for the slog handler.
//
// Concurrency model: snapshot is a *snapshot pointer behind an
// atomic.Pointer, so Global / LevelFor (called on every log line)
// stay lock-free. Mutations (admin endpoints) call DB.Upsert under
// a mutex so a refresh race can't lose a write, then publish a new
// snapshot. The mutex is touched only on admin writes — never on
// the hot read path.
type LogLevelService struct {
	repo     repository.Repository
	current  atomic.Pointer[snapshot]
	mu       sync.Mutex // serializes Upsert + snapshot publish
	logger   *slog.Logger
	envBoot  envBoot // captured at construction for "reset to env" semantics
	moduleCt moduleCatalog
}

// envBoot is the env-driven default the service falls back to when
// the DB document hasn't been seeded yet. Carries the same shape
// the StaticLevelResolver does at boot in shared/utils — so the
// behaviour is identical until an admin mutates the DB.
type envBoot struct {
	global slog.Level
	perMod map[string]slog.Level
}

// moduleCatalog is the list of module names the admin UI surfaces
// rows for. Populated at construction from the registered module
// set; the service does not enumerate Mongo collections to learn
// which modules exist.
type moduleCatalog struct {
	names []string
}

// NewLogLevelService builds the service with an explicit env-driven
// default and the catalog of module names the admin UI cares about.
// Pass logger=deps.Logger for boot diagnostics; pass moduleNames=
// list-of-registered-modules from the module registry. envGlobal /
// envPerModule capture the static-resolver values that were used
// during early boot so "reset to env" semantics work after admin
// edits.
func NewLogLevelService(repo repository.Repository, logger *slog.Logger, envGlobal slog.Level, envPerModule map[string]slog.Level, moduleNames []string) *LogLevelService {
	svc := &LogLevelService{
		repo:     repo,
		logger:   logger,
		envBoot:  envBoot{global: envGlobal, perMod: cloneLevelMap(envPerModule)},
		moduleCt: moduleCatalog{names: append([]string(nil), moduleNames...)},
	}
	// Seed snapshot from the env default. Load() below replaces it
	// with the persisted document if present.
	svc.publishSnapshot(envGlobal, envPerModule, time.Time{}, "")
	return svc
}

// Load reads the persisted document and publishes a snapshot if
// found. When the document doesn't exist, the service stays on the
// env-driven snapshot seeded by NewLogLevelService. Safe to call at
// boot (single-shot) or to refresh after an out-of-process mutation.
func (s *LogLevelService) Load(ctx context.Context) error {
	doc, err := s.repo.Get(ctx)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil // env defaults stand
		}
		return err
	}
	s.applyDoc(doc)
	return nil
}

// Global implements LevelResolver.
func (s *LogLevelService) Global() slog.Level {
	if snap := s.current.Load(); snap != nil {
		return snap.global
	}
	return slog.LevelInfo
}

// LevelFor implements LevelResolver.
func (s *LogLevelService) LevelFor(module string) (slog.Level, bool) {
	snap := s.current.Load()
	if snap == nil {
		return slog.LevelInfo, false
	}
	l, ok := snap.perModule[module]
	return l, ok
}

// SetGlobal updates the global threshold and persists. Mutates
// under the service mutex so concurrent admin writes serialize.
func (s *LogLevelService) SetGlobal(ctx context.Context, level models.LogLevel, actor string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cur := s.copyCurrentSnapshot()
	cur.global = level.Slog()
	cur.updatedAt = time.Now().UTC()
	cur.updatedBy = actor
	return s.persistAndPublish(ctx, cur)
}

// SetModule sets a per-module override. Passing a level identical
// to the current global still persists the row — operators can use
// it to "pin" a module against future global changes.
func (s *LogLevelService) SetModule(ctx context.Context, module string, level models.LogLevel, actor string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cur := s.copyCurrentSnapshot()
	cur.perModule[module] = level.Slog()
	cur.updatedAt = time.Now().UTC()
	cur.updatedBy = actor
	return s.persistAndPublish(ctx, cur)
}

// UnsetModule removes a per-module override so the module falls
// back to Global. Idempotent — returns nil even when no override
// existed (the resulting state matches the request).
func (s *LogLevelService) UnsetModule(ctx context.Context, module string, actor string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cur := s.copyCurrentSnapshot()
	delete(cur.perModule, module)
	cur.updatedAt = time.Now().UTC()
	cur.updatedBy = actor
	return s.persistAndPublish(ctx, cur)
}

// View renders the AdminView surface for the GET endpoint. Resolves
// the per-module effective level so the UI doesn't have to repeat
// the Global fallback logic.
func (s *LogLevelService) View() models.AdminView {
	snap := s.current.Load()
	view := models.AdminView{
		Global: levelToModelLevel(snap.global),
	}
	if snap.updatedAt != (time.Time{}) {
		view.UpdatedAt = snap.updatedAt
	}
	view.UpdatedBy = snap.updatedBy

	for _, name := range s.moduleCt.names {
		entry := models.AdminModuleEntry{Name: name}
		if l, ok := snap.perModule[name]; ok {
			entry.Effective = levelToModelLevel(l)
			entry.HasOverride = true
		} else {
			entry.Effective = view.Global
		}
		view.Modules = append(view.Modules, entry)
	}
	return view
}

// ResetToEnv reverts both global and per-module to the env-driven
// snapshot captured at NewLogLevelService time. Persists the result
// so a restart sees the same state.
func (s *LogLevelService) ResetToEnv(ctx context.Context, actor string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cur := &snapshot{
		global:    s.envBoot.global,
		perModule: cloneLevelMap(s.envBoot.perMod),
		updatedAt: time.Now().UTC(),
		updatedBy: actor,
	}
	return s.persistAndPublish(ctx, cur)
}

// ---- internal helpers ----

func (s *LogLevelService) copyCurrentSnapshot() *snapshot {
	cur := s.current.Load()
	if cur == nil {
		return &snapshot{global: s.envBoot.global, perModule: cloneLevelMap(s.envBoot.perMod)}
	}
	return &snapshot{
		global:    cur.global,
		perModule: cloneLevelMap(cur.perModule),
		updatedAt: cur.updatedAt,
		updatedBy: cur.updatedBy,
	}
}

func (s *LogLevelService) persistAndPublish(ctx context.Context, snap *snapshot) error {
	doc := &models.LogLevelDoc{
		ConfigKey: models.DefaultConfigKey,
		Global:    levelToModelLevel(snap.global),
		PerModule: levelMapToModelMap(snap.perModule),
		UpdatedAt: snap.updatedAt,
		UpdatedBy: snap.updatedBy,
	}
	if err := s.repo.Upsert(ctx, doc); err != nil {
		return err
	}
	s.current.Store(snap)
	return nil
}

func (s *LogLevelService) applyDoc(doc *models.LogLevelDoc) {
	perMod := map[string]slog.Level{}
	for k, v := range doc.PerModule {
		perMod[k] = v.Slog()
	}
	snap := &snapshot{
		global:    doc.Global.Slog(),
		perModule: perMod,
		updatedAt: doc.UpdatedAt,
		updatedBy: doc.UpdatedBy,
	}
	s.current.Store(snap)
}

func (s *LogLevelService) publishSnapshot(global slog.Level, perModule map[string]slog.Level, at time.Time, by string) {
	snap := &snapshot{
		global:    global,
		perModule: cloneLevelMap(perModule),
		updatedAt: at,
		updatedBy: by,
	}
	s.current.Store(snap)
}

func cloneLevelMap(in map[string]slog.Level) map[string]slog.Level {
	if in == nil {
		return map[string]slog.Level{}
	}
	out := make(map[string]slog.Level, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func levelToModelLevel(l slog.Level) models.LogLevel {
	switch {
	case l <= slog.LevelDebug:
		return models.LogLevelDebug
	case l <= slog.LevelInfo:
		return models.LogLevelInfo
	case l <= slog.LevelWarn:
		return models.LogLevelWarn
	default:
		return models.LogLevelError
	}
}

func levelMapToModelMap(in map[string]slog.Level) map[string]models.LogLevel {
	out := make(map[string]models.LogLevel, len(in))
	for k, v := range in {
		out[k] = levelToModelLevel(v)
	}
	return out
}
