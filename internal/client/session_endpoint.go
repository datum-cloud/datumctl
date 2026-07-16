package client

import (
	"encoding/base64"
	"fmt"

	"go.datum.net/datumctl/internal/authutil"
	"go.datum.net/datumctl/internal/datumconfig"
	customerrors "go.datum.net/datumctl/internal/errors"
)

// EndpointTLS carries the TLS settings a session's endpoint declares:
// the SNI/verification server name, whether verification is skipped, and the
// decoded certificate authority bundle.
type EndpointTLS struct {
	ServerName            string
	InsecureSkipTLSVerify bool
	CAData                []byte
}

// SessionEndpoint is the resolved connection identity for one datumctl
// session: the keyring user key that owns its credentials, the base API
// server URL (scheme included, no trailing slash), and the endpoint's TLS
// settings.
type SessionEndpoint struct {
	UserKey    string
	BaseServer string
	TLS        EndpointTLS
}

// ResolveSessionEndpoint picks a session from cfg and resolves its endpoint,
// using the same resolution order CustomConfigFlags.ToRESTConfig performs:
// the named session when sessionName is given (error if unknown), else the
// active session, else the keyring active user for pre-session setups; the
// base server from the session endpoint, falling back to the keyring API
// hostname; TLS from the session endpoint. The returned session is nil on
// the keyring-fallback path.
func ResolveSessionEndpoint(cfg *datumconfig.ConfigV1Beta1, sessionName string) (*datumconfig.Session, *SessionEndpoint, error) {
	var session *datumconfig.Session
	if sessionName != "" {
		session = cfg.SessionByName(sessionName)
		if session == nil {
			return nil, nil, customerrors.NewUserErrorWithHint(
				fmt.Sprintf("No session named %q.", sessionName),
				"Run 'datumctl auth list' to see the available session names.",
			)
		}
	} else {
		session = cfg.ActiveSessionEntry()
	}

	userKey, err := sessionUserKey(session)
	if err != nil {
		return nil, nil, err
	}
	baseServer, err := sessionBaseServer(userKey, session)
	if err != nil {
		return nil, nil, err
	}
	tlsSettings, err := sessionEndpointTLS(session)
	if err != nil {
		return nil, nil, err
	}

	return session, &SessionEndpoint{
		UserKey:    userKey,
		BaseServer: baseServer,
		TLS:        tlsSettings,
	}, nil
}

// sessionUserKey returns the keyring user key for a session, falling back to
// the keyring active user (which bootstraps pre-session setups) when the
// session is nil or carries no key.
func sessionUserKey(session *datumconfig.Session) (string, error) {
	if session != nil && session.UserKey != "" {
		return session.UserKey, nil
	}
	return authutil.GetUserKey()
}

// sessionBaseServer returns the base API server URL for a session, falling
// back to the API hostname recorded in the user's stored credentials.
func sessionBaseServer(userKey string, session *datumconfig.Session) (string, error) {
	if session != nil && session.Endpoint.Server != "" {
		return datumconfig.CleanBaseServer(datumconfig.EnsureScheme(session.Endpoint.Server)), nil
	}
	apiHostname, err := authutil.GetAPIHostnameForUser(userKey)
	if err != nil {
		return "", err
	}
	return datumconfig.CleanBaseServer(datumconfig.EnsureScheme(apiHostname)), nil
}

// sessionEndpointTLS extracts the endpoint TLS settings from a session,
// decoding the base64 certificate authority data. A nil session has no TLS
// settings.
func sessionEndpointTLS(session *datumconfig.Session) (EndpointTLS, error) {
	if session == nil {
		return EndpointTLS{}, nil
	}
	ep := session.Endpoint
	settings := EndpointTLS{
		ServerName:            ep.TLSServerName,
		InsecureSkipTLSVerify: ep.InsecureSkipTLSVerify,
	}
	if ep.CertificateAuthorityData != "" {
		decoded, err := base64.StdEncoding.DecodeString(ep.CertificateAuthorityData)
		if err != nil {
			return EndpointTLS{}, customerrors.WrapUserErrorWithHint(
				fmt.Sprintf("Could not decode the certificate authority data stored for session %q.", session.Name),
				"Run 'datumctl login' again to refresh the session's endpoint settings.",
				err,
			)
		}
		settings.CAData = decoded
	}
	return settings, nil
}
