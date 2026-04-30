package data

import "k8s.io/client-go/dynamic"

// newKubeResourceClientWithDynamic constructs a KubeResourceClient backed by the
// given dynamic client, bypassing the factory. For unit tests only.
func newKubeResourceClientWithDynamic(dc dynamic.Interface) *KubeResourceClient {
	return &KubeResourceClient{dc: dc}
}
