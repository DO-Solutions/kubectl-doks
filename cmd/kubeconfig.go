package cmd

import (
	"github.com/spf13/cobra"
)

// kubeconfigCmd represents the kubeconfig command
var kubeconfigCmd = &cobra.Command{
	Use:   "kubeconfig",
	Short: "Sync DOKS kubeconfig entries",
	Long:  `Commands to sync DigitalOcean Kubernetes Service (DOKS) kubeconfig entries.`,
}

func init() {
	rootCmd.AddCommand(kubeconfigCmd)
}
