package authutil

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"go.datum.net/datumctl/internal/datumconfig"
	"go.datum.net/datumctl/internal/keyring"
)

// GetUserKeyForCurrentContext resolves the user key bound to the current context.
// If the context does not yet define a user, it attempts a one-time migration
// from existing keyring mappings and persists the user reference into config.
func GetUserKeyForCurrentContext() (string, string, error) {
	cfg, ctxEntry, clusterEntry, err := datumconfig.LoadCurrentContext()
	if err != nil {
		return "", "", err
	}
	if ctxEntry == nil || clusterEntry == nil {
		return bootstrapUserFromKeyring(cfg)
	}

	if ctxEntry.Context.User != "" {
		if userEntry, ok := cfg.UserByName(ctxEntry.Context.User); ok && userEntry.User.Key != "" {
			return userEntry.User.Key, clusterEntry.Name, nil
		}
		// If user reference points directly to a key, accept it and backfill users.
		userKey := ctxEntry.Context.User
		if err := ensureUserEntry(cfg, ctxEntry, userKey); err != nil {
			return "", "", err
		}
		if err := datumconfig.Save(cfg); err != nil {
			return "", "", err
		}
		return userKey, clusterEntry.Name, nil
	}

	userKey, err := resolveUserKeyForCluster(clusterEntry.Name)
	if err != nil {
		return "", "", err
	}

	if err := ensureUserEntry(cfg, ctxEntry, userKey); err != nil {
		return "", "", err
	}
	if err := datumconfig.Save(cfg); err != nil {
		return "", "", err
	}

	return userKey, clusterEntry.Name, nil
}

func ensureUserEntry(cfg *datumconfig.Config, ctxEntry *datumconfig.NamedContext, userKey string) error {
	if cfg == nil || ctxEntry == nil {
		return fmt.Errorf("invalid config or context")
	}
	userName := userKey
	cfg.UpsertUser(datumconfig.NamedUser{
		Name: userName,
		User: datumconfig.User{Key: userKey},
	})
	ctxEntry.Context.User = userName
	cfg.UpsertContext(*ctxEntry)
	return nil
}

func resolveUserKeyForCluster(clusterName string) (string, error) {
	if clusterName == "" {
		return "", ErrNoCurrentContext
	}

	userKey, err := keyring.Get(ServiceName, clusterActiveUserKey(clusterName))
	if err == nil && userKey != "" {
		return migrateUserKeyIfNeeded(clusterName, userKey)
	}
	if err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return "", fmt.Errorf("failed to get active user for cluster %q: %w", clusterName, err)
	}

	legacyUserKey, err := keyring.Get(ServiceName, ActiveUserKey)
	if err == nil && legacyUserKey != "" {
		return migrateUserKeyIfNeeded(clusterName, legacyUserKey)
	}
	if err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return "", fmt.Errorf("failed to get legacy active user: %w", err)
	}

	return "", ErrNoActiveUserForCluster
}

func migrateUserKeyIfNeeded(clusterName, userKey string) (string, error) {
	creds, err := GetStoredCredentials(userKey)
	if err != nil {
		return "", err
	}
	if creds.Subject == "" || creds.Hostname == "" {
		return "", fmt.Errorf("stored credentials missing subject or hostname for %q", userKey)
	}

	newUserKey := fmt.Sprintf("%s@%s", creds.Subject, creds.Hostname)
	if newUserKey != userKey {
		credsJSON, err := json.Marshal(creds)
		if err != nil {
			return "", err
		}
		if err := keyring.Set(ServiceName, newUserKey, string(credsJSON)); err != nil {
			return "", err
		}
		if err := keyring.Set(ServiceName, clusterActiveUserKey(clusterName), newUserKey); err != nil {
			return "", err
		}
		if err := AddKnownUserKey(newUserKey); err != nil {
			return "", err
		}
		return newUserKey, nil
	}

	if err := keyring.Set(ServiceName, clusterActiveUserKey(clusterName), userKey); err != nil {
		return "", err
	}
	return userKey, nil
}

func bootstrapUserFromKeyring(cfg *datumconfig.Config) (string, string, error) {
	if cfg == nil {
		return "", "", ErrNoCurrentContext
	}

	legacyKey, err := keyring.Get(ServiceName, ActiveUserKey)
	if err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return "", "", fmt.Errorf("failed to read legacy active user: %w", err)
	}

	candidateKey := legacyKey
	if candidateKey == "" {
		knownUsersJSON, err := keyring.Get(ServiceName, KnownUsersKey)
		if err != nil && !errors.Is(err, keyring.ErrNotFound) {
			return "", "", fmt.Errorf("failed to read known users: %w", err)
		}
		if knownUsersJSON != "" {
			var knownUsers []string
			if err := json.Unmarshal([]byte(knownUsersJSON), &knownUsers); err != nil {
				return "", "", fmt.Errorf("failed to unmarshal known users: %w", err)
			}
			if len(knownUsers) == 1 {
				candidateKey = knownUsers[0]
			}
		}
	}

	if candidateKey == "" {
		return "", "", ErrNoCurrentContext
	}

	creds, err := GetStoredCredentials(candidateKey)
	if err != nil {
		return "", "", err
	}

	apiHostname := creds.APIHostname
	if apiHostname == "" {
		apiHostname, err = DeriveAPIHostname(creds.Hostname)
		if err != nil {
			return "", "", err
		}
	}

	clusterName := "datum-" + sanitizeClusterName(apiHostname)
	userKey, err := migrateUserKeyIfNeeded(clusterName, candidateKey)
	if err != nil {
		return "", "", err
	}

	cluster := datumconfig.Cluster{
		Server: datumconfig.CleanBaseServer(datumconfig.EnsureScheme(apiHostname)),
	}
	if err := cfg.ValidateCluster(cluster); err != nil {
		return "", "", err
	}

	ctx := datumconfig.Context{
		Cluster: clusterName,
		User:    userKey,
	}
	cfg.EnsureContextDefaults(&ctx)
	if err := cfg.ValidateContext(ctx); err != nil {
		return "", "", err
	}

	cfg.UpsertCluster(datumconfig.NamedCluster{
		Name:    clusterName,
		Cluster: cluster,
	})
	cfg.UpsertUser(datumconfig.NamedUser{
		Name: userKey,
		User: datumconfig.User{Key: userKey},
	})
	cfg.UpsertContext(datumconfig.NamedContext{
		Name:    clusterName,
		Context: ctx,
	})
	if cfg.CurrentContext == "" {
		cfg.CurrentContext = clusterName
	}

	if err := datumconfig.Save(cfg); err != nil {
		return "", "", err
	}

	return userKey, clusterName, nil
}

func sanitizeClusterName(apiHostname string) string {
	name := strings.TrimSpace(apiHostname)
	name = strings.TrimPrefix(name, "https://")
	name = strings.TrimPrefix(name, "http://")
	name = strings.TrimSuffix(name, "/")
	name = strings.NewReplacer(":", "-", "/", "-", " ", "-").Replace(name)
	return name
}
