// The suite is written with go-specs, like every other suite in the module.
// Its third-party import is unconstrained here: internal/architecture belongs
// to no ADR-0001 layer, so its own imports are ungoverned -- and even in a
// standard-library-only layer this import would be permitted, because it
// appears only in a _test.go file (ADR-0003).
//
// Unlike the rest of the module, these specs live in the package rather than
// in an external one. That is forced, not preferred: every file in this
// package is a _test.go file, so the rules, the allowlist, and the graph have
// no exported surface an external test package could reach.
package architecture

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/pablogore/go-specs/mock"
	"github.com/pablogore/go-specs/specs"
)

// TestArchitecture is the rule that guards the tree, plus the specs that keep
// the rule honest about itself.
//
// When the dependency-direction spec fails, the fix is one of exactly two
// things, and which one is a review decision, not a convenience call:
//
//  1. the import is wrong -- move the code so the dependency points inward; or
//  2. the edge is genuinely legitimate -- amend the ADR-0001 table and
//     allowedLayerEdges together, in the same change, with the justification
//     in the pull request.
//
// Growing the allowlist without that justification is treated exactly like the
// violation it would otherwise catch.
func TestArchitecture(t *testing.T) {
	specs.Describe(t, "the ADR-0001 dependency direction", func(s *specs.Spec) {
		s.When("the rules run against the real module", func(s *specs.Spec) {
			s.It("finds no package importing across a forbidden boundary", func(ctx *specs.Context) {
				g := loadGraph(ctx.T, moduleRoot(ctx.T))

				found := checkAll(g)
				if len(found) > 0 {
					ctx.T.Errorf("%d dependency-direction violation(s) in %s:\n\n%s",
						len(found), g.modulePath, formatViolations(found))
				}
				specs.EqualTo(ctx, len(found), 0)
			})

			// A path-based classifier that matches nothing still reports zero
			// violations, which looks identical to a clean tree. If a layer
			// disappears from the module, that is a fact worth failing on
			// rather than a suite that passes by checking nothing.
			//
			// internal/bootstrap and cmd/* are in the expectation now that #66
			// ATOM-BOOT-006 has introduced them. compositionRootRule is the
			// rule most exposed to a silent classifier match: if either
			// package vanished, it would report zero violations and read as
			// a clean tree.
			s.It("still classifies a package into every layer the module is supposed to have", func(ctx *specs.Context) {
				g := loadGraph(ctx.T, moduleRoot(ctx.T))

				seen := map[layer]int{}
				for _, p := range g.packages {
					if _, local := g.rel(p.ImportPath); local {
						seen[g.classify(p.ImportPath)]++
					}
				}

				for _, l := range []layer{
					layerApplication, layerDomain, layerPlatform,
					layerAdapters, layerCmd, layerBootstrap,
				} {
					if seen[l] == 0 {
						ctx.T.Errorf("no package classified as %s: either the layer was removed or the classifier stopped matching it", l)
					}
					ctx.Expect(seen[l] > 0).To(specs.BeTrue())
				}
			})
		})

		// These are the properties the allowlist has to keep to be an
		// allowlist at all rather than a list of remembered prohibitions.
		s.When("the allowlist is read as data", func(s *specs.Spec) {
			s.It("lists no edge twice", func(ctx *specs.Context) {
				seen := map[layerEdge]bool{}
				for _, edge := range allowedLayerEdges {
					key := layerEdge{from: edge.from, to: edge.to}
					if seen[key] {
						ctx.T.Errorf("duplicate allowlist entry %s -> %s", edge.from, edge.to)
					}
					ctx.Expect(seen[key]).To(specs.BeFalse())
					seen[key] = true
				}
			})

			s.It("names only ADR-0001 layers on both ends of every edge", func(ctx *specs.Context) {
				for _, edge := range allowedLayerEdges {
					ctx.Expect(isGovernedSource(edge.from)).To(specs.BeTrue())
					ctx.Expect(isGovernedSource(edge.to)).To(specs.BeTrue())
				}
			})

			// The load-bearing one: a blanket self-edge would quietly let any
			// layer sprawl internally. Only the two intra-layer edges
			// ADR-0001 actually states are permitted.
			s.It("permits a layer to import itself only where ADR-0001 says so", func(ctx *specs.Context) {
				allowedSelfEdges := []layer{layerPlatform, layerDomain}

				for _, edge := range allowedLayerEdges {
					if edge.from != edge.to {
						continue
					}
					if !slices.Contains(allowedSelfEdges, edge.from) {
						ctx.T.Errorf("allowlist entry %s -> %s: ADR-0001 states an intra-layer edge only for %v", edge.from, edge.to, allowedSelfEdges)
					}
					ctx.Expect(slices.Contains(allowedSelfEdges, edge.from)).To(specs.BeTrue())
				}
			})

			// A missing entry in either policy map silently reads as deny,
			// which is the safe direction but not an answer anyone wrote
			// down. Every governed layer states both, explicitly.
			s.It("states a production third-party policy for every governed layer", func(ctx *specs.Context) {
				for _, l := range governedSourceLayers {
					_, stated := externalImportsAllowed[l]
					if !stated {
						ctx.T.Errorf("layer %s has no third-party import policy: every governed layer must state one explicitly, since a missing entry silently reads as deny", l)
					}
					ctx.Expect(stated).To(specs.BeTrue())
				}
			})

			s.It("states a test-only third-party policy for every governed layer", func(ctx *specs.Context) {
				for _, l := range governedSourceLayers {
					_, stated := testOnlyExternalImportsAllowed[l]
					if !stated {
						ctx.T.Errorf("layer %s has no ADR-0003 test-only third-party policy: a missing entry silently reads as deny", l)
					}
					ctx.Expect(stated).To(specs.BeTrue())
				}
			})
		})

		// The two halves of ADR-0001 have to stay consistent: if they ever
		// contradicted, dependencyRule and moduleDependencyRule would
		// disagree about the same import and the table would have stopped
		// meaning one thing.
		s.When("the allowlist and the anti-edges are compared", func(s *specs.Spec) {
			s.It("classifies no edge as both allowed and forbidden", func(ctx *specs.Context) {
				for _, forbidden := range forbiddenLayerEdges {
					if layerEdgeAllowed(forbidden.from, forbidden.to) {
						ctx.T.Errorf("%s -> %s is both an ADR-0001 anti-edge and an allowlist entry", forbidden.from, forbidden.to)
					}
					ctx.Expect(layerEdgeAllowed(forbidden.from, forbidden.to)).To(specs.BeFalse())
				}
			})

			s.It("agrees with itself about which layers may import the domain", func(ctx *specs.Context) {
				for _, l := range governedSourceLayers {
					if l == layerDomain {
						continue
					}
					listed := slices.Contains(domainImporters, l)
					allowed := layerEdgeAllowed(l, layerDomain)
					if listed != allowed {
						ctx.T.Errorf("domainImporters and allowedLayerEdges disagree about %s -> %s", l, layerDomain)
					}
					specs.EqualTo(ctx, listed, allowed)
				}
			})
		})

		// A broken build has to say which package broke which rule, or the
		// failure costs more to diagnose than it saves.
		s.When("a violation is rendered for a failing build", func(s *specs.Spec) {
			s.It("names the package, the import, the rule, and the reason", func(ctx *specs.Context) {
				v := violation{
					rule:    "dependencyRule",
					pkg:     "internal/domain/execution",
					imports: "adapters/mcp",
					reason:  "the domain must not know about any concrete I/O implementation",
				}

				got := v.String()
				for _, want := range []string{v.pkg, v.imports, v.rule, v.reason} {
					if !strings.Contains(got, want) {
						ctx.T.Errorf("violation message %q does not name %q", got, want)
					}
					ctx.Expect(strings.Contains(got, want)).To(specs.BeTrue())
				}
			})
		})
	})
}

// TestToolchainBoundary specifies what graph loading does when the Go
// toolchain -- the only collaborator in this package -- does not cooperate.
//
// This is the one place a test double belongs here. Every rule is a pure
// function over a graph and gets no double at all; the fixtures under
// testdata/ run the real `go list` on purpose, because a hand-built graph
// would not be the graph production sees. What the real toolchain cannot show
// is its own failure modes, and those are exactly the modes that decide
// whether a green run means anything.
func TestToolchainBoundary(t *testing.T) {
	specs.Describe(t, "loading the import graph from the Go toolchain", func(s *specs.Spec) {
		s.When("the toolchain answers normally", func(s *specs.Spec) {
			s.It("issues exactly three subcommands, in the order the graph depends on", func(ctx *specs.Context) {
				run, calls := recordingRunner(nil)

				g, err := buildGraph(run, "/somewhere", []string{"GOWORK=off"})

				ctx.Expect(err == nil).To(specs.BeTrue())
				specs.EqualTo(ctx, g.modulePath, "example.test/mod")
				specs.EqualTo(ctx, len(g.packages), 1)

				// Order matters: the module path is what every import in the
				// listing is classified against, so it has to be resolved
				// first. The count is asserted too, because an extra
				// subprocess nobody specified is exactly how `go list std`
				// escaped this boundary before.
				specs.EqualTo(ctx, calls.CallCount(), 3)
				recorded := calls.Calls()
				specs.EqualTo(ctx, recorded[0].Args[2].(string), "list -m")
				specs.EqualTo(ctx, recorded[1].Args[2].(string), "list -json ./...")
				specs.EqualTo(ctx, recorded[2].Args[2].(string), "list std")
			})

			// The fixtures under testdata/ are each their own module, loaded
			// with GOWORK=off so an inherited workspace file cannot make the
			// toolchain reject them. That only holds if the variable reaches
			// the call that reads the fixture, so each subcommand is named
			// exactly -- mock.Any() here would be satisfied by `list -m`
			// alone and prove nothing about the listing.
			s.It("pins the caller's environment on both calls that read the module", func(ctx *specs.Context) {
				run, calls := recordingRunner(nil)

				_, err := buildGraph(run, "/somewhere", []string{"GOWORK=off"})

				ctx.Expect(err == nil).To(specs.BeTrue())
				ctx.Expect(calls.CalledWith(
					mock.Equal("/somewhere"), mock.Equal("GOWORK=off"), mock.Equal("list -m"),
				)).To(specs.BeTrue())
				ctx.Expect(calls.CalledWith(
					mock.Equal("/somewhere"), mock.Equal("GOWORK=off"), mock.Equal("list -json ./..."),
				)).To(specs.BeTrue())
			})

			// And the third one deliberately does not carry either: the
			// standard library is the same set whichever module is being
			// loaded, so a fixture's directory and GOWORK have nothing to say
			// about it. Stated as a spec so the asymmetry is a decision on
			// record rather than something a reader has to infer.
			s.It("asks for the standard library outside the module being loaded", func(ctx *specs.Context) {
				run, calls := recordingRunner(nil)

				_, err := buildGraph(run, "/somewhere", []string{"GOWORK=off"})

				ctx.Expect(err == nil).To(specs.BeTrue())
				ctx.Expect(calls.CalledWith(
					mock.Equal("."), mock.Equal(""), mock.Equal("list std"),
				)).To(specs.BeTrue())
			})

			s.It("classifies an import as standard library from what the toolchain listed", func(ctx *specs.Context) {
				run, _ := recordingRunner(map[string]string{"list std": "errors\nfmt\n"})

				g, err := buildGraph(run, "/somewhere", nil)

				ctx.Expect(err == nil).To(specs.BeTrue())
				specs.EqualTo(ctx, g.classify("fmt"), layerStdlib)
				// Not in the listing, so not standard library, however much
				// the path shape suggests it.
				specs.EqualTo(ctx, g.classify("unicode/utf8"), layerExternal)
			})
		})

		s.When("the toolchain fails", func(s *specs.Spec) {
			s.It("reports which step failed rather than reading as a clean tree", func(ctx *specs.Context) {
				boom := errors.New("go: cannot find main module")
				run, calls := failingRunner("list -m", boom)

				g, err := buildGraph(run, "/somewhere", nil)

				ctx.Expect(g == nil).To(specs.BeTrue())
				ctx.Expect(errors.Is(err, boom)).To(specs.BeTrue())
				ctx.Expect(strings.Contains(err.Error(), "resolving module path")).To(specs.BeTrue())
				// It gives up at the first failure instead of carrying on
				// with a half-built graph.
				specs.EqualTo(ctx, calls.CallCount(), 1)
			})

			s.It("reports a failure to list packages as its own step", func(ctx *specs.Context) {
				boom := errors.New("go: build constraints exclude all Go files")
				run, _ := failingRunner("list -json ./...", boom)

				_, err := buildGraph(run, "/somewhere", nil)

				ctx.Expect(errors.Is(err, boom)).To(specs.BeTrue())
				ctx.Expect(strings.Contains(err.Error(), "listing packages")).To(specs.BeTrue())
			})

			// This one used to end the test with t.Fatalf from outside the
			// runner, so it could not be specified at all. Losing the stdlib
			// set is not a small failure: every standard-library import would
			// classify as external and a healthy tree would come back as a
			// wall of violations.
			s.It("reports a failure to list the standard library as its own step", func(ctx *specs.Context) {
				boom := errors.New("go: cannot determine GOROOT")
				run, calls := failingRunner("list std", boom)

				g, err := buildGraph(run, "/somewhere", nil)

				ctx.Expect(g == nil).To(specs.BeTrue())
				ctx.Expect(errors.Is(err, boom)).To(specs.BeTrue())
				ctx.Expect(strings.Contains(err.Error(), "listing the standard library")).To(specs.BeTrue())
				// It got that far: the first two succeeded, so the failure is
				// attributed to the step that actually broke.
				specs.EqualTo(ctx, calls.CallCount(), 3)
			})
		})

		// The silent-pass hazard, stated as a spec: zero packages means every
		// rule reports zero violations, which is indistinguishable from a
		// tree that obeys every rule.
		s.When("the toolchain succeeds but reports nothing", func(s *specs.Spec) {
			s.It("fails instead of letting an empty graph pass every rule", func(ctx *specs.Context) {
				run, _ := recordingRunner(map[string]string{
					"list -m":          "example.test/mod\n",
					"list -json ./...": "",
				})

				g, err := buildGraph(run, "/empty", nil)

				ctx.Expect(g == nil).To(specs.BeTrue())
				ctx.Expect(err != nil).To(specs.BeTrue())
				ctx.Expect(strings.Contains(err.Error(), "returned no packages")).To(specs.BeTrue())

				// The hazard itself, spelled out: an empty graph really does
				// satisfy every rule, which is why the error above matters.
				specs.EqualTo(ctx, len(checkAll(&graph{modulePath: "example.test/mod"})), 0)
			})

			s.It("fails when the package list is not the JSON go list promises", func(ctx *specs.Context) {
				run, _ := recordingRunner(map[string]string{
					"list -m":          "example.test/mod\n",
					"list -json ./...": "this is not json",
				})

				_, err := buildGraph(run, "/broken", nil)

				ctx.Expect(err != nil).To(specs.BeTrue())
				ctx.Expect(strings.Contains(err.Error(), "decoding go list -json output")).To(specs.BeTrue())
			})
		})
	})
}
