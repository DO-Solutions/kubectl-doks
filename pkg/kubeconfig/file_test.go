package kubeconfig

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetKubeconfig(t *testing.T) {
	t.Run("path provided and file exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		kubeconfigFile := filepath.Join(tmpDir, "config")
		content := []byte("test-data")
		err := os.WriteFile(kubeconfigFile, content, 0600)
		require.NoError(t, err)

		path, data, err := GetKubeconfig(kubeconfigFile)
		require.NoError(t, err)
		assert.Equal(t, kubeconfigFile, path)
		assert.Equal(t, content, data)
	})

	t.Run("path provided and file does not exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		kubeconfigFile := filepath.Join(tmpDir, "non-existent-config")

		path, data, err := GetKubeconfig(kubeconfigFile)
		require.NoError(t, err)
		assert.Equal(t, kubeconfigFile, path)
		assert.Empty(t, data)
	})

	t.Run("default path", func(t *testing.T) {
		tmpHome := t.TempDir()
		t.Setenv("HOME", tmpHome)

		expectedPath := filepath.Join(tmpHome, ".kube", "config")
		path, data, err := GetKubeconfig("")
		require.NoError(t, err)
		assert.Equal(t, expectedPath, path)
		assert.Empty(t, data)
	})

	t.Run("path is a directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		_, _, err := GetKubeconfig(tmpDir)
		require.Error(t, err)
	})
}
