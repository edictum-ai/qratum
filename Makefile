GO ?= go
BIN := bin/qrt
export GOPATH ?= $(CURDIR)/.gopath
export GOCACHE ?= $(CURDIR)/.gocache
export TMPDIR ?= /tmp
export GOLANGCI_LINT_CACHE ?= $(CURDIR)/.golangci-lint-cache
GOBIN := $(GOPATH)/bin
GOLANGCI_LINT_VERSION := v2.12.2
GOVULNCHECK_VERSION := v1.3.0

export GOTOOLCHAIN ?= local
export GOFLAGS ?= -mod=readonly

.PHONY: build test test-race vet lint security supply-chain history-lint trust verify demo dogfood-demo clean

build:
	mkdir -p bin
	$(GO) build -o $(BIN) ./cmd/qrt

test: history-lint
	$(GO) test ./...

test-race:
	$(GO) test -race ./...

vet:
	$(GO) vet ./...

lint:
	env GOFLAGS= $(GO) install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	$(GOBIN)/golangci-lint run ./...

security:
	env GOFLAGS= $(GO) install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)
	$(GOBIN)/govulncheck ./...

supply-chain:
	sh scripts/check-supply-chain.sh

history-lint:
	$(GO) test ./cmd/qrt -run 'TestNoSecretInGolden/git-history-known-red' -v

trust: build
	mkdir -p .trust
	@QRATUM_HOME="$$(mktemp -d "$$PWD/.qratum-home.XXXXXX")"; \
	export QRATUM_HOME; \
	$(GO) run ./cmd/trustbench --qrt ./$(BIN) --json-out .trust/trust-scorecard.json

verify: supply-chain vet lint test test-race build demo dogfood-demo security trust

demo: build
	sh scripts/demo.sh ./$(BIN)

dogfood-demo: build
	rm -rf .qratum .qratum-home.*
	@QRATUM_HOME="$$(mktemp -d "$$PWD/.qratum-home.XXXXXX")"; \
	export QRATUM_HOME; \
	./$(BIN) dogfood import fixtures/dogfood/real-shaped-transcript.jsonl; \
	./$(BIN) dogfood latest

clean:
	rm -rf bin .qratum .qratum-home.* .trust .gopath .gocache .golangci-lint-cache
