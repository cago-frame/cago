
GOLANGCI_LINT_VERSION ?= v2.12.2
GOLANGCI_LINT := $(CURDIR)/bin/golangci-lint
GOLANGCI_LINT_BUILDER := $(CURDIR)/bin/golangci-lint-builder

.PHONY: check-golangci-lint lint lint-fix lint-config

check-cago:
ifneq ($(which cago),)
	go install github.com/cago-frame/cago/cmd/cago@latest
endif

check-mockgen:
ifneq ($(which mockgen),)
	go install go.uber.org/mock/mockgen@latest
endif

check-golangci-lint: $(GOLANGCI_LINT)

$(GOLANGCI_LINT_BUILDER):
	@mkdir -p $(dir $@)
	GOBIN=$(dir $@) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	mv $(dir $@)golangci-lint $@

$(GOLANGCI_LINT): $(GOLANGCI_LINT_BUILDER) .custom-gcl.yml
	$(GOLANGCI_LINT_BUILDER) custom

check-goconvey:
ifneq ($(which goconvey),)
	go install github.com/smartystreets/goconvey@latest
endif


proto:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		pkg/proto/user/user.proto

lint: check-golangci-lint
	$(GOLANGCI_LINT) run ./...

lint-fix: check-golangci-lint
	$(GOLANGCI_LINT) run --fix ./...

lint-config: check-golangci-lint
	$(GOLANGCI_LINT) config verify

test:
	go test -v ./...

coverage.out cover:
	go test -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -func=coverage.out

html-cover: coverage.out
	go tool cover -html=coverage.out
	go tool cover -func=coverage.out

generate: check-mockgen check-cago
	go generate ./...
	cago gen

goconvey: check-goconvey
	goconvey

GOOS=linux
GOARCH=amd64
APP_NAME=simple
APP_VERSION=1.0.0
LD_FLAGS=-w -s -X github.com/cago-frame/cago/configs.Version=${APP_VERSION}

SUFFIX=
ifeq ($(GOOS),windows)
	SUFFIX=.exe
endif

build:
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o bin/$(APP_NAME)_v$(APP_VERSION)$(SUFFIX) \
		-trimpath -ldflags "${LD_FLAGS}" ./cmd/app
