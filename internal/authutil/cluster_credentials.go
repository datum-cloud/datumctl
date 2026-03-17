package authutil

import (
	"errors"
	"fmt"

	"go.datum.net/datumctl/internal/keyring"
)

const clusterActiveUserKeyPrefix = "active_user.cluster."

func clusterActiveUserKey(clusterName string) string {
	return clusterActiveUserKeyPrefix + clusterName
}

// GetActiveUserKeyForCluster returns the user key mapped to the cluster.
func GetActiveUserKeyForCluster(clusterName string) (string, error) {
	if clusterName == "" {
		return "", ErrNoCurrentContext
	}

	userKey, err := keyring.Get(ServiceName, clusterActiveUserKey(clusterName))
	if err == nil && userKey != "" {
		return userKey, nil
	}
	if err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return "", fmt.Errorf("failed to get active user for cluster %q: %w", clusterName, err)
	}

	return "", ErrNoActiveUserForCluster
}

// SetActiveUserKeyForCluster stores a cluster-specific mapping to a user key.
func SetActiveUserKeyForCluster(clusterName, userKey string) error {
	if clusterName == "" || userKey == "" {
		return fmt.Errorf("cluster name and user key are required")
	}
	return keyring.Set(ServiceName, clusterActiveUserKey(clusterName), userKey)
}
