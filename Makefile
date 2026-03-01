
check-golangci-lint:
	@command -v golangci-lint >/dev/null 2>&1 || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

lint: check-golangci-lint
	golangci-lint run

lint-fix: check-golangci-lint
	golangci-lint run --fix

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
