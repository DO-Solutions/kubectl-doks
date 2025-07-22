package kubeconfig

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestGetClusterID(t *testing.T) {
	tests := []struct {
		name          string
		cluster       *api.Cluster
		expectedID    string
		expectedFound bool
	}{
		{
			name:          "no extensions",
			cluster:       &api.Cluster{},
			expectedID:    "",
			expectedFound: false,
		},
		{
			name: "extension exists",
			cluster: &api.Cluster{
				Extensions: map[string]runtime.Object{
					DigitalOceanClusterIDExtension: &runtime.Unknown{Raw: []byte(`{"id":"test-id"}`)},
				},
			},
			expectedID:    "test-id",
			expectedFound: true,
		},
		{
			name: "other extensions exist",
			cluster: &api.Cluster{
				Extensions: map[string]runtime.Object{
					"other-extension": &runtime.Unknown{Raw: []byte(`{"key":"value"}`)},
				},
			},
			expectedID:    "",
			expectedFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, found := GetClusterID(tt.cluster)
			assert.Equal(t, tt.expectedID, id)
			assert.Equal(t, tt.expectedFound, found)
		})
	}
}

func TestSetClusterID(t *testing.T) {
	tests := []struct {
		name    string
		cluster *api.Cluster
		idToSet string
	}{
		{
			name:    "add to new cluster",
			cluster: &api.Cluster{},
			idToSet: "new-id",
		},
		{
			name: "update existing extension",
			cluster: &api.Cluster{
				Extensions: map[string]runtime.Object{
					DigitalOceanClusterIDExtension: &runtime.Unknown{Raw: []byte(`{"id":"old-id"}`)},
				},
			},
			idToSet: "updated-id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetClusterID(tt.cluster, tt.idToSet)
			id, found := GetClusterID(tt.cluster)
			assert.True(t, found)
			assert.Equal(t, tt.idToSet, id)
		})
	}
}
