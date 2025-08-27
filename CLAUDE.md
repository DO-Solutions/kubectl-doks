# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

kubectl-doks is a Kubernetes CLI plugin written in Go that manages DigitalOcean Kubernetes (DOKS) kubeconfig entries. It allows users to synchronize all active DOKS clusters to their local `~/.kube/config` file, removing stale contexts and adding new ones.

## Common Commands

### Build and Development
- `make build` - Build the binary for current platform in `dist/` directory
- `make build-all` - Build for all platforms (Linux amd64, macOS amd64/arm64)
- `make test` - Run all tests with verbose output (`go test -v ./...`)
- `make lint` - Run golangci-lint for code quality checks
- `make clean` - Remove all built binaries from `dist/`

### Testing
- `go test ./...` - Run all tests
- `go test ./cmd` - Test command packages
- `go test ./pkg/kubeconfig` - Test kubeconfig utilities

### Release
- `make package-for-krew` - Build and package binaries for Krew distribution
- `make update-krew-manifest` - Update Krew manifest with new version/checksums

## Architecture

### Core Structure
- **main.go**: Entry point that calls `cmd.Execute()`
- **cmd/**: Cobra CLI command definitions and logic
- **do/**: DigitalOcean API client wrapper using godo library
- **pkg/kubeconfig/**: Core kubeconfig manipulation utilities

### Key Components

#### Command Structure (cmd/)
- `root.go`: Base command with global flags and authentication validation
- `kubeconfig.go`: Parent command for kubeconfig operations
- `sync.go`: Synchronizes all DOKS clusters (adds new, removes stale)
- `save.go`: Saves specific cluster or all new clusters without removing stale ones
- `auth.go`: Authentication utilities for DigitalOcean API
- `version.go`: Version command

#### Kubeconfig Management (pkg/kubeconfig/)
- `file.go`: File I/O operations for kubeconfig
- `merge.go`: Merging cluster credentials into existing kubeconfig
- `prune.go`: Removing stale contexts from kubeconfig
- `backup.go`: Creating backups before modifications
- `extension.go`: Managing DigitalOcean cluster ID extensions in kubeconfig

#### DigitalOcean Integration (do/)
- `client.go`: API client wrapper with authentication context support

### Key Design Patterns
- Uses Cobra for CLI structure with persistent flags
- Implements kubeconfig extensions (`digitalocean.com/cluster-id`) to track cluster identity across recreations
- Supports multiple authentication methods (API tokens, doctl contexts)
- Creates automatic backups before kubeconfig modifications
- Distinguishes between "sync" (add new + remove stale) and "save" (add new only) operations

### Dependencies
- **cobra**: CLI framework
- **godo**: DigitalOcean API client
- **k8s.io/client-go**: Kubernetes client library for kubeconfig handling
- **viper**: Configuration management
- **testify**: Testing framework

### Testing Strategy
- Unit tests for all major components with `_test.go` files
- Test coverage includes authentication, kubeconfig operations, and API client functionality
- Uses testify for assertions and mocking