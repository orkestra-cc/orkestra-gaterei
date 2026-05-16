package module

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

// loggerCapturingModule records the deps.Logger it received in Init so
// the test can verify the registry passed a per-module-scoped logger
// (ADR-0005 §1.4). Goes through the same Init code path real modules
// do — no special-casing.
type loggerCapturingModule struct {
	BaseModule
	name      string
	gotLogger *slog.Logger
}

func (m *loggerCapturingModule) Name() string             { return m.name }
func (m *loggerCapturingModule) Category() ModuleCategory { return CategoryToggleable }
func (m *loggerCapturingModule) Init(deps *Dependencies) error {
	m.gotLogger = deps.Logger
	return nil
}

func TestModuleRegistry_InitAllDecoratesPerModuleLogger(t *testing.T) {
	buf := &bytes.Buffer{}
	root := slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	reg := NewModuleRegistry(root)
	ragModule := &loggerCapturingModule{name: "rag"}
	authModule := &loggerCapturingModule{name: "auth"}
	reg.Register(ragModule)
	reg.Register(authModule)

	deps := &Dependencies{
		Services: NewServiceRegistry(),
		Logger:   root,
	}
	if err := reg.InitAll(deps); err != nil {
		t.Fatalf("InitAll: %v", err)
	}

	for _, tc := range []struct {
		name   string
		module *loggerCapturingModule
	}{
		{"rag", ragModule},
		{"auth", authModule},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.module.gotLogger == nil {
				t.Fatalf("module %s did not receive a logger", tc.name)
			}
			buf.Reset()
			tc.module.gotLogger.Info("hello")
			line := buf.String()
			var parsed map[string]any
			if err := json.Unmarshal(bytes.TrimSpace([]byte(line)), &parsed); err != nil {
				t.Fatalf("invalid JSON %q: %v", line, err)
			}
			if parsed["module"] != tc.name {
				t.Errorf("module attr = %v, want %q (line: %s)", parsed["module"], tc.name, line)
			}
		})
	}
}

func TestModuleRegistry_DepsForSharesPointers(t *testing.T) {
	root := slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil))
	reg := NewModuleRegistry(root)

	services := NewServiceRegistry()
	deps := &Dependencies{Services: services, Logger: root}

	scoped := reg.depsFor(deps, "rag")
	if scoped == deps {
		t.Errorf("depsFor returned the same pointer; must be a shallow copy")
	}
	if scoped.Services != services {
		t.Errorf("Services pointer must be shared across the copy")
	}
	if scoped.Logger == deps.Logger {
		t.Errorf("Logger must be replaced, not shared")
	}
}

func TestModuleRegistry_DepsForNilSafe(t *testing.T) {
	root := slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil))
	reg := NewModuleRegistry(root)

	if got := reg.depsFor(nil, "rag"); got != nil {
		t.Errorf("depsFor(nil, ...) = %v, want nil", got)
	}

	deps := &Dependencies{Services: NewServiceRegistry()} // Logger=nil
	scoped := reg.depsFor(deps, "rag")
	if scoped == nil {
		t.Fatalf("depsFor with nil Logger should still return a copy")
	}
	if scoped.Logger != nil {
		t.Errorf("nil Logger must remain nil to avoid hiding wiring bugs; got %v", scoped.Logger)
	}
}

func TestModuleRegistry_DepsForEmptyNameSkipsDecoration(t *testing.T) {
	buf := &bytes.Buffer{}
	root := slog.New(slog.NewJSONHandler(buf, nil))
	reg := NewModuleRegistry(root)
	deps := &Dependencies{Services: NewServiceRegistry(), Logger: root}

	scoped := reg.depsFor(deps, "")
	scoped.Logger.Info("bare")
	if strings.Contains(buf.String(), `"module"`) {
		t.Errorf("empty module name must not stamp a module attr, got %s", buf.String())
	}
}
