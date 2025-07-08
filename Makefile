# Makefile for kubectl-doks

BIN_NAME := kubectl-doks
GO := go

.PHONY: build
# Variables
VERSION := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
COMMIT := $(shell git rev-parse --short HEAD)

build:
	$(GO) build -ldflags="-X 'github.com/DO-Solutions/kubectl-doks/cmd.version=$(VERSION)' -X 'github.com/DO-Solutions/kubectl-doks/cmd.commit=$(COMMIT)'" -o dist/$(BIN_NAME) .

.PHONY: clean
clean:
	rm -f dist/$(BIN_NAME)

.PHONY: test
test:
	$(GO) test -v ./...

.PHONY: lint
lint:
	golangci-lint run

.PHONY: install
install: build
	mv dist/$(BIN_NAME) $(GOPATH)/bin/
