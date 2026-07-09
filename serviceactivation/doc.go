// Package serviceactivation implements the client-side UX for enabling a Datum
// Cloud service via the service catalog's ServiceEntitlement API.
//
// It is a sibling to the deliberately dependency-free plugin package: importing
// serviceactivation pulls in the service-catalog API types and Kubernetes client
// machinery, so the plugin package's "environment variables and subprocesses
// only" charter is preserved by not importing this package from there.
//
// The package is service-agnostic: a caller supplies a Config naming the service
// (object name, canonical name, display noun, and the plugin-local access verb
// used in next-step copy) plus an injected client and IO streams. Everything
// else — the entitlement state model, prompt/TTY/exit-code conventions, wait
// mechanics, and the user-facing copy — lives here so consumers converge instead
// of forking the flow.
package serviceactivation
