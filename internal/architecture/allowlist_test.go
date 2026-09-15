package architecture

// This file is the ADR-0001 dependency table expressed as Go data. It is the
// single place a new cross-boundary edge can be legalised, and it is
// default-deny: an edge that is not listed here is a violation, so a new
// inward or cross-boundary import must be added in the same change that
// introduces it, with review.
//
// ADR-0001, verbatim -- the allowed edges:
//
//	cmd/*                -> internal/bootstrap
//	internal/bootstrap   -> internal/application, internal/domain/*, adapters/*, platform/*
//	adapters/*           -> internal/application
//	adapters/*           -> internal/domain/*      [ports and contractual types only]
//	internal/application -> internal/domain/*
//	platform/*           -> stdlib, platform/* internals
//	internal/domain/*    -> stdlib, other internal/domain/* packages (explicitly shared concepts only)
//
// and the anti-edges:
//
//	internal/domain/*    -X-> internal/application
//	internal/domain/*    -X-> adapters/*
//	internal/domain/*    -X-> platform/*
//	internal/application -X-> adapters/* (concrete)
//	cmd/*                -X-> adapters/* (concrete)
//	cmd/*                -X-> internal/application, internal/domain/*, platform/* (directly)
//
// allowedLayerEdges below is that first block, entry for entry, in the same
// order. Nothing else is in it. Note what is deliberately absent: there is no
// blanket "a layer may import itself" entry. Only the two intra-layer edges
// the table actually states -- platform/* internals and domain-to-domain --
// are allowed. adapters/* importing another adapters/* package, or cmd/*
// importing another cmd/* package, is denied until the table says otherwise.

// layerEdge is one allowed edge between two ADR-0001 layers.
type layerEdge struct {
	from layer
	to   layer
	// note carries the ADR-0001 qualifier for edges that have one. It is
	// documentation: go list cannot enforce a qualifier about intent.
	note string
}

var allowedLayerEdges = []layerEdge{
	{from: layerCmd, to: layerBootstrap},

	{from: layerBootstrap, to: layerApplication},
	{from: layerBootstrap, to: layerDomain},
	{from: layerBootstrap, to: layerAdapters},
	{from: layerBootstrap, to: layerPlatform},

	{from: layerAdapters, to: layerApplication},
	{from: layerAdapters, to: layerDomain, note: "ports and contractual types only -- reviewed convention, not mechanically enforced"},

	{from: layerApplication, to: layerDomain},

	{from: layerPlatform, to: layerPlatform, note: "platform/* internals"},

	{from: layerDomain, to: layerDomain, note: "explicitly shared concepts only"},
}

func layerEdgeAllowed(from, to layer) bool {
	for _, edge := range allowedLayerEdges {
		if edge.from == from && edge.to == to {
			return true
		}
	}
	return false
}

// externalImportsAllowed says whether a layer may import a package from
// another module (a third-party dependency).
//
// The ADR-0001 rows for internal/domain/* and platform/* both read "stdlib"
// -- those two layers are standard-library-only, so a third-party import from
// either is a violation. The remaining layers exist precisely to hold the
// outside world, so third-party imports there are unconstrained by this rule.
var externalImportsAllowed = map[layer]bool{
	layerCmd:         true,
	layerBootstrap:   true,
	layerApplication: true,
	layerAdapters:    true,
	layerDomain:      false,
	layerPlatform:    false,
}

// domainImporters are the layers ADR-0001 permits to import internal/domain/*
// at all. Everything else -- cmd/*, platform/*, and every pre-ATOM-BOOT
// package in the module -- is denied by restrictedImportRule.
var domainImporters = []layer{
	layerBootstrap,
	layerApplication,
	layerAdapters,
	layerDomain,
}

// There is deliberately no exemption or grandfathering list here.
//
// ADR-0002 removed the pre-ATOM-BOOT tree outright -- the module is a
// greenfield baseline whose only packages are the ADR-0001 layers -- so there
// is no legacy edge to carry, and inventing a mechanism for edges that do not
// exist would be speculative complexity the issue's budget does not allow.
// The only way to legalise a new edge is to put it in the ADR-0001 table and
// in allowedLayerEdges above, in the same change that needs it, with review.
