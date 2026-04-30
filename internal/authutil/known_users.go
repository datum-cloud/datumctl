package authutil

import (
	"encoding/json"
	"errors"
	"fmt"

	"go.datum.net/datumctl/internal/keyring"
)

// AddKnownUserKey adds a userKey (subject@hostname) to the known_users list in the keyring.
func AddKnownUserKey(newUserKey string) error {
	knownUsers := []string{}

	knownUsersJSON, err := keyring.Get(ServiceName, KnownUsersKey)
	if err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return fmt.Errorf("failed to get known users list from keyring: %w", err)
	}

	if err == nil && knownUsersJSON != "" {
		if err := json.Unmarshal([]byte(knownUsersJSON), &knownUsers); err != nil {
			return fmt.Errorf("failed to unmarshal known users list: %w", err)
		}
	}

	found := false
	for _, key := range knownUsers {
		if key == newUserKey {
			found = true
			break
		}
	}

	if !found {
		knownUsers = append(knownUsers, newUserKey)
		updatedJSON, err := json.Marshal(knownUsers)
		if err != nil {
			return fmt.Errorf("failed to marshal updated known users list: %w", err)
		}
		if err := keyring.Set(ServiceName, KnownUsersKey, string(updatedJSON)); err != nil {
			return fmt.Errorf("failed to store updated known users list: %w", err)
		}
	}

	return nil
}
