package architecture

import (
	"slices"
	"strings"
	"testing"
)

// TestDependencyDirection is the rule that guards the tree. It runs every
// ADR-0001 rule against the real module and fails with the violating package,
// the violating import, the rule that was broken, and why the rule exists.
//
// When this fails, the fix is one of exactly two things, and which one is a
// review decision, not a convenience call:
//
//  1. the import is wrong -- move the code so the dependency points inward; or
//  2. the edge is genuinely legitimate -- amend the ADR-0001 table and
//     allowedLayerEdges together, in the same change, with the justification
//     in the pull request.
//
// Growing the allowlist without that justification is treated exactly like the
// violation it would otherwise catch.
func TestDependencyDirection(t *testing.T) {
	g := loadGraph(t, moduleRoot(t))

	if found := checkAll(g); len(found) > 0 {
		t.Fatalf("%d dependency-direction violation(s) in %s:\n\n%s",
			len(found), g.modulePath, formatViolations(found))
	}
}

// TestGovernedLayersArePresent guards against the rules silently going quiet.
// A path-based classifier that matches nothing still reports zero violations,
// which looks identical to a clean tree. If a layer disappears from the
// module, that is a fact worth failing on rather than a test that passes by
// checking nothing.
//
// internal/bootstrap and cmd/* are deliberately absent from this list: #66
// ATOM-BOOT-006 introduces them, and their rules are proven by fixtures until
// then.
func TestGovernedLayersArePresent(t *testing.T) {
	g := loadGraph(t, moduleRoot(t))

	want := []layer{layerApplication, layerDomain, layerPlatform, layerAdapters}

	seen := map[layer]int{}
	for _, p := range g.packages {
		if _, local := g.rel(p.ImportPath); local {
			seen[g.classify(p.ImportPath)]++
		}
	}

	for _, l := range want {
		if seen[l] == 0 {
			t.Errorf("no package classified as %s: either the layer was removed or the classifier stopped matching it", l)
		}
	}
}

// TestAllowlistIsDefaultDeny asserts the properties the allowlist has to keep
// to be an allowlist at all: no duplicate entries, no entry naming a
// non-layer, and -- the load-bearing one -- no blanket self-edge that would
// quietly let any layer sprawl internally. Only the two intra-layer edges
// ADR-0001 actually states are permitted.
func TestAllowlistIsDefaultDeny(t *testing.T) {
	allowedSelfEdges := []layer{layerPlatform, layerDomain}

	seen := map[layerEdge]bool{}
	for _, edge := range allowedLayerEdges {
		key := layerEdge{from: edge.from, to: edge.to}
		if seen[key] {
			t.Errorf("duplicate allowlist entry %s -> %s", edge.from, edge.to)
		}
		seen[key] = true

		if !isGovernedSource(edge.from) {
			t.Errorf("allowlist entry %s -> %s: %s is not an ADR-0001 layer", edge.from, edge.to, edge.from)
		}
		if !isGovernedSource(edge.to) {
			t.Errorf("allowlist entry %s -> %s: %s is not an ADR-0001 layer", edge.from, edge.to, edge.to)
		}
		if edge.from == edge.to && !slices.Contains(allowedSelfEdges, edge.from) {
			t.Errorf("allowlist entry %s -> %s: ADR-0001 states an intra-layer edge only for %v", edge.from, edge.to, allowedSelfEdges)
		}
	}

	for _, l := range governedSourceLayers {
		if _, ok := externalImportsAllowed[l]; !ok {
			t.Errorf("layer %s has no third-party import policy: every governed layer must state one explicitly, since a missing entry silently reads as deny", l)
		}
		if _, ok := testOnlyExternalImportsAllowed[l]; !ok {
			t.Errorf("layer %s has no test-only third-party import policy (ADR-0003): every governed layer must state one explicitly, since a missing entry silently reads as deny", l)
		}
	}
}

// TestAllowlistAndAntiEdgesDoNotContradict asserts the two halves of ADR-0001
// stay consistent: an edge cannot be both allowed and an anti-edge. If they
// ever contradicted, dependencyRule and moduleDependencyRule would disagree
// about the same import and the table would have stopped meaning one thing.
func TestAllowlistAndAntiEdgesDoNotContradict(t *testing.T) {
	for _, forbidden := range forbiddenLayerEdges {
		if layerEdgeAllowed(forbidden.from, forbidden.to) {
			t.Errorf("%s -> %s is both an ADR-0001 anti-edge and an allowlist entry", forbidden.from, forbidden.to)
		}
	}

	for _, l := range governedSourceLayers {
		if l != layerDomain && slices.Contains(domainImporters, l) != layerEdgeAllowed(l, layerDomain) {
			t.Errorf("domainImporters and allowedLayerEdges disagree about %s -> %s", l, layerDomain)
		}
	}
}

// TestViolationMessageNamesPackageAndRule pins the failure-message contract:
// a broken build has to say which package broke which rule, or the failure
// costs more to diagnose than it saves.
func TestViolationMessageNamesPackageAndRule(t *testing.T) {
	v := violation{
		rule:    "dependencyRule",
		pkg:     "internal/domain/execution",
		imports: "adapters/mcp",
		reason:  "the domain must not know about any concrete I/O implementation",
	}

	got := v.String()
	for _, want := range []string{v.pkg, v.imports, v.rule, v.reason} {
		if !strings.Contains(got, want) {
			t.Errorf("violation message %q does not name %q", got, want)
		}
	}
}
