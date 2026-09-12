package plugindispatch

import "testing"

func TestResolveLegacyAlias(t *testing.T) {
	cases := []struct {
		name       string
		wantPlugin string
		wantOK     bool
	}{
		{"ai", "assistant", true},
		{"assistant", "", false},
		{"compute", "", false},
		{"", "", false},
	}
	for _, tc := range cases {
		plugin, ok := ResolveLegacyAlias(tc.name)
		if ok != tc.wantOK || plugin != tc.wantPlugin {
			t.Errorf("ResolveLegacyAlias(%q) = (%q, %v), want (%q, %v)",
				tc.name, plugin, ok, tc.wantPlugin, tc.wantOK)
		}
	}
}
