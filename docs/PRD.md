# **Product Requirements Document**

**Plugin Name:** `kubectl-doks`
**Language:** Go
**Repository:** to be created under `github.com/DO-Solutions/kubectl-doks`

---

## 1. Overview & Goals

* **Purpose:** Provide a seamless `kubectl` plugin for managing DigitalOcean Kubernetes (DOKS) credentials in the local `~/.kube/config` file.
* **Primary Goals:**
    1. **Sync** all active DOKS clusters across one or more DO authentication contexts, adding missing contexts and removing stale entries.
    2. **Save** a single cluster’s credentials—by name or interactively—while optionally setting it as the current context.
    3. Work entirely in Go, leveraging existing DO libraries where possible, with idiomatic CLI design (using Cobra).

---

## 2. Functional Requirements

### 2.1 `kubectl doks kubeconfig sync`

| Requirement            | Detail                                                                                                                            |
| ---------------------- |-----------------------------------------------------------------------------------------------------------------------------------|
| Authentication sources | `--access-token`, `--auth-context` (multiple), or `--all-auth-contexts`. Only one kind of Auth Source can be specified at a time. |
| Discovery              | Query all specified DO contexts/teams for clusters via godo/doctl library                                                         |
| Prefix filter          | Only manage kubeconfig `contexts` starting with `do-`                                                                             |
| Backup                 | Always copy existing `~/.kube/config` → `~/.kube/config.kubectl-doks.bak` before changes                                          |
| Add missing contexts   | For each live cluster, fetch kubeconfig and merge                                                                                 |
| Remove stale contexts  | Detect contexts in local file whose cluster is deleted; remove context, user, cluster entries                                     |
| Current-context        | Must not modify                                                                                                                   |
| Verbose logging        | `-v/--verbose`: report teams queried, contexts added/removed, path of kubeconfig updated                                          |
| Exit-on-error          | Fatal if any DO context fails to respond; aggregate errors                                                                        |

Verbose logging example:

```
$ doctl doks kubeconfig sync --all-auth-contexts --verbose
Notice: Getting cluster credentials from Team "do-test1"
Notice: Getting cluster credentials from Team "do-test2"
Notice: Adding contexts: do-sfo3-test1, do-sfo3-test2, do-nyc1-new-test
Notice: Removing contexts: do-sfo3-test1-old
Notice: Syncing cluster credentials to kubeconfig file found in "/home/username/.kube/config"
```

### 2.2 `kubectl doks kubeconfig save [<cluster-name>]`

| Requirement         | Detail                                                                                 |
| ------------------- |----------------------------------------------------------------------------------------|
| Authentication      | Same global flags as `sync`                                                            |
| Cluster selection   | 1) Named argument 2) Interactive TUI list (filterable)                                 |
| Interactive UI      | Use a survey-like library for arrow-key selection and incremental search               |
| Fetch & merge       | Retrieve kubeconfig for selected cluster and merge into `~/.kube/config`               |
| Backup              | Same backup behavior                                                                   |
| Set current-context | `--set-current-context` (default `true`): set `current-context` to newly added context |
| Verbose             | See below for example                                                                  |

Verbose logging example:

```
$ doctl doks kubeconfig save test1 --verbose
Notice: Adding context: do-sfo3-test1
Notice: Saving cluster credentials to kubeconfig file found in "/home/username/.kube/config"
Notice: Setting current-context to do-sfo3-test1
```

---

## 3. Non-Functional & Global Technical Details

* **Language & Libraries**:
    * **CLI framework:** [spf13/cobra](https://github.com/spf13/cobra)
    * **DO API client:** Prefer reuse of **doctl** logic if exposed as library; otherwise use [godo](https://github.com/digitalocean/godo)
    * **Kubeconfig manipulation:** [`client-go/tools/clientcmd`](https://pkg.go.dev/k8s.io/client-go/tools/clientcmd)
    * **Interactive prompt:** e.g. [AlecAivazis/survey](https://github.com/AlecAivazis/survey)

* **Configuration**:
    * Support overriding DO API URL (`--api-url`) and doctl config file path (`--config`)
    * Environment fallback for `DIGITALOCEAN_ACCESS_TOKEN` and default doctl context

* **Error Handling**:
    * Validate mutual exclusivity of auth flags
    * Aggregate and surface all DO context errors
    * Exit with non-zero code on any fatal error

* **Logging**:
    * Default: quiet, no output unless error encountered
    * Verbose: detailed logs as shown in examples

* **Backup Strategy**:
    * Single backup per run: overwrite if backup file exists
    * Ensure atomic write: write to temp file then rename

* **Testing**:
    * Unit tests for:
        * Flag parsing and validation
        * Cluster list diffs (add/remove detection)
        * Kubeconfig merge and prune logic (using in-memory temp files)
    * Integration tests:
        * Mock DO API with httptest server to simulate multiple contexts and error scenarios
    * Makefile with the following targets:
        * `lint` validates that go files are formatted and followed best practices using static analisys
        * `test-unit` runs unit tests
        * `test-integration` run integration tests
        * `test-all` runs unit and then integration tests

* **Build & Release**:
    * Go modules enabled
    * Makefile with the following targets:
        * `build` builds binary
        * `clean` removes old binary or any temp files created during build. 
    * GitHub Actions workflow:
        * `make lint` and `make test-all` on PRs
        * Build binaries for Linux/macOS (amd64)
        * Publish GitHub release artifacts (tar.gz)

---

## 4. Architecture & Code Organization

```text
cmd/
  root.go         # cobra root (plugin entrypoint)
  root_test.go    # unit test (if applicable)
  sync.go         # sync command implementation
  sync_test.go    # unit test (if applicable)
  save.go         # save command implementation
  save_test.go    # unit test (if applicable)

do/
  client.go       # wrapper over doctl/godo for cluster listing and kubeconfig retrieval
  client_test.go  # unit test (if applicable)

pkg/
  kubeconfig/
    merge.go        # functions to merge new kubeconfig blobs into existing
    merge_test.go   # unit test (if applicable)
    prune.go        # functions to remove stale contexts
    prune_test.go   # unit test (if applicable)

internal/
  ui/
    prompt.go        # interactive cluster selector
    prompt_test.go   # unit test (if applicable)

util/
  backup.go         # atomic backup logic
  backup_test.go    # unit test (if applicable)
  logging.go        # verbose vs. quiet logging helpers
  logging_test.go   # unit test (if applicable)

test/
  integration/
    # integration tests
```

### Directory Conventions

* `cmd/`: Contains the entrypoint and command implementations for the Cobra-based CLI. Each top-level command (e.g., `sync`, `save`) lives here.
* `do/`: Encapsulates DigitalOcean API interactions, abstracting away whether we use `doctl` libraries or direct `godo` calls.
* `pkg/`: Public Go packages meant to be imported by external code. This holds reusable, domain-specific logic (e.g., `kubeconfig` merge/prune functions) that could be leveraged in other projects.
* `internal/`: Private packages that are only available within this module. Use this for code that should not be consumed by external consumers—such as the UI prompt implementation.
* `util/`: Helper code that spans multiple areas of the codebase but isn’t domain-specific enough for `pkg/`. Examples include atomic backup logic and logging abstractions.

---

## 5. External References & Reuse

* **doctl**

    * Commands: [commands/kubernetes.go](https://github.com/digitalocean/doctl/blob/main/commands/kubernetes.go)
    * Core logic: [do/kubernetes.go](https://github.com/digitalocean/doctl/blob/main/do/kubernetes.go)

* **godo**

    * Kubernetes API: [kubernetes.go](https://github.com/digitalocean/godo/blob/main/kubernetes.go)

> Evaluate whether `doctl` can be imported as a module to reuse CLI flags, authentication setup, and kubeconfig generation logic. If `doctl` doesn’t expose the needed interfaces, wrap `godo` directly.
