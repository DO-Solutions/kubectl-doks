# kubectl-doks

A Kubernetes CLI plugin to manage DigitalOcean Kubernetes (DOKS) kubeconfig entries. Easily synchronize all active DOKS clusters to your local `~/.kube/config` and remove stale contexts, or save cluster credentials without removing any contexts.

This plugin is ideal for environments where clusters are created and destroyed frequently: it keeps your local kubeconfig in sync by removing credentials for deleted clusters and adding new ones in a single command. You can stay in the familiar `kubectl` context without switching back to `doctl`.

---

## Installation

### Install via krew

1.  Add the custom krew index from this repository:

    ```bash
    kubectl krew index add kubectl-doks https://github.com/DO-Solutions/kubectl-doks.git
    ```

2.  Install the `doks` plugin:

    ```bash
    kubectl krew install kubectl-doks/doks 
    ```

### Prebuilt Binary

Download the appropriate binary for your operating system and architecture from the [GitHub Releases](https://github.com/DO-Solutions/kubectl-doks/releases) page.

#### Linux (amd64)

```bash
curl -LO https://github.com/DO-Solutions/kubectl-doks/releases/latest/download/kubectl-doks-linux-amd64.tar.gz
tar xvf kubectl-doks-linux-amd64.tar.gz
sudo install kubectl-doks-linux-amd64 /usr/local/bin/kubectl-doks
```

#### macOS (arm64)

```bash
curl -LO https://github.com/DO-Solutions/kubectl-doks/releases/latest/download/kubectl-doks-darwin-arm64.tar.gz
tar xvf kubectl-doks-darwin-arm64.tar.gz
sudo install kubectl-doks-darwin-arm64 /usr/local/bin/kubectl-doks
```

#### macOS (amd64)

```bash
curl -LO https://github.com/DO-Solutions/kubectl-doks/releases/latest/download/kubectl-doks-darwin-amd64.tar.gz
tar xvf kubectl-doks-darwin-amd64.tar.gz
sudo install kubectl-doks-darwin-amd64 /usr/local/bin/kubectl-doks
```

After installation, verify it's available:


```bash
kubectl plugin list
# should show "doks" in the list of installed plugins
```

---

## Usage

```bash
# Synchronize all DOKS clusters to ~/.kube/config
kubectl doks kubeconfig sync [flags]

# Save credentials for a single named cluster or all new clusters
kubectl doks kubeconfig save [<cluster-name>] [flags]
```

### Commands

#### `kubeconfig sync`

*   **Description**: Fetches all DOKS clusters from the configured DigitalOcean authentication contexts and synchronizes your local `~/.kube/config` file.
*   **Behavior**:
    *   Creates a backup of the existing kubeconfig at `~/.kube/config.kubectl-doks.bak` before modifying it.
    *   **Adds** contexts for any new clusters found on DigitalOcean that are not in your local kubeconfig.
    *   **Removes** stale contexts (and related cluster/user entries) from your kubeconfig if the corresponding cluster no longer exists on DigitalOcean. It only removes contexts prefixed with `do-`.
    *   By default, it will set the `current-context` if the current-context is not set (which could have been a stale context that was removed) and only one new context is added. This can be disabled with `--set-current-context=false`.

#### `kubeconfig save [<cluster-name>]`

*   **Description**: Fetches credentials and merges them into `~/.kube/config`. This command has two modes of operation depending on whether a cluster name is provided.
*   **Behavior**:
    *   **When a `<cluster-name>` is provided**: It saves the credentials for that specific cluster. This is functionally equivalent to `doctl kubernetes cluster kubeconfig save <cluster-name>`.
    *   **When `<cluster-name>` is omitted**: It saves the credentials for **all** available clusters that are not already in your kubeconfig. This is useful for adding all new clusters without removing old ones.
    *   By default, it sets the `current-context` in two cases:
        *   When saving a single, named cluster.
        *   When saving all clusters, if only one new context is added and no `current-context` is already set.
    *   This behavior can be disabled with `--set-current-context=false`.

#### `version`

*   **Description**: Print the version number of kubectl-doks.
*   **Behavior**: Prints the version number of kubectl-doks to the console.

---

## Flags

| Flag | Description | Applicable To |
| --- | --- | --- |
| `-t`, `--access-token` | DigitalOcean API V2 token (can be specified multiple times) | global |
| `-u`, `--api-url` | Override the default DigitalOcean API endpoint | global |
| `-c`, `--config` | Path to `doctl` config file | global |
| `--auth-context` | Use this `doctl` authentication context (can be specified multiple times) | global |
| `--all-auth-contexts` | Include all `doctl` authentication contexts | global |
| `-v`, `--verbose` | Enable verbose output (reports added/removed contexts, teams queried, etc.) | global |
| `--set-current-context` | Set `current-context` after a `save` or `sync` operation (default: `true`). See command descriptions for specific behavior. | global |

**Notes**:

*   You must provide an authentication method via one of the following (in order of precedence): `--access-token`, `--auth-context`, `--all-auth-contexts`, or the `DIGITALOCEAN_ACCESS_TOKEN` environment variable. If none are provided, the plugin will attempt to use your current `doctl` configuration.
*   Combining `--access-token`, `--auth-context`, and `--all-auth-contexts` is not allowed; the plugin will exit with an error if more than one of these modes is used.

---

## Kubeconfig Modification Details

When you use the `kubeconfig sync` or `kubeconfig save` commands the plugin modifies your kubeconfig file to include a DigitalOcean-specific extension. This helps the tool track clusters more accurately, especially when a cluster is deleted and recreated with the same name.

Specifically, it adds an extension named `digitalocean.com/cluster-id` to each cluster entry in your kubeconfig. This extension stores the unique ID of the DOKS cluster.

When `kubeconfig sync` is run, it compares the cluster ID from the DigitalOcean API with the one stored in the kubeconfig extension. If the IDs do not match, `kubectl-doks` recognizes that the cluster has been recreated. It then updates the kubeconfig with the new cluster's credentials, ensuring that you are always connecting to the correct cluster instance. This prevents issues where `kubectl` might try to connect to a stale or non-existent cluster that happened to share a name with a new one.

---

## Examples

```bash
# Sync all clusters for the current doctl context.
# This adds new clusters and removes stale ones.
kubectl doks kubeconfig sync

# Saves all clusters for the current doctl context.
# This adds new clusters and but does not remove stale ones.
kubectl doks kubeconfig save

# Sync all clusters for all doctl contexts.
# This adds new clusters and removes stale ones.
kubectl doks kubeconfig sync --all-auth-contexts

# Sync clusters using a specific API token.
kubectl doks kubeconfig sync -t $DIGITALOCEAN_ACCESS_TOKEN

# Save credentials for a single named cluster and switch the current context to it.
kubectl doks kubeconfig save my-cluster-name

# Save credentials for all new/missing clusters from multiple specified teams
# without changing the current context.
kubectl doks kubeconfig save --auth-context test-team-1 --auth-context test-team-2

# Save a single cluster but prevent changing the current context.
kubectl doks kubeconfig save my-cluster-name --set-current-context=false
```

---

## Error Handling

*   Fatal if unable to reach any specified team (reports all failures at once).
*   Exits with non-zero status on invalid flags or API errors.

---

## Contributing

Contributions welcome! Please open issues and pull requests against the `main` branch.

---

## Release Process

This project uses GitHub Actions to automate the release process. A new release is created whenever a tag matching the pattern `v*` (e.g., `v1.0.0`) is pushed to the repository.

The release workflow automatically performs the following steps:

1.  **Builds and Packages Binaries**: It cross-compiles the `kubectl-doks` binary for various platforms and packages them into `.tar.gz` archives.
2.  **Updates Krew Manifest**: It updates the `krew-index/plugins/doks.yaml` file with the new version and SHA256 checksums for the packaged binaries.
3.  **Commits Manifest**: It commits the updated krew manifest back to the repository.
4.  **Creates GitHub Release**: It creates a new GitHub release corresponding to the pushed tag.
5.  **Attaches Artifacts**: The packaged `.tar.gz` archives are attached as downloadable artifacts to the GitHub release.
