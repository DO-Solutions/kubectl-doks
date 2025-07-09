package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Global flags
	accessTokens    []string
	authContexts    []string
	allAuthContexts bool
	apiURL          string
	configFile      string
	verbose         bool
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
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "Path to doctl config file (default: $HOME/.config/doctl/config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
}

// validateAuthFlags ensures that at least one authentication method is specified.
func validateAuthFlags(cmd *cobra.Command, args []string) error {
	// Skip validation for help and version commands
	if cmd.Name() == "help" || cmd.Name() == "version" {
		return nil
	}
	if err := validateAuthSources(); err != nil {
		return err
	}

	// Check if at least one authentication method is provided via flags, environment variables, or a doctl config file.
	flagsProvided := len(accessTokens) > 0 || len(authContexts) > 0 || allAuthContexts
	envProvided := os.Getenv("DIGITALOCEAN_ACCESS_TOKEN") != ""

	if !flagsProvided && !envProvided {
		// If no explicit auth is given, check for the implicit doctl config file.
		configPath := configFile
		if configPath == "" {
			configPath = getDoctlConfigPath()
		}

		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			return fmt.Errorf("no authentication method provided; please use flags, set DIGITALOCEAN_ACCESS_TOKEN, or configure doctl")
		}
	}

	return nil
}
