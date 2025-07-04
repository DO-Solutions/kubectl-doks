package kubeconfig

import (
	"reflect"
	"testing"

	k8sclientcmd "k8s.io/client-go/tools/clientcmd"
	k8sclientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// Test data
var (
	srcKubeconfig = []byte(`apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: c291cmNlLWNhLWRhdGE=
    server: https://source.example.com
  name: source-cluster
contexts:
- context:
    cluster: source-cluster
    user: source-user
  name: source-context
current-context: source-context
kind: Config
users:
- name: source-user
  user:
    client-certificate-data: c291cmNlLWNlcnQtZGF0YQ==
    client-key-data: c291cmNlLWtleS1kYXRh`)
	newKubeconfig = []byte(`apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: bmV3LWNhLWRhdGE=
    server: https://new.example.com
  name: new-cluster
contexts:
- context:
    cluster: new-cluster
    user: new-user
  name: new-context
current-context: new-context
kind: Config
users:
- name: new-user
  user:
    client-certificate-data: bmV3LWNlcnQtZGF0YQ==
    client-key-data: bmV3LWtleS1kYXRh`)
	emptyConfig = []byte{}
	invalidConfig = []byte(`not-valid-yaml!`)
)

func TestMergeConfig(t *testing.T) {
	t.Run("Successful merge without setting current context", func(t *testing.T) {
		merged, err := MergeConfig(srcKubeconfig, newKubeconfig, false)
		if err != nil {
			t.Fatalf("MergeConfig failed: %v", err)
		}

		// Parse the merged config to verify
		config, err := k8sclientcmd.Load(merged)
		if err != nil {
			t.Fatalf("Failed to parse merged config: %v", err)
		}

		// Verify both clusters exist
		if _, exists := config.Clusters["source-cluster"]; !exists {
			t.Error("source-cluster not found in merged config")
		}
		if _, exists := config.Clusters["new-cluster"]; !exists {
			t.Error("new-cluster not found in merged config")
		}

		// Verify both contexts exist
		if _, exists := config.Contexts["source-context"]; !exists {
			t.Error("source-context not found in merged config")
		}
		if _, exists := config.Contexts["new-context"]; !exists {
			t.Error("new-context not found in merged config")
		}

		// Verify both users exist
		if _, exists := config.AuthInfos["source-user"]; !exists {
			t.Error("source-user not found in merged config")
		}
		if _, exists := config.AuthInfos["new-user"]; !exists {
			t.Error("new-user not found in merged config")
		}

		// Verify current context is still the context from the source config
		if config.CurrentContext != "source-context" {
			t.Errorf("Expected current-context to be 'source-context', got %q", config.CurrentContext)
		}
	})

	t.Run("Empty source config", func(t *testing.T) {
		_, err := MergeConfig(emptyConfig, newKubeconfig, false)
		if err == nil {
			t.Fatal("Expected error for empty source config, got nil")
		}
	})

	t.Run("Empty new config", func(t *testing.T) {
		_, err := MergeConfig(srcKubeconfig, emptyConfig, false)
		if err == nil {
			t.Fatal("Expected error for empty new config, got nil")
		}
	})

	t.Run("Invalid source config", func(t *testing.T) {
		_, err := MergeConfig(invalidConfig, newKubeconfig, false)
		if err == nil {
			t.Fatal("Expected error for invalid source config, got nil")
		}
	})

	t.Run("Invalid new config", func(t *testing.T) {
		_, err := MergeConfig(srcKubeconfig, invalidConfig, false)
		if err == nil {
			t.Fatal("Expected error for invalid new config, got nil")
		}
	})

	t.Run("Successful merge with setting current context", func(t *testing.T) {
		merged, err := MergeConfig(srcKubeconfig, newKubeconfig, true)
		if err != nil {
			t.Fatalf("MergeConfig failed: %v", err)
		}

		// Parse the merged config to verify
		config, err := k8sclientcmd.Load(merged)
		if err != nil {
			t.Fatalf("Failed to parse merged config: %v", err)
		}

		// Verify current context is set to the one from new config
		if config.CurrentContext != "new-context" {
			t.Errorf("Expected current-context to be 'new-context', got %q", config.CurrentContext)
		}
	})
}

func TestMergeConfigWithSetCurrentContext(t *testing.T) {
	t.Run("New config with no current context", func(t *testing.T) {
		// Create a new config with no current-context
		configObj := k8sclientcmdapi.NewConfig()
		configObj.Clusters["test-cluster"] = &k8sclientcmdapi.Cluster{Server: "https://test.example.com"}
		configObj.AuthInfos["test-user"] = &k8sclientcmdapi.AuthInfo{Token: "test-token"}
		configObj.Contexts["test-context"] = &k8sclientcmdapi.Context{Cluster: "test-cluster", AuthInfo: "test-user"}
		// Explicitly set current-context to empty string
		configObj.CurrentContext = ""

		configBytes, err := k8sclientcmd.Write(*configObj)
		if err != nil {
			t.Fatalf("Failed to serialize test config: %v", err)
		}

		_, err = MergeConfig(srcKubeconfig, configBytes, true)
		if err == nil {
			t.Fatal("Expected error for new config with no current context when setting current context, got nil")
		}
	})
}

// Helper function to test internal mergeKubeConfigObjects function
func TestMergeKubeConfigObjects(t *testing.T) {
	// Parse test configs
	srcObj, err := k8sclientcmd.Load(srcKubeconfig)
	if err != nil {
		t.Fatalf("Failed to parse source config: %v", err)
	}

	newObj, err := k8sclientcmd.Load(newKubeconfig)
	if err != nil {
		t.Fatalf("Failed to parse new config: %v", err)
	}

	// Create a copy of srcObj for testing
	target := srcObj.DeepCopy()

	// Merge the objects
	mergeKubeConfigObjects(target, newObj)

	// Check that target now has all elements from both configs
	// Check clusters
	if !reflect.DeepEqual(target.Clusters["new-cluster"], newObj.Clusters["new-cluster"]) {
		t.Error("New cluster was not properly merged")
	}
	if !reflect.DeepEqual(target.Clusters["source-cluster"], srcObj.Clusters["source-cluster"]) {
		t.Error("Source cluster was modified")
	}

	// Check contexts
	if !reflect.DeepEqual(target.Contexts["new-context"], newObj.Contexts["new-context"]) {
		t.Error("New context was not properly merged")
	}
	if !reflect.DeepEqual(target.Contexts["source-context"], srcObj.Contexts["source-context"]) {
		t.Error("Source context was modified")
	}

	// Check auth infos (users)
	if !reflect.DeepEqual(target.AuthInfos["new-user"], newObj.AuthInfos["new-user"]) {
		t.Error("New user was not properly merged")
	}
	if !reflect.DeepEqual(target.AuthInfos["source-user"], srcObj.AuthInfos["source-user"]) {
		t.Error("Source user was modified")
	}

	// Check current context was not updated by mergeKubeConfigObjects
	// This is now controlled by the setCurrentContext parameter in MergeConfig
	if target.CurrentContext != "source-context" {
		t.Errorf("Current context should not be updated by mergeKubeConfigObjects. Got %q, want %q", target.CurrentContext, "source-context")
	}
}
