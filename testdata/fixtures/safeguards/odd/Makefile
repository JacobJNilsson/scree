VERSION := $(shell git describe --tags)

.PHONY: build release setup

build:
	go build -ldflags "-X main.version=$(shell git describe --tags)" \
		./...

release: build
	@echo releasing $(VERSION)

setup:
	git config core.hooksPath .githooks
