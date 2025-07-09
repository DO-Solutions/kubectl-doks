# Makefile for kubectl-doks

BIN_NAME := kubectl-doks
GO := go
KREW_MANIFEST_TEMPLATE := plugins/doks.yaml.tpl
KREW_TEMPLATE := plugins/doks.yaml

.PHONY: build
# Variables
VERSION := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
COMMIT := $(shell git rev-parse --short HEAD)

build:
	$(GO) build -ldflags="-X 'github.com/DO-Solutions/kubectl-doks/cmd.version=$(VERSION)' -X 'github.com/DO-Solutions/kubectl-doks/cmd.commit=$(COMMIT)'" -o dist/$(BIN_NAME) .

.PHONY: build-all
build-all: build-linux build-macos-amd64 build-macos-arm64

.PHONY: build-linux
build-linux:
	GOOS=linux GOARCH=amd64 $(GO) build -ldflags="-X 'github.com/DO-Solutions/kubectl-doks/cmd.version=$(VERSION)' -X 'github.com/DO-Solutions/kubectl-doks/cmd.commit=$(COMMIT)'" -o dist/$(BIN_NAME)-linux-amd64 .

.PHONY: build-macos-amd64
build-macos-amd64:
	GOOS=darwin GOARCH=amd64 $(GO) build -ldflags="-X 'github.com/DO-Solutions/kubectl-doks/cmd.version=$(VERSION)' -X 'github.com/DO-Solutions/kubectl-doks/cmd.commit=$(COMMIT)'" -o dist/$(BIN_NAME)-darwin-amd64 .

.PHONY: build-macos-arm64
build-macos-arm64:
	GOOS=darwin GOARCH=arm64 $(GO) build -ldflags="-X 'github.com/DO-Solutions/kubectl-doks/cmd.version=$(VERSION)' -X 'github.com/DO-Solutions/kubectl-doks/cmd.commit=$(COMMIT)'" -o dist/$(BIN_NAME)-darwin-arm64 .

.PHONY: clean
clean:
	rm -f dist/$(BIN_NAME)*

.PHONY: test
test:
	$(GO) test -v ./...

.PHONY: lint
lint:
	golangci-lint run

.PHONY: install
install: build
	mv dist/$(BIN_NAME) $(GOPATH)/bin/

.PHONY: package-for-krew
package-for-krew: build-all
	tar -C dist -czf dist/kubectl-doks-linux-amd64.tar.gz kubectl-doks-linux-amd64
	tar -C dist -czf dist/kubectl-doks-darwin-amd64.tar.gz kubectl-doks-darwin-amd64
	tar -C dist -czf dist/kubectl-doks-darwin-arm64.tar.gz kubectl-doks-darwin-arm64

.PHONY: update-krew-manifest
update-krew-manifest:
	VERSION_NO_V=$(VERSION:v%=%); \
	LINUX_AMD64_SHA256=$$(shasum -a 256 dist/kubectl-doks-linux-amd64.tar.gz | awk '{ print $$1 }'); \
	DARWIN_AMD64_SHA256=$$(shasum -a 256 dist/kubectl-doks-darwin-amd64.tar.gz | awk '{ print $$1 }'); \
	DARWIN_ARM64_SHA256=$$(shasum -a 256 dist/kubectl-doks-darwin-arm64.tar.gz | awk '{ print $$1 }'); \
	sed -e "s|v__VERSION__|$(VERSION)|g" \
	    -e "s|__LINUX_AMD64_SHA256__|$$LINUX_AMD64_SHA256|g" \
	    -e "s|__DARWIN_AMD64_SHA256__|$$DARWIN_AMD64_SHA256|g" \
	    -e "s|__DARWIN_ARM64_SHA256__|$$DARWIN_ARM64_SHA256|g" \
	    $(KREW_MANIFEST_TEMPLATE) > $(KREW_TEMPLATE)
