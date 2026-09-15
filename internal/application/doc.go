// Package application is the internal/application layer defined by
// ADR-0001. It sits between adapters and internal/domain: adapters call
// application services rather than domain packages directly. It is
// scaffolded empty by ATOM-BOOT-003; application services land with the
// epics that need them.
package application
