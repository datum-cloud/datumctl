package plugin

import (
	"encoding/json"
	"fmt"
	"os"
)

// Manifest describes a plugin binary. Plugin binaries should call ServeManifest(m)
// at the top of main() to handle the --plugin-manifest protocol automatically.
type Manifest struct {
	Name               string `json:"name"`
	Version            string `json:"version"`
	Description        string `json:"description"`
	MinDatumctlVersion string `json:"min_datumctl_version,omitempty"`
	APIVersion         int    `json:"api_version"`
	MinAPIVersion      int    `json:"min_api_version,omitempty"`
}

// ServeManifest checks os.Args for --plugin-manifest. If found, it prints m as JSON
// to stdout and exits 0. This must be called before cobra.Execute() so the manifest
// is served even if cobra flag parsing would otherwise fail.
func ServeManifest(m Manifest) {
	for _, arg := range os.Args[1:] {
		if arg == "--plugin-manifest" {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			if err := enc.Encode(m); err != nil {
				fmt.Fprintf(os.Stderr, "plugin manifest encode error: %v\n", err)
				os.Exit(1)
			}
			os.Exit(0)
		}
	}
}
