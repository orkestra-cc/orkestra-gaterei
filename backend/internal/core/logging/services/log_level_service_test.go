package services

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/orkestra/backend/internal/core/logging/models"
	"github.com/orkestra/backend/internal/core/logging/repository"
)

// fakeRepo is an in-memory Repository for unit tests. Wraps a mutex
// around the single doc since concurrent SetGlobal/SetModule paths
// would otherwise race in -race mode.
type fakeRepo struct {
	mu  sync.Mutex
	doc *models.LogLevelDoc
	err error // injectable for failure-path tests
}

func (r *fakeRepo) Get(_ context.Context) (*models.LogLevelDoc, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return nil, r.err
	}
	if r.doc == nil {
		return nil, repository.ErrNotFound
	}
	clone := *r.doc
	return &clone, nil
}

func (r *fakeRepo) Upsert(_ context.Context, doc *models.LogLevelDoc) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return r.err
	}
	clone := *doc
	r.doc = &clone
	return nil
}

func newTestSvc(t *testing.T) (*LogLevelService, *fakeRepo) {
	t.Helper()
	repo := &fakeRepo{}
	logger := slog.New(slog.NewTextHandler(testWriter{t: t}, &slog.HandlerOptions{Level: slog.LevelDebug}))
	svc := NewLogLevelService(repo, logger, slog.LevelInfo, map[string]slog.Level{
		"rag": slog.LevelDebug,
	}, []string{"rag", "billing", "auth"})
	return svc, repo
}

type testWriter struct{ t *testing.T }

func (w testWriter) Write(p []byte) (int, error) {
	w.t.Log(string(p))
	return len(p), nil
}

func TestLogLevelService_EnvDefaultsWhenNothingPersisted(t *testing.T) {
	svc, _ := newTestSvc(t)

	if got := svc.Global(); got != slog.LevelInfo {
		t.Errorf("Global = %v, want info", got)
	}
	if l, ok := svc.LevelFor("rag"); !ok || l != slog.LevelDebug {
		t.Errorf("LevelFor(rag) = %v,%v want debug,true", l, ok)
	}
	if _, ok := svc.LevelFor("billing"); ok {
		t.Errorf("LevelFor(billing) returned ok=true without an env override")
	}
}

func TestLogLevelService_LoadFromDB(t *testing.T) {
	svc, repo := newTestSvc(t)

	repo.doc = &models.LogLevelDoc{
		ConfigKey: models.DefaultConfigKey,
		Global:    models.LogLevelWarn,
		PerModule: map[string]models.LogLevel{"billing": models.LogLevelError},
	}

	if err := svc.Load(context.Background()); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := svc.Global(); got != slog.LevelWarn {
		t.Errorf("Global = %v, want warn", got)
	}
	if l, ok := svc.LevelFor("billing"); !ok || l != slog.LevelError {
		t.Errorf("LevelFor(billing) = %v,%v want error,true", l, ok)
	}
	// rag was in the env seed but the persisted doc didn't include it
	// — Load REPLACES the snapshot wholesale; rag override is gone.
	if _, ok := svc.LevelFor("rag"); ok {
		t.Errorf("LevelFor(rag) should NOT be set after a Load that doesn't include it")
	}
}

func TestLogLevelService_SetGlobalPersistsAndPublishes(t *testing.T) {
	svc, repo := newTestSvc(t)

	if err := svc.SetGlobal(context.Background(), models.LogLevelError, "test-user"); err != nil {
		t.Fatalf("SetGlobal: %v", err)
	}

	// In-memory snapshot updated.
	if got := svc.Global(); got != slog.LevelError {
		t.Errorf("Global = %v, want error", got)
	}
	// Persisted.
	if repo.doc == nil || repo.doc.Global != models.LogLevelError {
		t.Errorf("repo doc not persisted: %+v", repo.doc)
	}
	if repo.doc.UpdatedBy != "test-user" {
		t.Errorf("UpdatedBy = %q, want test-user", repo.doc.UpdatedBy)
	}
}

func TestLogLevelService_SetModule_AddAndUnset(t *testing.T) {
	svc, repo := newTestSvc(t)

	if err := svc.SetModule(context.Background(), "billing", models.LogLevelWarn, "u"); err != nil {
		t.Fatalf("SetModule: %v", err)
	}
	if l, ok := svc.LevelFor("billing"); !ok || l != slog.LevelWarn {
		t.Errorf("billing override missing")
	}
	if repo.doc.PerModule["billing"] != models.LogLevelWarn {
		t.Errorf("billing not persisted: %+v", repo.doc.PerModule)
	}

	if err := svc.UnsetModule(context.Background(), "billing", "u"); err != nil {
		t.Fatalf("UnsetModule: %v", err)
	}
	if _, ok := svc.LevelFor("billing"); ok {
		t.Errorf("billing override should have been removed")
	}
	if _, persisted := repo.doc.PerModule["billing"]; persisted {
		t.Errorf("billing should be absent from persisted doc")
	}
}

func TestLogLevelService_UnsetModule_Idempotent(t *testing.T) {
	svc, _ := newTestSvc(t)
	if err := svc.UnsetModule(context.Background(), "nonexistent", "u"); err != nil {
		t.Errorf("UnsetModule for missing key should be a no-op, got %v", err)
	}
}

func TestLogLevelService_View(t *testing.T) {
	svc, _ := newTestSvc(t)
	_ = svc.SetModule(context.Background(), "billing", models.LogLevelError, "u")

	view := svc.View()
	if view.Global != models.LogLevelInfo {
		t.Errorf("view.Global = %v, want info", view.Global)
	}
	if len(view.Modules) != 3 {
		t.Errorf("expected 3 module rows, got %d", len(view.Modules))
	}

	byName := map[string]models.AdminModuleEntry{}
	for _, m := range view.Modules {
		byName[m.Name] = m
	}
	if e := byName["rag"]; !e.HasOverride || e.Effective != models.LogLevelDebug {
		t.Errorf("rag entry = %+v", e)
	}
	if e := byName["billing"]; !e.HasOverride || e.Effective != models.LogLevelError {
		t.Errorf("billing entry = %+v", e)
	}
	if e := byName["auth"]; e.HasOverride || e.Effective != models.LogLevelInfo {
		t.Errorf("auth should inherit Global without override, got %+v", e)
	}
}

func TestLogLevelService_ResetToEnv(t *testing.T) {
	svc, _ := newTestSvc(t)
	_ = svc.SetGlobal(context.Background(), models.LogLevelError, "u")
	_ = svc.SetModule(context.Background(), "billing", models.LogLevelWarn, "u")

	if err := svc.ResetToEnv(context.Background(), "u"); err != nil {
		t.Fatalf("ResetToEnv: %v", err)
	}
	if got := svc.Global(); got != slog.LevelInfo {
		t.Errorf("Global after reset = %v, want info", got)
	}
	if _, ok := svc.LevelFor("billing"); ok {
		t.Errorf("billing override should be gone after reset")
	}
	if l, ok := svc.LevelFor("rag"); !ok || l != slog.LevelDebug {
		t.Errorf("rag env seed should be restored after reset")
	}
}

func TestLogLevelService_PersistFailurePreservesSnapshot(t *testing.T) {
	svc, repo := newTestSvc(t)
	// Capture pre-mutation state.
	preGlobal := svc.Global()

	repo.err = errors.New("boom")
	err := svc.SetGlobal(context.Background(), models.LogLevelError, "u")
	if err == nil {
		t.Fatalf("expected error from broken repo")
	}
	// In-memory snapshot must NOT advance on persist failure.
	if svc.Global() != preGlobal {
		t.Errorf("snapshot updated despite persist failure: %v -> %v", preGlobal, svc.Global())
	}
}

func TestLogLevelService_ConcurrentReadsAndWrites(t *testing.T) {
	// Smoke test under -race: many concurrent readers (Global / LevelFor)
	// while a writer mutates SetModule. atomic.Pointer snapshot keeps
	// reads consistent without locking the hot path.
	svc, _ := newTestSvc(t)
	var (
		stop atomic.Bool
		wg   sync.WaitGroup
	)
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for !stop.Load() {
				_ = svc.Global()
				_, _ = svc.LevelFor("rag")
				_, _ = svc.LevelFor("billing")
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for n := 0; n < 200; n++ {
			lvl := models.LogLevelInfo
			if n%2 == 0 {
				lvl = models.LogLevelDebug
			}
			if err := svc.SetModule(context.Background(), "billing", lvl, "u"); err != nil {
				t.Errorf("SetModule: %v", err)
				return
			}
		}
		stop.Store(true)
	}()
	wg.Wait()
}
