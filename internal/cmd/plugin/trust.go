package plugin

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"k8s.io/kubectl/pkg/util/templates"

	customerrors "go.datum.net/datumctl/internal/errors"
	"go.datum.net/datumctl/internal/plugindispatch"
	"go.datum.net/datumctl/internal/pluginstore"
)

// hashFile computes the SHA256 hex digest of the file at path.
func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func trustCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trust <name>",
		Short: "Trust an unmanaged plugin so datumctl will run it",
		Long: templates.LongDesc(`
			Record that an unmanaged (PATH-installed) plugin is trusted so datumctl
			will run it.

			When datumctl finds a plugin binary on your PATH that it did not install,
			it blocks execution and shows an error. Use 'trust' to explicitly allow a
			specific unmanaged plugin to run.

			The resolved path and a fingerprint of the binary are recorded. If the
			binary changes after trust is granted, datumctl will refuse to run it until
			you re-run 'datumctl plugin trust <name>'.

			To revoke trust, use 'datumctl plugin untrust <name>'.`),
		Example: templates.Examples(`
			# Trust the dns plugin found on PATH
			datumctl plugin trust dns`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			pluginsDir, err := resolvePluginsDir(cmd)
			if err != nil {
				return err
			}

			// Resolve the binary path via the shared plugin resolver, which checks
			// the managed dir first (generic and legacy names) and then PATH for a
			// recognized plugin prefix (milo- or datumctl-).
			resolvedPath, _, findErr := plugindispatch.FindPlugin(name, pluginsDir)
			if findErr != nil {
				return customerrors.NewUserError(fmt.Sprintf("cannot find a 'milo-%s' or 'datumctl-%s' plugin in the managed directory or PATH", name, name))
			}

			// Resolve symlinks exactly once. The resolved path is used for both the
			// trust record and subsequent hash verification, eliminating TOCTOU from
			// symlink flips.
			if abs, absErr := filepath.EvalSymlinks(resolvedPath); absErr == nil {
				resolvedPath = abs
			}

			// Compute SHA256 of the binary at trust time. This is stored alongside
			// the path so that if the binary is replaced after trust is granted,
			// the hash mismatch will block execution.
			digest, err := hashFile(resolvedPath)
			if err != nil {
				return fmt.Errorf("hash plugin binary: %w", err)
			}

			manifest, err := pluginstore.Load(pluginsDir)
			if err != nil {
				return fmt.Errorf("load plugins manifest: %w", err)
			}
			if manifest.Trusted == nil {
				manifest.Trusted = make(map[string]*pluginstore.TrustedEntry)
			}
			manifest.Trusted[name] = &pluginstore.TrustedEntry{
				Path:      resolvedPath,
				SHA256:    digest,
				TrustedAt: time.Now().UTC(),
			}
			if err := pluginstore.Save(pluginsDir, manifest); err != nil {
				return fmt.Errorf("save plugins manifest: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Trusted %s (%s)\n", name, resolvedPath)
			fmt.Fprintf(cmd.OutOrStdout(), "To revoke: datumctl plugin untrust %s\n", name)
			return nil
		},
	}
	cmd.ValidArgsFunction = installedPluginNames
	return cmd
}

func untrustCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "untrust <name>",
		Short: "Remove a plugin from the trusted list",
		Long: templates.LongDesc(`
			Remove a previously trusted unmanaged plugin from the trusted list.

			After running this command, invoking the plugin will return an error until
			you explicitly trust it again with 'datumctl plugin trust <name>'.`),
		Example: templates.Examples(`
			# Revoke trust for the dns plugin
			datumctl plugin untrust dns`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			pluginsDir, err := resolvePluginsDir(cmd)
			if err != nil {
				return err
			}

			manifest, err := pluginstore.Load(pluginsDir)
			if err != nil {
				return fmt.Errorf("load plugins manifest: %w", err)
			}

			if manifest.Trusted == nil || manifest.Trusted[name] == nil {
				return customerrors.NewUserError(fmt.Sprintf("plugin %q is not in the trusted list", name))
			}

			delete(manifest.Trusted, name)
			if err := pluginstore.Save(pluginsDir, manifest); err != nil {
				return fmt.Errorf("save plugins manifest: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Revoked trust for %s\n", name)
			return nil
		},
	}
	return cmd
}
