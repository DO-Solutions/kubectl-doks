package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/DO-Solutions/kubectl-doks/do"
	"github.com/DO-Solutions/kubectl-doks/pkg/kubeconfig"
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
	RunE: func(cmd *cobra.Command, args []string) error {
		var existingConfigBytes []byte
		var err error
		kubeConfigPath, existingConfigBytes, err = kubeconfig.GetKubeconfig(kubeConfigPath)
		if err != nil {
			return err
		}

		ctx := context.Background()

		tokens, err := getAllAccessTokens()
		if err != nil {
			return err
		}

		var allClusters []do.Cluster
		clusterIDToClient := make(map[string]*do.Client)

		for _, token := range tokens {
			client, err := do.NewClient(token, apiURL)
			if err != nil {
				return fmt.Errorf("creating DigitalOcean client: %w", err)
			}

			clusters, err := client.ListClusters(ctx)
			if err != nil {
				return fmt.Errorf("fetching clusters for a token: %w", err)
			}

			for _, cluster := range clusters {
				allClusters = append(allClusters, cluster)
				clusterIDToClient[cluster.ID] = client
			}
		}

		if len(allClusters) == 0 {
			fmt.Println("No DOKS clusters found to sync.")
			return nil
		}

		prunedConfigBytes, removedContexts, err := kubeconfig.PruneConfig(existingConfigBytes, allClusters)
		if err != nil {
			return fmt.Errorf("pruning kubeconfig: %w", err)
		}

		currentConfigBytes := prunedConfigBytes
		var addedContexts []string

		configObj, err := k8sclientcmd.Load(currentConfigBytes)
		if err != nil {
			if len(currentConfigBytes) == 0 {
				configObj = k8sclientcmdapi.NewConfig()
			} else {
				return fmt.Errorf("parsing kubeconfig: %w", err)
			}
		}

		for _, cluster := range allClusters {
			expectedContextName := fmt.Sprintf("do-%s-%s", cluster.Region, cluster.Name)
			if _, exists := configObj.Contexts[expectedContextName]; exists {
				continue
			}

			client, ok := clusterIDToClient[cluster.ID]
			if !ok {
				return fmt.Errorf("could not find a client for cluster %s", cluster.Name)
			}

			kubeConfigBytes, err := client.GetKubeConfig(ctx, cluster.ID)
			if err != nil {
				return fmt.Errorf("getting kubeconfig for cluster %s: %w", cluster.Name, err)
			}

			var mergedConfigBytes []byte
			if len(currentConfigBytes) == 0 {
				mergedConfigBytes = kubeConfigBytes
			} else {
				mergedConfigBytes, err = kubeconfig.MergeConfig(currentConfigBytes, kubeConfigBytes, false)
				if err != nil {
					return fmt.Errorf("merging kubeconfig for cluster %s: %w", cluster.Name, err)
				}
			}

			currentConfigBytes = mergedConfigBytes
			addedContexts = append(addedContexts, expectedContextName)

			configObj, err = k8sclientcmd.Load(currentConfigBytes)
			if err != nil {
				return fmt.Errorf("reloading kubeconfig after merge: %w", err)
			}
		}

		if len(removedContexts) > 0 || len(addedContexts) > 0 {
			backupPath := kubeConfigPath + ".kubectl-doks.bak"
			if verbose {
				fmt.Printf("Notice: Creating backup of kubeconfig at %s\n", backupPath)
			}
			if _, err := os.Stat(kubeConfigPath); err == nil {
				if err := kubeconfig.BackupKubeconfig(kubeConfigPath, backupPath); err != nil {
					return fmt.Errorf("backing up kubeconfig: %w", err)
				}
			}

			if verbose && len(removedContexts) > 0 {
				fmt.Printf("Notice: Removing stale contexts: %v\n", removedContexts)
			}

			if verbose && len(addedContexts) > 0 {
				fmt.Printf("Notice: Adding contexts: %v\n", addedContexts)
			}

			config, err := k8sclientcmd.Load(currentConfigBytes)
			if err != nil {
				return fmt.Errorf("loading final kubeconfig: %w", err)
			}

			originalConfig, _ := k8sclientcmd.Load(existingConfigBytes)
			originalCurrentContext := ""
			if originalConfig != nil {
				originalCurrentContext = originalConfig.CurrentContext
			}

			contextRemoved := false
			for _, r := range removedContexts {
				if r == originalCurrentContext {
					contextRemoved = true
					break
				}
			}

			if setCurrentContext && len(addedContexts) == 1 && (config.CurrentContext == "" || contextRemoved) {
				config.CurrentContext = addedContexts[0]
				if verbose {
					fmt.Printf("Notice: Set current-context to %q\n", addedContexts[0])
				}
			}

			finalConfigBytes, err := k8sclientcmd.Write(*config)
			if err != nil {
				return fmt.Errorf("serializing final kubeconfig: %w", err)
			}

			if err := os.WriteFile(kubeConfigPath, finalConfigBytes, 0600); err != nil {
				return fmt.Errorf("writing updated kubeconfig: %w", err)
			}
			if verbose {
				fmt.Printf("Notice: Successfully synced %d DOKS cluster(s) to your kubeconfig file.\n", len(allClusters))
			}
		} else {
			if verbose {
				fmt.Println("Notice: Kubeconfig is already up to date.")
			}
		}
		return nil
	},
}

func init() {
	kubeconfigCmd.AddCommand(syncCmd)
}
