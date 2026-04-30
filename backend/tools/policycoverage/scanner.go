// Package policycoverage scans the Orkestra backend for permission and
// capability key drift between route middleware ("what the route asks for")
// and module catalogs ("what the system declares exists").
//
// The scanner is a standalone cross-package AST walker rather than a
// go/analysis Analyzer because the reconciliation is inherently
// cross-package — a permission is declared in one module's module.go and
// consumed in another module's route registration. go/analysis works
// one package at a time and would need fact-passing to stitch that back
// together; go/packages lets us load everything once and walk freely.
//
// Phase 5.1 of the tenancy plan (ADR-0001 / project_tenancy_plan_v2.md):
// the scanner is the data-collection half; the reporter formats the diff;
// the CLI wraps both. Cedar-action reconciliation is also collected here
// but treated as informational — permission keys and Cedar action names
// are deliberately disjoint during Phase 1 shadow mode, and the report
// surfaces the overlap (or absence thereof) without failing CI.
package policycoverage

import (
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/tools/go/packages"
)

// Callers identifies one of the three gate middleware we track. The scanner
// groups usage sites by Caller so the report can say e.g. "permission X is
// used by RequirePermission but never declared" without losing the
// distinction between a permission and a capability consumer.
type Caller string

const (
	CallerRequirePermission       Caller = "RequirePermission"
	CallerRequireSystemPermission Caller = "RequireSystemPermission"
	CallerRequireCapability       Caller = "RequireCapability"
)

// Position is a filename + line pointer for diagnostic output. Stored as a
// flat string so the report doesn't need to carry the go/token.FileSet
// around; the scanner converts on the fly.
type Position struct {
	File string
	Line int
}

func (p Position) String() string { return fmt.Sprintf("%s:%d", p.File, p.Line) }

// Usage is a single call site of a gate middleware with a literal string
// key. Dynamic call sites (the key comes from a variable) are collected
// separately into Findings.Dynamic so they surface in the report as
// potentially-missed drift.
type Usage struct {
	Key    string
	Caller Caller
	Pos    Position
}

// Declaration is a single catalog entry — either a PermissionSpec.Key or a
// Capability.ID literal inside a module's Permissions()/Capabilities()
// method. The Owner field is the literal from the same composite literal
// (struct field Module), which lets the report group drift by module.
type Declaration struct {
	Key   string
	Owner string
	Pos   Position
}

// Findings is the raw collection the scanner produces, before any
// reconciliation. The reporter turns this into the coverage report.
type Findings struct {
	DeclaredPermissions  []Declaration
	DeclaredCapabilities []Declaration
	UsedPermissions      []Usage
	UsedCapabilities     []Usage
	Dynamic              []Usage // non-literal first argument — flagged so the audit is complete
	CedarActions         []string
	CedarSuffixes        []string // suffix literals referenced by context.action_suffix == "X" clauses
	CedarModules         []string // module literals referenced by context.action_module == "X" clauses
	Packages             int
}

// Scan loads every Go package under the given patterns and walks their AST,
// collecting declarations and usages. cedarDir (when non-empty) is scanned
// with a regex for Action::"..." literals so the report can show the
// overlap between permission keys and Cedar actions. Errors loading
// individual packages are logged but do not abort the scan — a partial
// report is more useful than no report.
func Scan(patterns []string, cedarDir string) (*Findings, error) {
	cfg := &packages.Config{
		Mode:  packages.NeedName | packages.NeedFiles | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo,
		Tests: false,
	}
	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		return nil, fmt.Errorf("policycoverage: load packages: %w", err)
	}

	findings := &Findings{Packages: len(pkgs)}
	for _, pkg := range pkgs {
		// Skip pure analyzer/tool packages — they declare helpers that
		// look like gates but never ship into production routing.
		if strings.HasPrefix(pkg.PkgPath, "github.com/orkestra/backend/tools/") {
			continue
		}
		for _, file := range pkg.Syntax {
			if file == nil {
				continue
			}
			// Ask the FileSet for the file name directly rather than
			// pairing pkg.Syntax[i] with pkg.CompiledGoFiles[i] — the
			// two slices are not required to line up across go/packages
			// versions, and the empty filenames that result turn the
			// report's Sites column into noise like ":42".
			filename := pkg.Fset.Position(file.Pos()).Filename
			scanFile(pkg.Fset, file, filename, findings)
		}
	}
	sort.Slice(findings.DeclaredPermissions, func(i, j int) bool {
		return findings.DeclaredPermissions[i].Key < findings.DeclaredPermissions[j].Key
	})
	sort.Slice(findings.DeclaredCapabilities, func(i, j int) bool {
		return findings.DeclaredCapabilities[i].Key < findings.DeclaredCapabilities[j].Key
	})
	sort.Slice(findings.UsedPermissions, func(i, j int) bool {
		if findings.UsedPermissions[i].Key != findings.UsedPermissions[j].Key {
			return findings.UsedPermissions[i].Key < findings.UsedPermissions[j].Key
		}
		return findings.UsedPermissions[i].Pos.String() < findings.UsedPermissions[j].Pos.String()
	})
	sort.Slice(findings.UsedCapabilities, func(i, j int) bool {
		if findings.UsedCapabilities[i].Key != findings.UsedCapabilities[j].Key {
			return findings.UsedCapabilities[i].Key < findings.UsedCapabilities[j].Key
		}
		return findings.UsedCapabilities[i].Pos.String() < findings.UsedCapabilities[j].Pos.String()
	})

	if cedarDir != "" {
		actions, suffixes, modules, err := scanCedar(cedarDir)
		if err != nil {
			return findings, fmt.Errorf("policycoverage: scan cedar: %w", err)
		}
		findings.CedarActions = actions
		findings.CedarSuffixes = suffixes
		findings.CedarModules = modules
	}
	return findings, nil
}

// scanFile walks a single parsed file, feeding usages and declarations into
// the accumulator. The two pattern we recognize:
//
//   - Method Permissions() / Capabilities() on a module receiver. The return
//     statement is a composite literal slice; each element is a struct
//     literal with a "Key" (permission) or "ID" (capability) field whose
//     RHS is a basic string literal. Elements whose Key is a non-literal
//     are ignored (we can't statically know their value).
//   - Call expression where the selector is RequirePermission /
//     RequireSystemPermission / RequireCapability and the first argument
//     is a basic string literal. Non-literal first args are collected as
//     Dynamic.
func scanFile(fset *token.FileSet, file *ast.File, filename string, out *Findings) {
	// Strip absolute paths down to a repo-relative path for stable output.
	// Host checkouts have ".../backend/internal/..." while the dev
	// container lives at "/app/internal/...". Anchor on "/internal/"
	// and keep everything from "internal/" onwards; that form is stable
	// regardless of where the scan runs.
	rel := filename
	if i := strings.Index(filename, "/internal/"); i >= 0 {
		rel = filename[i+1:] // keep the "internal/..." prefix
	}
	rel = filepath.ToSlash(rel)

	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			if node.Body == nil || node.Recv == nil || node.Name == nil {
				return true
			}
			switch node.Name.Name {
			case "Permissions":
				collectCatalog(fset, rel, node.Body, "Key", func(key, owner string, pos Position) {
					out.DeclaredPermissions = append(out.DeclaredPermissions, Declaration{Key: key, Owner: owner, Pos: pos})
				})
			case "Capabilities":
				collectCatalog(fset, rel, node.Body, "ID", func(key, owner string, pos Position) {
					out.DeclaredCapabilities = append(out.DeclaredCapabilities, Declaration{Key: key, Owner: owner, Pos: pos})
				})
			}
		case *ast.CallExpr:
			collectUsage(fset, rel, node, out)
		}
		return true
	})
}

// collectCatalog walks the body of a Permissions() or Capabilities()
// method and emits one callback per composite-literal entry. keyField is
// the struct field whose string value identifies the catalog entry
// ("Key" for permissions, "ID" for capabilities). The "Module" field is
// captured alongside, when present, as the owner tag.
func collectCatalog(fset *token.FileSet, relFile string, body *ast.BlockStmt, keyField string, emit func(key, owner string, pos Position)) {
	ast.Inspect(body, func(n ast.Node) bool {
		cl, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}
		var key, owner string
		for _, elt := range cl.Elts {
			kv, ok := elt.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			name, ok := kv.Key.(*ast.Ident)
			if !ok {
				continue
			}
			if name.Name == keyField {
				if s, ok := stringLit(kv.Value); ok {
					key = s
				}
			}
			if name.Name == "Module" {
				if s, ok := stringLit(kv.Value); ok {
					owner = s
				}
			}
		}
		if key != "" {
			p := fset.Position(cl.Pos())
			emit(key, owner, Position{File: relFile, Line: p.Line})
		}
		return true
	})
}

// collectUsage extracts RequirePermission / RequireSystemPermission /
// RequireCapability call sites. We match on the selector's final identifier
// regardless of the receiver type so a handful of alternate receivers
// (e.g. AuthMiddleware vs JWTValidator) both light up.
func collectUsage(fset *token.FileSet, relFile string, call *ast.CallExpr, out *Findings) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}
	if len(call.Args) == 0 {
		return
	}
	var caller Caller
	switch sel.Sel.Name {
	case "RequirePermission":
		caller = CallerRequirePermission
	case "RequireSystemPermission":
		caller = CallerRequireSystemPermission
	case "RequireCapability":
		caller = CallerRequireCapability
	default:
		return
	}
	p := fset.Position(call.Pos())
	pos := Position{File: relFile, Line: p.Line}
	key, ok := stringLit(call.Args[0])
	if !ok {
		out.Dynamic = append(out.Dynamic, Usage{Caller: caller, Pos: pos})
		return
	}
	u := Usage{Key: key, Caller: caller, Pos: pos}
	if caller == CallerRequireCapability {
		out.UsedCapabilities = append(out.UsedCapabilities, u)
	} else {
		out.UsedPermissions = append(out.UsedPermissions, u)
	}
}

// stringLit unwraps a BasicLit string expression, returning the underlying
// Go string value. Returns (,"", false) for non-literal expressions like
// identifiers, constants, or concatenations — those are reported as Dynamic.
func stringLit(e ast.Expr) (string, bool) {
	bl, ok := e.(*ast.BasicLit)
	if !ok || bl.Kind != token.STRING {
		return "", false
	}
	s, err := strconv.Unquote(bl.Value)
	if err != nil {
		return "", false
	}
	return s, true
}

// cedarActionRE matches Action::"name" literals in .cedar source. Cedar
// names are alphanumeric with dots and underscores — the character class
// matches the subset we actually use.
var cedarActionRE = regexp.MustCompile(`Action::"([a-zA-Z0-9._-]+)"`)

// cedarSuffixRE matches `context.action_suffix == "X"` (and the symmetric
// `"X" == context.action_suffix`) clauses. These are the indirect coverage
// path: a permission `foo.bar.read` is Cedar-reachable when some policy
// includes `context.action_suffix == "read"` even if no `Action::"foo.bar.read"`
// literal exists. Whitespace tolerance matches what the formatter emits.
var cedarSuffixRE = regexp.MustCompile(`(?:context\.action_suffix\s*==\s*"([a-zA-Z0-9_-]+)"|"([a-zA-Z0-9_-]+)"\s*==\s*context\.action_suffix)`)

// cedarModuleRE mirrors cedarSuffixRE for the `context.action_module == "X"`
// clause. Module coverage lets a single policy cover every permission under
// one module prefix (billing.*, payments.*, subscriptions.*) without
// enumerating each action — used by the org_billing role policy.
var cedarModuleRE = regexp.MustCompile(`(?:context\.action_module\s*==\s*"([a-zA-Z0-9_-]+)"|"([a-zA-Z0-9_-]+)"\s*==\s*context\.action_module)`)

// scanCedar reads every .cedar file in dir and returns the union of
// Action::"name" literals, context.action_suffix == "X" suffix literals,
// and context.action_module == "X" module literals across all files. All
// three are deduplicated and sorted for stable output.
func scanCedar(dir string) ([]string, []string, []string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, nil, err
	}
	actions := map[string]struct{}{}
	suffixes := map[string]struct{}{}
	modules := map[string]struct{}{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".cedar") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, nil, nil, err
		}
		src := string(data)
		for _, m := range cedarActionRE.FindAllStringSubmatch(src, -1) {
			actions[m[1]] = struct{}{}
		}
		for _, m := range cedarSuffixRE.FindAllStringSubmatch(src, -1) {
			// Either capture group 1 or 2 is set depending on operand order.
			lit := m[1]
			if lit == "" {
				lit = m[2]
			}
			if lit != "" {
				suffixes[lit] = struct{}{}
			}
		}
		for _, m := range cedarModuleRE.FindAllStringSubmatch(src, -1) {
			lit := m[1]
			if lit == "" {
				lit = m[2]
			}
			if lit != "" {
				modules[lit] = struct{}{}
			}
		}
	}
	return sortedSet(actions), sortedSet(suffixes), sortedSet(modules), nil
}

// sortedSet returns the keys of a set-shaped map as a sorted slice.
func sortedSet(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
