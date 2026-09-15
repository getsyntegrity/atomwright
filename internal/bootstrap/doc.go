// Package bootstrap is the single composition root defined by ADR-0001.
// It is the one package allowed to see concrete implementations from every
// layer at once -- internal/application, internal/domain/*, adapters/*,
// and platform/* -- and it wires them by calling constructors, never
// through a reflection-based DI container.
//
// cmd/* sees this package and the standard library, nothing else. That
// keeps every entry point (CLI today; MCP and TUI as #9-#19 land) a thin
// process shell over one shared wiring, and it is enforced mechanically by
// compositionRootRule in internal/architecture.
//
// Today the wiring is small because the layers it wires are still
// scaffolds: New builds the platform logger and nothing more. Each
// functional epic adds its own constructor call here as it lands, which is
// the point -- there is exactly one place to add it.
package bootstrap
