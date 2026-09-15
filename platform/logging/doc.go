// Package logging is the cross-cutting platform logger defined by
// ADR-0001. Like every platform package it must never import
// internal/domain. It is scaffolded empty by ATOM-BOOT-003; the minimal
// log/slog wrapper lands with ATOM-BOOT-006, when make check and CI
// wiring need it.
package logging
