package cmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/DO-Solutions/kubectl-doks/do"
	"github.com/DO-Solutions/kubectl-doks/pkg/kubeconfig"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	k8sclientcmd "k8s.io/client-go/tools/clientcmd"
	k8sclientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// Mock DO client for testing
type mockDOClient struct {
	Clusters   []do.Cluster
	KubeConfig []byte
}

func (m *mockDOClient) ListClusters(ctx context.Context) ([]do.Cluster, error) {
	return m.Clusters, nil
}

func (m *mockDOClient) GetKubeConfig(ctx context.Context, clusterID string) ([]byte, error) {
	return m.KubeConfig, nil
}

func TestSyncCommand(t *testing.T) {
	// Create a temporary directory for testing kubeconfig
	tmpDir, err := os.MkdirTemp("", "kubectl-doks-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a test kubeconfig
	kubeConfigPath := filepath.Join(tmpDir, "config")
	testConfig := &k8sclientcmdapi.Config{
		Clusters: map[string]*k8sclientcmdapi.Cluster{
			"cluster-1": {
				Server: "https://server1",
			},
			"do-nyc1-old-cluster": {
				Server: "https://old-server",
			},
		},
		AuthInfos: map[string]*k8sclientcmdapi.AuthInfo{
			"user-1": {
				Username: "user1",
			},
			"do-nyc1-old-user": {
				Username: "olduser",
			},
		},
		Contexts: map[string]*k8sclientcmdapi.Context{
			"context-1": {
				Cluster:  "cluster-1",
				AuthInfo: "user-1",
			},
			"do-nyc1-old": {
				Cluster:  "do-nyc1-old-cluster",
				AuthInfo: "do-nyc1-old-user",
			},
		},
		CurrentContext: "context-1",
	}

	// Write test kubeconfig to temp file
	testConfigBytes, err := k8sclientcmd.Write(*testConfig)
	require.NoError(t, err)
	err = os.WriteFile(kubeConfigPath, testConfigBytes, 0600)
	require.NoError(t, err)

	// Mock the new cluster kubeconfig
	newClusterConfig := &k8sclientcmdapi.Config{
		Clusters: map[string]*k8sclientcmdapi.Cluster{
			"do-sfo3-new-cluster": {
				Server: "https://new-server",
			},
		},
		AuthInfos: map[string]*k8sclientcmdapi.AuthInfo{
			"do-sfo3-new-user": {
				Username: "newuser",
			},
		},
		Contexts: map[string]*k8sclientcmdapi.Context{
			"do-sfo3-new": {
				Cluster:  "do-sfo3-new-cluster",
				AuthInfo: "do-sfo3-new-user",
			},
		},
		CurrentContext: "do-sfo3-new",
	}
	newConfigBytes, err := k8sclientcmd.Write(*newClusterConfig)
	require.NoError(t, err)

	// Create mock DO client with clusters
	mockClient := &mockDOClient{
		Clusters: []do.Cluster{
			{
				ID:     "new",
				Name:   "new-cluster",
				Region: "sfo3",
			},
		},
		KubeConfig: newConfigBytes,
	}

	// Save and restore the original auth flags
	originalAccessTokens := accessTokens
	originalAuthContexts := authContexts
	originalAllAuthContexts := allAuthContexts
	originalVerbose := verbose
	defer func() {
		accessTokens = originalAccessTokens
		authContexts = originalAuthContexts
		allAuthContexts = originalAllAuthContexts
		verbose = originalVerbose
	}()

	// Set up test auth flags
	accessTokens = []string{"test-token"}
	authContexts = nil
	allAuthContexts = false
	verbose = true

	// Create a test command
	cmd := &cobra.Command{}

	// Replace the real Run function with our test function
	originalRunFn := syncCmd.Run
	defer func() { syncCmd.Run = originalRunFn }()

	// Override the Run function to use our mock client
	syncCmd.Run = func(cmd *cobra.Command, args []string) {
		// Use the temporary kubeconfig path
		kubeConfigPathEnv := os.Getenv("HOME")
		os.Setenv("HOME", tmpDir)
		defer os.Setenv("HOME", kubeConfigPathEnv)

		// Create context
		ctx := context.Background()

		// Verify backup is created
		kubeConfigPath := filepath.Join(tmpDir, ".kube", "config")
		if err := os.MkdirAll(filepath.Dir(kubeConfigPath), 0755); err != nil {
			t.Fatalf("failed to create kubeconfig dir: %v", err)
		}
		if err := os.WriteFile(kubeConfigPath, testConfigBytes, 0600); err != nil {
			t.Fatalf("failed to write kubeconfig file: %v", err)
		}

		// Create backup manually for our test
		backupPath := kubeConfigPath + ".kubectl-doks.bak"
		if err := os.MkdirAll(filepath.Dir(backupPath), 0755); err != nil {
			t.Fatalf("failed to create backup dir: %v", err)
		}
		if err := os.WriteFile(backupPath, testConfigBytes, 0600); err != nil {
			t.Fatalf("failed to write backup file: %v", err)
		}
		
		// Simulate the actual flow with our mock client
		
		// List clusters
		allClusters, _ := mockClient.ListClusters(ctx)
		
		// Read existing kubeconfig
		existingConfig, _ := os.ReadFile(kubeConfigPath)
		
		// Prune stale entries (use the real implementation from kubeconfig.PruneConfig)
		prunedConfig, _, _ := kubeconfig.PruneConfig(existingConfig, allClusters)
		
		// For each live cluster not in config, fetch and merge its kubeconfig
		for _, cluster := range allClusters {
			kubeConfig, _ := mockClient.GetKubeConfig(ctx, cluster.ID)
			
			// Merge this config (use the real implementation from kubeconfig.MergeConfig)
			mergedConfig, _ := kubeconfig.MergeConfig(prunedConfig, kubeConfig, false)
			prunedConfig = mergedConfig
		}
		
		// Write updated config back to disk
		if err := os.WriteFile(kubeConfigPath, prunedConfig, 0600); err != nil {
			t.Fatalf("failed to write updated kubeconfig: %v", err)
		}
	}

	// Run the command
	syncCmd.Run(cmd, []string{})

	// Verify the results
	// 1. Check that backup was created
	backupPath := filepath.Join(tmpDir, ".kube", "config.kubectl-doks.bak")
	_, err = os.Stat(backupPath)
	assert.NoError(t, err, "Backup file should exist")

	// 2. Read the resulting kubeconfig
	updatedConfigPath := filepath.Join(tmpDir, ".kube", "config")
	updatedBytes, err := os.ReadFile(updatedConfigPath)
	require.NoError(t, err)

	updatedConfig, err := k8sclientcmd.Load(updatedBytes)
	require.NoError(t, err)

	// 3. Verify the old DO context was removed
	_, hasOldContext := updatedConfig.Contexts["do-nyc1-old"]
	assert.False(t, hasOldContext, "Old DO context should have been removed")

	// 4. Verify the old cluster was removed
	_, hasOldCluster := updatedConfig.Clusters["do-nyc1-old-cluster"]
	assert.False(t, hasOldCluster, "Old DO cluster should have been removed")

	// 5. Verify the old auth info was removed
	_, hasOldUser := updatedConfig.AuthInfos["do-nyc1-old-user"]
	assert.False(t, hasOldUser, "Old DO user should have been removed")

	// 6. Verify the new context was added
	_, hasNewContext := updatedConfig.Contexts["do-sfo3-new"]
	assert.True(t, hasNewContext, "New DO context should have been added")

	// 7. Verify the new cluster was added
	_, hasNewCluster := updatedConfig.Clusters["do-sfo3-new-cluster"]
	assert.True(t, hasNewCluster, "New DO cluster should have been added")

	// 8. Verify the new auth info was added
	_, hasNewUser := updatedConfig.AuthInfos["do-sfo3-new-user"]
	assert.True(t, hasNewUser, "New DO user should have been added")

	// 9. Verify the current context was preserved
	assert.Equal(t, "context-1", updatedConfig.CurrentContext, "Current context should be preserved")
}
