# The statement coverage bar in percent.
COVERAGE_MIN ?= 90
export COVERAGE_MIN

.PHONY: build test lint vet check setup

build:
	go build ./...

test:
	go test -race -coverprofile=coverage.out ./...
	sh scripts/coverage-gate.sh coverage.out

lint:
	golangci-lint run ./...

vet:
	go vet ./...

check: vet lint test build

setup:
	git config core.hooksPath .githooks
