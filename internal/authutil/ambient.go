// Package authutil: ambient-token mode.
//
// Ambient-token mode lets a trusted host process (e.g. the cloud-portal
// embedded terminal) hand datumctl a pre-obtained bearer token and API
// endpoint via environment variables, bypassing the keyring-based OAuth
// flow entirely. In this mode:
//
//   - All credential lookups (GetActiveCredentials, GetTokenSource, etc.)
//     return a synthesized in-memory identity rather than reading the
//     OS keyring.
//   - Commands that mutate authentication or context state (login, logout,
//     auth switch, ctx use, auth update-kubeconfig) are rejected with
//     ErrAmbientReadOnly so the host can guarantee a pinned identity and
//     context for the lifetime of the process.
//
// The host is responsible for setting DATUM_PROJECT or DATUM_ORGANIZATION
// to pin the context; those env vars are already honored by
// internal/client/factory.go.
package authutil

import (
	"errors"
	"fmt"
	"os"
	"time"

	"go.datum.net/datumctl/internal/datumconfig"
	customerrors "go.datum.net/datumctl/internal/errors"
	"golang.org/x/oauth2"
)

// Environment variable names that control ambient-token mode.
const (
	// AmbientTokenEnv, when non-empty, activates ambient-token mode and
	// supplies the bearer access token used for all API calls.
	AmbientTokenEnv = "DATUMCTL_TOKEN"

	// AmbientAPIHostnameEnv supplies the API server hostname
	// (e.g. "api.datum.net") when ambient-token mode is active. Required
	// when DATUMCTL_TOKEN is set because no keyring entry is consulted.
	AmbientAPIHostnameEnv = "DATUM_API_HOSTNAME"

	// AmbientUserEmailEnv optionally populates the synthesized user email
	// so that "datumctl whoami" and similar commands show the right user.
	AmbientUserEmailEnv = "DATUMCTL_USER_EMAIL"

	// AmbientUserSubjectEnv optionally populates the synthesized user
	// subject (OIDC "sub" claim). Defaults to "ambient" when unset.
	AmbientUserSubjectEnv = "DATUMCTL_USER_SUBJECT"
)

// Synthesized identifiers used when ambient-token mode is active. These are
// deliberately constant so the various subsystems that key off user id / session
// name all agree within a single process.
const (
	AmbientUserKey     = "ambient@datumctl"
	AmbientSessionName = "ambient"
	ambientCredType    = "ambient"
)

// ErrAmbientReadOnly is returned by any command that mutates authentication
// or context state while ambient-token mode is active.
var ErrAmbientReadOnly = customerrors.NewUserErrorWithHint(
	"This command is disabled because datumctl was started with DATUMCTL_TOKEN set.",
	"Authentication and context are managed by the host environment (e.g. the cloud-portal embedded terminal). Run datumctl outside of that environment to change them.",
)

// HasAmbientToken reports whether DATUMCTL_TOKEN is non-empty, i.e. whether
// ambient-token mode is active for this process.
func HasAmbientToken() bool {
	return os.Getenv(AmbientTokenEnv) != ""
}

// GuardAmbientMutation returns ErrAmbientReadOnly when ambient-token mode is
// active. Commands that change auth or context state should call this at the
// top of their RunE to reject the invocation cleanly.
func GuardAmbientMutation() error {
	if HasAmbientToken() {
		return ErrAmbientReadOnly
	}
	return nil
}

// ambientToken returns the raw bearer token from DATUMCTL_TOKEN.
func ambientToken() (string, error) {
	tok := os.Getenv(AmbientTokenEnv)
	if tok == "" {
		return "", errors.New(AmbientTokenEnv + " is not set")
	}
	return tok, nil
}

// ambientAPIHostname returns the API hostname from DATUM_API_HOSTNAME.
func ambientAPIHostname() (string, error) {
	h := os.Getenv(AmbientAPIHostnameEnv)
	if h == "" {
		return "", fmt.Errorf("%s must be set when %s is set", AmbientAPIHostnameEnv, AmbientTokenEnv)
	}
	return h, nil
}

// ambientSubject returns the synthesized user subject, defaulting to "ambient".
func ambientSubject() string {
	if s := os.Getenv(AmbientUserSubjectEnv); s != "" {
		return s
	}
	return "ambient"
}

// ambientEmail returns the synthesized user email.
func ambientEmail() string {
	return os.Getenv(AmbientUserEmailEnv)
}

// ambientCredentials builds an in-memory StoredCredentials from the ambient
// environment. Used by the various Get*Credentials / Get*APIHostname
// short-circuits below.
func ambientCredentials() (*StoredCredentials, error) {
	tok, err := ambientToken()
	if err != nil {
		return nil, err
	}
	host, err := ambientAPIHostname()
	if err != nil {
		return nil, err
	}
	email := ambientEmail()
	return &StoredCredentials{
		APIHostname: host,
		// Hostname (auth server) is unused in ambient mode. Leave blank so any
		// accidental code path that tries to use it fails loudly rather than
		// hitting a wrong endpoint.
		Hostname:  "",
		UserEmail: email,
		UserName:  email,
		Subject:   ambientSubject(),
		Token: &oauth2.Token{
			AccessToken: tok,
			TokenType:   "Bearer",
			// Far-future expiry so the oauth2 stack never attempts a "refresh";
			// the host is expected to restart datumctl with a fresh token when
			// the real upstream token rotates.
			Expiry: time.Now().Add(24 * time.Hour),
		},
		CredentialType: ambientCredType,
	}, nil
}

// ambientSession returns a synthesized v1beta1 Session for ambient mode so
// that code paths expecting a Session still work.
func ambientSession() (*datumconfig.Session, error) {
	host, err := ambientAPIHostname()
	if err != nil {
		return nil, err
	}
	email := ambientEmail()
	return &datumconfig.Session{
		Name:      AmbientSessionName,
		UserKey:   AmbientUserKey,
		UserEmail: email,
		UserName:  email,
		Endpoint: datumconfig.Endpoint{
			Server: datumconfig.CleanBaseServer(datumconfig.EnsureScheme(host)),
		},
	}, nil
}

// ambientTokenSource is an oauth2.TokenSource backed by DATUMCTL_TOKEN. It
// re-reads the env on each call so a long-lived datumctl process picks up an
// updated token without having to restart.
type ambientTokenSource struct{}

func (ambientTokenSource) Token() (*oauth2.Token, error) {
	tok, err := ambientToken()
	if err != nil {
		return nil, err
	}
	return &oauth2.Token{
		AccessToken: tok,
		TokenType:   "Bearer",
		Expiry:      time.Now().Add(24 * time.Hour),
	}, nil
}
