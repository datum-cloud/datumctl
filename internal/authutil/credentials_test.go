package authutil

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	customerrors "go.datum.net/datumctl/internal/errors"
	"go.datum.net/datumctl/internal/keyring"
	"golang.org/x/oauth2"
)

const testUserKey = "maya@datum.net@auth.datum.net"

// seedInteractiveCreds stores interactive credentials for testUserKey and
// returns them. expiry in the past forces the next Token() call to refresh.
func seedInteractiveCreds(t *testing.T, accessToken, refreshToken, tokenURL string, expiry time.Time) *StoredCredentials {
	t.Helper()
	creds := &StoredCredentials{
		Hostname:         "auth.datum.net",
		APIHostname:      "api.datum.net",
		ClientID:         "client-id",
		EndpointTokenURL: tokenURL,
		UserEmail:        "maya@datum.net",
		Token: &oauth2.Token{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			Expiry:       expiry,
		},
	}
	blob, err := json.Marshal(creds)
	if err != nil {
		t.Fatalf("marshal creds: %v", err)
	}
	if err := keyring.Set(ServiceName, testUserKey, string(blob)); err != nil {
		t.Fatalf("seed keyring: %v", err)
	}
	return creds
}

func TestPersistingTokenSource_ValidTokenFastPath(t *testing.T) {
	mockKeyring(t)
	seedInteractiveCreds(t, "fresh-token", "refresh-token", "http://127.0.0.1:1/token", time.Now().Add(time.Hour))

	source, err := GetTokenSourceForUser(context.Background(), testUserKey)
	if err != nil {
		t.Fatalf("GetTokenSourceForUser: %v", err)
	}
	token, err := source.Token()
	if err != nil {
		t.Fatalf("Token: %v", err)
	}
	if token.AccessToken != "fresh-token" {
		t.Errorf("access token = %q, want the stored token with no refresh", token.AccessToken)
	}
}

// TestPersistingTokenSource_LogoutMidRunFailsWithoutResurrection pins the
// proxy's mid-run logout contract: once the keyring entry is deleted, the
// next refresh fails with a UserError pointing at 'datumctl login', and the
// deleted entry is NOT silently re-created from in-memory state.
func TestPersistingTokenSource_LogoutMidRunFailsWithoutResurrection(t *testing.T) {
	mockKeyring(t)
	seedInteractiveCreds(t, "old-token", "refresh-token", "http://127.0.0.1:1/token", time.Now().Add(-time.Minute))

	source, err := GetTokenSourceForUser(context.Background(), testUserKey)
	if err != nil {
		t.Fatalf("GetTokenSourceForUser: %v", err)
	}

	// The user logs out: the keyring entry disappears.
	if err := keyring.Delete(ServiceName, testUserKey); err != nil {
		t.Fatalf("delete creds: %v", err)
	}

	_, err = source.Token()
	if err == nil {
		t.Fatal("Token after logout: want an error, got none")
	}
	userErr, ok := customerrors.IsUserError(err)
	if !ok {
		t.Fatalf("error = %v (%T), want a UserError", err, err)
	}
	if !strings.Contains(userErr.Hint, "datumctl login") {
		t.Errorf("hint = %q, want it to point at 'datumctl login'", userErr.Hint)
	}
	if _, getErr := keyring.Get(ServiceName, testUserKey); !errors.Is(getErr, keyring.ErrNotFound) {
		t.Errorf("keyring entry after failed refresh: err = %v, want ErrNotFound (the logged-out session must not be resurrected)", getErr)
	}
}

// TestPersistingTokenSource_RecoversAfterReLogin pins the recovery half of
// the same contract: when the user logs back in to the same session, an
// already-running token source adopts the new stored credentials on its next
// refresh — no restart required.
func TestPersistingTokenSource_RecoversAfterReLogin(t *testing.T) {
	mockKeyring(t)
	seedInteractiveCreds(t, "stale-token", "dead-refresh-token", "http://127.0.0.1:1/token", time.Now().Add(-time.Minute))

	source, err := GetTokenSourceForUser(context.Background(), testUserKey)
	if err != nil {
		t.Fatalf("GetTokenSourceForUser: %v", err)
	}

	// The user re-logs in: the keyring entry is replaced with fresh creds.
	seedInteractiveCreds(t, "relogin-token", "new-refresh-token", "http://127.0.0.1:1/token", time.Now().Add(time.Hour))

	token, err := source.Token()
	if err != nil {
		t.Fatalf("Token after re-login: %v", err)
	}
	if token.AccessToken != "relogin-token" {
		t.Errorf("access token = %q, want the re-login token adopted from the keyring", token.AccessToken)
	}
}

// TestPersistingTokenSource_RefreshPersistsToKeyring exercises a real refresh
// against a fake token endpoint and asserts the refreshed token is written
// back to the keyring.
func TestPersistingTokenSource_RefreshPersistsToKeyring(t *testing.T) {
	mockKeyring(t)

	tokenEndpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"access_token":"refreshed-token","token_type":"Bearer","refresh_token":"refresh-token-2","expires_in":3600}`)
	}))
	defer tokenEndpoint.Close()

	seedInteractiveCreds(t, "expired-token", "refresh-token", tokenEndpoint.URL, time.Now().Add(-time.Minute))

	source, err := GetTokenSourceForUser(context.Background(), testUserKey)
	if err != nil {
		t.Fatalf("GetTokenSourceForUser: %v", err)
	}
	token, err := source.Token()
	if err != nil {
		t.Fatalf("Token: %v", err)
	}
	if token.AccessToken != "refreshed-token" {
		t.Errorf("access token = %q, want %q", token.AccessToken, "refreshed-token")
	}

	persisted, err := GetStoredCredentials(testUserKey)
	if err != nil {
		t.Fatalf("GetStoredCredentials after refresh: %v", err)
	}
	if persisted.Token.AccessToken != "refreshed-token" {
		t.Errorf("persisted access token = %q, want the refreshed token", persisted.Token.AccessToken)
	}
}

// TestPersistingTokenSource_InvalidGrantIsUserError pins the flagship failure
// message: a dead refresh token surfaces a UserError whose hint names the
// real login command ('datumctl login', not the nonexistent 'auth login').
func TestPersistingTokenSource_InvalidGrantIsUserError(t *testing.T) {
	mockKeyring(t)

	tokenEndpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error":"invalid_grant","error_description":"refresh token revoked"}`)
	}))
	defer tokenEndpoint.Close()

	seedInteractiveCreds(t, "expired-token", "revoked-refresh-token", tokenEndpoint.URL, time.Now().Add(-time.Minute))

	source, err := GetTokenSourceForUser(context.Background(), testUserKey)
	if err != nil {
		t.Fatalf("GetTokenSourceForUser: %v", err)
	}
	_, err = source.Token()
	if err == nil {
		t.Fatal("Token with a revoked refresh token: want an error")
	}
	userErr, ok := customerrors.IsUserError(err)
	if !ok {
		t.Fatalf("error = %v (%T), want a UserError", err, err)
	}
	if !strings.Contains(userErr.Hint, "`datumctl login`") {
		t.Errorf("hint = %q, want it to reference `datumctl login`", userErr.Hint)
	}
	if strings.Contains(userErr.Hint, "auth login") {
		t.Errorf("hint = %q references the nonexistent 'datumctl auth login'", userErr.Hint)
	}
}
