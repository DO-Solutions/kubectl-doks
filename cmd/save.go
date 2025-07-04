package cmd

import (
	"fmt"

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
	Run: func(cmd *cobra.Command, args []string) {
		// This will be implemented in a future milestone
		fmt.Println("save command called - functionality will be implemented in a future milestone")
	},
}

func init() {
	kubeconfigCmd.AddCommand(saveCmd)

	// Local flags for the save command
	saveCmd.Flags().BoolVar(&setCurrentContext, "set-current-context", true, "Set current-context to the new context")
}
