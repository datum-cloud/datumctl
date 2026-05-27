package pluginstore

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// GetMatchingPlatform returns the first Platform whose Selector matches
// {os: goos, arch: goarch}. A nil or empty Selector matches everything.
func GetMatchingPlatform(plugin *Plugin, goos, goarch string) (*Platform, error) {
	for i := range plugin.Spec.Platforms {
		p := &plugin.Spec.Platforms[i]
		if p.Selector == nil || (len(p.Selector.MatchLabels) == 0 && len(p.Selector.MatchExpressions) == 0) {
			return p, nil
		}
		sel, err := metav1.LabelSelectorAsSelector(p.Selector)
		if err != nil {
			return nil, fmt.Errorf("invalid selector for platform: %w", err)
		}
		if sel.Matches(labels.Set{"os": goos, "arch": goarch}) {
			return p, nil
		}
	}
	return nil, fmt.Errorf("no platform found matching os=%s arch=%s", goos, goarch)
}
