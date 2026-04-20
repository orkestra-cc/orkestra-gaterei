package policycoverage

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

// parseToFindings is a test helper that runs the file-level scanner against
// an inline source snippet. It synthesizes a packages.Package result just
// enough that scanFile can walk the AST — we deliberately avoid go/packages
// here so tests stay hermetic and don't require the surrounding module.
func parseToFindings(t *testing.T, src string) *Findings {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "addon.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	out := &Findings{Packages: 1}
	scanFile(fset, file, "internal/addons/fake/module.go", out)
	return out
}

func TestScan_DeclaredPermissionsAndCapabilities(t *testing.T) {
	src := `package fake

type PermissionSpec struct{ Key, Module, Description string; System bool }
type Capability struct{ ID, Module, Title, Description string; Published bool }
type FakeModule struct{}

func (m *FakeModule) Permissions() []PermissionSpec {
	return []PermissionSpec{
		{Key: "fake.thing.read", Module: "fake", Description: "read things"},
		{Key: "fake.thing.create", Module: "fake"},
	}
}

func (m *FakeModule) Capabilities() []Capability {
	return []Capability{
		{ID: "fake.access", Module: "fake", Title: "Fake", Published: true},
	}
}
`
	f := parseToFindings(t, src)
	if len(f.DeclaredPermissions) != 2 {
		t.Fatalf("expected 2 permissions, got %d: %+v", len(f.DeclaredPermissions), f.DeclaredPermissions)
	}
	byKey := map[string]Declaration{}
	for _, d := range f.DeclaredPermissions {
		byKey[d.Key] = d
	}
	if d, ok := byKey["fake.thing.read"]; !ok || d.Owner != "fake" {
		t.Errorf("missing or bad fake.thing.read: %+v", d)
	}
	if d, ok := byKey["fake.thing.create"]; !ok || d.Owner != "fake" {
		t.Errorf("missing or bad fake.thing.create: %+v", d)
	}
	if len(f.DeclaredCapabilities) != 1 || f.DeclaredCapabilities[0].Key != "fake.access" {
		t.Errorf("unexpected capabilities: %+v", f.DeclaredCapabilities)
	}
}

func TestScan_UsageSites(t *testing.T) {
	src := `package fake

type router struct{}
type mw struct{}
func (m *mw) RequirePermission(p string) func() {}
func (m *mw) RequireSystemPermission(p string) func() {}
func (m *mw) RequireCapability(c string) func() {}

func wire(m *mw) {
	m.RequirePermission("fake.thing.read")
	m.RequireSystemPermission("system.fake.admin")
	m.RequireCapability("fake.access")
	k := "dynamic.key"
	m.RequirePermission(k)
}
`
	f := parseToFindings(t, src)
	if len(f.UsedPermissions) != 2 {
		t.Fatalf("expected 2 permission usages, got %d: %+v", len(f.UsedPermissions), f.UsedPermissions)
	}
	if len(f.UsedCapabilities) != 1 || f.UsedCapabilities[0].Key != "fake.access" {
		t.Errorf("unexpected capability usage: %+v", f.UsedCapabilities)
	}
	if len(f.Dynamic) != 1 {
		t.Errorf("expected 1 dynamic call, got %d: %+v", len(f.Dynamic), f.Dynamic)
	}
}

func TestReconcile_FlagsDrift(t *testing.T) {
	f := &Findings{
		DeclaredPermissions: []Declaration{
			{Key: "fake.used", Owner: "fake", Pos: Position{File: "a.go", Line: 1}},
			{Key: "fake.dead", Owner: "fake", Pos: Position{File: "a.go", Line: 2}},
		},
		DeclaredCapabilities: []Declaration{
			{Key: "fake.access", Owner: "fake", Pos: Position{File: "a.go", Line: 3}},
			{Key: "fake.unused_cap", Owner: "fake", Pos: Position{File: "a.go", Line: 4}},
		},
		UsedPermissions: []Usage{
			{Key: "fake.used", Caller: CallerRequirePermission, Pos: Position{File: "b.go", Line: 10}},
			{Key: "fake.typo", Caller: CallerRequirePermission, Pos: Position{File: "b.go", Line: 11}},
		},
		UsedCapabilities: []Usage{
			{Key: "fake.access", Caller: CallerRequireCapability, Pos: Position{File: "b.go", Line: 12}},
			{Key: "fake.ghost_cap", Caller: CallerRequireCapability, Pos: Position{File: "b.go", Line: 13}},
		},
	}
	report := Reconcile(f, nil)

	wantCategories := map[string]Severity{
		"permission.used.undeclared:fake.typo":        SeverityError,
		"permission.declared.unused:fake.dead":        SeverityError,
		"capability.used.undeclared:fake.ghost_cap":   SeverityError,
		"capability.declared.unused:fake.unused_cap":  SeverityWarn,
	}
	found := map[string]Severity{}
	for _, d := range report.Diagnostics {
		found[d.Category+":"+d.Key] = d.Severity
	}
	for k, v := range wantCategories {
		if got, ok := found[k]; !ok {
			t.Errorf("missing diagnostic %s", k)
		} else if got != v {
			t.Errorf("diagnostic %s: want severity %s, got %s", k, v, got)
		}
	}
	if !report.HasErrors() {
		t.Errorf("expected HasErrors to be true, got false")
	}
}

func TestReconcile_BaselineSuppresses(t *testing.T) {
	f := &Findings{
		UsedPermissions: []Usage{
			{Key: "fake.typo", Caller: CallerRequirePermission, Pos: Position{File: "b.go", Line: 11}},
		},
	}
	baseline := map[string]bool{"permission.used.undeclared:fake.typo": true}
	report := Reconcile(f, baseline)
	if report.HasErrors() {
		t.Errorf("baseline should suppress error, got: %+v", report.Diagnostics)
	}
	if len(report.Diagnostics) != 0 {
		t.Errorf("expected no surviving diagnostics, got %d", len(report.Diagnostics))
	}
}

func TestReconcile_CedarReconciliation(t *testing.T) {
	f := &Findings{
		DeclaredPermissions: []Declaration{
			{Key: "system.users.admin", Owner: "authz"},
			{Key: "fake.other", Owner: "fake"},
		},
		UsedPermissions: []Usage{
			{Key: "system.users.admin", Caller: CallerRequireSystemPermission, Pos: Position{File: "b.go", Line: 1}},
			{Key: "fake.other", Caller: CallerRequirePermission, Pos: Position{File: "b.go", Line: 2}},
		},
		CedarActions: []string{"system.users.admin", "system.tenants.admin"},
	}
	report := Reconcile(f, nil)
	if report.HasErrors() {
		t.Errorf("unexpected errors: %+v", report.Diagnostics)
	}
	if len(report.Cedar.MatchedPermissions) != 1 || report.Cedar.MatchedPermissions[0] != "system.users.admin" {
		t.Errorf("unexpected matched: %+v", report.Cedar.MatchedPermissions)
	}
	if len(report.Cedar.UnmatchedCedar) != 1 || report.Cedar.UnmatchedCedar[0] != "system.tenants.admin" {
		t.Errorf("unexpected unmatched: %+v", report.Cedar.UnmatchedCedar)
	}
}

// Sanity-check the package compiles as a library (no side-effects on import).
var _ = ast.Inspect
var _ = token.NewFileSet
