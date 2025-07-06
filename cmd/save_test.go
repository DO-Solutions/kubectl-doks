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
const initialKubeconfigForSave = `
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

const mockKubeconfigForSave = `
apiVersion: v1
clusters:
- cluster:
    server: https://new-cluster-server
  name: do-sfo3-new-cluster
contexts:
- context:
    cluster: do-sfo3-new-cluster
    user: do-sfo3-new-cluster-admin
  name: do-sfo3-new-cluster
current-context: do-sfo3-new-cluster
kind: Config
users:
- name: do-sfo3-new-cluster-admin
  user:
    token: new-token
`

func TestSaveCommand(t *testing.T) {
	// 1. Create a mock API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/kubernetes/clusters" {
			clusters := []*godo.KubernetesCluster{
				{
					ID:         "new-cluster-id",
					Name:       "new-cluster",
					RegionSlug: "sfo3",
				},
			}
			response := struct {
				KubernetesClusters []*godo.KubernetesCluster `json:"kubernetes_clusters"`
			}{
				KubernetesClusters: clusters,
			}
			w.Header().Set("Content-Type", "application/json")
			require.NoError(t, json.NewEncoder(w).Encode(response))
		} else if r.URL.Path == "/v2/kubernetes/clusters/new-cluster-id/kubeconfig" {
			fmt.Fprint(w, mockKubeconfigForSave)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// 2. Set up temporary environment and flags
	tmpDir := t.TempDir()

	kubeConfigDir := filepath.Join(tmpDir, ".kube")
	require.NoError(t, os.MkdirAll(kubeConfigDir, 0755))
	finalKubeConfigPath := filepath.Join(kubeConfigDir, "config")
	require.NoError(t, os.WriteFile(finalKubeConfigPath, []byte(initialKubeconfigForSave), 0600))

	originalHome, err := os.UserHomeDir()
	require.NoError(t, err)
	t.Setenv("HOME", tmpDir)
	defer t.Setenv("HOME", originalHome)

	originalAPIURL := apiURL
	apiURL = server.URL
	defer func() { apiURL = originalAPIURL }()

	originalAccessTokens := accessTokens
	accessTokens = []string{"test-token"}
	defer func() { accessTokens = originalAccessTokens }()

	originalKubeConfigPath := kubeConfigPath
	kubeConfigPath = ""
	defer func() { kubeConfigPath = originalKubeConfigPath }()

	// 3. Run the command
	err = saveCmd.RunE(saveCmd, []string{"new-cluster"})
	require.NoError(t, err)

	// 4. Verify the results
	updatedBytes, err := os.ReadFile(finalKubeConfigPath)
	require.NoError(t, err)

	updatedKubeconfig, err := k8sclientcmd.Load(updatedBytes)
	require.NoError(t, err)

	// Verify new context
	assert.Contains(t, updatedKubeconfig.Contexts, "do-sfo3-new-cluster", "Context for new cluster should exist")
	// Verify old context still exists
	assert.Contains(t, updatedKubeconfig.Contexts, "do-nyc1-old-cluster", "Old context should still exist")

	// Verify new cluster
	assert.Contains(t, updatedKubeconfig.Clusters, "do-sfo3-new-cluster", "New cluster should exist")
	// Verify old cluster still exists
	assert.Contains(t, updatedKubeconfig.Clusters, "do-nyc1-old-cluster", "Old cluster should still exist")

	// Verify new user
	assert.Contains(t, updatedKubeconfig.AuthInfos, "do-sfo3-new-cluster-admin", "User for new cluster should exist")
	// Verify old user still exists
	assert.Contains(t, updatedKubeconfig.AuthInfos, "do-nyc1-old-cluster-admin", "Old user should still exist")
}
