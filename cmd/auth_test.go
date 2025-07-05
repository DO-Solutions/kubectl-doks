package cmd

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

// setup and teardown for global flags
func setup(t *testing.T) {
	// Reset global flags before each test
	accessTokens = nil
	authContexts = nil
	allAuthContexts = false
	configFile = ""
	t.Setenv("DIGITALOCEAN_ACCESS_TOKEN", "")
}

func TestValidateAuthSources(t *testing.T) {
	tests := []struct {
		name        string
		setup       func()
		expectError bool
	}{
		{
			name: "No flags",
			setup: func() {
				accessTokens = nil
				authContexts = nil
				allAuthContexts = false
			},
			expectError: false,
		},
		{
			name: "Only access-token",
			setup: func() {
				accessTokens = []string{"token1"}
				authContexts = nil
				allAuthContexts = false
			},
			expectError: false,
		},
		{
			name: "Only auth-context",
			setup: func() {
				accessTokens = nil
				authContexts = []string{"context1"}
				allAuthContexts = false
			},
			expectError: false,
		},
		{
			name: "Only all-auth-contexts",
			setup: func() {
				accessTokens = nil
				authContexts = nil
				allAuthContexts = true
			},
			expectError: false,
		},
		{
			name: "Access token and auth context should error",
			setup: func() {
				accessTokens = []string{"token1"}
				authContexts = []string{"context1"}
				allAuthContexts = false
			},
			expectError: true,
		},
		{
			name: "Access token and all auth contexts should error",
			setup: func() {
				accessTokens = []string{"token1"}
				authContexts = nil
				allAuthContexts = true
			},
			expectError: true,
		},
		{
			name: "Auth context and all auth contexts should error",
			setup: func() {
				accessTokens = nil
				authContexts = []string{"context1"}
				allAuthContexts = true
			},
			expectError: true,
		},
		{
			name: "Access token, auth context and all auth contexts should error",
			setup: func() {
				accessTokens = []string{"token1"}
				authContexts = []string{"context1"}
				allAuthContexts = true
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setup(t)
			tt.setup()
			err := validateAuthSources()
			if (err != nil) != tt.expectError {
				t.Errorf("validateAuthSources() error = %v, wantErr %v", err, tt.expectError)
			}
		})
	}
}

func TestUnique(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{"empty slice", []string{}, []string{}},
		{"no duplicates", []string{"a", "b", "c"}, []string{"a", "b", "c"}},
		{"with duplicates", []string{"a", "b", "a", "c", "b"}, []string{"a", "b", "c"}},
		{"all duplicates", []string{"a", "a", "a"}, []string{"a"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := unique(tt.input)
			sort.Strings(got)
			sort.Strings(tt.want)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("unique() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetDoctlConfigPath(t *testing.T) {
	t.Run("configFile flag is set", func(t *testing.T) {
		setup(t)
		expectedPath := "/custom/path/config.yaml"
		configFile = expectedPath
		path := getDoctlConfigPath()
		if path != expectedPath {
			t.Errorf("getDoctlConfigPath() = %v, want %v", path, expectedPath)
		}
	})

	t.Run("configFile flag is not set", func(t *testing.T) {
		setup(t)
		homeDir := t.TempDir()
		t.Setenv("HOME", homeDir)
		expectedPath := filepath.Join(homeDir, ".config", "doctl", "config.yaml")
		path := getDoctlConfigPath()
		if path != expectedPath {
			t.Errorf("getDoctlConfigPath() = %v, want %v", path, expectedPath)
		}
	})
}

func createMockDoctlConfig(t *testing.T, content string) string {
	t.Helper()
	tmpFile, err := os.CreateTemp(t.TempDir(), "config-*.yaml")
	if err != nil {
			t.Fatalf("Failed to create temp config file: %v", err)
	}
	if _, err := tmpFile.Write([]byte(content)); err != nil {
			t.Fatalf("Failed to write to temp config file: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
			t.Fatalf("Failed to close temp config file: %v", err)
	}
	return tmpFile.Name()
}

const mockConfigContent = `
auth-contexts:
  context1:
    access-token: token1
  context2:
    access-token: token2
  context3:
    access-token: token3
context: context1
`

func TestGetAllAccessTokens(t *testing.T) {
	mockConfigPath := createMockDoctlConfig(t, mockConfigContent)

	tests := []struct {
		name        string
		setup       func()
		want        []string
		expectError bool
	}{
		{
			name: "using --access-token flag",
			setup: func() {
				accessTokens = []string{"flag-token1", "flag-token2"}
			},
			want:        []string{"flag-token1", "flag-token2"},
			expectError: false,
		},
		{
			name: "using --auth-context flag",
			setup: func() {
				authContexts = []string{"context2", "context3"}
				configFile = mockConfigPath
			},
			want:        []string{"token2", "token3"},
			expectError: false,
		},
		{
			name: "using --all-auth-contexts flag",
			setup: func() {
				allAuthContexts = true
				configFile = mockConfigPath
			},
			want:        []string{"token1", "token2", "token3"},
			expectError: false,
		},
		{
			name: "using DIGITALOCEAN_ACCESS_TOKEN env var",
			setup: func() {
				t.Setenv("DIGITALOCEAN_ACCESS_TOKEN", "env-token")
			},
			want:        []string{"env-token"},
			expectError: false,
		},
		{
			name: "using current doctl context",
			setup: func() {
				configFile = mockConfigPath
			},
			want:        []string{"token1"},
			expectError: false,
		},
		{
			name: "precedence: --access-token wins over everything",
			setup: func() {
				accessTokens = []string{"flag-token"}
				authContexts = []string{"context1"}
				allAuthContexts = true
				t.Setenv("DIGITALOCEAN_ACCESS_TOKEN", "env-token")
				configFile = mockConfigPath
			},
			want:        []string{"flag-token"},
			expectError: false,
		},
		{
			name: "precedence: --auth-context wins over env and current context",
			setup: func() {
				authContexts = []string{"context2"}
				t.Setenv("DIGITALOCEAN_ACCESS_TOKEN", "env-token")
				configFile = mockConfigPath
			},
			want:        []string{"token2"},
			expectError: false,
		},
		{
			name: "precedence: env var wins over current context",
			setup: func() {
				t.Setenv("DIGITALOCEAN_ACCESS_TOKEN", "env-token")
				configFile = mockConfigPath
			},
			want:        []string{"env-token"},
			expectError: false,
		},
		{
			name: "no tokens found for specified context",
			setup: func() {
				authContexts = []string{"non-existent-context"}
				configFile = mockConfigPath
			},
			want:        nil,
			expectError: true,
		},
		{
			name: "config file not found",
			setup: func() {
				authContexts = []string{"context1"}
				configFile = "/path/to/non/existent/config.yaml"
			},
			want:        nil,
			expectError: true,
		},
		{
			name: "no auth methods provided and no config file",
			setup: func() {
				// This test relies on the default config path not existing.
				t.Setenv("HOME", t.TempDir())
			},
			want:        nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setup(t)
			tt.setup()

			got, err := getAllAccessTokens()

			if (err != nil) != tt.expectError {
				t.Errorf("getAllAccessTokens() error = %v, wantErr %v", err, tt.expectError)
				return
			}

			// Sort slices for consistent comparison
			sort.Strings(got)
			sort.Strings(tt.want)

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getAllAccessTokens() = %v, want %v", got, tt.want)
			}
		})
	}
}

