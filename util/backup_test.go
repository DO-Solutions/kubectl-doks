package util

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBackupKubeconfig(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "backup-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test source file
	srcContent := "test kubeconfig content"
	srcPath := filepath.Join(tmpDir, "config")
	if err := os.WriteFile(srcPath, []byte(srcContent), 0644); err != nil {
		t.Fatalf("Failed to create test source file: %v", err)
	}

	// Define backup path
	backupPath := filepath.Join(tmpDir, "backup", "config.bak")

	// Test successful backup
	t.Run("Successful backup", func(t *testing.T) {
		// Set specific permissions on source file for testing
		expectedMode := os.FileMode(0640)
		if err := os.Chmod(srcPath, expectedMode); err != nil {
			t.Fatalf("Failed to set permissions on source file: %v", err)
		}

		// Get source file info before backup
		srcInfo, err := os.Stat(srcPath)
		if err != nil {
			t.Fatalf("Failed to stat source file: %v", err)
		}

		err = BackupKubeconfig(srcPath, backupPath)
		if err != nil {
			t.Fatalf("BackupKubeconfig failed: %v", err)
		}

		// Verify backup file exists
		backupContent, err := os.ReadFile(backupPath)
		if err != nil {
			t.Fatalf("Failed to read backup file: %v", err)
		}

		// Verify content matches
		if string(backupContent) != srcContent {
			t.Fatalf("Backup content doesn't match source. Got %q, want %q", string(backupContent), srcContent)
		}

		// Verify permissions match
		backupInfo, err := os.Stat(backupPath)
		if err != nil {
			t.Fatalf("Failed to stat backup file: %v", err)
		}

		if backupInfo.Mode().Perm() != srcInfo.Mode().Perm() {
			t.Fatalf("Backup file permissions don't match source. Got %v, want %v", 
				backupInfo.Mode().Perm(), srcInfo.Mode().Perm())
		}
	})

	// Test backup with nonexistent source
	t.Run("Nonexistent source", func(t *testing.T) {
		nonExistentPath := filepath.Join(tmpDir, "nonexistent")
		err := BackupKubeconfig(nonExistentPath, backupPath)
		if err == nil {
			t.Fatal("Expected error for nonexistent source, got nil")
		}
		if !strings.Contains(err.Error(), "does not exist") {
			t.Fatalf("Expected 'does not exist' error, got: %v", err)
		}
	})

	// Test backup with inaccessible backup directory
	t.Run("Inaccessible backup directory", func(t *testing.T) {
		// Skip this test if running as root since root can write anywhere
		if os.Geteuid() == 0 {
			t.Skip("Skipping test since running as root")
		}

		// Try to write to a protected system directory
		protectedPath := "/proc/something-not-writable"
		err := BackupKubeconfig(srcPath, protectedPath)
		if err == nil {
			t.Fatal("Expected error for protected backup path, got nil")
		}
	})
}

func TestExpandPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get user home directory: %v", err)
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Just tilde", "~", home},
		{"Tilde with slash", "~/", home},
		{"Tilde with path", "~/Documents", filepath.Join(home, "Documents")},
		{"No tilde", "/etc/hosts", "/etc/hosts"},
		{"Empty path", "", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := expandPath(tc.input)
			if result != tc.expected {
				t.Fatalf("expandPath(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}
