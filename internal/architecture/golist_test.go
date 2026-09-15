// The architecture package is test-only. It makes the ADR-0001 dependency
// table executable (issue #65, ATOM-BOOT-005) by shelling out to the Go
// toolchain -- go list -m for the module path, go list -json ./... for the
// package import graph, and go list std for the standard-library set -- and
// evaluating that graph against a default-deny allowlist expressed as Go data.
//
// Nothing here ships in a binary: every file in this package is a _test.go
// file, so the rules cost the production build nothing and cannot be imported
// by accident.
//
// Known limit of a go list-based mechanism, stated up front: go list sees
// import edges between packages, not what a package does with what it imports.
// Every edge and anti-edge in the ADR-0001 table is distinguishable at the
// package-path level and is therefore enforced here. The one thing this
// mechanism cannot see is intent inside an allowed edge -- see the comment on
// restrictedImportRule in rules_test.go.
package architecture

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
)

// layer is an ADR-0001 architectural layer. Classification is purely
// package-path based, which is what makes every rule in this package
// mechanically decidable.
type layer string

const (
	layerCmd         layer = "cmd/*"
	layerBootstrap   layer = "internal/bootstrap"
	layerApplication layer = "internal/application"
	layerDomain      layer = "internal/domain/*"
	layerPlatform    layer = "platform/*"
	layerAdapters    layer = "adapters/*"

	// layerUngoverned is a module-local package that belongs to no layer in
	// the ADR-0001 table -- after ADR-0002 stripped the pre-ATOM-BOOT tree,
	// the only such package is this test package itself. Its own imports are
	// not checked, but it is still a governed *target*: no governed layer may
	// import it (moduleDependencyRule) and it may not import the domain
	// (restrictedImportRule).
	layerUngoverned layer = "ungoverned"

	// layerExternal is a package from another module (third party).
	layerExternal layer = "external"

	// layerStdlib is the Go standard library, allowed from everywhere.
	layerStdlib layer = "stdlib"
)

// governedSourceLayers are the layers ATOM-BOOT created. Only packages in
// these layers have their outgoing imports checked.
var governedSourceLayers = []layer{
	layerCmd,
	layerBootstrap,
	layerApplication,
	layerDomain,
	layerPlatform,
	layerAdapters,
}

func isGovernedSource(l layer) bool { return slices.Contains(governedSourceLayers, l) }

// pkg is the slice of `go list -json` output this package needs.
type pkg struct {
	ImportPath   string
	Imports      []string
	TestImports  []string
	XTestImports []string
}

// allImports merges production, in-package test, and external test imports.
// A domain package that reaches for an adapter only from its test file has
// still broken the dependency direction, so tests are not a loophole.
func (p pkg) allImports() []string {
	merged := slices.Concat(p.Imports, p.TestImports, p.XTestImports)
	slices.Sort(merged)
	return slices.Compact(merged)
}

// graph is an import graph plus everything needed to classify a path.
type graph struct {
	modulePath string
	packages   []pkg
	stdlib     map[string]bool
}

// rel returns the module-relative path of importPath, and whether importPath
// belongs to this module at all.
func (g *graph) rel(importPath string) (string, bool) {
	if importPath == g.modulePath {
		return ".", true
	}
	prefix := g.modulePath + "/"
	if !strings.HasPrefix(importPath, prefix) {
		return "", false
	}
	return strings.TrimPrefix(importPath, prefix), true
}

// classify maps an import path to its ADR-0001 layer.
func (g *graph) classify(importPath string) layer {
	if g.stdlib[importPath] {
		return layerStdlib
	}
	rel, local := g.rel(importPath)
	if !local {
		return layerExternal
	}
	switch {
	case rel == "internal/bootstrap" || strings.HasPrefix(rel, "internal/bootstrap/"):
		return layerBootstrap
	case rel == "internal/application" || strings.HasPrefix(rel, "internal/application/"):
		return layerApplication
	case strings.HasPrefix(rel, "internal/domain/"):
		return layerDomain
	case strings.HasPrefix(rel, "platform/"):
		return layerPlatform
	case strings.HasPrefix(rel, "adapters/"):
		return layerAdapters
	case strings.HasPrefix(rel, "cmd/"):
		return layerCmd
	default:
		return layerUngoverned
	}
}

// display renders an import path the way a violation message should show it:
// module-relative for module-local packages, verbatim for anything else.
func (g *graph) display(importPath string) string {
	if rel, local := g.rel(importPath); local {
		return rel
	}
	return importPath
}

var (
	stdlibOnce sync.Once
	stdlibSet  map[string]bool
	stdlibErr  error
)

// stdlibPackages shells out to `go list std` once per test binary. The result
// is toolchain truth rather than the "first path segment has no dot"
// heuristic, which misclassifies vendored and versioned paths.
func stdlibPackages(t *testing.T) map[string]bool {
	t.Helper()
	stdlibOnce.Do(func() {
		out, err := runGo(".", nil, "list", "std")
		if err != nil {
			stdlibErr = err
			return
		}
		stdlibSet = make(map[string]bool, 512)
		for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
			if line = strings.TrimSpace(line); line != "" {
				stdlibSet[line] = true
			}
		}
	})
	if stdlibErr != nil {
		t.Fatalf("go list std: %v", stdlibErr)
	}
	return stdlibSet
}

// runGo executes a go subcommand in dir with extra environment entries.
func runGo(dir string, extraEnv []string, args ...string) (string, error) {
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), extraEnv...)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", &goListError{args: args, dir: dir, stderr: stderr.String(), err: err}
	}
	return string(out), nil
}

type goListError struct {
	args   []string
	dir    string
	stderr string
	err    error
}

func (e *goListError) Error() string {
	return "go " + strings.Join(e.args, " ") + " in " + e.dir + ": " + e.err.Error() + "\n" + e.stderr
}

// loadGraph builds the import graph of the module rooted at dir.
//
// extraEnv exists for the testdata fixtures: each fixture is its own module
// nested under testdata/, so it is loaded with GOWORK=off. ATOM-BOOT-004
// decided this repository has no go.work, which makes that a no-op here today
// -- but a workspace file inherited from a contributor's environment, or a
// GOWORK pointing anywhere at all, would make the toolchain reject a fixture
// module for not being listed in it. Pinning the variable keeps the fixtures
// loading identically on every machine.
func loadGraph(t *testing.T, dir string, extraEnv ...string) *graph {
	t.Helper()

	modulePath, err := runGo(dir, extraEnv, "list", "-m")
	if err != nil {
		t.Fatalf("resolving module path: %v", err)
	}

	listed, err := runGo(dir, extraEnv, "list", "-json", "./...")
	if err != nil {
		t.Fatalf("listing packages: %v", err)
	}

	g := &graph{
		modulePath: strings.TrimSpace(modulePath),
		stdlib:     stdlibPackages(t),
	}
	decoder := json.NewDecoder(strings.NewReader(listed))
	for decoder.More() {
		var p pkg
		if err := decoder.Decode(&p); err != nil {
			t.Fatalf("decoding go list -json output: %v", err)
		}
		g.packages = append(g.packages, p)
	}
	if len(g.packages) == 0 {
		t.Fatalf("go list -json ./... in %s returned no packages", dir)
	}
	return g
}

// moduleRoot walks up from the test's working directory to the directory
// holding go.mod, so the rules are evaluated against the whole module rather
// than just internal/architecture.
func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("no go.mod found above %s", dir)
		}
		dir = parent
	}
}
