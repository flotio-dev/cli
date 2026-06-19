# Flotio CLI — Makefile
#
# Canonical build (matches CI). devenv.nix lives in the PARENT directory, so
# devenv-based targets `cd` up one level before running:
#
#   cd <parent> && devenv shell go build -C cli \
#       -ldflags "-X github.com/flotio-dev/cli/cmd.version=<X.Y.Z>" -o bin/flotio .
#
# Use `make help` to list targets.

# --- Toolchain / paths (literals — safe to expand at parse time) ---
GO         ?= go
MODULE     := github.com/flotio-dev/cli
BINARY     := flotio
BIN_DIR    := bin
BIN        := $(BIN_DIR)/$(BINARY)
# devenv.nix lives in the parent directory (../../ of module root's parent).
DEVENV_DIR := ..

# --- Version (lazy: only evaluated when a recipe references it) ---
# Derives the version from the latest git tag, strips a leading "v" to match CI
# (build.yml does VERSION="${VERSION#v}"), and falls back to "dev".
# Override anytime with: make VERSION=1.2.3 build
VERSION ?= $(shell v=$$(git describe --tags --always --dirty 2>/dev/null || echo dev); echo "$${v#v}")
LDFLAGS  = -X $(MODULE)/cmd.version=$(VERSION)

# --- Install destination (lazy: only evaluated by the `install` target) ---
GOBIN_ENV  = $(shell $(GO) env GOBIN 2>/dev/null)
GOPATH_ENV = $(shell $(GO) env GOPATH 2>/dev/null)
INSTALL_DIR = $(if $(GOBIN_ENV),$(GOBIN_ENV),$(if $(GOPATH_ENV),$(GOPATH_ENV)/bin,$(HOME)/.local/bin))

.DEFAULT_GOAL := help

.PHONY: help build build-go build-dev test test-go run clean install check-devenv

help: ## Show available targets
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# --- Internal helper ---
check-devenv:
	@command -v devenv >/dev/null 2>&1 || { \
		echo "error: 'devenv' not found on PATH."; \
		echo "       Enter the dev shell first (devenv shell) or use 'make build-go'."; \
		exit 1; }

# --- Build ---
build: check-devenv ## Build bin/flotio with the current version (via devenv)
	cd $(DEVENV_DIR) && devenv shell $(GO) build -C cli -ldflags "$(LDFLAGS)" -o $(BIN) .

build-go: ## Build bin/flotio with version ldflags using plain 'go build' (no devenv)
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BIN) .

build-dev: ## Build bin/flotio without ldflags (version reports "dev")
	$(GO) build -o $(BIN) .

# --- Test ---
test: ## Run the test suite (via devenv if available, else plain 'go test')
	@if command -v devenv >/dev/null 2>&1; then \
		cd $(DEVENV_DIR) && devenv shell $(GO) test -C cli ./...; \
	else \
		echo "(devenv not found — running go test directly)"; \
		$(GO) test ./...; \
	fi

test-go: ## Run the test suite with plain 'go test' (no devenv)
	$(GO) test ./...

# --- Run from source ---
run: ## Run the CLI from source (e.g. make run ARGS="version")
	$(GO) run . $(ARGS)

# --- Clean ---
clean: ## Remove build artifacts
	rm -rf $(BIN_DIR)

# --- Install ---
install: build ## Build and install flotio to $$GOBIN (or $$GOPATH/bin)
	@mkdir -p "$(INSTALL_DIR)"
	cp $(BIN) "$(INSTALL_DIR)/$(BINARY)"
	@echo "✓ Installed $(BINARY) to $(INSTALL_DIR)"
