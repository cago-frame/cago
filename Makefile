
GOLANGCI_LINT_VERSION ?= v2.12.2
GOLANGCI_LINT := $(CURDIR)/bin/golangci-lint
GOLANGCI_LINT_BUILDER := $(CURDIR)/bin/golangci-lint-builder
CAGOLINT_SOURCES := $(shell find tools/cagolint -type f \( -name '*.go' -o -name 'go.mod' -o -name 'go.sum' \))

.PHONY: check-golangci-lint lint lint-config lint-fix lint-plugin-test test cover html-cover install

check-golangci-lint: $(GOLANGCI_LINT)

$(GOLANGCI_LINT_BUILDER):
	@mkdir -p $(dir $@)
	GOBIN=$(dir $@) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	mv $(dir $@)golangci-lint $@

$(GOLANGCI_LINT): $(GOLANGCI_LINT_BUILDER) .custom-gcl.yml $(CAGOLINT_SOURCES)
	$(GOLANGCI_LINT_BUILDER) custom

lint-plugin-test: check-golangci-lint
	cd tools/cagolint && go test ./...
	$(GOLANGCI_LINT) run ./examples/simple/...

lint-config: check-golangci-lint
	$(GOLANGCI_LINT) config verify

lint: lint-plugin-test lint-config
	$(GOLANGCI_LINT) run

lint-fix: lint-config
	$(GOLANGCI_LINT) run --fix

test:
	go test -v ./...

coverage.out cover:
	go test -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -func=coverage.out

html-cover: coverage.out
	go tool cover -html=coverage.out
	go tool cover -func=coverage.out

install:
	go install ./cmd/cago
