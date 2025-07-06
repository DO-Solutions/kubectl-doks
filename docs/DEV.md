# Development Roadmap for `kubectl-doks`

This roadmap breaks down the work into incremental, testable, and shippable milestones. Each step should be in a state where it can be unit testes and merged into `main`.

---

## Milestone 1: Project Bootstrap & CLI Scaffolding

**Objectives**:
* Initialize Go module and repository structure.
* Add Cobra CLI framework with root command (`kubectl doks`).
* Implement global flags and validation (mutual exclusivity of `--access-token`, `--auth-context`, `--all-auth-contexts`).
* Write unit tests for flag parsing and validation logic.

**Deliverables**:
1. `go.mod`, `go.sum` initialized.
2. `cmd/root.go` with `--access-token`, `--auth-context`, `--all-auth-contexts`, `--api-url`, `--config`, `--expiry-seconds`, `--verbose` flags.
3. Validation that exactly one of the auth flags is used.
4. Unit tests for flag parsing in `cmd/root_test.go`.

---

## Milestone 2: DO API Client Abstraction

**Objectives**:
* Create a `do/client.go` that wraps authentication and cluster listing using `godo` or `doctl` libraries.
* Expose methods:
    * `ListClusters(ctx) ([]Cluster, error)`
    * `GetKubeConfig(ctx, clusterID) ([]byte, error)`
* Write unit tests using an HTTP mock server to simulate API responses and error scenarios.

**Deliverables**:
1. `do/client.go` with authentication setup reading flags and environment.
2. `ListClusters` and `GetKubeConfig` implementations.
3. Unit tests in `do/client_test.go` using `httptest` to mock DO API.

---

## Milestone 3: Kubeconfig Backup & Merge Utilities

**Objectives**:
* Implement atomic backup of `~/.kube/config` → `~/.kube/config.kubectl-doks.bak` in `util/backup.go`.
* Create `pkg/kubeconfig/merge.go`:
    * Parse existing kubeconfig.
    * Merge raw kubeconfig bytes into the existing structure.
* Unit tests for backup logic and merge functions using temp files and in-memory data.

**Deliverables**:
1. `util/backup.go` with `BackupKubeconfig(srcPath, backupPath) error`.
2. `pkg/kubeconfig/merge.go` with `MergeConfig(srcConfig, newConfig []byte) ([]byte, error)`.
3. Unit tests under `pkg/kubeconfig/merge_test.go` and `utils/backup_test.go`.

---

## Milestone 4: Kubeconfig Prune Utility

**Objectives**:
* In `pkg/kubeconfig/prune.go`, implement logic to remove contexts, clusters, and users whose context names start with `do-` but whose corresponding cluster no longer exists.
* Unit tests to verify contexts removed and dependencies preserved for other contexts.

**Deliverables**:

1. `PruneConfig(config []byte, liveClusters []Cluster) ([]byte, []string /*removedContexts*/, error)`.
2. Tests in `pkg/kubeconfig/prune_test.go` covering:
    * Removing single/multiple stale contexts.
    * No-op when all contexts are valid.

---

## Milestone 5: `sync` Command Implementation

**Objectives**:
* Put it all together in `cmd/sync.go`:
    1. Backup existing config.
    2. For each auth context, call `ListClusters`.
    3. Read existing kubeconfig.
    4. Prune stale entries.
    5. For each live cluster not in config, fetch and merge its kubeconfig.
    6. Write updated config back to disk.
    7. Log verbose notices if `--verbose`.
* Unit/Integration tests to simulate full sync flow with mocked `do` and in-memory kubeconfig.

**Deliverables**:
1. `cmd/sync.go` with full sync logic.
2. Integration test in `tests/integration/sync_integration_test.go` using `httptest` and temp files.

---

## Milestone 6: Interactive Cluster Selector & `save` Command

**Objectives**:
* Add `internal/ui/prompt.go` with a survey-based selector:
    * Accepts a list of clusters and returns the selected one.
* Implement `cmd/save.go`:
    1. Backup existing config.
    2. If cluster name provided, validate existence; otherwise prompt.
    3. Fetch that cluster’s kubeconfig.
    4. Merge into existing config.
    5. Set `current-context` if flag is true.
* Unit tests for prompt (mocked user input) and save logic.

**Deliverables**:
1. `internal/ui/prompt.go` using `survey`.
2. `cmd/save.go` fully functional.
3. Tests in `internal/ui/prompt_test.go` and `cmd/save_test.go`.


