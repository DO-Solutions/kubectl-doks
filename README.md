# kubectl-doks

A Kubernetes CLI plugin to manage DigitalOcean Kubernetes (DOKS) kubeconfig entries. Easily synchronize all active DOKS clusters to your local `~/.kube/config` and remove stale contexts, or save a single cluster’s credentials interactively or by name.

This plugin is ideal for environments where clusters are created and destroyed frequently: it keeps your local kubeconfig in sync by removing credentials for deleted clusters and adding new ones in a single command. You can stay in the familiar `kubectl` context without switching back to `doctl`, and the built‑in text‑based UI makes selecting programmatically generated cluster names simple, even when you don’t remember the exact identifier.

---

## Installation

### Prebuilt Binary

1. Download the latest release from GitHub:

   ```bash
   curl -LO https://github.com/DO-Solutions/kubectl-doks/releases/download/vX.Y.Z/kubectl-doks_$(uname | tr '[:upper:]' '[:lower:]')_amd64.tar.gz
   tar -xzvf kubectl-doks_*.tar.gz
   mv kubectl-doks /usr/local/bin/
   ```
2. Verify installation:

   ```bash
   kubectl plugin list
   # should show "doks" in the list of installed plugins
   ```

### Install via Custom krew Index

1. Add your custom krew index (if not already added):

   ```bash
   kubectl krew index add mykrew https://github.com/DO-Solutions/krew-index.git
   ```
2. Install the plugin:

   ```bash
   kubectl krew install doks
   ```

---

## Usage

```bash
# Synchronize all DOKS clusters to ~/.kube/config
kubectl doks kubeconfig sync [flags]

# Save a single cluster’s credentials (by name or interactively)
kubectl doks kubeconfig save [<cluster-name>] [flags]
```

### Commands

#### `kubeconfig sync`

* **Description**: Fetches all reachable teams’ DOKS clusters and ensures your local `~/.kube/config` contains only the contexts matching existing clusters (contexts start with `do-`).
* **Behavior**:
    * Creates a backup of the existing kubeconfig at `~/.kube/config.kubectl-doks.bak` before modifying it
    * Adds missing contexts for active clusters and removes contexts (and related cluster/users) for deleted clusters
    * Does *not* change `current-context`

#### `kubeconfig save [cluster-name]`

* **Description**: Fetches a single cluster’s credentials and merges them into `~/.kube/config`. 
* **Behavior**:
    * Works the same way `doctl kubernetes cluster kubeconfig save` works if `<cluster-name>` is provided.
    * If `<cluster-name>` is omitted, launches an interactive prompt to pick a cluster
    * By default, sets `current-context` to the newly saved context

---

## Flags

| Flag                           | Description                                                                 | Applicable To |
| ------------------------------ | --------------------------------------------------------------------------- | ------------- |
| `-t`, `--access-token`         | DigitalOcean API V2 token                                                   | global        |
| `-u`, `--api-url`              | Override the default DigitalOcean API endpoint                              | global        |
| `-c`, `--config`               | Path to `doctl` config file (default: `$HOME/.config/doctl/config.yaml`)    | global        |
| `--auth-context`               | Use this `doctl` authentication context (can specify multiple times)        | global        |
| `--all-auth-contexts`          | Include all `doctl` authentication contexts                                 | global        |
| `--expiry-seconds` `<seconds>` | Credential TTL in seconds; auto-renewal is enabled by default               | global        |
| `-v`, `--verbose`              | Enable verbose output (reports added/removed contexts, teams queried, etc.) | global        |
| `--set-current-context`        | *(save only)* Set `current-context` to the new context (default: `true`)    | save          |

**Notes**:

* `--access-token` and `--auth-context` flags may each be specified multiple times to fetch clusters from multiple teams.
* Combining `--access-token`, `--auth-context`, and `--all-auth-contexts` is not allowed; the plugin will exit with an error if more than one of these modes is used.

---

## Examples

```bash
# Sync all clusters for default doctl context
kubectl doks kubeconfig sync

# Sync, using a specific token and config file
kubectl doks kubeconfig sync -t $DIGITALOCEAN_ACCESS_TOKEN -c ~/.config/doctl/custom.yaml

# Save credentials for a named cluster and switch context
# This is functionally the same as 'doctl kubernetes cluster kubeconfig save do-test-1'   
kubectl doks kubeconfig save do-test-1

# Save interactively from multiple teams
kubectl doks kubeconfig save --auth-context test-team-1 --auth-context test-team-2
```

---

## Error Handling

* Fatal if unable to reach any specified team (reports all failures at once).
* Exits with non-zero status on invalid flags or API errors.

---

## Contributing

Contributions welcome! Please open issues and pull requests against the `main` branch.
