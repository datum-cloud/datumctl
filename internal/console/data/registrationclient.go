package data

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"go.datum.net/datumctl/internal/client"
)

// ResourceRegistration is the TUI-facing projection of the quota service's
// ResourceRegistration CRD.
type ResourceRegistration struct {
	Group       string // derived from spec.resourceType prefix before last "/"
	Name        string // derived from spec.resourceType suffix after last "/"
	Description string // spec.description — "" means miss
}

// ResourceRegistrationClient is implemented by the platform-API-backed client.
type ResourceRegistrationClient interface {
	ListResourceRegistrations(ctx context.Context) ([]ResourceRegistration, error)
	InvalidateRegistrationCache()
}

// ResourceRegistrationsLoadedMsg carries the result of a ListResourceRegistrations call.
// On error, Registrations is nil; surfaces fall back silently.
type ResourceRegistrationsLoadedMsg struct {
	Registrations []ResourceRegistration
	Err           error
	Unauthorized  bool // true on 403 — used to mute logging; no user-facing difference
}

// ResolveDescription looks up (group, name) in registrations and returns
// Description when found and non-empty. Returns "" on miss, nil/empty slice,
// or empty Description. Case-sensitive; first match wins.
func ResolveDescription(registrations []ResourceRegistration, group, name string) string {
	for _, r := range registrations {
		if r.Group == group && r.Name == name && r.Description != "" {
			return r.Description
		}
	}
	return ""
}

// SplitResourceType splits a fully-qualified resource type string into (group, name).
// For strings without "/", group is "" and name is the whole string.
func SplitResourceType(rt string) (group, name string) {
	if idx := strings.LastIndex(rt, "/"); idx >= 0 {
		return rt[:idx], rt[idx+1:]
	}
	return "", rt
}

// KubeResourceRegistrationClient fetches ResourceRegistration objects from the
// platform API with a 5-minute in-memory TTL cache.
type KubeResourceRegistrationClient struct {
	factory   *client.DatumCloudFactory
	mu        sync.Mutex
	cached    []ResourceRegistration
	fetchedAt time.Time
	ttl       time.Duration
}

func NewKubeResourceRegistrationClient(factory *client.DatumCloudFactory) *KubeResourceRegistrationClient {
	return &KubeResourceRegistrationClient{factory: factory, ttl: 5 * time.Minute}
}

func (k *KubeResourceRegistrationClient) ListResourceRegistrations(ctx context.Context) ([]ResourceRegistration, error) {
	k.mu.Lock()
	cachedAt := k.fetchedAt
	cached := k.cached
	k.mu.Unlock()

	if !cachedAt.IsZero() && time.Since(cachedAt) < k.ttl {
		return cached, nil
	}

	regs, err := k.fetch(ctx)
	if err != nil {
		if isRegistrationUnauthorized(err) {
			// 403 — cache nil with TTL to prevent re-fetch storm.
			k.mu.Lock()
			k.cached = nil
			k.fetchedAt = time.Now()
			k.mu.Unlock()
		}
		// Non-403 errors: don't set fetchedAt so retry is allowed on next call.
		return nil, err
	}

	k.mu.Lock()
	k.cached = regs
	k.fetchedAt = time.Now()
	k.mu.Unlock()

	return regs, nil
}

func (k *KubeResourceRegistrationClient) InvalidateRegistrationCache() {
	k.mu.Lock()
	k.cached = nil
	k.fetchedAt = time.Time{}
	k.mu.Unlock()
}

func (k *KubeResourceRegistrationClient) fetch(ctx context.Context) ([]ResourceRegistration, error) {
	gvr, err := k.findRegistrationGVR()
	if err != nil {
		// CRD not installed — return empty, not an error.
		return nil, nil
	}

	dc, err := k.factory.DynamicClient()
	if err != nil {
		return nil, fmt.Errorf("dynamic client: %w", err)
	}

	list, err := dc.Resource(gvr).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	regs := make([]ResourceRegistration, 0, len(list.Items))
	for _, item := range list.Items {
		spec, _ := item.Object["spec"].(map[string]any)
		rt := stringField(spec, "resourceType")
		g, n := SplitResourceType(rt)
		regs = append(regs, ResourceRegistration{
			Group:       g,
			Name:        n,
			Description: stringField(spec, "description"),
		})
	}

	return regs, nil
}

func (k *KubeResourceRegistrationClient) findRegistrationGVR() (schema.GroupVersionResource, error) {
	dc, err := k.factory.ToDiscoveryClient()
	if err != nil {
		return schema.GroupVersionResource{}, fmt.Errorf("discovery client: %w", err)
	}
	lists, err := dc.ServerPreferredResources()
	if err != nil && lists == nil {
		return schema.GroupVersionResource{}, fmt.Errorf("server preferred resources: %w", err)
	}
	for _, list := range lists {
		gv, parseErr := schema.ParseGroupVersion(list.GroupVersion)
		if parseErr != nil {
			continue
		}
		if gv.Group != "quota.miloapis.com" {
			continue
		}
		for _, r := range list.APIResources {
			if r.Name == "resourceregistrations" {
				return schema.GroupVersionResource{
					Group:    gv.Group,
					Version:  gv.Version,
					Resource: "resourceregistrations",
				}, nil
			}
		}
	}
	return schema.GroupVersionResource{}, fmt.Errorf("resourceregistrations not found in discovery")
}

func isRegistrationUnauthorized(err error) bool {
	return k8serrors.IsForbidden(err) || k8serrors.IsUnauthorized(err)
}
