package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Global flags
	accessTokens      []string
	authContexts      []string
	allAuthContexts   bool
	apiURL            string
	configFile        string
	expirySeconds     int
	verbose           bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "doks",
	Short: "A kubectl plugin to sync DigitalOcean Kubernetes (DOKS) kubeconfig entries",
	Long: `kubectl-doks is a Kubernetes CLI plugin to sync DigitalOcean Kubernetes (DOKS) kubeconfig entries.

Easily synchronize all active DOKS clusters to your local ~/.kube/config and remove stale contexts,
or save a single cluster's credentials interactively or by name.`,
	PersistentPreRunE: validateAuthFlags,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Global flags for authentication and configuration
	rootCmd.PersistentFlags().StringSliceVarP(&accessTokens, "access-token", "t", nil,
		"DigitalOcean API V2 token (can specify multiple times)")
	rootCmd.PersistentFlags().StringSliceVarP(&authContexts, "auth-context", "", nil,
		"Use this doctl authentication context (can specify multiple times)")
	rootCmd.PersistentFlags().BoolVarP(&allAuthContexts, "all-auth-contexts", "", false, "Include all doctl authentication contexts")
	rootCmd.PersistentFlags().StringVarP(&apiURL, "api-url", "u", "", "Override the default DigitalOcean API endpoint")
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "",
		"Path to doctl config file (default: $HOME/.config/doctl/config.yaml)")
	rootCmd.PersistentFlags().IntVarP(&expirySeconds, "expiry-seconds", "", 0,
		"Credential TTL in seconds; auto-renewal is enabled by default")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
}

// validateAuthFlags ensures that exactly one authentication method is specified
func validateAuthFlags(cmd *cobra.Command, args []string) error {
	// Skip validation for help command
	if cmd.Name() == "help" {
		return nil
	}

	// Count how many authentication methods are specified
	methodCount := 0

	if len(accessTokens) > 0 {
		methodCount++
	}

	if len(authContexts) > 0 {
		methodCount++
	}

	if allAuthContexts {
		methodCount++
	}

	// Check if DIGITALOCEAN_ACCESS_TOKEN environment variable is set
	if os.Getenv("DIGITALOCEAN_ACCESS_TOKEN") != "" && methodCount == 0 {
		// Using environment variable as default
		return nil
	}

	switch methodCount {
	case 0:
		return errors.New("no authentication method specified; " +
			"use --access-token, --auth-context, --all-auth-contexts, " +
			"or set DIGITALOCEAN_ACCESS_TOKEN environment variable")
	case 1:
		return nil
	default:
		return fmt.Errorf("multiple authentication methods specified; " +
			"use exactly one of: --access-token, --auth-context, or --all-auth-contexts")
	}
}
