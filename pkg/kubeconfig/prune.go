package kubeconfig

import (
	"fmt"
	"strings"

	"github.com/DO-Solutions/kubectl-doks/do"
	k8sclientcmd "k8s.io/client-go/tools/clientcmd"
)

// PruneConfig removes contexts, clusters, and users whose context names start with 'do-' 
// but whose corresponding cluster no longer exists in the list of liveClusters.
// It returns the pruned configuration as a byte array along with a slice of removed context names.
func PruneConfig(config []byte, liveClusters []do.Cluster) ([]byte, []string, error) {
	if len(config) == 0 {
		return nil, nil, fmt.Errorf("config cannot be empty")
	}

	// Parse the kubeconfig
	configObj, err := k8sclientcmd.Load(config)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse kubeconfig: %v", err)
	}

	// Create a map of existing cluster IDs for quick lookup
	// The context name format for DO clusters is typically 'do-<region>-<cluster-id>'
	clusterIDMap := make(map[string]bool)
	for _, cluster := range liveClusters {
		clusterIDMap[cluster.ID] = true
	}

	// Track removed contexts
	removedContexts := []string{}

	// Find and collect DO contexts that no longer have a corresponding live cluster
	contextsToRemove := []string{}
	for contextName := range configObj.Contexts {
		// Only process contexts that start with 'do-'
		if len(contextName) > 3 && contextName[:3] == "do-" {
			// Extract the cluster ID from the context
			// The cluster ID is typically the last part of the context name after the last dash
			parts := strings.Split(contextName, "-")
			if len(parts) >= 3 {
				clusterID := parts[len(parts)-1]
				
				// Check if the cluster ID exists in the live clusters
				if !clusterIDMap[clusterID] {
					contextsToRemove = append(contextsToRemove, contextName)
				}
			}
		}
	}

	// Remove stale contexts and their associated clusters and users
	for _, contextName := range contextsToRemove {
		ctx, exists := configObj.Contexts[contextName]
		if !exists {
			continue
		}

		// Delete the context
		delete(configObj.Contexts, contextName)
		removedContexts = append(removedContexts, contextName)

		// Get the cluster and user associated with this context
		clusterName := ctx.Cluster
		userName := ctx.AuthInfo

		// Check if the cluster is used by any remaining contexts
		clusterInUse := false
		for _, ctx := range configObj.Contexts {
			if ctx.Cluster == clusterName {
				clusterInUse = true
				break
			}
		}

		// If not in use, delete the cluster
		if !clusterInUse {
			delete(configObj.Clusters, clusterName)
		}

		// Check if the user is used by any remaining contexts
		userInUse := false
		for _, ctx := range configObj.Contexts {
			if ctx.AuthInfo == userName {
				userInUse = true
				break
			}
		}

		// If not in use, delete the user
		if !userInUse {
			delete(configObj.AuthInfos, userName)
		}
	}

	// If current context was removed, clear it
	if configObj.CurrentContext != "" {
		_, exists := configObj.Contexts[configObj.CurrentContext]
		if !exists {
			configObj.CurrentContext = ""
		}
	}

	// Convert the pruned config back to bytes
	prunedConfig, err := k8sclientcmd.Write(*configObj)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to serialize pruned config: %v", err)
	}

	return prunedConfig, removedContexts, nil
}
