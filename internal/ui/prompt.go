package ui

import (
	"errors"

	"github.com/AlecAivazis/survey/v2"
	"github.com/DO-Solutions/kubectl-doks/do"
)

// Cluster selects a cluster from a list of clusters and returns the selected cluster.
// If the list is empty, it returns an error.
func Cluster(clusters []do.Cluster) (do.Cluster, error) {
	var selected do.Cluster

	if len(clusters) == 0 {
		return selected, errors.New("no clusters found")
	}

	clusterNames := make([]string, len(clusters))
	for i, c := range clusters {
		clusterNames[i] = c.Name
	}

	var selectedClusterName string
	prompt := &survey.Select{
		Message: "Select a cluster:",
		Options: clusterNames,
	}

	err := survey.AskOne(prompt, &selectedClusterName, survey.WithKeepFilter(true))
	if err != nil {
		return selected, err
	}

	for _, c := range clusters {
		if c.Name == selectedClusterName {
			selected = c
			break
		}
	}

	return selected, nil
}
