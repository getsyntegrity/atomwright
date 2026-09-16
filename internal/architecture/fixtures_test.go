package architecture

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/pablogore/go-specs/specs"
)

// Every rule in this package ships with at least one deliberately-violating
// module under testdata/. A rule with no negative fixture is not a rule, it is
// a hope: a path-based check that silently stopped matching reports zero
// violations, which is indistinguishable from a clean tree. The fixtures are
// what make a green run mean something.
//
// Each fixture is a self-contained Go module with its own go.mod, so the same
// go list machinery that inspects the real tree inspects the fixtures too --
// the rules are never handed a hand-built graph they would not see in
// production. The testdata/ directory name is what keeps the toolchain from
// compiling these deliberately-wrong packages as part of the real module.
//
// Fixtures are loaded with GOWORK=off so that an inherited workspace file
// cannot change how they load -- see loadGraph.
//
// CONTRIBUTING's module-layout rule (ATOM-BOOT-004) says never to add a go.mod
// under internal/. These fixture modules are the documented exception, and
// CONTRIBUTING names it: a testdata/ module is invisible to the toolchain --
// it is never built, tested, or released with the real module -- and go list
// needs a module boundary to resolve the deliberately-wrong import graphs at
// all. The rule is about the production tree, and these are not in it.

// fixtureEnv is the environment every fixture load needs. See above.
var fixtureEnv = []string{"GOWORK=off"}

// edge is one expected (package, import) pair in a fixture's violations.
type edge struct {
	pkg     string
	imports string
}

type fixtureCase struct {
	// dir is the fixture module under testdata/.
	dir string
	// why states the violation the fixture exists to prove is caught.
	why string
	// wantRules is the EXACT set of rules expected to fire. Exact, not
	// "at least": a rule firing where it should not is as much a defect as
	// a rule staying silent where it should not.
	wantRules []string
	// wantEdges are violations that must appear by name, so a fixture
	// cannot pass by failing for some unrelated reason.
	wantEdges []edge
}

func fixtureCases() []fixtureCase {
	return []fixtureCase{
		{
			dir:       "domain_imports_application",
			why:       "the domain is the centre; it must not depend on the layer above it",
			wantRules: []string{"dependencyRule", "moduleDependencyRule"},
			wantEdges: []edge{{"internal/domain/execution", "internal/application"}},
		},
		{
			dir:       "application_imports_adapter",
			why:       "adapters call application services, never the other way round",
			wantRules: []string{"dependencyRule", "moduleDependencyRule"},
			wantEdges: []edge{{"internal/application", "adapters/mcp"}},
		},
		{
			dir:       "platform_imports_domain",
			why:       "platform is domain-agnostic infrastructure, and is not a permitted importer of the domain",
			wantRules: []string{"dependencyRule", "restrictedImportRule", "moduleDependencyRule"},
			wantEdges: []edge{{"platform/logging", "internal/domain/execution"}},
		},
		{
			// Acceptance criterion 5 of issue #65: a cmd/* package reaching
			// past the composition root into a concrete adapter, platform,
			// application, or domain package must fail.
			dir:       "cmd_imports_layers_directly",
			why:       "cmd/* sees only internal/bootstrap and the standard library; internal/bootstrap is the single composition root",
			wantRules: []string{"restrictedImportRule", "compositionRootRule", "moduleDependencyRule"},
			wantEdges: []edge{
				{"cmd/atomwright", "adapters/vcs/gitworktree"},
				{"cmd/atomwright", "platform/logging"},
				{"cmd/second", "internal/application"},
				{"cmd/second", "internal/domain/execution"},
			},
		},
		{
			// The importing package belongs to no ADR-0001 layer, so its own
			// imports are not governed -- but the domain is still protected
			// as a target. Only restrictedImportRule may fire here.
			dir:       "foreign_imports_domain",
			why:       "only bootstrap, application, adapters, and the domain itself may import internal/domain/*",
			wantRules: []string{"restrictedImportRule"},
			wantEdges: []edge{{"internal/tooling", "internal/domain/execution"}},
		},
		{
			// Not an ADR-0001 anti-edge, and not in the allowlist either.
			// This is the fixture that proves the allowlist is default-deny
			// rather than a list of remembered prohibitions.
			dir:       "application_imports_platform",
			why:       "an edge absent from the ADR-0001 table is denied even though no anti-edge names it",
			wantRules: []string{"moduleDependencyRule"},
			wantEdges: []edge{{"internal/application", "platform/logging"}},
		},
		{
			dir:       "domain_imports_non_layer",
			why:       "ADR-0001 confines internal/domain/* to the standard library and other domain packages",
			wantRules: []string{"moduleDependencyRule"},
			wantEdges: []edge{{"internal/domain/execution", "internal/tooling"}},
		},
		{
			// ADR-0003 acceptance criterion 1: the amendment is scoped to
			// test files, so production code is untouched by it.
			dir:       "domain_imports_thirdparty",
			why:       "a third-party import in production code of internal/domain/* is still denied after ADR-0003",
			wantRules: []string{"moduleDependencyRule"},
			wantEdges: []edge{{"internal/domain/execution", "example.test/thirdparty"}},
		},
		{
			// ADR-0003 acceptance criterion 2: the permission is pinned by
			// a fixture rather than by the absence of one.
			dir:       "domain_test_imports_thirdparty",
			why:       "a third-party import reached only from a _test.go file is never in the shipped binary, so ADR-0003 permits it",
			wantRules: nil,
		},
		{
			// The positive control. Without it, a rule set that flagged
			// everything would pass every negative fixture above.
			dir:       "allowed_tree",
			why:       "every edge ADR-0001 allows, and nothing else, must produce no violations",
			wantRules: nil,
		},
	}
}

// TestFixtures is the meta-suite: it proves each rule actually fails on the
// violation it claims to catch, and that the claims themselves stay honest.
//
// The fixture matrix is enumerated as one generated It per fixture rather than
// through the go-specs path builder. s.Paths explores a parameter space, taking
// the Cartesian product of independent dimensions; this is not one. It is an
// explicit list of (dir, wantRules, wantEdges, why) fixtures, each a hand-chosen
// violation paired with the exact rules it must provoke, and no two of them
// combine. Generating the specs in a loop is that same enumeration, is
// deterministic, and names each fixture's claim.
func TestFixtures(t *testing.T) {
	specs.Describe(t, "the testdata fixtures that keep the rules honest", func(s *specs.Spec) {
		s.When("a fixture module is loaded and checked", func(s *specs.Spec) {
			for _, tc := range fixtureCases() {
				s.It("proves "+tc.dir+": "+tc.why, func(ctx *specs.Context) {
					dir, err := filepath.Abs(filepath.Join("testdata", tc.dir))
					if err != nil {
						ctx.T.Fatalf("resolving fixture path: %v", err)
					}
					found := checkAll(loadGraph(ctx.T, dir, fixtureEnv...))

					gotRules := firedRules(found)
					wantRules := slices.Clone(tc.wantRules)
					slices.Sort(wantRules)
					if !slices.Equal(gotRules, wantRules) {
						ctx.T.Errorf("fixture %s (%s)\nrules fired: %v\nrules wanted: %v\n\nviolations:\n%s",
							tc.dir, tc.why, gotRules, wantRules, formatViolations(found))
					}
					ctx.Expect(slices.Equal(gotRules, wantRules)).To(specs.BeTrue())

					// Naming the edges is what stops a fixture passing because
					// the right rule fired for the wrong import.
					for _, want := range tc.wantEdges {
						if !containsEdge(found, want) {
							ctx.T.Errorf("fixture %s: no violation reported for %s -> %s\n\nviolations:\n%s",
								tc.dir, want.pkg, want.imports, formatViolations(found))
						}
						ctx.Expect(containsEdge(found, want)).To(specs.BeTrue())
					}
				})
			}
		})

		s.When("the rule set and the fixtures are compared", func(s *specs.Spec) {
			// The completeness bar issue #65 sets: a rule without a fixture
			// proving it fails is not considered done. Add a rule without a
			// fixture and this fails, which is the point.
			s.It("has a fixture proving every rule fails on the violation it targets", func(ctx *specs.Context) {
				covered := map[string]string{}
				for _, tc := range fixtureCases() {
					for _, name := range tc.wantRules {
						if _, ok := covered[name]; !ok {
							covered[name] = tc.dir
						}
					}
				}

				for _, r := range allRules() {
					_, ok := covered[r.name()]
					if !ok {
						ctx.T.Errorf("rule %s has no negative testdata fixture: add one under testdata/ proving the rule fails on the violation it targets", r.name())
					}
					ctx.Expect(ok).To(specs.BeTrue())
				}
			})

			// Guards the meta-suite against its own typo: a fixture expecting
			// a rule name nothing produces would assert nothing at all.
			s.It("expects no rule name the rule set does not produce", func(ctx *specs.Context) {
				known := make([]string, 0, len(allRules()))
				for _, r := range allRules() {
					known = append(known, r.name())
				}

				for _, tc := range fixtureCases() {
					for _, name := range tc.wantRules {
						if !slices.Contains(known, name) {
							ctx.T.Errorf("fixture %s expects unknown rule %q; known rules: %s", tc.dir, name, strings.Join(known, ", "))
						}
						ctx.Expect(slices.Contains(known, name)).To(specs.BeTrue())
					}
				}
			})

			// The generated specs above are named after the fixture
			// directory, so two cases sharing one directory would read as two
			// independent claims while proving the same thing once.
			s.It("registers each fixture directory exactly once", func(ctx *specs.Context) {
				seen := map[string]bool{}
				for _, tc := range fixtureCases() {
					if seen[tc.dir] {
						ctx.T.Errorf("fixture directory %s is registered more than once", tc.dir)
					}
					ctx.Expect(seen[tc.dir]).To(specs.BeFalse())
					seen[tc.dir] = true
				}
				specs.EqualTo(ctx, len(seen), len(fixtureCases()))
			})
		})
	})
}

func firedRules(found []violation) []string {
	var names []string
	for _, v := range found {
		if !slices.Contains(names, v.rule) {
			names = append(names, v.rule)
		}
	}
	slices.Sort(names)
	return names
}

func containsEdge(found []violation, want edge) bool {
	for _, v := range found {
		if v.pkg == want.pkg && v.imports == want.imports {
			return true
		}
	}
	return false
}
