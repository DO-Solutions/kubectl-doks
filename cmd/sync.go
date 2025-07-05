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
	k8sclientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// syncCmd represents the sync command
var kubeConfigPath string

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Synchronize all DOKS clusters to the kubeconfig file",
	Long: `Fetches all reachable DOKS clusters and ensures that the local kubeconfig file
is synchronized with the clusters' credentials.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Determine kubeconfig path.
		if kubeConfigPath == "" {
			homedir, err := os.UserHomeDir()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error finding home directory: %v\n", err)
				os.Exit(1)
			}
			kubeConfigPath = filepath.Join(homedir, ".kube", "config")
		}

		ctx := context.Background()

		// Get all DigitalOcean access tokens.
		tokens, err := getAllAccessTokens()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		// For each token, get all clusters and store a client that can access them.
		var allClusters []do.Cluster
		clusterIDToClient := make(map[string]*do.Client)

		for _, token := range tokens {
			client, err := do.NewClient(token, apiURL)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating DigitalOcean client: %v\n", err)
				os.Exit(1)
			}

			clusters, err := client.ListClusters(ctx)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error fetching clusters for a token: %v\n", err)
				os.Exit(1)
			}

			for _, cluster := range clusters {
				allClusters = append(allClusters, cluster)
				clusterIDToClient[cluster.ID] = client
			}
		}

		if len(allClusters) == 0 {
			fmt.Println("No DOKS clusters found to sync.")
			return
		}

		// Read existing kubeconfig or start with an empty one.
		existingConfigBytes, err := os.ReadFile(kubeConfigPath)
		if err != nil {
			if !os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "Error reading kubeconfig at %s: %v\n", kubeConfigPath, err)
				os.Exit(1)
			}
			existingConfigBytes = []byte{}
		}

		// Prune stale entries from the config.
		prunedConfigBytes, removedContexts, err := kubeconfig.PruneConfig(existingConfigBytes, allClusters)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error pruning kubeconfig: %v\n", err)
			os.Exit(1)
		}
		if verbose && len(removedContexts) > 0 {
			fmt.Printf("Notice: Removing stale contexts: %v\n", removedContexts)
		}

		currentConfigBytes := prunedConfigBytes
		var addedContexts []string

		configObj, err := k8sclientcmd.Load(currentConfigBytes)
		if err != nil {
			if len(currentConfigBytes) == 0 {
				configObj = k8sclientcmdapi.NewConfig()
			} else {
				fmt.Fprintf(os.Stderr, "Error parsing kubeconfig: %v\n", err)
				os.Exit(1)
			}
		}

		// For each live cluster, add it to the config if it's not already there.
		for _, cluster := range allClusters {
			expectedContextName := fmt.Sprintf("do-%s-%s", cluster.Region, cluster.Name)
			if _, exists := configObj.Contexts[expectedContextName]; exists {
				continue // Skip if context already exists.
			}

			client, ok := clusterIDToClient[cluster.ID]
			if !ok {
				fmt.Fprintf(os.Stderr, "Error: could not find a client for cluster %s\n", cluster.Name)
				os.Exit(1)
			}

			kubeConfigBytes, err := client.GetKubeConfig(ctx, cluster.ID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting kubeconfig for cluster %s: %v\n", cluster.Name, err)
				os.Exit(1)
			}

			var mergedConfigBytes []byte
			if len(currentConfigBytes) == 0 {
				mergedConfigBytes = kubeConfigBytes
			} else {
				mergedConfigBytes, err = kubeconfig.MergeConfig(currentConfigBytes, kubeConfigBytes, false)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error merging kubeconfig for cluster %s: %v\n", cluster.Name, err)
					os.Exit(1)
				}
			}
			
			currentConfigBytes = mergedConfigBytes
			addedContexts = append(addedContexts, expectedContextName)

			configObj, err = k8sclientcmd.Load(currentConfigBytes)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reloading kubeconfig after merge: %v\n", err)
				os.Exit(1)
			}
		}

		if verbose && len(addedContexts) > 0 {
			fmt.Printf("Notice: Adding contexts: %v\n", addedContexts)
		}

		if len(removedContexts) > 0 || len(addedContexts) > 0 {
			backupPath := kubeConfigPath + ".kubectl-doks.bak"
			if verbose {
				fmt.Printf("Notice: Creating backup of kubeconfig at %s\n", backupPath)
			}
			// Only backup if the file exists
			if _, err := os.Stat(kubeConfigPath); err == nil {
				if err := util.BackupKubeconfig(kubeConfigPath, backupPath); err != nil {
					fmt.Fprintf(os.Stderr, "Error backing up kubeconfig: %v\n", err)
					os.Exit(1)
				}
			}

			if err := os.WriteFile(kubeConfigPath, currentConfigBytes, 0600); err != nil {
				fmt.Fprintf(os.Stderr, "Error writing updated kubeconfig: %v\n", err)
				os.Exit(1)
			}
			if verbose {
				fmt.Printf("Successfully synced %d DOKS cluster(s) to your kubeconfig file.\n", len(allClusters))
			}
		} else {
			if verbose {
				fmt.Println("Kubeconfig is already up to date.")
			}
		}
	},
}

func init() {
	kubeconfigCmd.AddCommand(syncCmd)
}
