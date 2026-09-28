# The statement coverage bar in percent. The scripts read it from the environment.
COVERAGE_MIN ?= 90
export COVERAGE_MIN

.PHONY: build test gate-test lint lint-fast vet tidy tidy-check antislop check check-scoped setup clean golden

build:
	go build ./...

test:
	go test -race -coverprofile=coverage.out ./...
	sh scripts/coverage-gate.sh coverage.out

# Only packages with golden files define the -update flag.
golden:
	go test ./internal/inventory -update

gate-test:
	sh scripts/coverage-gate-test.sh

lint:
	golangci-lint run ./...

lint-fast:
	golangci-lint run --fast-only ./...

vet:
	go vet ./...

tidy:
	go mod tidy

# The hooks run after staging, so this target reports a needed change and never rewrites go.mod.
tidy-check:
	go mod tidy -diff

# anti-slop-go v1.3.0 requires Go 1.24 or newer. GOTOOLCHAIN=auto downloads a newer toolchain.
# GOTOOLCHAIN=local on Go 1.23 fails.
antislop:
	go run github.com/JacobJNilsson/anti-slop-go/cmd/antislop@v1.3.0 ./...

check: tidy-check vet lint gate-test test build antislop

check-scoped:
	sh scripts/precommit-check.sh

setup:
	git config core.hooksPath .githooks

clean:
	rm -f coverage.out coverage.scoped.out
