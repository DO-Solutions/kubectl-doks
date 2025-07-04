# Makefile for kubectl-doks

BIN_NAME := kubectl-doks
GO := go

.PHONY: build
build:
	$(GO) build -o dist/$(BIN_NAME) .

.PHONY: clean
clean:
	rm -f dist/$(BIN_NAME)

.PHONY: test-unit
test-unit:
	$(GO) test -v ./...

.PHONY: test-integration
test-integration:
	$(GO) test -v ./test/integration/...

.PHONY: test-all
test-all: test-unit test-integration

.PHONY: lint
lint:
	golangci-lint run

.PHONY: install
install: build
	mv dist/$(BIN_NAME) $(GOPATH)/bin/
