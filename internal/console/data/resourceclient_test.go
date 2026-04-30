package data

import (
	"context"
	"testing"

	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	clienttesting "k8s.io/client-go/testing"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestKubeResourceClient_DeleteResource_PropagationBackground verifies AC#14:
// DeleteResource sends metav1.DeleteOptions with an explicit Background propagation
// policy — not nil (server default) and not Foreground. A future change to
// resourceclient.go that drops or changes the policy will fail this test in CI.
func TestKubeResourceClient_DeleteResource_PropagationBackground(t *testing.T) {
	t.Parallel()

	scheme := k8sruntime.NewScheme()
	fakeClient := dynamicfake.NewSimpleDynamicClient(scheme)
	rc := newKubeResourceClientWithDynamic(fakeClient)

	rt := ResourceType{
		Group:      "",
		Version:    "v1",
		Name:       "pods",
		Kind:       "Pod",
		Namespaced: true,
	}

	// The fake records the Delete action before checking the object tracker,
	// so the action is captured regardless of whether the pod exists.
	_ = rc.DeleteResource(context.Background(), rt, "datumctl-test-pod", "default")

	for _, a := range fakeClient.Actions() {
		da, ok := a.(clienttesting.DeleteAction)
		if !ok {
			continue
		}
		opts := da.GetDeleteOptions()
		if opts.PropagationPolicy == nil {
			t.Fatal("AC#14: PropagationPolicy = nil, want explicit Background (nil means server default, which can drift)")
		}
		if *opts.PropagationPolicy != metav1.DeletePropagationBackground {
			t.Errorf("AC#14: PropagationPolicy = %v, want %v", *opts.PropagationPolicy, metav1.DeletePropagationBackground)
		}
		return
	}
	t.Fatal("AC#14: no DeleteAction recorded — DeleteResource did not call the dynamic client")
}
