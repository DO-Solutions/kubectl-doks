package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	// version is the version of the CLI. This is set at build time.
	version = "v0.0.0"
	// commit is the git commit hash of the CLI. This is set at build time.
	commit = "dev"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of kubectl-doks",
	Long:  `All software has versions. This is kubectl-doks's`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("%s-%s\n", version, commit)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
