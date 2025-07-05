package kubeconfig

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// BackupKubeconfig creates a backup of the kubeconfig file at srcPath and saves it to backupPath.
// It performs an atomic write by first writing to a temporary file and then renaming it.
// The backup file will have the same file permissions as the source file.
// Both paths can include tilde (~) which will be expanded to the user's home directory.
func BackupKubeconfig(srcPath, backupPath string) error {
	// Expand any ~ in paths
	srcPath = expandPath(srcPath)
	backupPath = expandPath(backupPath)

	// Verify source file exists
	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		return fmt.Errorf("source file %s does not exist", srcPath)
	} else if err != nil {
		return fmt.Errorf("error checking source file: %v", err)
	}

	// Ensure backup directory exists
	backupDir := filepath.Dir(backupPath)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory %s: %v", backupDir, err)
	}

	// Create a temporary file in the same directory as the backup
	tmpFile, err := os.CreateTemp(backupDir, ".kubectl-doks-backup-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name()) // Clean up temp file in case of error

	// Open source file for reading
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %v", err)
	}
	defer srcFile.Close()

	// Copy content to temp file
	if _, err = io.Copy(tmpFile, srcFile); err != nil {
		tmpFile.Close() // Close before returning error
		return fmt.Errorf("failed to copy source file content: %v", err)
	}

	// Get the source file permissions
	srcInfo, err := os.Stat(srcPath)
	if err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to stat source file: %v", err)
	}

	// Set permissions of the temp file to match the source
	if err := os.Chmod(tmpFile.Name(), srcInfo.Mode()); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to set permissions on temp file: %v", err)
	}

	// Close the file to flush changes to disk
	if err = tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temporary file: %v", err)
	}

	// Atomically rename the temp file to the backup file
	if err = os.Rename(tmpFile.Name(), backupPath); err != nil {
		return fmt.Errorf("failed to rename temp file to backup file: %v", err)
	}

	return nil
}

// expandPath expands the tilde (~) character in a path to the user's home directory
func expandPath(path string) string {
	if path == "~" || path == "~/" {
		homedir, err := os.UserHomeDir()
		if err != nil {
			return path // Return original path if we can't expand
		}
		return homedir
	} else if len(path) > 2 && path[:2] == "~/" {
		homedir, err := os.UserHomeDir()
		if err != nil {
			return path // Return original path if we can't expand
		}
		return filepath.Join(homedir, path[2:])
	}
	return path
}
