package cmd

import (
	"github.com/DO-Solutions/kubectl-doks/pkg/kubeconfig"
	k8sclientcmdapi "k8s.io/client-go/tools/clientcmd/api"
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
	err = syncCmd.RunE(syncCmd, []string{})
	require.NoError(t, err)

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

	// Verify cluster ID extension was added
	cluster1, exists := updatedKubeconfig.Clusters["do-nyc1-doks-cluster-1"]
	require.True(t, exists)
	cluster1ID, found := kubeconfig.GetClusterID(cluster1)
	assert.True(t, found)
	assert.Equal(t, "cluster-1-id", cluster1ID)

	cluster2, exists := updatedKubeconfig.Clusters["do-sfo3-doks-cluster-2"]
	require.True(t, exists)
	cluster2ID, found := kubeconfig.GetClusterID(cluster2)
	assert.True(t, found)
	assert.Equal(t, "cluster-2-id", cluster2ID)
}

func TestSyncCommandContextHandling(t *testing.T) {
	setup := func(t *testing.T, initialKubeconfig string, server *httptest.Server) (string, func()) {
		tmpDir := t.TempDir()
		kubeConfigDir := filepath.Join(tmpDir, ".kube")
		require.NoError(t, os.MkdirAll(kubeConfigDir, 0755))
		finalKubeConfigPath := filepath.Join(kubeConfigDir, "config")
		require.NoError(t, os.WriteFile(finalKubeConfigPath, []byte(initialKubeconfig), 0600))

		originalHome, err := os.UserHomeDir()
		require.NoError(t, err)
		t.Setenv("HOME", tmpDir)

		originalAPIURL := apiURL
		apiURL = server.URL

		originalAccessTokens := accessTokens
		accessTokens = []string{"test-token"}

		originalKubeConfigPath := kubeConfigPath
		kubeConfigPath = ""

		return finalKubeConfigPath, func() {
			t.Setenv("HOME", originalHome)
			apiURL = originalAPIURL
			accessTokens = originalAccessTokens
			kubeConfigPath = originalKubeConfigPath
		}
	}

	t.Run("set new context when old is removed and one new is added", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/v2/kubernetes/clusters" {
				clusters := []*godo.KubernetesCluster{{ID: "cluster-1-id", Name: "doks-cluster-1", RegionSlug: "nyc1"}}
				response := struct{ KubernetesClusters []*godo.KubernetesCluster `json:"kubernetes_clusters"` }{KubernetesClusters: clusters}
				w.Header().Set("Content-Type", "application/json")
				require.NoError(t, json.NewEncoder(w).Encode(response))
			} else if r.URL.Path == "/v2/kubernetes/clusters/cluster-1-id/kubeconfig" {
				fmt.Fprint(w, mockKubeconfig1ForSync)
			}
		}))
		defer server.Close()

		finalKubeConfigPath, cleanup := setup(t, initialKubeconfigForSync, server)
		defer cleanup()

		setCurrentContext = true
		err := syncCmd.RunE(syncCmd, []string{})
		require.NoError(t, err)

		updatedBytes, err := os.ReadFile(finalKubeConfigPath)
		require.NoError(t, err)
		updatedKubeconfig, err := k8sclientcmd.Load(updatedBytes)
		require.NoError(t, err)
		assert.Equal(t, "do-nyc1-doks-cluster-1", updatedKubeconfig.CurrentContext)
	})

	t.Run("do not set new context when flag is false", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/v2/kubernetes/clusters" {
				clusters := []*godo.KubernetesCluster{{ID: "cluster-1-id", Name: "doks-cluster-1", RegionSlug: "nyc1"}}
				response := struct{ KubernetesClusters []*godo.KubernetesCluster `json:"kubernetes_clusters"` }{KubernetesClusters: clusters}
				w.Header().Set("Content-Type", "application/json")
				require.NoError(t, json.NewEncoder(w).Encode(response))
			} else if r.URL.Path == "/v2/kubernetes/clusters/cluster-1-id/kubeconfig" {
				fmt.Fprint(w, mockKubeconfig1ForSync)
			}
		}))
		defer server.Close()

		finalKubeConfigPath, cleanup := setup(t, initialKubeconfigForSync, server)
		defer cleanup()

		setCurrentContext = false
		err := syncCmd.RunE(syncCmd, []string{})
		require.NoError(t, err)

		updatedBytes, err := os.ReadFile(finalKubeConfigPath)
		require.NoError(t, err)
		updatedKubeconfig, err := k8sclientcmd.Load(updatedBytes)
		require.NoError(t, err)
		assert.Equal(t, "", updatedKubeconfig.CurrentContext)
	})

	t.Run("do not set new context if multiple clusters are added", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/v2/kubernetes/clusters" {
				clusters := []*godo.KubernetesCluster{
					{ID: "cluster-1-id", Name: "doks-cluster-1", RegionSlug: "nyc1"},
					{ID: "cluster-2-id", Name: "doks-cluster-2", RegionSlug: "sfo3"},
				}
				response := struct{ KubernetesClusters []*godo.KubernetesCluster `json:"kubernetes_clusters"` }{KubernetesClusters: clusters}
				w.Header().Set("Content-Type", "application/json")
				require.NoError(t, json.NewEncoder(w).Encode(response))
			} else if r.URL.Path == "/v2/kubernetes/clusters/cluster-1-id/kubeconfig" {
				fmt.Fprint(w, mockKubeconfig1ForSync)
			} else if r.URL.Path == "/v2/kubernetes/clusters/cluster-2-id/kubeconfig" {
				fmt.Fprint(w, mockKubeconfig2ForSync)
			}
		}))
		defer server.Close()

		finalKubeConfigPath, cleanup := setup(t, initialKubeconfigForSync, server)
		defer cleanup()

		setCurrentContext = true
		err := syncCmd.RunE(syncCmd, []string{})
		require.NoError(t, err)

		updatedBytes, err := os.ReadFile(finalKubeConfigPath)
		require.NoError(t, err)
		updatedKubeconfig, err := k8sclientcmd.Load(updatedBytes)
		require.NoError(t, err)
		assert.Equal(t, "", updatedKubeconfig.CurrentContext)
	})

	t.Run("remove stale contexts when no clusters are found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/v2/kubernetes/clusters" {
				clusters := []*godo.KubernetesCluster{}
				response := struct{ KubernetesClusters []*godo.KubernetesCluster `json:"kubernetes_clusters"` }{KubernetesClusters: clusters}
				w.Header().Set("Content-Type", "application/json")
				require.NoError(t, json.NewEncoder(w).Encode(response))
			}
		}))
		defer server.Close()

		finalKubeConfigPath, cleanup := setup(t, initialKubeconfigForSync, server)
		defer cleanup()

		setCurrentContext = true
		err := syncCmd.RunE(syncCmd, []string{})
		require.NoError(t, err)

		updatedBytes, err := os.ReadFile(finalKubeConfigPath)
		require.NoError(t, err)
		updatedKubeconfig, err := k8sclientcmd.Load(updatedBytes)
		require.NoError(t, err)

		assert.NotContains(t, updatedKubeconfig.Contexts, "do-nyc1-old-cluster", "Old context should have been removed")
		assert.NotContains(t, updatedKubeconfig.Clusters, "do-nyc1-old-cluster", "Old cluster should have been removed")
		assert.NotContains(t, updatedKubeconfig.AuthInfos, "do-nyc1-old-cluster-admin", "Old user should have been removed")
		assert.Empty(t, updatedKubeconfig.CurrentContext, "Current context should be empty")
	})
}

func TestSyncCommandWithRecreatedCluster(t *testing.T) {
	// 1. Create a mock API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/kubernetes/clusters" {
			clusters := []*godo.KubernetesCluster{
				{
					ID:         "new-recreated-cluster-id",
					Name:       "doks-recreated-cluster",
					RegionSlug: "nyc1",
				},
			}
			response := struct {
				KubernetesClusters []*godo.KubernetesCluster `json:"kubernetes_clusters"`
			}{
				KubernetesClusters: clusters,
			}
			w.Header().Set("Content-Type", "application/json")
			require.NoError(t, json.NewEncoder(w).Encode(response))
		} else if r.URL.Path == "/v2/kubernetes/clusters/new-recreated-cluster-id/kubeconfig" {
			fmt.Fprint(w, `
apiVersion: v1
clusters:
- cluster:
    server: https://new-recreated-cluster-server
  name: do-nyc1-doks-recreated-cluster
contexts:
- context:
    cluster: do-nyc1-doks-recreated-cluster
    user: do-nyc1-doks-recreated-cluster-admin
  name: do-nyc1-doks-recreated-cluster
current-context: do-nyc1-doks-recreated-cluster
kind: Config
users:
- name: do-nyc1-doks-recreated-cluster-admin
  user:
    token: new-recreated-cluster-token
`)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// 2. Set up temporary environment and flags
	tmpDir := t.TempDir()

	// Create mock kubeconfig with an old cluster ID
	kubeConfigDir := filepath.Join(tmpDir, ".kube")
	require.NoError(t, os.MkdirAll(kubeConfigDir, 0755))
	finalKubeConfigPath := filepath.Join(kubeConfigDir, "config")

	// Create a kubeconfig object programmatically to add the extension
	initialConfig := k8sclientcmdapi.NewConfig()
	clusterName := "do-nyc1-doks-recreated-cluster"
	cluster := k8sclientcmdapi.NewCluster()
	cluster.Server = "https://old-recreated-cluster-server"
	kubeconfig.SetClusterID(cluster, "old-recreated-cluster-id")
	initialConfig.Clusters[clusterName] = cluster

	// Add context and user for completeness
	contextName := clusterName
	context := k8sclientcmdapi.NewContext()
	context.Cluster = clusterName
	context.AuthInfo = clusterName + "-admin"
	initialConfig.Contexts[contextName] = context
	initialConfig.CurrentContext = contextName

	authInfo := k8sclientcmdapi.NewAuthInfo()
	authInfo.Token = "old-token"
	initialConfig.AuthInfos[context.AuthInfo] = authInfo

	initialBytes, err := k8sclientcmd.Write(*initialConfig)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(finalKubeConfigPath, initialBytes, 0600))

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

	// Enable verbose logging to check output
	verbose = true
	defer func() { verbose = false }()

	// 3. Run the command
	err = syncCmd.RunE(syncCmd, []string{})
	require.NoError(t, err)

	// 4. Verify the results
	updatedBytes, err := os.ReadFile(finalKubeConfigPath)
	require.NoError(t, err)

	updatedKubeconfig, err := k8sclientcmd.Load(updatedBytes)
	require.NoError(t, err)

	// Verify that the cluster was updated, not just added
	assert.Len(t, updatedKubeconfig.Clusters, 1, "There should be only one cluster in the config")
	updatedCluster, exists := updatedKubeconfig.Clusters["do-nyc1-doks-recreated-cluster"]
	require.True(t, exists, "Recreated cluster should exist in config")

	// Verify the server URL has been updated
	assert.Equal(t, "https://new-recreated-cluster-server", updatedCluster.Server, "Cluster server URL should be updated")

	// Verify the cluster ID has been updated
	newID, found := kubeconfig.GetClusterID(updatedCluster)
	assert.True(t, found, "Cluster ID extension should be found")
	assert.Equal(t, "new-recreated-cluster-id", newID, "Cluster ID should be updated to the new ID")
}
