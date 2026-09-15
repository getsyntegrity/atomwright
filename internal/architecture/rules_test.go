package architecture

import (
	"fmt"
	"slices"
	"strings"
)

// violation is one broken edge, reported with enough detail that a failing
// build names the offending package, the offending import, the rule, and why
// the rule exists.
type violation struct {
	rule    string
	pkg     string // module-relative package path
	imports string // module-relative (or verbatim, if outside the module) import path
	reason  string
}

func (v violation) String() string {
	return fmt.Sprintf("%s imports %s\n    rule:   %s\n    reason: %s", v.pkg, v.imports, v.rule, v.reason)
}

// rule is one executable dependency-direction check over the import graph.
type rule interface {
	name() string
	check(g *graph) []violation
}

// allRules is the complete rule set. Every rule here has at least one
// deliberately-violating fixture under testdata/ -- see fixtures_test.go. A
// rule without a negative fixture is not considered complete, and
// TestEveryRuleHasANegativeFixture enforces that.
func allRules() []rule {
	return []rule{
		dependencyRule{},
		restrictedImportRule{},
		compositionRootRule{},
		moduleDependencyRule{},
	}
}

// eachGovernedImport walks every import of every governed-source package in
// the module, skipping self-imports (an external test package importing the
// package under test).
func eachGovernedImport(g *graph, visit func(pkgRel string, from layer, importPath string, to layer)) {
	for _, p := range g.packages {
		pkgRel, local := g.rel(p.ImportPath)
		if !local {
			continue
		}
		from := g.classify(p.ImportPath)
		if !isGovernedSource(from) {
			continue
		}
		for _, importPath := range p.allImports() {
			if importPath == p.ImportPath {
				continue
			}
			visit(pkgRel, from, importPath, g.classify(importPath))
		}
	}
}

// ---------------------------------------------------------------------------
// 1. dependencyRule
// ---------------------------------------------------------------------------

// forbiddenLayerEdges are the ADR-0001 anti-edges that describe the direction
// of dependency itself: inward-only, with the domain at the centre.
var forbiddenLayerEdges = []struct {
	from   layer
	to     layer
	reason string
}{
	{layerDomain, layerApplication, "ADR-0001 anti-edge: internal/domain/* must never import internal/application -- the domain is the centre and depends on nothing above it"},
	{layerDomain, layerAdapters, "ADR-0001 anti-edge: internal/domain/* must never import adapters/* -- the domain must not know about any concrete I/O implementation"},
	{layerDomain, layerPlatform, "ADR-0001 anti-edge: internal/domain/* must never import platform/* -- cross-cutting infrastructure is not a domain concern"},
	{layerApplication, layerAdapters, "ADR-0001 anti-edge: internal/application must never import a concrete adapters/* package -- adapters call application services, never the other way round"},
	{layerPlatform, layerDomain, "ADR-0001 anti-edge: platform/* must never import internal/domain/* -- platform is domain-agnostic infrastructure"},
}

// dependencyRule enforces the ADR-0001 anti-edge block: the directions that
// must never exist regardless of what any allowlist says.
type dependencyRule struct{}

func (dependencyRule) name() string { return "dependencyRule" }

func (r dependencyRule) check(g *graph) []violation {
	var found []violation
	eachGovernedImport(g, func(pkgRel string, from layer, importPath string, to layer) {
		for _, forbidden := range forbiddenLayerEdges {
			if forbidden.from == from && forbidden.to == to {
				found = append(found, violation{
					rule:    r.name(),
					pkg:     pkgRel,
					imports: g.display(importPath),
					reason:  forbidden.reason,
				})
			}
		}
	})
	return found
}

// ---------------------------------------------------------------------------
// 2. restrictedImportRule
// ---------------------------------------------------------------------------

// restrictedImportRule guards internal/domain/* as an import target: only
// internal/bootstrap, internal/application, adapters/*, and other
// internal/domain/* packages may import it. Every other package in the module
// -- cmd/*, platform/*, and any package belonging to no layer -- is denied.
//
// WHAT THIS RULE DOES NOT PROVE, AND MUST NOT BE MISTAKEN FOR:
//
// ADR-0001 permits adapters/* -> internal/domain/* for "ports and contractual
// types only", never to call domain logic. This rule enforces the EDGE, not
// the REASON for the edge. An adapter importing a domain package to implement
// a port and an adapter importing the same package to call domain logic
// produce the identical adapters/x -> internal/domain/y edge in the go list
// graph; nothing in that graph distinguishes them.
//
// So: a green run of this rule means "no package outside the permitted layers
// imports the domain". It does NOT mean "every adapter uses the domain only
// for ports and contractual types". That half stays a REVIEWED CONVENTION --
// a code-review concern, not an assertion this test makes.
//
// Telling the two apart would need either AST/go-types analysis of what is
// referenced across the edge (explicitly rejected by issue #65 as
// disproportionate to one sub-rule), or a physically separate importable
// location for contracts, e.g. internal/domain/<ctx>/ports, which adapters may
// import while the rest of internal/domain/<ctx> stays off-limits. The second
// is the upgrade path: it turns the distinction back into a plain
// package-path check, and this rule tightens to that sub-path with no
// analyzer. Neither is built here, and until one exists, do not read a green
// suite as a guarantee it does not provide.
type restrictedImportRule struct{}

func (restrictedImportRule) name() string { return "restrictedImportRule" }

func (r restrictedImportRule) check(g *graph) []violation {
	var found []violation
	for _, p := range g.packages {
		pkgRel, local := g.rel(p.ImportPath)
		if !local {
			continue
		}
		from := g.classify(p.ImportPath)
		for _, importPath := range p.allImports() {
			if importPath == p.ImportPath {
				continue
			}
			if g.classify(importPath) != layerDomain {
				continue
			}
			if slices.Contains(domainImporters, from) {
				continue
			}
			found = append(found, violation{
				rule:    r.name(),
				pkg:     pkgRel,
				imports: g.display(importPath),
				reason: fmt.Sprintf(
					"ADR-0001 restricts internal/domain/* to importers in %s; %s is classified as %s",
					joinLayers(domainImporters), pkgRel, from),
			})
		}
	}
	return found
}

func joinLayers(layers []layer) string {
	parts := make([]string, 0, len(layers))
	for _, l := range layers {
		parts = append(parts, string(l))
	}
	return strings.Join(parts, ", ")
}

// ---------------------------------------------------------------------------
// 3. compositionRootRule
// ---------------------------------------------------------------------------

// compositionRootRule enforces that internal/bootstrap is the only place that
// wires concrete implementations together. A cmd/* package -- cmd/atomwright,
// or any future entry point -- may reach internal/bootstrap and the standard
// library, and nothing else from the ADR-0001 layers.
//
// This one is fully mechanical with no semantic ambiguity: "does cmd/* import
// an adapters/*, internal/application, internal/domain/*, or platform/*
// package at all" is a plain package-path question.
//
// No cmd/* package exists yet -- #66 ATOM-BOOT-006 lands cmd/atomwright and
// internal/bootstrap -- so this rule is vacuously satisfied by the current
// tree. That is exactly why its negative fixtures matter: they are what proves
// the rule works, and they are in place before the package it guards exists.
// Anything a cmd/* package imports that is neither a layer nor the standard
// library is caught by moduleDependencyRule, not here.
type compositionRootRule struct{}

func (compositionRootRule) name() string { return "compositionRootRule" }

func (r compositionRootRule) check(g *graph) []violation {
	forbiddenFromCmd := []layer{layerAdapters, layerApplication, layerDomain, layerPlatform}

	var found []violation
	eachGovernedImport(g, func(pkgRel string, from layer, importPath string, to layer) {
		if from != layerCmd || !slices.Contains(forbiddenFromCmd, to) {
			return
		}
		found = append(found, violation{
			rule:    r.name(),
			pkg:     pkgRel,
			imports: g.display(importPath),
			reason: fmt.Sprintf(
				"ADR-0001 anti-edge: cmd/* sees only internal/bootstrap and the standard library, never %s directly -- internal/bootstrap is the single composition root",
				to),
		})
	})
	return found
}

// ---------------------------------------------------------------------------
// 4. moduleDependencyRule
// ---------------------------------------------------------------------------

// moduleDependencyRule is the default-deny allowlist. Every import of every
// governed package must be one of:
//
//   - the standard library, allowed everywhere;
//   - an edge between ADR-0001 layers listed in allowedLayerEdges;
//   - a third-party module, and only from a layer whose ADR-0001 row is not
//     restricted to the standard library (see externalImportsAllowed).
//
// Anything else fails, including an import of a module package that belongs to
// no layer at all. There is no exemption list to fall back on: see the closing
// comment in allowlist_test.go. This is what makes the boundary default-deny rather
// than "forbid the handful of directions somebody remembered to write down":
// dependencyRule and compositionRootRule name the anti-edges explicitly so
// their failure messages are specific, and this rule catches everything
// nobody thought to name.
type moduleDependencyRule struct{}

func (moduleDependencyRule) name() string { return "moduleDependencyRule" }

func (r moduleDependencyRule) check(g *graph) []violation {
	var found []violation
	eachGovernedImport(g, func(pkgRel string, from layer, importPath string, to layer) {
		switch to {
		case layerStdlib:
			return
		case layerExternal:
			if externalImportsAllowed[from] {
				return
			}
			found = append(found, violation{
				rule:    r.name(),
				pkg:     pkgRel,
				imports: importPath,
				reason: fmt.Sprintf(
					"ADR-0001 restricts %s to the standard library and its own layer; a third-party import is not allowed there", from),
			})
		case layerUngoverned:
			found = append(found, violation{
				rule:    r.name(),
				pkg:     pkgRel,
				imports: g.display(importPath),
				reason: fmt.Sprintf(
					"default-deny: %s is an ADR-0001 layer and %s is a module package that belongs to no layer in the dependency table; give it a home in one of the layers, or add the edge to the table with review",
					from, g.display(importPath)),
			})
		default:
			if layerEdgeAllowed(from, to) {
				return
			}
			found = append(found, violation{
				rule:    r.name(),
				pkg:     pkgRel,
				imports: g.display(importPath),
				reason: fmt.Sprintf(
					"default-deny: the edge %s -> %s is not in the ADR-0001 dependency table; add it there, with review, in the same change that needs it",
					from, to),
			})
		}
	})
	return found
}

// checkAll runs every rule and returns the violations in rule order.
func checkAll(g *graph) []violation {
	var found []violation
	for _, r := range allRules() {
		found = append(found, r.check(g)...)
	}
	return found
}

// formatViolations renders violations for a test failure message.
func formatViolations(found []violation) string {
	var b strings.Builder
	for i, v := range found {
		if i > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(v.String())
	}
	return b.String()
}
