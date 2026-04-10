package authutil

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

// machineAccountKeyDir returns the directory used to store machine account PEM key files.
// It does not create the directory.
func machineAccountKeyDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to determine user config directory: %w", err)
	}
	return filepath.Join(configDir, "datumctl", "machine-accounts"), nil
}

// MachineAccountKeyFilePath returns the on-disk path where the PEM key for
// the given userKey is stored. It does not create the directory.
func MachineAccountKeyFilePath(userKey string) (string, error) {
	dir, err := machineAccountKeyDir()
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(userKey))
	filename := hex.EncodeToString(sum[:]) + ".pem"
	return filepath.Join(dir, filename), nil
}

// WriteMachineAccountKeyFile atomically writes the PEM key to disk for the
// given userKey and returns the absolute path. Creates the parent directory
// with mode 0700 if needed. The file is written with mode 0600.
func WriteMachineAccountKeyFile(userKey, pemKey string) (string, error) {
	dir, err := machineAccountKeyDir()
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("failed to create machine account key directory %s: %w", dir, err)
	}

	destPath, err := MachineAccountKeyFilePath(userKey)
	if err != nil {
		return "", err
	}

	tmpPath := destPath + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(pemKey), 0600); err != nil {
		return "", fmt.Errorf("failed to write machine account key to %s: %w", tmpPath, err)
	}

	if err := os.Rename(tmpPath, destPath); err != nil {
		// Best-effort cleanup of the temp file on rename failure.
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("failed to move machine account key to %s: %w", destPath, err)
	}

	return destPath, nil
}

// ReadMachineAccountKeyFile reads a PEM key from the given path.
func ReadMachineAccountKeyFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read machine account key file %s: %w", path, err)
	}
	return string(data), nil
}

// RemoveMachineAccountKeyFile deletes the PEM key file for the given userKey.
// Returns nil if the file does not exist.
func RemoveMachineAccountKeyFile(userKey string) error {
	path, err := MachineAccountKeyFilePath(userKey)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove machine account key file %s: %w", path, err)
	}
	return nil
}
