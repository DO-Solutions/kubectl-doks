# Makefile for kubectl-doks

BIN_NAME := kubectl-doks
GO := go

.PHONY: build
build:
	$(GO) build -o dist/$(BIN_NAME) .

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
