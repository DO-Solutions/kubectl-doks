package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// syncCmd represents the sync command
var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Synchronize all DOKS clusters to ~/.kube/config",
	Long: `Fetches all reachable teams' DOKS clusters and ensures your local ~/.kube/config
contains only the contexts matching existing clusters (contexts start with do-).`,
	Run: func(cmd *cobra.Command, args []string) {
		// This will be implemented in a future milestone
		fmt.Println("sync command called - functionality will be implemented in a future milestone")
	},
}

func init() {
	kubeconfigCmd.AddCommand(syncCmd)
}
