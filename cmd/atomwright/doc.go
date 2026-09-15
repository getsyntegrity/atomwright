// Command atomwright is Atomwright's single composition-root binary.
//
// It is owned by #66 ATOM-BOOT-006, the issue that turned the ADR-0001
// skeleton into a running program. ADR-0001 keeps this package thin on
// purpose: it handles process-level concerns only -- signals, streams,
// exit codes -- and hands everything else to internal/bootstrap. It
// imports internal/bootstrap and the standard library, and nothing else
// from the module; compositionRootRule in internal/architecture fails the
// build if that ever stops being true.
//
// Later entry points (MCP and TUI, as #9-#19 land) get their own cmd/*
// package over the same wiring rather than growing this one.
package main
