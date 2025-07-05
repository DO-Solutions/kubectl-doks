package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/DO-Solutions/kubectl-doks/do"
	"github.com/DO-Solutions/kubectl-doks/pkg/kubeconfig"
	"github.com/DO-Solutions/kubectl-doks/pkg/ui"
	"github.com/spf13/cobra"
)

var (
	setCurrentContext bool
)

// saveCmd represents the save command
var saveCmd = &cobra.Command{
	Use:   "save [<cluster-name>]",
	Short: "Save a single cluster's credentials",
	Long: `Fetches a single cluster's credentials and merges them into ~/.kube/config.
If no cluster name is provided, launches an interactive prompt to pick a cluster.`,
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

		var selectedCluster do.Cluster

		if len(args) > 0 {
			clusterName := args[0]
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
		} else {
			selectedCluster, err = ui.Cluster(allClusters)
			if err != nil {
				return fmt.Errorf("selecting cluster: %w", err)
			}
		}

		client := clusterToClient[selectedCluster.ID]
		kubeConfigBytes, err := client.GetKubeConfig(ctx, selectedCluster.ID)
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

		mergedConfigBytes, err := kubeconfig.MergeConfig(existingConfigBytes, kubeConfigBytes, setCurrentContext)
		if err != nil {
			return fmt.Errorf("merging kubeconfig for cluster %s: %w", selectedCluster.Name, err)
		}

		if err := os.WriteFile(kubeConfigPath, mergedConfigBytes, 0600); err != nil {
			return fmt.Errorf("writing updated kubeconfig: %w", err)
		}

		if verbose {
			fmt.Printf("Notice: Saved credentials for cluster %q to %s\n", selectedCluster.Name, kubeConfigPath)
			if setCurrentContext {
				contextName := fmt.Sprintf("do-%s-%s", selectedCluster.Region, selectedCluster.Name)
				fmt.Printf("Notice: Set current-context to %q\n", contextName)
			}
		}

		return nil
	},
}

func init() {
	kubeconfigCmd.AddCommand(saveCmd)
	saveCmd.Flags().BoolVar(&setCurrentContext, "set-current-context", true, "Set current-context to the new context")
}
