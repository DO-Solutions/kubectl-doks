package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	k8sclientcmd "k8s.io/client-go/tools/clientcmd"
)

// Mock kubeconfig data
const initialKubeconfigForSync = `
apiVersion: v1
clusters:
- cluster:
    server: https://old-cluster-server
  name: do-nyc1-old-cluster
contexts:
- context:
    cluster: do-nyc1-old-cluster
    user: do-nyc1-old-cluster-admin
  name: do-nyc1-old-cluster
current-context: do-nyc1-old-cluster
kind: Config
users:
- name: do-nyc1-old-cluster-admin
  user:
    token: old-token
`

const mockKubeconfig1ForSync = `
apiVersion: v1
clusters:
- cluster:
    server: https://cluster-1-server
  name: do-nyc1-doks-cluster-1
contexts:
- context:
    cluster: do-nyc1-doks-cluster-1
    user: do-nyc1-doks-cluster-1-admin
  name: do-nyc1-doks-cluster-1
current-context: do-nyc1-doks-cluster-1
kind: Config
users:
- name: do-nyc1-doks-cluster-1-admin
  user:
    token: cluster-1-token
`

const mockKubeconfig2ForSync = `
apiVersion: v1
clusters:
- cluster:
    server: https://cluster-2-server
  name: do-sfo3-doks-cluster-2
contexts:
- context:
    cluster: do-sfo3-doks-cluster-2
    user: do-sfo3-doks-cluster-2-admin
  name: do-sfo3-doks-cluster-2
current-context: do-sfo3-doks-cluster-2
kind: Config
users:
- name: do-sfo3-doks-cluster-2-admin
  user:
    token: cluster-2-token
`

func TestSyncCommand(t *testing.T) {
	// 1. Create a mock API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/kubernetes/clusters" {
			// Respond with a list of clusters
			clusters := []*godo.KubernetesCluster{
				{
					ID:         "cluster-1-id",
					Name:       "doks-cluster-1",
					RegionSlug: "nyc1",
				},
				{
					ID:         "cluster-2-id",
					Name:       "doks-cluster-2",
					RegionSlug: "sfo3",
				},
			}
			// The godo client expects the response to be wrapped in a JSON object
			// with a key matching the resource type.
			response := struct {
				KubernetesClusters []*godo.KubernetesCluster `json:"kubernetes_clusters"`
			}{
				KubernetesClusters: clusters,
			}
			w.Header().Set("Content-Type", "application/json")
			require.NoError(t, json.NewEncoder(w).Encode(response))
		} else if r.URL.Path == "/v2/kubernetes/clusters/cluster-1-id/kubeconfig" {
			// Respond with kubeconfig for cluster 1
			fmt.Fprint(w, mockKubeconfig1ForSync)
		} else if r.URL.Path == "/v2/kubernetes/clusters/cluster-2-id/kubeconfig" {
			// Respond with kubeconfig for cluster 2
			fmt.Fprint(w, mockKubeconfig2ForSync)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// 2. Set up temporary environment and flags
	tmpDir := t.TempDir()

	// Create mock kubeconfig
	kubeConfigDir := filepath.Join(tmpDir, ".kube")
	require.NoError(t, os.MkdirAll(kubeConfigDir, 0755))
	finalKubeConfigPath := filepath.Join(kubeConfigDir, "config")
	require.NoError(t, os.WriteFile(finalKubeConfigPath, []byte(initialKubeconfigForSync), 0600))

	// Override HOME to our temp dir
	originalHome, err := os.UserHomeDir()
	require.NoError(t, err)
	t.Setenv("HOME", tmpDir)
	defer t.Setenv("HOME", originalHome)

	// Override API URL to point to our mock server
	originalAPIURL := apiURL
	apiURL = server.URL
	defer func() { apiURL = originalAPIURL }()

	// Provide a token directly to bypass config file logic
	originalAccessTokens := accessTokens
	accessTokens = []string{"test-token"}
	defer func() { accessTokens = originalAccessTokens }()

	// Reset kubeConfigPath flag to ensure it uses the default path within our temp HOME
	originalKubeConfigPath := kubeConfigPath
	kubeConfigPath = ""
	defer func() { kubeConfigPath = originalKubeConfigPath }()

	// 3. Run the command
	syncCmd.Run(syncCmd, []string{})

	// 4. Verify the results
	// Check that backup was created
	backupPath := finalKubeConfigPath + ".kubectl-doks.bak"
	_, err = os.Stat(backupPath)
	assert.NoError(t, err, "Backup file should exist")

	// Read the resulting kubeconfig
	updatedBytes, err := os.ReadFile(finalKubeConfigPath)
	require.NoError(t, err)

	updatedKubeconfig, err := k8sclientcmd.Load(updatedBytes)
	require.NoError(t, err)

	// Verify contexts
	assert.Contains(t, updatedKubeconfig.Contexts, "do-nyc1-doks-cluster-1", "Context for cluster 1 should exist")
	assert.Contains(t, updatedKubeconfig.Contexts, "do-sfo3-doks-cluster-2", "Context for cluster 2 should exist")
	assert.NotContains(t, updatedKubeconfig.Contexts, "do-nyc1-old-cluster", "Old context should have been removed")

	// Verify clusters
	assert.Contains(t, updatedKubeconfig.Clusters, "do-nyc1-doks-cluster-1", "Cluster 1 should exist")
	assert.Contains(t, updatedKubeconfig.Clusters, "do-sfo3-doks-cluster-2", "Cluster 2 should exist")
	assert.NotContains(t, updatedKubeconfig.Clusters, "do-nyc1-old-cluster", "Old cluster should have been removed")

	// Verify users
	assert.Contains(t, updatedKubeconfig.AuthInfos, "do-nyc1-doks-cluster-1-admin", "User for cluster 1 should exist")
	assert.Contains(t, updatedKubeconfig.AuthInfos, "do-sfo3-doks-cluster-2-admin", "User for cluster 2 should exist")
	assert.NotContains(t, updatedKubeconfig.AuthInfos, "do-nyc1-old-cluster-admin", "Old user should have been removed")
}
