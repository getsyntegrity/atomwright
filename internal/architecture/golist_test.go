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
	"fmt"
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
// still broken the dependency direction, so tests are not a loophole for any
// layer edge. The one thing tests do change is third-party reach -- see
// importOrigin and ADR-0003.
func (p pkg) allImports() []string {
	merged := slices.Concat(p.Imports, p.TestImports, p.XTestImports)
	slices.Sort(merged)
	return slices.Compact(merged)
}

// importOrigin says which go list import set an import came from. ADR-0003
// amends ADR-0001 so a standard-library-only layer may take a third-party
// import through TestImports/XTestImports but never through Imports, and that
// is only decidable if the sets stay distinguishable.
type importOrigin int

const (
	// originProduction is an import from go list's Imports set: a file the
	// non-test build compiles and links into the shipped binary.
	originProduction importOrigin = iota
	// originTest is an import seen only in TestImports or XTestImports.
	// The Go toolchain excludes _test.go files from a non-test build, so
	// nothing reached this way is linked into a shipped binary.
	originTest
)

// classifiedImport is one import path together with where it came from.
type classifiedImport struct {
	path   string
	origin importOrigin
}

// classifiedImports returns every import of p attributed to an origin.
//
// FAIL CLOSED: an import that cannot be attributed to a known set is treated
// as a production import, because originProduction is the denying side of
// every ADR-0003 permission. Production also wins over test when a path
// appears in both sets: it is linked into the binary either way. So the only
// imports that gain the test-only permission are the ones positively proven
// to come from TestImports or XTestImports alone.
func (p pkg) classifiedImports() []classifiedImport {
	origins := make(map[string]importOrigin, len(p.Imports)+len(p.TestImports)+len(p.XTestImports))
	for _, path := range slices.Concat(p.TestImports, p.XTestImports) {
		origins[path] = originTest
	}
	for _, path := range p.Imports { // second, so production overwrites test
		origins[path] = originProduction
	}

	all := p.allImports()
	out := make([]classifiedImport, 0, len(all))
	for _, path := range all {
		origin, known := origins[path]
		if !known {
			origin = originProduction
		}
		out = append(out, classifiedImport{path: path, origin: origin})
	}
	return out
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
	stdlibOut  string
	stdlibErr  error
)

// cachingStdlibRunner memoises `go list std` across the test binary. The
// standard library is the same for every fixture, and loadGraph runs once per
// fixture, so without this each one pays for another subprocess.
//
// The cache is a decoration of the runner rather than a branch inside
// buildGraph on purpose. buildGraph issues all three subcommands
// unconditionally, which is what makes their order observable, and a fake
// runner handed to a spec is never wrapped -- so a spec sees every call the
// real one would make.
func cachingStdlibRunner(run goRunner) goRunner {
	return func(dir string, extraEnv []string, args ...string) (string, error) {
		if strings.Join(args, " ") != "list std" {
			return run(dir, extraEnv, args...)
		}
		stdlibOnce.Do(func() { stdlibOut, stdlibErr = run(dir, extraEnv, args...) })
		return stdlibOut, stdlibErr
	}
}

// parseStdlib turns `go list std` output into a set. Using toolchain truth
// rather than the "first path segment has no dot" heuristic matters: that
// heuristic misclassifies vendored and versioned paths.
func parseStdlib(out string) map[string]bool {
	set := make(map[string]bool, 512)
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			set[line] = true
		}
	}
	return set
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

	g, err := buildGraph(cachingStdlibRunner(runGo), dir, extraEnv)
	if err != nil {
		t.Fatalf("%v", err)
	}
	return g
}

// goRunner executes one go subcommand. Graph loading goes through this
// indirection rather than calling runGo directly because the Go toolchain is
// the one real collaborator this package has: every rule is a pure function
// over a graph, and the only I/O in the package is the three subprocesses that
// produce it. Standing that boundary in is what makes the failure paths
// specifiable without a broken Go installation.
type goRunner func(dir string, extraEnv []string, args ...string) (string, error)

// buildGraph builds the import graph of the module rooted at dir.
//
// It reports failures instead of ending the test, which is what separates it
// from loadGraph: the interesting thing about this function is what it does
// when the toolchain does not cooperate, and a t.Fatalf cannot be observed.
func buildGraph(run goRunner, dir string, extraEnv []string) (*graph, error) {
	modulePath, err := run(dir, extraEnv, "list", "-m")
	if err != nil {
		return nil, fmt.Errorf("resolving module path: %w", err)
	}

	listed, err := run(dir, extraEnv, "list", "-json", "./...")
	if err != nil {
		return nil, fmt.Errorf("listing packages: %w", err)
	}

	// The standard-library set is the last of the three, and it is the one
	// call deliberately not made in dir with extraEnv: the standard library
	// does not depend on which module is being loaded, and a fixture's
	// GOWORK=off has nothing to say about it.
	//
	// It fails like the other two rather than ending the test. A missing
	// stdlib set classifies every standard-library import as external, which
	// would turn a healthy tree into a wall of violations -- a failure worth
	// naming for what it is.
	stdlibListed, err := run(".", nil, "list", "std")
	if err != nil {
		return nil, fmt.Errorf("listing the standard library: %w", err)
	}

	g := &graph{
		modulePath: strings.TrimSpace(modulePath),
		stdlib:     parseStdlib(stdlibListed),
	}
	decoder := json.NewDecoder(strings.NewReader(listed))
	for decoder.More() {
		var p pkg
		if err := decoder.Decode(&p); err != nil {
			return nil, fmt.Errorf("decoding go list -json output: %w", err)
		}
		g.packages = append(g.packages, p)
	}
	// An empty package list is the silent-pass hazard this whole package is
	// built to avoid: every rule reports zero violations over zero packages,
	// which is indistinguishable from a clean tree. Fail instead.
	if len(g.packages) == 0 {
		return nil, fmt.Errorf("go list -json ./... in %s returned no packages", dir)
	}
	return g, nil
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
