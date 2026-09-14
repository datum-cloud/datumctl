package cmd

import (
	"testing"

	"go.datum.net/datumctl/internal/plugindispatch"
)

// The root command routes an unknown name to a plugin only when Cobra has not
// already claimed it. Re-adding a built-in for an aliased name would take that
// route away without any build failure, so pin it.
func TestLegacyAliasIsNotShadowedByBuiltIn(t *testing.T) {
	root := RootCmd()
	for _, name := range []string{"ai"} {
		if _, aliased := plugindispatch.ResolveLegacyAlias(name); !aliased {
			t.Fatalf("%q is expected to be a legacy alias", name)
		}
		if plugindispatch.IsBuiltIn(root, name) {
			t.Errorf("%q is both a built-in command and a legacy plugin alias; the alias never fires", name)
		}
	}
}
