package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

func newHandlerWithLevels(t *testing.T, levels map[string]slog.Level, global slog.Level) (*slog.Logger, *bytes.Buffer) {
	t.Helper()
	buf := &bytes.Buffer{}
	base := slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	h := &PerModuleLevelHandler{base: base, levels: levels, global: global}
	return slog.New(h), buf
}

func parseLines(t *testing.T, raw []byte) []map[string]any {
	t.Helper()
	var out []map[string]any
	for _, line := range bytes.Split(bytes.TrimSpace(raw), []byte("\n")) {
		if len(line) == 0 {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal(line, &m); err != nil {
			t.Fatalf("invalid JSON %q: %v", string(line), err)
		}
		out = append(out, m)
	}
	return out
}

func TestPerModuleLevelHandler_GlobalLevelGatesBareLogger(t *testing.T) {
	logger, buf := newHandlerWithLevels(t, nil, slog.LevelInfo)
	logger.Debug("dropped")
	logger.Info("kept")
	logger.Warn("kept-warn")

	lines := parseLines(t, buf.Bytes())
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines (info + warn), got %d: %v", len(lines), lines)
	}
	if lines[0]["msg"] != "kept" || lines[1]["msg"] != "kept-warn" {
		t.Errorf("got %v", lines)
	}
}

func TestPerModuleLevelHandler_PerModuleOverride_Enables(t *testing.T) {
	logger, buf := newHandlerWithLevels(t, map[string]slog.Level{"rag": slog.LevelDebug}, slog.LevelInfo)

	rag := logger.With(slog.String("module", "rag"))
	rag.Debug("rag-debug-kept")
	rag.Info("rag-info-kept")

	// Bare logger still gated globally — debug dropped.
	logger.Debug("global-debug-dropped")

	lines := parseLines(t, buf.Bytes())
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %v", len(lines), lines)
	}
	if lines[0]["msg"] != "rag-debug-kept" {
		t.Errorf("expected rag-debug-kept first, got %v", lines[0])
	}
	if lines[0]["module"] != "rag" {
		t.Errorf("module attr lost: %v", lines[0])
	}
}

func TestPerModuleLevelHandler_PerModuleOverride_Suppresses(t *testing.T) {
	// Module quieter than global — only warn+ from rag should survive
	// even when global is info.
	logger, buf := newHandlerWithLevels(t, map[string]slog.Level{"rag": slog.LevelWarn}, slog.LevelInfo)

	rag := logger.With(slog.String("module", "rag"))
	rag.Info("rag-info-dropped")
	rag.Warn("rag-warn-kept")

	logger.Info("global-info-kept")

	lines := parseLines(t, buf.Bytes())
	msgs := make([]string, 0, len(lines))
	for _, l := range lines {
		msgs = append(msgs, l["msg"].(string))
	}
	for _, want := range []string{"rag-warn-kept", "global-info-kept"} {
		found := false
		for _, m := range msgs {
			if m == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected msg %q in output, got %v", want, msgs)
		}
	}
	for _, banned := range []string{"rag-info-dropped"} {
		for _, m := range msgs {
			if m == banned {
				t.Errorf("dropped msg %q still appeared: %v", banned, msgs)
			}
		}
	}
}

func TestPerModuleLevelHandler_NestedWithDoesNotResetModule(t *testing.T) {
	logger, buf := newHandlerWithLevels(t, map[string]slog.Level{"rag": slog.LevelDebug}, slog.LevelInfo)

	rag := logger.With(slog.String("module", "rag"))
	scoped := rag.With(slog.String("op", "ingest"))
	scoped.Debug("nested-debug-kept")

	lines := parseLines(t, buf.Bytes())
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d: %v", len(lines), lines)
	}
	if lines[0]["module"] != "rag" {
		t.Errorf("nested .With lost module attr: %v", lines[0])
	}
	if lines[0]["op"] != "ingest" {
		t.Errorf("nested .With dropped op attr: %v", lines[0])
	}
}

func TestPerModuleLevelHandler_NonModuleAttrsPassThrough(t *testing.T) {
	logger, buf := newHandlerWithLevels(t, nil, slog.LevelInfo)
	logger.With(slog.String("svc", "thing"), slog.Int("n", 42)).Info("hi")

	lines := parseLines(t, buf.Bytes())
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	if lines[0]["svc"] != "thing" || int(lines[0]["n"].(float64)) != 42 {
		t.Errorf("non-module attrs dropped: %v", lines[0])
	}
	if _, ok := lines[0]["module"]; ok {
		t.Errorf("bare logger should not stamp module attr: %v", lines[0])
	}
}

func TestPerModuleLevelHandler_WithGroupPreservesModuleAndLevels(t *testing.T) {
	logger, buf := newHandlerWithLevels(t, map[string]slog.Level{"rag": slog.LevelDebug}, slog.LevelInfo)

	rag := logger.With(slog.String("module", "rag")).WithGroup("g")
	rag.Debug("kept", slog.String("k", "v"))

	out := buf.String()
	if !strings.Contains(out, `"module":"rag"`) {
		t.Errorf("module attr lost across WithGroup, got %s", out)
	}
	if !strings.Contains(out, `"g":{`) {
		t.Errorf("group not applied, got %s", out)
	}
}

func TestPerModuleLevelHandler_EnabledIsCheapWhenSuppressed(t *testing.T) {
	// Even when the level is suppressed via per-module override, Enabled
	// must return false (so slog skips record construction entirely).
	// This is the perf-critical guarantee for noisy modules.
	logger, buf := newHandlerWithLevels(t, map[string]slog.Level{"rag": slog.LevelWarn}, slog.LevelInfo)
	rag := logger.With(slog.String("module", "rag"))
	if rag.Enabled(context.Background(), slog.LevelInfo) {
		t.Errorf("Enabled should be false for rag at info when module level is warn")
	}
	if !rag.Enabled(context.Background(), slog.LevelWarn) {
		t.Errorf("Enabled should be true for rag at warn")
	}
	rag.Info("dropped")
	if buf.Len() != 0 {
		t.Errorf("Enabled=false should suppress emission, got %q", buf.String())
	}
}

func TestParseSlogLevel(t *testing.T) {
	tests := []struct {
		in   string
		want slog.Level
		ok   bool
	}{
		{"debug", slog.LevelDebug, true},
		{"DEBUG", slog.LevelDebug, true},
		{" Info ", slog.LevelInfo, true},
		{"warn", slog.LevelWarn, true},
		{"warning", slog.LevelWarn, true},
		{"error", slog.LevelError, true},
		{"trace", slog.LevelInfo, false},
		{"", slog.LevelInfo, false},
	}
	for _, tc := range tests {
		got, ok := parseSlogLevel(tc.in)
		if ok != tc.ok {
			t.Errorf("parseSlogLevel(%q) ok=%v, want %v", tc.in, ok, tc.ok)
		}
		if ok && got != tc.want {
			t.Errorf("parseSlogLevel(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestLoadPerModuleLevels_EnvOverrides(t *testing.T) {
	t.Setenv("LOG_LEVEL_RAG", "debug")
	t.Setenv("LOG_LEVEL_BILLING", "warn")
	t.Setenv("LOG_LEVEL_BOGUS", "babble") // unparseable — must be silently dropped
	t.Setenv("LOG_LEVEL", "info")         // bare LOG_LEVEL must NOT appear in the map

	out := loadPerModuleLevels()
	if out["rag"] != slog.LevelDebug {
		t.Errorf("rag = %v, want debug", out["rag"])
	}
	if out["billing"] != slog.LevelWarn {
		t.Errorf("billing = %v, want warn", out["billing"])
	}
	if _, ok := out["bogus"]; ok {
		t.Errorf("unparseable LOG_LEVEL_BOGUS must be dropped, got %v", out)
	}
	if _, ok := out[""]; ok {
		t.Errorf("bare LOG_LEVEL must not pollute the map, got %v", out)
	}
}
