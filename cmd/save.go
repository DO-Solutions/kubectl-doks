package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/DO-Solutions/kubectl-doks/do"
	"github.com/DO-Solutions/kubectl-doks/pkg/kubeconfig"
	"github.com/spf13/cobra"
	k8sclientcmd "k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

// saveCmd represents the save command
var saveCmd = &cobra.Command{
	Use:   "save [<cluster-name>]",
	Short: "Save cluster credentials",
	Long: `Fetches cluster credentials and merges them into ~/.kube/config.
If a cluster name is provided, it saves that specific cluster's credentials.
If no cluster name is provided, it saves the credentials for all available clusters.`,
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
		clusterToClient := make(map[string]*do.Client)

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
				clusterToClient[cluster.ID] = client
			}
		}

		if len(allClusters) == 0 {
			fmt.Println("No DOKS clusters found.")
			return nil
		}

		if len(args) > 0 {
			clusterName := args[0]
			var selectedCluster do.Cluster
			found := false
			for _, cluster := range allClusters {
				if cluster.Name == clusterName {
					selectedCluster = cluster
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("cluster %q not found", clusterName)
			}

			client := clusterToClient[selectedCluster.ID]
			kubeConfigBytes, err := client.GetKubeConfig(ctx, selectedCluster.ID, expirySeconds)
			if err != nil {
				return fmt.Errorf("getting kubeconfig for cluster %s: %w", selectedCluster.Name, err)
			}

			backupPath := kubeConfigPath + ".kubectl-doks.bak"
			if verbose {
				fmt.Printf("Notice: Creating backup of kubeconfig at %s\n", backupPath)
			}
			if _, err := os.Stat(kubeConfigPath); err == nil {
				if err := kubeconfig.BackupKubeconfig(kubeConfigPath, backupPath); err != nil {
					return fmt.Errorf("backing up kubeconfig: %w", err)
				}
			}

			var mergedConfigBytes []byte
			if len(existingConfigBytes) == 0 {
				mergedConfigBytes = kubeConfigBytes
			} else {
				mergedConfigBytes, err = kubeconfig.MergeConfig(existingConfigBytes, kubeConfigBytes, false) // Always merge with false first
				if err != nil {
					return fmt.Errorf("merging kubeconfig for cluster %s: %w", selectedCluster.Name, err)
				}
			}

			config, err := k8sclientcmd.Load(mergedConfigBytes)
			if err != nil {
				return fmt.Errorf("reloading kubeconfig after merge: %w", err)
			}

			contextName := fmt.Sprintf("do-%s-%s", selectedCluster.Region, selectedCluster.Name)
			if cluster, ok := config.Clusters[contextName]; ok {
				kubeconfig.SetClusterID(cluster, selectedCluster.ID)
			}

			if setCurrentContext {
				config.CurrentContext = contextName
			}

			mergedConfigBytes, err = k8sclientcmd.Write(*config)
			if err != nil {
				return fmt.Errorf("serializing modified kubeconfig: %w", err)
			}

			if err := os.WriteFile(kubeConfigPath, mergedConfigBytes, 0600); err != nil {
				return fmt.Errorf("writing updated kubeconfig: %w", err)
			}

			if verbose {
				fmt.Printf("Notice: Saved credentials for cluster %q to %s\n", selectedCluster.Name, kubeConfigPath)
				if setCurrentContext {
					fmt.Printf("Notice: Set current-context to %q\n", contextName)
				}
			}
		} else {
			// If no cluster name is provided, save all clusters.
			currentConfigBytes := existingConfigBytes
			var addedContexts []string

			configObj, err := k8sclientcmd.Load(currentConfigBytes)
			if err != nil {
				if len(currentConfigBytes) == 0 {
					configObj = api.NewConfig()
				} else {
					return fmt.Errorf("parsing kubeconfig: %w", err)
				}
			}

			for _, cluster := range allClusters {
				expectedContextName := fmt.Sprintf("do-%s-%s", cluster.Region, cluster.Name)
				if _, exists := configObj.Contexts[expectedContextName]; exists && !force {
					continue
				}

				client := clusterToClient[cluster.ID]
				kubeConfigBytes, err := client.GetKubeConfig(ctx, cluster.ID, expirySeconds)
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

				// Reload config object to add extension and check for next cluster
				configObj, err = k8sclientcmd.Load(mergedConfigBytes)
				if err != nil {
					return fmt.Errorf("reloading kubeconfig after merge: %w", err)
				}

				if c, ok := configObj.Clusters[expectedContextName]; ok {
					kubeconfig.SetClusterID(c, cluster.ID)
				}

				currentConfigBytes, err = k8sclientcmd.Write(*configObj)
				if err != nil {
					return fmt.Errorf("serializing intermediate kubeconfig: %w", err)
				}

				addedContexts = append(addedContexts, expectedContextName)
			}

			if len(addedContexts) > 0 {
				backupPath := kubeConfigPath + ".kubectl-doks.bak"
				if verbose {
					fmt.Printf("Notice: Creating backup of kubeconfig at %s\n", backupPath)
				}
				if _, err := os.Stat(kubeConfigPath); err == nil {
					if err := kubeconfig.BackupKubeconfig(kubeConfigPath, backupPath); err != nil {
						return fmt.Errorf("backing up kubeconfig: %w", err)
					}
				}

				if verbose {
					if expirySeconds == 0 {
						fmt.Printf("Notice: Adding contexts: %v without expiration.\n", addedContexts)
					} else {
						fmt.Printf("Notice: Adding contexts: %v with expiration set to %d seconds.\n", addedContexts, expirySeconds)
					}
				}

				config, err := k8sclientcmd.Load(currentConfigBytes)
				if err != nil {
					return fmt.Errorf("loading final kubeconfig: %w", err)
				}

				if setCurrentContext && len(addedContexts) == 1 && config.CurrentContext == "" {
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
					fmt.Printf("Notice: Successfully saved %d DOKS cluster(s) to your kubeconfig file.\n", len(addedContexts))
				}
			} else {
				if verbose {
					fmt.Println("Notice: Kubeconfig is already up to date.")
				}
			}
		}
		return nil
	},
}

func init() {
	kubeconfigCmd.AddCommand(saveCmd)
}