package utils

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"
)

// captureHandler is a minimal slog.Handler that appends every record it
// receives to a slice. Useful as a test double for verifying fanout
// behavior without depending on the JSON encoder's format.
type captureHandler struct {
	level slog.Level
	calls *[]string
	attrs []slog.Attr
	group string
	err   error
}

func (h *captureHandler) Enabled(_ context.Context, l slog.Level) bool {
	return l >= h.level
}

func (h *captureHandler) Handle(_ context.Context, r slog.Record) error {
	var b strings.Builder
	if h.group != "" {
		b.WriteString(h.group + ".")
	}
	b.WriteString(r.Message)
	for _, a := range h.attrs {
		b.WriteString(" " + a.Key + "=" + a.Value.String())
	}
	r.Attrs(func(a slog.Attr) bool {
		b.WriteString(" " + a.Key + "=" + a.Value.String())
		return true
	})
	*h.calls = append(*h.calls, b.String())
	return h.err
}

func (h *captureHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	clone := *h
	clone.attrs = append(append([]slog.Attr{}, h.attrs...), attrs...)
	return &clone
}

func (h *captureHandler) WithGroup(name string) slog.Handler {
	clone := *h
	clone.group = name
	return &clone
}

func TestFanoutHandler_DispatchesToAll(t *testing.T) {
	var aCalls, bCalls []string
	a := &captureHandler{level: slog.LevelDebug, calls: &aCalls}
	b := &captureHandler{level: slog.LevelDebug, calls: &bCalls}

	logger := slog.New(NewFanoutHandler(a, b))
	logger.Info("hello")

	if len(aCalls) != 1 || aCalls[0] != "hello" {
		t.Errorf("handler a did not receive 'hello': %v", aCalls)
	}
	if len(bCalls) != 1 || bCalls[0] != "hello" {
		t.Errorf("handler b did not receive 'hello': %v", bCalls)
	}
}

func TestFanoutHandler_RespectsPerHandlerLevel(t *testing.T) {
	// Stdout-mock (debug) sees everything; OTLP-mock (warn) only sees warn+.
	var stdoutCalls, otlpCalls []string
	stdoutMock := &captureHandler{level: slog.LevelDebug, calls: &stdoutCalls}
	otlpMock := &captureHandler{level: slog.LevelWarn, calls: &otlpCalls}

	logger := slog.New(NewFanoutHandler(stdoutMock, otlpMock))
	logger.Info("info-line")
	logger.Warn("warn-line")

	if len(stdoutCalls) != 2 {
		t.Errorf("stdout should see both lines, got %v", stdoutCalls)
	}
	if len(otlpCalls) != 1 || otlpCalls[0] != "warn-line" {
		t.Errorf("otlp should only see warn-line, got %v", otlpCalls)
	}
}

func TestFanoutHandler_DropsNilHandlers(t *testing.T) {
	var calls []string
	live := &captureHandler{level: slog.LevelDebug, calls: &calls}

	// Constructing with a nil entry must not panic — the production
	// wiring passes telemetry.InitLogs.Handler which is nil when
	// OTLP logs are disabled.
	logger := slog.New(NewFanoutHandler(live, nil))
	logger.Info("survived-nil-dropped")

	if len(calls) != 1 {
		t.Errorf("expected single dispatch, got %v", calls)
	}
}

func TestFanoutHandler_EmptyFanoutIsNoOp(t *testing.T) {
	h := NewFanoutHandler()
	if h.Enabled(context.Background(), slog.LevelError) {
		t.Errorf("empty fanout should report Enabled=false at every level")
	}
	logger := slog.New(h)
	logger.Error("dropped") // must not panic
}

func TestFanoutHandler_AggregatesErrors(t *testing.T) {
	var aCalls, bCalls []string
	errBoom := errors.New("boom")
	a := &captureHandler{level: slog.LevelDebug, calls: &aCalls, err: errBoom}
	b := &captureHandler{level: slog.LevelDebug, calls: &bCalls} // healthy

	h := NewFanoutHandler(a, b)
	rec := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
	err := h.Handle(context.Background(), rec)

	if err == nil || !errors.Is(err, errBoom) {
		t.Errorf("expected aggregated error containing boom, got %v", err)
	}
	if len(aCalls) != 1 || len(bCalls) != 1 {
		t.Errorf("error on one handler should not prevent the other from dispatching (a=%v b=%v)", aCalls, bCalls)
	}
}

func TestFanoutHandler_WithAttrsPropagates(t *testing.T) {
	var aCalls, bCalls []string
	a := &captureHandler{level: slog.LevelDebug, calls: &aCalls}
	b := &captureHandler{level: slog.LevelDebug, calls: &bCalls}

	logger := slog.New(NewFanoutHandler(a, b)).With(slog.String("svc", "test"))
	logger.Info("hi")

	if !strings.Contains(aCalls[0], "svc=test") {
		t.Errorf("a missing svc=test: %v", aCalls)
	}
	if !strings.Contains(bCalls[0], "svc=test") {
		t.Errorf("b missing svc=test: %v", bCalls)
	}
}

func TestFanoutHandler_WithGroupPropagates(t *testing.T) {
	var aCalls, bCalls []string
	a := &captureHandler{level: slog.LevelDebug, calls: &aCalls}
	b := &captureHandler{level: slog.LevelDebug, calls: &bCalls}

	logger := slog.New(NewFanoutHandler(a, b)).WithGroup("scoped")
	logger.Info("hi")

	if !strings.HasPrefix(aCalls[0], "scoped.") {
		t.Errorf("a missing group prefix: %v", aCalls)
	}
	if !strings.HasPrefix(bCalls[0], "scoped.") {
		t.Errorf("b missing group prefix: %v", bCalls)
	}
}

func TestFanoutHandler_RecordCloneIsolatesHandlers(t *testing.T) {
	// Integration check: a real JSON handler downstream must see the
	// unmodified record. captureHandler.Handle iterates record attrs
	// non-destructively, but a real handler may mutate via AddAttrs.
	// Confirm a real JSON handler receives the right output even
	// when paired with another handler.
	jsonBuf := &bytes.Buffer{}
	jsonHandler := slog.NewJSONHandler(jsonBuf, &slog.HandlerOptions{Level: slog.LevelDebug})
	var captureCalls []string
	capture := &captureHandler{level: slog.LevelDebug, calls: &captureCalls}

	logger := slog.New(NewFanoutHandler(jsonHandler, capture))
	logger.Info("clone-check", slog.String("k", "v"))

	if !strings.Contains(jsonBuf.String(), `"msg":"clone-check"`) {
		t.Errorf("json handler missing msg, got %q", jsonBuf.String())
	}
	if !strings.Contains(jsonBuf.String(), `"k":"v"`) {
		t.Errorf("json handler missing k=v, got %q", jsonBuf.String())
	}
	if len(captureCalls) != 1 || !strings.Contains(captureCalls[0], "k=v") {
		t.Errorf("capture handler missing k=v: %v", captureCalls)
	}
}
