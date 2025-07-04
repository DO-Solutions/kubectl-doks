package kubeconfig

import (
	"errors"
	"fmt"

	k8sclientcmd "k8s.io/client-go/tools/clientcmd"
	k8sclientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// MergeConfig merges the raw kubeconfig bytes from newConfig into the srcConfig.
// If setCurrentContext is true, the current-context will be set to the one from newConfig,
// if newConfig doesn't have a current-context set, an error will be returned.
// It returns the merged configuration as a byte array.
func MergeConfig(srcConfig, newConfig []byte, setCurrentContext bool) ([]byte, error) {
	if len(srcConfig) == 0 {
		return nil, errors.New("source config cannot be empty")
	}

	if len(newConfig) == 0 {
		return nil, errors.New("new config cannot be empty")
	}

	// Parse the source config
	configObj, err := k8sclientcmd.Load(srcConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to parse source kubeconfig: %v", err)
	}

	// Parse the new config
	newConfigObj, err := k8sclientcmd.Load(newConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to parse new kubeconfig: %v", err)
	}

	// If we're setting current context, check that the new config has one set
	if setCurrentContext && newConfigObj.CurrentContext == "" {
		return nil, errors.New("cannot set current context: new config does not have a current context set")
	}

	// Merge the configs
	mergeKubeConfigObjects(configObj, newConfigObj)

	// If setCurrentContext is true, ensure the current context is set to the one from the new config
	if setCurrentContext {
		configObj.CurrentContext = newConfigObj.CurrentContext
	}

	// Convert the merged config back to bytes
	mergedConfig, err := k8sclientcmd.Write(*configObj)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize merged config: %v", err)
	}

	return mergedConfig, nil
}

// mergeKubeConfigObjects merges the new config object into the target config object
func mergeKubeConfigObjects(target *k8sclientcmdapi.Config, newConfig *k8sclientcmdapi.Config) {
	// Merge clusters
	for key, cluster := range newConfig.Clusters {
		target.Clusters[key] = cluster
	}

	// Merge auth info (users)
	for key, authInfo := range newConfig.AuthInfos {
		target.AuthInfos[key] = authInfo
	}

	// Merge contexts
	for key, context := range newConfig.Contexts {
		target.Contexts[key] = context
	}
}


