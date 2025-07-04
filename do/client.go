package do

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/digitalocean/godo"
)

// Cluster represents a DigitalOcean Kubernetes cluster with minimal required fields
type Cluster struct {
	ID     string
	Name   string
	Region string
}

// Client provides an interface to interact with DigitalOcean Kubernetes API
type Client struct {
	godoClient *godo.Client
}

// NewClient creates a new DO API client with the given access token
func NewClient(accessToken string, apiURL string) (*Client, error) {
	if accessToken == "" {
		return nil, errors.New("access token is required")
	}

	// Create the client using the token directly
	client := godo.NewFromToken(accessToken)

	// Set custom API URL if provided
	if apiURL != "" {
		// Parse the custom URL
		customURL, err := url.Parse(apiURL)
		if err != nil {
			return nil, fmt.Errorf("invalid API URL: %v", err)
		}
		client.BaseURL = customURL
	}
	return &Client{godoClient: client}, nil
}

// NewClientFromEnv creates a new DO API client using the DIGITALOCEAN_ACCESS_TOKEN environment variable
func NewClientFromEnv(apiURL string) (*Client, error) {
	token := os.Getenv("DIGITALOCEAN_ACCESS_TOKEN")
	if token == "" {
		return nil, errors.New("DIGITALOCEAN_ACCESS_TOKEN environment variable is not set")
	}

	return NewClient(token, apiURL)
}

// ListClusters returns a list of all Kubernetes clusters in the account
func (c *Client) ListClusters(ctx context.Context) ([]Cluster, error) {
	opt := &godo.ListOptions{}
	var allClusters []Cluster

	for {
		clusters, resp, err := c.godoClient.Kubernetes.List(ctx, opt)
		if err != nil {
			return nil, fmt.Errorf("error listing clusters: %v", err)
		}

		for _, cluster := range clusters {
			allClusters = append(allClusters, Cluster{
				ID:     cluster.ID,
				Name:   cluster.Name,
				Region: cluster.RegionSlug,
			})
		}

		// Check if we've reached the last page
		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}

		// Get the next page
		page, err := resp.Links.CurrentPage()
		if err != nil {
			return nil, fmt.Errorf("error getting current page: %v", err)
		}

		opt.Page = page + 1
	}

	return allClusters, nil
}

// GetKubeConfig returns the kubeconfig for a specific cluster as a byte array
func (c *Client) GetKubeConfig(ctx context.Context, clusterID string) ([]byte, error) {
	if strings.TrimSpace(clusterID) == "" {
		return nil, errors.New("cluster ID cannot be empty")
	}

	kubeConfig, _, err := c.godoClient.Kubernetes.GetKubeConfig(ctx, clusterID)
	if err != nil {
		return nil, fmt.Errorf("error retrieving kubeconfig for cluster %s: %v", clusterID, err)
	}

	return kubeConfig.KubeconfigYAML, nil
}
