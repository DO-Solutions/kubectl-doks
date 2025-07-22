package kubeconfig

import (
	"encoding/json"

	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/apimachinery/pkg/runtime"
)

// DigitalOceanClusterIDExtension is the name of the extension used to store the DigitalOcean cluster ID.
const DigitalOceanClusterIDExtension = "digitalocean.com/cluster-id"

// GetClusterID retrieves the DigitalOcean cluster ID from a kubeconfig cluster's extensions.
// It returns the ID and true if the extension is found, otherwise it returns an empty string and false.
func GetClusterID(cluster *api.Cluster) (string, bool) {
	extension, ok := cluster.Extensions[DigitalOceanClusterIDExtension]
	if !ok {
		return "", false
	}

	unknown, ok := extension.(*runtime.Unknown)
	if !ok {
		return "", false
	}

	var data map[string]string
	if err := json.Unmarshal(unknown.Raw, &data); err != nil {
		return "", false
	}

	id, ok := data["id"]
	return id, ok
}

// SetClusterID adds or updates the DigitalOcean cluster ID in a kubeconfig cluster's extensions.
func SetClusterID(cluster *api.Cluster, id string) {
	if cluster.Extensions == nil {
		cluster.Extensions = make(map[string]runtime.Object)
	}

	cluster.Extensions[DigitalOceanClusterIDExtension] = &runtime.Unknown{Raw: []byte(`{"id":"` + id + `"}`)}
}
