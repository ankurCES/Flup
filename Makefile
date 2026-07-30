# Makefile for Flup — a TUI HTTP benchmarking tool.
# github.com/ankurCES/Flup

# ---- Toolchain ----
GO            ?= go
GOVERSION    := $(shell $(GO) version | awk '{print $$3}')
GOBIN         ?= $$(go env GOPATH)/bin
BIN_DIR       ?= bin

# ---- Project metadata ----
BINARY        ?= flup
SERVER_BIN    ?= bench_server
PKG           := ./...
MAIN          := ./cmd/$(BINARY)
SERVER_MAIN   := ./cmd/$(SERVER_BIN)
LDFLAGS       := -s -w -X main.version=$(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

# ---- Phony targets ----
.PHONY: help all build build-server test test-race test-cover lint vet fmt tidy run run-server install clean check deps

# Default target: show help.
help: ## Show this help message.
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

all: vet lint test build ## Run vet, lint, test, and build.

# ---- Build ----
build: ## Build the flup binary into ./bin/.
	@mkdir -p $(BIN_DIR)
	$(GO) build -trimpath -ldflags '$(LDFLAGS)' -o $(BIN_DIR)/$(BINARY) $(MAIN)
	@echo "✓ built $(BIN_DIR)/$(BINARY)"

build-server: ## Build the bundled test server into ./bin/.
	@mkdir -p $(BIN_DIR)
	$(GO) build -trimpath -ldflags '$(LDFLAGS)' -o $(BIN_DIR)/$(SERVER_BIN) $(SERVER_MAIN)
	@echo "✓ built $(BIN_DIR)/$(SERVER_BIN)"

# ---- Test ----
test: ## Run unit tests.
	$(GO) test -count=1 -timeout 120s $(PKG)

test-race: ## Run unit tests with the race detector enabled.
	$(GO) test -race -count=1 -timeout 180s $(PKG)

test-cover: ## Run unit tests and write a coverage profile to coverage.out.
	$(GO) test -coverprofile=coverage.out -covermode=atomic $(PKG)
	@$(GO) tool cover -func=coverage.out | tail -n 1

# ---- Static analysis ----
vet: ## Run go vet on all packages.
	$(GO) vet $(PKG)

fmt: ## Run gofmt -s -w on all Go files.
	$(GO) fmt $(PKG)

tidy: ## Sync go.mod / go.sum with the source tree.
	$(GO) mod tidy

lint: vet ## Alias for the local lint chain (vet + format check).

# ---- Run ----
run: build ## Build and launch the TUI.
	./$(BIN_DIR)/$(BINARY)

run-server: build-server ## Build and launch the bundled test server.
	./$(BIN_DIR)/$(SERVER_BIN)

# ---- Install ----
install: ## Install flup into $$GOBIN.
	$(GO) install -ldflags '$(LDFLAGS)' $(MAIN)
	@echo "✓ installed $(BINARY) to $(GOBIN)"

# ---- Utility ----
deps: ## Download and verify Go module dependencies.
	$(GO) mod download
	$(GO) mod verify

check: ## Quick pre-flight: version + module sanity.
	@echo "go:  $(GOVERSION)"
	@$(GO) env GOMOD GOFLAGS GOPROXY

clean: ## Remove build artifacts and coverage files.
	rm -rf $(BIN_DIR) coverage.out coverage.html *.test
	@echo "✓ cleaned"

# Fail fast on recipes that don't match.
%:
	@: