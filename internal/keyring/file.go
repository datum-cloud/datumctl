package keyring

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// fileBackend stores secrets in a JSON file on disk. It is used as a fallback
// when the system keyring is unavailable. The file is written with 0600
// permissions but is otherwise unencrypted, so secrets stored this way must be
// treated as sensitive on-disk data.
type fileBackend struct {
	path string
	mu   sync.Mutex
}

// fileStore maps service name to user key to secret value.
type fileStore map[string]map[string]string

func newFileBackend() (*fileBackend, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get home dir: %w", err)
	}
	return &fileBackend{path: filepath.Join(home, ".datumctl", "credentials.json")}, nil
}

func (f *fileBackend) load() (fileStore, error) {
	data, err := os.ReadFile(f.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fileStore{}, nil
		}
		return nil, err
	}
	store := fileStore{}
	if len(data) == 0 {
		return store, nil
	}
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, fmt.Errorf("parse credentials file %s: %w", f.path, err)
	}
	return store, nil
}

func (f *fileBackend) save(store fileStore) error {
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal credentials: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(f.path), 0o700); err != nil {
		return fmt.Errorf("create credentials dir: %w", err)
	}
	return os.WriteFile(f.path, data, 0o600)
}

func (f *fileBackend) Set(service, user, secret string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	store, err := f.load()
	if err != nil {
		return err
	}
	if store[service] == nil {
		store[service] = map[string]string{}
	}
	store[service][user] = secret
	return f.save(store)
}

func (f *fileBackend) Get(service, user string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	store, err := f.load()
	if err != nil {
		return "", err
	}
	users, ok := store[service]
	if !ok {
		return "", ErrNotFound
	}
	secret, ok := users[user]
	if !ok {
		return "", ErrNotFound
	}
	return secret, nil
}

func (f *fileBackend) Delete(service, user string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	store, err := f.load()
	if err != nil {
		return err
	}
	users, ok := store[service]
	if !ok {
		return ErrNotFound
	}
	if _, ok := users[user]; !ok {
		return ErrNotFound
	}
	delete(users, user)
	if len(users) == 0 {
		delete(store, service)
	}
	return f.save(store)
}

// hasContent reports whether the backing file exists and is non-empty.
func (f *fileBackend) hasContent() bool {
	info, err := os.Stat(f.path)
	if err != nil {
		return false
	}
	return info.Size() > 0
}
