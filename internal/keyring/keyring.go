// Package keyring is a simple wrapper that adds timeouts to the zalando/go-keyring package.
// Taken from: https://github.com/cli/cli/blob/6c5145166003ac6fb952c5c591a6f3bdeea10465/internal/keyring/keyring.go
//
// In addition to the upstream behavior, this package transparently falls back
// to an on-disk JSON file when the system keyring is not available, mirroring
// what tools like the GitHub and Docker CLIs do. The fallback engages silently;
// callers that want to surface the change in storage (for example, the login
// command) can invoke WarnIfFallbackActive once the operation is complete.
package keyring

import (
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	keyring "github.com/zalando/go-keyring"
)

var ErrNotFound = keyring.ErrNotFound

type TimeoutError struct {
	message string
}

func (e *TimeoutError) Error() string {
	return e.message
}

var (
	fallbackMu     sync.Mutex
	fileFallback   *fileBackend
	lastKeyringErr error
	fallbackLogged bool
)

// activeFile returns the file backend if fallback has already been triggered
// in this process, or if a credentials file from a prior fallback already
// exists on disk. Returns nil when the keyring should still be tried.
func activeFile() *fileBackend {
	fallbackMu.Lock()
	defer fallbackMu.Unlock()

	if fileFallback != nil {
		return fileFallback
	}
	fb, err := newFileBackend()
	if err != nil {
		return nil
	}
	if fb.hasContent() {
		fileFallback = fb
		return fileFallback
	}
	return nil
}

// switchToFile activates the file backend after a keyring failure, recording
// the underlying error so callers can surface it later via WarnIfFallbackActive.
func switchToFile(keyringErr error) (*fileBackend, error) {
	fallbackMu.Lock()
	defer fallbackMu.Unlock()

	if fileFallback == nil {
		fb, err := newFileBackend()
		if err != nil {
			return nil, err
		}
		fileFallback = fb
	}
	lastKeyringErr = keyringErr
	return fileFallback, nil
}

// WarnIfFallbackActive emits a one-time warning to w when the credential
// store has fallen back to insecure on-disk storage. It is intended to be
// called from interactive command flows (such as login) so that storage
// changes surface to the user without spamming the warning on every command.
// No-op if fallback is not engaged or the warning has already been printed.
func WarnIfFallbackActive(w io.Writer) {
	fallbackMu.Lock()
	defer fallbackMu.Unlock()

	if fileFallback == nil || fallbackLogged {
		return
	}
	if lastKeyringErr != nil {
		fmt.Fprintf(w, "warning: system keyring unavailable (%v); falling back to insecure file storage at %s\n", lastKeyringErr, fileFallback.path)
	} else {
		fmt.Fprintf(w, "warning: using insecure file-based credential storage at %s\n", fileFallback.path)
	}
	fallbackLogged = true
}

// Set secret in keyring for user.
func Set(service, user, secret string) error {
	if fb := activeFile(); fb != nil {
		return fb.Set(service, user, secret)
	}

	ch := make(chan error, 1)
	go func() {
		defer close(ch)
		ch <- keyring.Set(service, user, secret)
	}()
	var err error
	select {
	case err = <-ch:
	case <-time.After(3 * time.Second):
		err = &TimeoutError{"timeout while trying to set secret in keyring"}
	}
	if err == nil {
		return nil
	}
	fb, ferr := switchToFile(err)
	if ferr != nil {
		return err
	}
	return fb.Set(service, user, secret)
}

// Get secret from keyring given service and user name.
func Get(service, user string) (string, error) {
	if fb := activeFile(); fb != nil {
		return fb.Get(service, user)
	}

	ch := make(chan struct {
		val string
		err error
	}, 1)
	go func() {
		defer close(ch)
		val, err := keyring.Get(service, user)
		ch <- struct {
			val string
			err error
		}{val, err}
	}()
	var res struct {
		val string
		err error
	}
	select {
	case res = <-ch:
	case <-time.After(3 * time.Second):
		res.err = &TimeoutError{"timeout while trying to get secret from keyring"}
	}
	if res.err == nil {
		return res.val, nil
	}
	if errors.Is(res.err, keyring.ErrNotFound) {
		return "", ErrNotFound
	}
	fb, ferr := switchToFile(res.err)
	if ferr != nil {
		return "", res.err
	}
	return fb.Get(service, user)
}

// Delete secret from keyring.
func Delete(service, user string) error {
	if fb := activeFile(); fb != nil {
		return fb.Delete(service, user)
	}

	ch := make(chan error, 1)
	go func() {
		defer close(ch)
		ch <- keyring.Delete(service, user)
	}()
	var err error
	select {
	case err = <-ch:
	case <-time.After(3 * time.Second):
		err = &TimeoutError{"timeout while trying to delete secret from keyring"}
	}
	if err == nil {
		return nil
	}
	if errors.Is(err, keyring.ErrNotFound) {
		return ErrNotFound
	}
	fb, ferr := switchToFile(err)
	if ferr != nil {
		return err
	}
	return fb.Delete(service, user)
}

func MockInit() {
	resetFallback()
	keyring.MockInit()
}

func MockInitWithError(err error) {
	resetFallback()
	keyring.MockInitWithError(err)
}

// resetFallback clears any cached fallback state. Used by mock initializers so
// tests start from a clean slate.
func resetFallback() {
	fallbackMu.Lock()
	defer fallbackMu.Unlock()
	fileFallback = nil
	lastKeyringErr = nil
	fallbackLogged = false
}
