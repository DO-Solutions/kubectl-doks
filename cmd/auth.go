package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/viper"
)

// validateAuthSources ensures that mutually exclusive authentication flags are not used together.
func validateAuthSources() error {
    authMethods := 0
    if len(accessTokens) > 0 {
        authMethods++
    }
    if len(authContexts) > 0 {
        authMethods++
    }
    if allAuthContexts {
        authMethods++
    }

    if authMethods > 1 {
        return fmt.Errorf("only one of --access-token, --auth-context, or --all-auth-contexts flags can be specified")
    }
    return nil
}

// getAllAccessTokens gathers access tokens following a specific precedence order:
// 1. --access-token flags
// 2. --auth-context or --all-auth-contexts flags (from doctl config)
// 3. DIGITALOCEAN_ACCESS_TOKEN environment variable
// 4. Current doctl authentication context
func getAllAccessTokens() ([]string, error) {
	// 1. --access-token
	if len(accessTokens) > 0 {
		return unique(accessTokens), nil
	}

	// We might need the doctl config for the next steps.
	doctlConfig, err := loadDoctlConfig()
	if err != nil {
		return nil, err
	}

	// 2. --auth-context or --all-auth-contexts
	if len(authContexts) > 0 || allAuthContexts {
		if doctlConfig == nil {
			// Config file does not exist, but flags were provided that require it.
			return nil, fmt.Errorf("doctl config file not found at %q", getDoctlConfigPath())
		}
		tokens, err := getTokensFromDoctlConfig(doctlConfig)
		if err != nil {
			return nil, err
		}
		if len(tokens) > 0 {
			return tokens, nil
		}
		return nil, fmt.Errorf("no tokens found for the specified auth contexts")
	}

	// 3. Environment variables
	if token := os.Getenv("DIGITALOCEAN_ACCESS_TOKEN"); token != "" {
		return []string{token}, nil
	}

	// 4. Current doctl authentication context
	if doctlConfig != nil {
		tokens, err := getCurrentDoctlContextToken(doctlConfig)
		if err != nil {
			return nil, err
		}
		if len(tokens) > 0 {
			return tokens, nil
		}
	}

	return nil, fmt.Errorf("no DigitalOcean access token found")
}

// loadDoctlConfig loads the doctl configuration file.
func loadDoctlConfig() (*viper.Viper, error) {
	v := viper.New()
	cfgFile := getDoctlConfigPath()
	v.SetConfigFile(cfgFile)

	if err := v.ReadInConfig(); err != nil {
		if os.IsNotExist(err) {
			return nil, nil // Not an error if config doesn't exist.
		}
		return nil, fmt.Errorf("failed to read doctl config file at %q: %w", cfgFile, err)
	}
	return v, nil
}

// getTokensFromDoctlConfig retrieves tokens from specified doctl auth contexts.
func getTokensFromDoctlConfig(v *viper.Viper) ([]string, error) {
	var contextsToUse []string
	if allAuthContexts {
		settings := v.AllSettings()
		if authContextsMap, ok := settings["auth-contexts"].(map[string]interface{}); ok {
			for name := range authContextsMap {
				contextsToUse = append(contextsToUse, name)
			}
		} else if v.IsSet("access-token") {
			// If no auth-contexts map exists, but there's a top-level token,
			// consider 'default' as the only available context.
			contextsToUse = append(contextsToUse, "default")
		}
	} else {
		contextsToUse = authContexts
	}

	var tokens []string
	for _, context := range contextsToUse {
		var token string
		if context == "default" {
			// The 'default' context can be a named context or refer to the top-level token.
			// If 'auth-contexts.default' is 'true' or not set, use the top-level token.
			if !v.IsSet("auth-contexts."+context) || v.GetString("auth-contexts."+context) == "true" {
				token = v.GetString("access-token")
			} else {
				// It's a named context with its own token.
				token = v.GetString("auth-contexts.default")
			}
		} else {
			token = v.GetString(fmt.Sprintf("auth-contexts.%s", context))
		}
		if token != "" {
			tokens = append(tokens, token)
		}
	}

	return unique(tokens), nil
}

// getCurrentDoctlContextToken retrieves the token from the current doctl context.
func getCurrentDoctlContextToken(v *viper.Viper) ([]string, error) {
	currentContext := v.GetString("context")
	if currentContext == "" {
		// If 'context' is not explicitly set, doctl uses 'default'.
		currentContext = "default"
	}

	var token string
	// The 'default' context is special: it might not exist in 'auth-contexts'.
	if currentContext == "default" {
		// The 'default' context is indicated by 'auth-contexts.default: true',
		// or by the absence of an 'auth-contexts' map.
		if !v.IsSet("auth-contexts") || v.GetString("auth-contexts.default") == "true" {
			token = v.GetString("access-token")
		} else {
			// This case handles if 'default' is a named context with its own token.
			token = v.GetString("auth-contexts.default")
		}
	} else {
		// For any other named context.
		token = v.GetString(fmt.Sprintf("auth-contexts.%s", currentContext))
	}

	if token == "" {
		return nil, nil // No token found for the context.
	}

	return []string{token}, nil
}

// getDoctlConfigPath determines the path to the doctl config file based on the OS.
func getDoctlConfigPath() string {
	if configFile != "" {
		return configFile
	}
	home, err := os.UserHomeDir()
	if err != nil {
		// Unable to get home directory, return empty string and let viper handle it.
		return ""
	}

	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "doctl", "config.yaml")
	default: // Defaults to Linux/other Unix-like systems path
		return filepath.Join(home, ".config", "doctl", "config.yaml")
	}
}

// unique returns a new slice with duplicate strings removed.
func unique(slice []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range slice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}
