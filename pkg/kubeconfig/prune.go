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
		return []byte{}, nil, nil
	}

	// Parse the kubeconfig
	configObj, err := k8sclientcmd.Load(config)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse kubeconfig: %v", err)
	}

	// Create a map of live cluster context names for quick lookup
	liveContexts := make(map[string]bool)
	for _, cluster := range liveClusters {
		contextName := fmt.Sprintf("do-%s-%s", cluster.Region, cluster.Name)
		liveContexts[contextName] = true
	}

	var removedContexts []string
	for contextName, context := range configObj.Contexts {
		// A context is managed by us if it starts with do- and the cluster and user match the expected format.
		isManaged := strings.HasPrefix(contextName, "do-") &&
			context.Cluster == contextName &&
			context.AuthInfo == contextName+"-admin"

		if isManaged {
			if !liveContexts[contextName] {
				removedContexts = append(removedContexts, contextName)
			}
		}
	}

	// Remove stale contexts and their associated clusters and users
	for _, contextName := range removedContexts {
		ctx, exists := configObj.Contexts[contextName]
		if !exists {
			continue
		}

		// Delete the context
		delete(configObj.Contexts, contextName)

		// Get the cluster and user associated with this context
		clusterName := ctx.Cluster
		userName := ctx.AuthInfo

		// Check if the cluster is used by any remaining contexts
		clusterInUse := false
		for _, otherCtx := range configObj.Contexts {
			if otherCtx.Cluster == clusterName {
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
		for _, otherCtx := range configObj.Contexts {
			if otherCtx.AuthInfo == userName {
				userInUse = true
				break
			}
		}

		// If not in use, delete the user
		if !userInUse {
			delete(configObj.AuthInfos, userName)
		}
	}

	// If the current context was removed, clear it
	currentContextRemoved := false
	for _, removed := range removedContexts {
		if configObj.CurrentContext == removed {
			currentContextRemoved = true
			break
		}
	}
	if currentContextRemoved {
		configObj.CurrentContext = ""
	}

	// Write the pruned config back to bytes
	prunedConfig, err := k8sclientcmd.Write(*configObj)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to write pruned kubeconfig: %v", err)
	}

	return prunedConfig, removedContexts, nil
}
