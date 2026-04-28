package authutil

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// serviceAccountKeyDir returns the directory used to store service account PEM key files.
// It does not create the directory.
func serviceAccountKeyDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to determine user config directory: %w", err)
	}
	return filepath.Join(configDir, "datumctl", "service-accounts"), nil
}

// ServiceAccountKeyFilePath returns the on-disk path where the PEM key for
// the given userKey is stored. It does not create the directory.
func ServiceAccountKeyFilePath(userKey string) (string, error) {
	dir, err := serviceAccountKeyDir()
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(userKey))
	filename := hex.EncodeToString(sum[:]) + ".pem"
	return filepath.Join(dir, filename), nil
}

// WriteServiceAccountKeyFile atomically writes the PEM key to disk for the
// given userKey and returns the absolute path. Creates the parent directory
// with mode 0700 if needed. The file is written with mode 0600.
func WriteServiceAccountKeyFile(userKey, pemKey string) (string, error) {
	dir, err := serviceAccountKeyDir()
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("failed to create service account key directory %s: %w", dir, err)
	}

	// Tighten perms on the directory even when it already existed with looser ones.
	if err := os.Chmod(dir, 0700); err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("failed to set permissions on service account key directory %s: %w", dir, err)
	}

	destPath, err := ServiceAccountKeyFilePath(userKey)
	if err != nil {
		return "", err
	}

	// Use a unique temp filename so concurrent logins for the same account
	// do not race on the same .tmp path and corrupt each other's writes.
	tmpFile, err := os.CreateTemp(dir, filepath.Base(destPath)+".tmp.*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file for service account key in %s: %w", dir, err)
	}
	tmpName := tmpFile.Name()

	// Ensure the temp file is removed on any error path. After a successful
	// Rename the file no longer exists at tmpName, so Remove is a no-op.
	var writeErr error
	defer func() {
		if writeErr != nil {
			_ = os.Remove(tmpName)
		}
	}()

	// Be explicit about mode even though CreateTemp already uses 0600.
	if writeErr = tmpFile.Chmod(0600); writeErr != nil {
		_ = tmpFile.Close()
		writeErr = fmt.Errorf("failed to set permissions on temp key file %s: %w", tmpName, writeErr)
		return "", writeErr
	}

	if _, writeErr = tmpFile.Write([]byte(pemKey)); writeErr != nil {
		_ = tmpFile.Close()
		writeErr = fmt.Errorf("failed to write service account key to %s: %w", tmpName, writeErr)
		return "", writeErr
	}

	if writeErr = tmpFile.Close(); writeErr != nil {
		writeErr = fmt.Errorf("failed to close temp key file %s: %w", tmpName, writeErr)
		return "", writeErr
	}

	if writeErr = os.Rename(tmpName, destPath); writeErr != nil {
		writeErr = fmt.Errorf("failed to move service account key to %s: %w", destPath, writeErr)
		return "", writeErr
	}

	return destPath, nil
}

// ReadServiceAccountKeyFile reads a PEM key from the given path.
func ReadServiceAccountKeyFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read service account key file %s: %w", path, err)
	}
	return string(data), nil
}

// RemoveServiceAccountKeyFile deletes the PEM key file for the given userKey.
// Returns nil if the file does not exist.
func RemoveServiceAccountKeyFile(userKey string) error {
	path, err := ServiceAccountKeyFilePath(userKey)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove service account key file %s: %w", path, err)
	}
	return nil
}
