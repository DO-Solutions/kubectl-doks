package kubeconfig

import (
	"testing"

	"github.com/DO-Solutions/kubectl-doks/do"
	"github.com/stretchr/testify/assert"
	k8sclientcmd "k8s.io/client-go/tools/clientcmd"
)

// Test data for a kubeconfig with multiple contexts, some of which are DigitalOcean contexts
const testKubeconfig = `
apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: dGVzdC1jbHVzdGVyLWNhLWRhdGE=
    server: https://cluster1.example.com
  name: do-nyc1-cluster1
- cluster:
    certificate-authority-data: dGVzdC1jbHVzdGVyLWNhLWRhdGE=
    server: https://cluster2.example.com
  name: do-ams3-cluster2
- cluster:
    certificate-authority-data: dGVzdC1jbHVzdGVyLWNhLWRhdGE=
    server: https://other-cluster.example.com
  name: other-cluster
contexts:
- context:
    cluster: do-nyc1-cluster1
    user: do-nyc1-cluster1-admin
  name: do-nyc1-cluster1
- context:
    cluster: do-ams3-cluster2
    user: do-ams3-cluster2-admin
  name: do-ams3-cluster2
- context:
    cluster: other-cluster
    user: other-cluster-admin
  name: other-cluster
current-context: do-nyc1-cluster1
kind: Config
preferences: {}
users:
- name: do-nyc1-cluster1-admin
  user:
    token: token1
- name: do-ams3-cluster2-admin
  user:
    token: token2
- name: other-cluster-admin
  user:
    token: token3
`

// TestPruneConfig_AllContextsValid tests that the function doesn't remove any contexts
// when all DO contexts have corresponding clusters in the liveClusters list
func TestPruneConfig_AllContextsValid(t *testing.T) {
	liveClusters := []do.Cluster{
		{ID: "cluster1", Name: "cluster1", Region: "nyc1"},
		{ID: "cluster2", Name: "cluster2", Region: "ams3"},
	}

	// Prune the config
	prunedConfig, removedContexts, err := PruneConfig([]byte(testKubeconfig), liveClusters)

	// Check results
	assert.NoError(t, err, "PruneConfig should not return an error")
	assert.Empty(t, removedContexts, "No contexts should be removed")

	// Parse the pruned config to verify no changes were made
	configObj, err := k8sclientcmd.Load(prunedConfig)
	assert.NoError(t, err, "Failed to parse pruned config")
	
	// Verify all contexts still exist
	assert.Len(t, configObj.Contexts, 3, "All contexts should still exist")
	assert.Contains(t, configObj.Contexts, "do-nyc1-cluster1", "do-nyc1-cluster1 context should exist")
	assert.Contains(t, configObj.Contexts, "do-ams3-cluster2", "do-ams3-cluster2 context should exist")
	assert.Contains(t, configObj.Contexts, "other-cluster", "other-cluster context should exist")

	// Verify current context is preserved
	assert.Equal(t, "do-nyc1-cluster1", configObj.CurrentContext, "Current context should be preserved")
}

// TestPruneConfig_RemoveStaleContext tests that the function correctly removes contexts
// that don't have corresponding clusters in the liveClusters list
func TestPruneConfig_RemoveStaleContext(t *testing.T) {
	liveClusters := []do.Cluster{
		// Only cluster1 is active, cluster2 has been deleted
		{ID: "cluster1", Name: "cluster1", Region: "nyc1"},
	}

	// Prune the config
	prunedConfig, removedContexts, err := PruneConfig([]byte(testKubeconfig), liveClusters)

	// Check results
	assert.NoError(t, err, "PruneConfig should not return an error")
	assert.Equal(t, []string{"do-ams3-cluster2"}, removedContexts, "do-ams3-cluster2 context should be removed")

	// Parse the pruned config to verify changes
	configObj, err := k8sclientcmd.Load(prunedConfig)
	assert.NoError(t, err, "Failed to parse pruned config")

	// Verify contexts
	assert.Len(t, configObj.Contexts, 2, "Should have 2 contexts remaining")
	assert.Contains(t, configObj.Contexts, "do-nyc1-cluster1", "do-nyc1-cluster1 context should exist")
	assert.NotContains(t, configObj.Contexts, "do-ams3-cluster2", "do-ams3-cluster2 context should be removed")
	assert.Contains(t, configObj.Contexts, "other-cluster", "other-cluster context should exist")

	// Verify clusters
	assert.Len(t, configObj.Clusters, 2, "Should have 2 clusters remaining")
	assert.Contains(t, configObj.Clusters, "do-nyc1-cluster1", "do-nyc1-cluster1 cluster should exist")
	assert.NotContains(t, configObj.Clusters, "do-ams3-cluster2", "do-ams3-cluster2 cluster should be removed")

	// Verify users
	assert.Len(t, configObj.AuthInfos, 2, "Should have 2 users remaining")
	assert.Contains(t, configObj.AuthInfos, "do-nyc1-cluster1-admin", "do-nyc1-cluster1-admin user should exist")
	assert.NotContains(t, configObj.AuthInfos, "do-ams3-cluster2-admin", "do-ams3-cluster2-admin user should be removed")

	// Verify current context is preserved
	assert.Equal(t, "do-nyc1-cluster1", configObj.CurrentContext, "Current context should be preserved")
}

// TestPruneConfig_CurrentContextRemoved tests that the function clears the current-context
// if the current context is removed
func TestPruneConfig_CurrentContextRemoved(t *testing.T) {
	// Modify the test config to have the current-context set to the context we'll remove
	modifiedConfig := testKubeconfig
	modifiedConfig = modifyCurrentContext(modifiedConfig, "do-ams3-cluster2")

	liveClusters := []do.Cluster{
		// Only cluster1 is active, cluster2 has been deleted
		{ID: "cluster1", Name: "cluster1", Region: "nyc1"},
	}

	// Prune the config
	prunedConfig, removedContexts, err := PruneConfig([]byte(modifiedConfig), liveClusters)

	// Check results
	assert.NoError(t, err, "PruneConfig should not return an error")
	assert.Equal(t, []string{"do-ams3-cluster2"}, removedContexts, "do-ams3-cluster2 context should be removed")

	// Parse the pruned config to verify changes
	configObj, err := k8sclientcmd.Load(prunedConfig)
	assert.NoError(t, err, "Failed to parse pruned config")

	// Verify current context is cleared
	assert.Empty(t, configObj.CurrentContext, "Current context should be cleared")
}

// TestPruneConfig_EmptyConfig tests that the function handles an empty config gracefully.
func TestPruneConfig_EmptyConfig(t *testing.T) {
	liveClusters := []do.Cluster{
		{ID: "cluster1", Name: "cluster1", Region: "nyc1"},
	}

	// Prune an empty config
	prunedConfig, removedContexts, err := PruneConfig([]byte{}, liveClusters)

	// Check result
	assert.NoError(t, err, "PruneConfig should not return an error for empty config")
	assert.Empty(t, prunedConfig, "Pruned config should be empty")
	assert.Empty(t, removedContexts, "No contexts should be removed")
}

// Helper function to modify the current-context in a kubeconfig string
func modifyCurrentContext(kubeconfig string, newCurrentContext string) string {
	// Parse the config
	configObj, err := k8sclientcmd.Load([]byte(kubeconfig))
	if err != nil {
		return kubeconfig // Return original if parsing fails
	}

	// Set the new current-context
	configObj.CurrentContext = newCurrentContext

	// Convert back to bytes
	modifiedConfig, err := k8sclientcmd.Write(*configObj)
	if err != nil {
		return kubeconfig // Return original if serialization fails
	}

	return string(modifiedConfig)
}
