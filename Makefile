GO ?= go
BIN := bin/qrt
GOBIN := $(shell $(GO) env GOPATH)/bin
GOLANGCI_LINT_VERSION := v2.12.2
GOVULNCHECK_VERSION := v1.3.0

export GOTOOLCHAIN ?= local
export GOFLAGS ?= -mod=readonly

.PHONY: build test test-race vet lint security supply-chain verify demo dogfood-demo clean

build:
	mkdir -p bin
	$(GO) build -o $(BIN) ./cmd/qrt

test:
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

verify: supply-chain vet lint test test-race build demo dogfood-demo security

demo: build
	sh scripts/demo.sh ./$(BIN)

dogfood-demo: build
	rm -rf .qratum
	./$(BIN) dogfood import fixtures/dogfood/real-shaped-transcript.jsonl
	./$(BIN) dogfood latest

clean:
	rm -rf bin .qratum
