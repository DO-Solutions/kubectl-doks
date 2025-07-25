package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestValidateAuthFlags(t *testing.T) {
	tests := []struct {
		name            string
		accessTokens    []string
		authContexts    []string
		allAuthContexts bool
		envToken        string
		expectError     bool
	}{
		{
			name:            "no auth method",
			accessTokens:    nil,
			authContexts:    nil,
			allAuthContexts: false,
			envToken:        "",
			expectError:     true,
		},
		{
			name:            "access token only",
			accessTokens:    []string{"token123"},
			authContexts:    nil,
			allAuthContexts: false,
			envToken:        "",
			expectError:     false,
		},
		{
			name:            "auth context only",
			accessTokens:    nil,
			authContexts:    []string{"context1"},
			allAuthContexts: false,
			envToken:        "",
			expectError:     false,
		},
		{
			name:            "all auth contexts only",
			accessTokens:    nil,
			authContexts:    nil,
			allAuthContexts: true,
			envToken:        "",
			expectError:     false,
		},
		{
			name:            "env token only",
			accessTokens:    nil,
			authContexts:    nil,
			allAuthContexts: false,
			envToken:        "env-token",
			expectError:     false,
		},
		{
			name:            "multiple methods - token and context",
			accessTokens:    []string{"token123"},
			authContexts:    []string{"context1"},
			allAuthContexts: false,
			envToken:        "",
			expectError:     true,
		},
		{
			name:            "multiple methods - token and all contexts",
			accessTokens:    []string{"token123"},
			authContexts:    nil,
			allAuthContexts: true,
			envToken:        "",
			expectError:     true,
		},
		{
			name:            "multiple methods - context and all contexts",
			accessTokens:    nil,
			authContexts:    []string{"context1"},
			allAuthContexts: true,
			envToken:        "",
			expectError:     true,
		},
		{
			name:            "multiple tokens",
			accessTokens:    []string{"token1", "token2"},
			authContexts:    nil,
			allAuthContexts: false,
			envToken:        "",
			expectError:     false,
		},
		{
			name:            "multiple contexts",
			accessTokens:    nil,
			authContexts:    []string{"context1", "context2"},
			allAuthContexts: false,
			envToken:        "",
			expectError:     false,
		},
		{
			name:            "env token and accessToken	",
			accessTokens:    []string{"token123"},
			authContexts:    nil,
			allAuthContexts: false,
			envToken:        "env-token",
			expectError:     false,
		},
		{
			name:            "env token and authContext	",
			accessTokens:    nil,
			authContexts:    []string{"context1"},
			allAuthContexts: false,
			envToken:        "env-token",
			expectError:     false,
		},
		{
			name:            "env token and allAuthContexts	",
			accessTokens:    nil,
			authContexts:    nil,
			allAuthContexts: true,
			envToken:        "env-token",
			expectError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup global flags for the test case.
			accessTokens = tt.accessTokens
			authContexts = tt.authContexts
			allAuthContexts = tt.allAuthContexts

			// Set environment variables for the test case.
			t.Setenv("DIGITALOCEAN_ACCESS_TOKEN", tt.envToken)

			// Special handling for the 'no auth method' test to ensure it runs in an isolated environment.
			if tt.name == "no auth method" {
				// Create a temporary directory to act as the user's home directory.
				// This ensures the test doesn't depend on a real doctl config file.
				t.Setenv("HOME", t.TempDir())
			}

			// Create a dummy command for testing.
			cmd := &cobra.Command{Use: "test"}

			// Execute validation.
			err := validateAuthFlags(cmd, []string{})

			// Check result.
			if (err != nil) != tt.expectError {
				t.Errorf("validateAuthFlags() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

func TestValidateAuthFlags_HelpCommand(t *testing.T) {
	// Tests that the help command always bypasses validation
	cmd := &cobra.Command{Use: "help"}
	err := validateAuthFlags(cmd, []string{})
	if err != nil {
		t.Errorf("validateAuthFlags() should not return error for help command, got: %v", err)
	}
}
