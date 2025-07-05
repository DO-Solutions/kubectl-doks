package kubeconfig

import (
	"fmt"
	"os"
	"path/filepath"
)

// GetKubeconfig returns the path and content of the kubeconfig file.
// If the provided path is empty, it defaults to ~/.kube/config.
// If the file does not exist, it returns the resolved path, an empty byte slice for the content, and no error.
func GetKubeconfig(path string) (string, []byte, error) {
	if path == "" {
		homedir, err := os.UserHomeDir()
		if err != nil {
			return "", nil, fmt.Errorf("finding home directory: %w", err)
		}
		path = filepath.Join(homedir, ".kube", "config")
	}

	configBytes, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return path, []byte{}, nil // File doesn't exist, return empty config.
		}
		return path, nil, fmt.Errorf("reading kubeconfig at %s: %w", path, err)
	}
	return path, configBytes, nil
}
