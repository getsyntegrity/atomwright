# Atomwright gates.
#
# `make check` is the whole gate, and CI runs this same target -- no
# CI-only step, no local-only step. A gate that exists on one side and not
# the other is treated as broken (#66 ATOM-BOOT-006), and
# internal/gates/gates_test.go fails the build if the two drift apart.

GO ?= go
# gofmt lives in GOROOT/bin, which is not always on PATH even when go is.
GOFMT ?= $(shell $(GO) env GOROOT)/bin/gofmt
BINARY ?= atomwright

.DEFAULT_GOAL := check

.PHONY: check fmt vet tidy build test arch binary help

## check: run every gate -- this is what CI runs
check: fmt vet tidy build test arch

## fmt: fail if any file is not gofmt-clean
#
# The `|| exit 1` is load-bearing: without it a gofmt that cannot run at
# all (missing binary, wrong GOROOT) leaves `unformatted` empty and the
# recipe reports success, which is a formatting gate that is green because
# it never ran.
fmt:
	@unformatted=$$($(GOFMT) -l .) || { echo "gofmt failed to run: $(GOFMT)"; exit 1; }; \
	if [ -n "$$unformatted" ]; then \
		echo "gofmt needs to run on:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi

## vet: run the standard-library static analysis suite
vet:
	$(GO) vet ./...

## tidy: fail if go.mod or go.sum would change (checks, never rewrites)
tidy:
	$(GO) mod tidy -diff

## build: compile every package
build:
	$(GO) build ./...

## test: run the full test suite
test:
	$(GO) test ./...

# The architecture checks are part of ./... and so already ran under
# `test`. They are named separately because ADR-0001's dependency
# direction is a gate in its own right: when it fails, the failure should
# be legible as an architecture failure, and it can be run alone while
# fixing one.
## arch: run the ADR-0001 dependency-direction checks on their own
arch:
	$(GO) test ./internal/architecture/...

## binary: build the atomwright binary into the repository root
binary:
	$(GO) build -o $(BINARY) ./cmd/atomwright

## help: list the available targets
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed -e 's/^## //' | awk -F ': ' '{printf "  %-10s %s\n", $$1, $$2}'
