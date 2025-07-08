# Makefile for kubectl-doks

BIN_NAME := kubectl-doks
GO := go

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
	rm -f dist/$(BIN_NAME) dist/$(BIN_NAME)-*

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
package-for-krew:
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X 'github.com/DO-Solutions/kubectl-doks/cmd.version=$(VERSION)' -X 'github.com/DO-Solutions/kubectl-doks/cmd.commit=$(COMMIT)'" -o dist/kubectl-doks-linux-amd64 .
	tar -C dist -czf dist/kubectl-doks-linux-amd64.tar.gz kubectl-doks-linux-amd64
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w -X 'github.com/DO-Solutions/kubectl-doks/cmd.version=$(VERSION)' -X 'github.com/DO-Solutions/kubectl-doks/cmd.commit=$(COMMIT)'" -o dist/kubectl-doks-darwin-amd64 .
	tar -C dist -czf dist/kubectl-doks-darwin-amd64.tar.gz kubectl-doks-darwin-amd64
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w -X 'github.com/DO-Solutions/kubectl-doks/cmd.version=$(VERSION)' -X 'github.com/DO-Solutions/kubectl-doks/cmd.commit=$(COMMIT)'" -o dist/kubectl-doks-darwin-arm64 .
	tar -C dist -czf dist/kubectl-doks-darwin-arm64.tar.gz kubectl-doks-darwin-arm64

.PHONY: update-krew-manifest
update-krew-manifest:
	KREW_TEMPLATE=krew-index/plugins/kubectl-doks.yaml
	VERSION_NO_V=$(VERSION:v%=%)
	sed -i "" "s|v__VERSION__|$(VERSION)|g" $(KREW_TEMPLATE)
	LINUX_AMD64_SHA256=$$(shasum -a 256 dist/kubectl-doks-linux-amd64.tar.gz | awk '{ print $$1 }')
	sed -i "" "s|__LINUX_AMD64_SHA256__|$$LINUX_AMD64_SHA256|g" $(KREW_TEMPLATE)
	DARWIN_AMD64_SHA256=$$(shasum -a 256 dist/kubectl-doks-darwin-amd64.tar.gz | awk '{ print $$1 }')
	sed -i "" "s|__DARWIN_AMD64_SHA256__|$$DARWIN_AMD64_SHA256|g" $(KREW_TEMPLATE)
	DARWIN_ARM64_SHA256=$$(shasum -a 256 dist/kubectl-doks-darwin-arm64.tar.gz | awk '{ print $$1 }')
	sed -i "" "s|__DARWIN_ARM64_SHA256__|$$DARWIN_ARM64_SHA256|g" $(KREW_TEMPLATE)
