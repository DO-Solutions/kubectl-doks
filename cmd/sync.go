package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/DO-Solutions/kubectl-doks/do"
	"github.com/DO-Solutions/kubectl-doks/pkg/kubeconfig"
	"github.com/DO-Solutions/kubectl-doks/util"
	"github.com/spf13/cobra"
	k8sclientcmd "k8s.io/client-go/tools/clientcmd"
)

// syncCmd represents the sync command
var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Synchronize all DOKS clusters to ~/.kube/config (or specified config file)",
	Long: `Fetches all reachable teams' DOKS clusters and ensures your local ~/.kube/config (or specified config file)
contains only the contexts matching existing clusters (contexts start with do-).`,
	Run: func(cmd *cobra.Command, args []string) {
		// Kubeconfig paths
		var kubeConfigPath string
		if configFile != "" {
			// Use the specified config path from the flag
			kubeConfigPath = configFile
		} else {
			// Use default location
			homedir, err := os.UserHomeDir()
			if err != nil {
				fmt.Printf("Error finding home directory: %v\n", err)
				os.Exit(1)
			}
			kubeConfigPath = filepath.Join(homedir, ".kube", "config")
		}
		
		// We don't need to validate auth flags here as it's done in PersistentPreRunE
		// Create context
		ctx := context.Background()
		
		
		// Read existing kubeconfig
		existingConfig, err := os.ReadFile(kubeConfigPath)
		if err != nil {
			fmt.Printf("Error reading kubeconfig: %v\n", err)
			os.Exit(1)
		}
		
		// Track all clusters across all auth contexts and map them to their access token.
		allClusters := []do.Cluster{}
		clusterIDToAccessToken := make(map[string]string)

		// For each auth context, call ListClusters
		for _, token := range accessTokens {
			if verbose {
				fmt.Println("Notice: Getting cluster credentials using provided access token")
			}

			// Create a new DO client
			client, err := do.NewClient(token, apiURL)
			if err != nil {
				fmt.Printf("Error creating DO client: %v\n", err)
				os.Exit(1)
			}

			// List clusters
			clusters, err := client.ListClusters(ctx)
			if err != nil {
				fmt.Printf("Error listing clusters: %v\n", err)
				os.Exit(1)
			}

			// Add to our collection of all clusters and record which token discovered them
			for _, c := range clusters {
				allClusters = append(allClusters, c)
				clusterIDToAccessToken[c.ID] = token
			}
		}
		
		// Prune stale entries
		prunedConfig, removedContexts, err := kubeconfig.PruneConfig(existingConfig, allClusters)
		if err != nil {
			fmt.Printf("Error pruning kubeconfig: %v\n", err)
			os.Exit(1)
		}
		
		if verbose && len(removedContexts) > 0 {
			fmt.Printf("Notice: Removing contexts: %v\n", removedContexts)
		}
		
		// Keep track of the current config as we make changes
		currentConfig := prunedConfig
		
		// For each live cluster not in config, fetch and merge its kubeconfig
		// First, parse the current config to check which clusters we already have
		configObj, err := k8sclientcmd.Load(currentConfig)
		if err != nil {
			fmt.Printf("Error parsing kubeconfig: %v\n", err)
			os.Exit(1)
		}
		
		// Track added contexts for verbose output
		addedContexts := []string{}
		
		// For each cluster, check if we need to add it
		for _, cluster := range allClusters {
			// Check if this cluster is already in the config
			// The context name format for DO clusters is typically 'do-<region>-<cluster-id>'
			expectedContextName := fmt.Sprintf("do-%s-%s", cluster.Region, cluster.ID)
			
			// Skip if already in config
			if _, exists := configObj.Contexts[expectedContextName]; exists {
				continue
			}
			
			// Get the token that works for this cluster.
			token, exists := clusterIDToAccessToken[cluster.ID]
			if !exists {
				// This should not happen if the cluster is in allClusters.
				fmt.Printf("Error: No valid access token found for cluster %s, aborting\n", cluster.ID)
				os.Exit(1)
			}
			
			// Create a client with the working token
			client, err := do.NewClient(token, apiURL)
			if err != nil {
				fmt.Printf("Error creating DO client: %v\n", err)
				os.Exit(1)
			}
			
			// Fetch the kubeconfig
			kubeConfig, err := client.GetKubeConfig(ctx, cluster.ID)
			if err != nil {
				fmt.Printf("Error getting kubeconfig for cluster %s: %v\n", cluster.ID, err)
				os.Exit(1)
			}
			
			// Merge this config
			mergedConfig, err := kubeconfig.MergeConfig(currentConfig, kubeConfig, false)
			if err != nil {
				fmt.Printf("Error merging kubeconfig for cluster %s: %v\n", cluster.ID, err)
				os.Exit(1)
			}
			
			// Update our current config
			currentConfig = mergedConfig
			addedContexts = append(addedContexts, expectedContextName)
			
			// Refresh the configObj for the next iteration
			configObj, err = k8sclientcmd.Load(currentConfig)
			if err != nil {
				fmt.Printf("Error parsing updated kubeconfig: %v\n", err)
				os.Exit(1)
			}
		}
		
		if verbose && len(addedContexts) > 0 {
			fmt.Printf("Notice: Adding contexts: %v\n", addedContexts)
		}

		// Backup existing config
		backupPath := kubeConfigPath + ".kubectl-doks.bak"
		if verbose {
			fmt.Printf("Notice: Creating backup of kubeconfig at %s\n", backupPath)
		}
		
		if err := util.BackupKubeconfig(kubeConfigPath, backupPath); err != nil {
			fmt.Printf("Error backing up kubeconfig: %v\n", err)
			os.Exit(1)
		}
		
		// Write updated config back to disk
		if err := os.WriteFile(kubeConfigPath, currentConfig, 0600); err != nil {
			fmt.Printf("Error writing updated kubeconfig: %v\n", err)
			os.Exit(1)
		}
		
		if verbose {
			fmt.Printf("Notice: Syncing cluster credentials to kubeconfig file found in %q\n", kubeConfigPath)
		}
	},
}

func init() {
	kubeconfigCmd.AddCommand(syncCmd)
}
