package plugin

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	componentversion "k8s.io/component-base/version"
	"k8s.io/kubectl/pkg/util/templates"

	"go.datum.net/datumctl/internal/client"
	customerrors "go.datum.net/datumctl/internal/errors"
	"go.datum.net/datumctl/internal/pluginstore"
)

func upgradeCmd(_ *client.DatumCloudFactory) *cobra.Command {
	var all bool
	cmd := &cobra.Command{
		Use:   "upgrade [name]",
		Short: "Upgrade installed plugins to their latest versions",
		Long: templates.LongDesc(`
			Download and install the latest release of a managed plugin, or of every
			managed plugin with --all.

			Each plugin is upgraded from the catalog (or GitHub repository) it was
			originally installed from, using the same flow as 'datumctl plugin
			install': SHA256 verification, manifest check, and compatibility
			validation. A plugin that is already at its latest version is left
			untouched.

			With --all, a plugin that fails to upgrade does not stop the others; the
			failures are listed together once every plugin has been attempted.`),
		Example: templates.Examples(`
			# Upgrade the dns plugin to the latest version
			datumctl plugin upgrade dns

			# Upgrade every installed plugin
			datumctl plugin upgrade --all`),
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch {
			case all && len(args) > 0:
				return customerrors.NewUserError("--all upgrades every installed plugin and cannot be combined with a plugin name")
			case !all && len(args) == 0:
				return customerrors.NewUserError("specify a plugin name, or use --all to upgrade every installed plugin")
			}

			pluginsDir, err := resolvePluginsDir(cmd)
			if err != nil {
				return err
			}

			manifest, err := pluginstore.Load(pluginsDir)
			if err != nil {
				return fmt.Errorf("load plugins manifest: %w", err)
			}

			currentVersion := componentversion.Get().GitVersion

			if all {
				return upgradeAll(cmd, pluginsDir, manifest, currentVersion)
			}
			return upgradeOne(cmd, pluginsDir, manifest, args[0], currentVersion)
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "Upgrade every installed plugin")
	cmd.ValidArgsFunction = installedPluginNames
	return cmd
}

// upgradeOne upgrades a single named plugin and reports the outcome.
func upgradeOne(cmd *cobra.Command, pluginsDir string, manifest *pluginstore.Manifest, name, currentVersion string) error {
	entry, ok := manifest.Plugins[name]
	if !ok {
		return customerrors.NewUserError(fmt.Sprintf("plugin %q is not installed; run 'datumctl plugin install' first", name))
	}

	u, err := newUpgrader(cmd, pluginsDir, manifest, currentVersion)
	if err != nil {
		return err
	}

	from := entry.Version
	to, upgraded, err := u.upgrade(name, entry)
	if err != nil {
		var fe *pluginstore.IndexFetchError
		if errors.As(err, &fe) {
			return indexFetchUserError(err)
		}
		return customerrors.NewUserError(fmt.Sprintf("upgrade plugin %s: %v", name, err))
	}
	reportUpgrade(cmd, name, from, to, upgraded)
	return nil
}

// upgradeAll upgrades every plugin recorded in plugins.json. Plugins are
// processed in name order; a failure is recorded and the remaining plugins are
// still attempted. Each successful upgrade is saved immediately so progress is
// kept even if a later plugin fails.
func upgradeAll(cmd *cobra.Command, pluginsDir string, manifest *pluginstore.Manifest, currentVersion string) error {
	if len(manifest.Plugins) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No managed plugins installed.")
		return nil
	}

	u, err := newUpgrader(cmd, pluginsDir, manifest, currentVersion)
	if err != nil {
		return err
	}

	names := make([]string, 0, len(manifest.Plugins))
	for name := range manifest.Plugins {
		names = append(names, name)
	}
	sort.Strings(names)

	var errs []string
	for _, name := range names {
		entry := manifest.Plugins[name]
		from := entry.Version
		to, upgraded, err := u.upgrade(name, entry)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", name, err))
			continue
		}
		reportUpgrade(cmd, name, from, to, upgraded)
	}

	if len(errs) > 0 {
		return customerrors.NewUserError("some plugins failed to upgrade:\n  " + strings.Join(errs, "\n  "))
	}
	return nil
}

// reportUpgrade prints the one-line status for a plugin that was processed
// without error.
func reportUpgrade(cmd *cobra.Command, name, from, to string, upgraded bool) {
	if upgraded {
		fmt.Fprintf(cmd.OutOrStdout(), "Upgraded %s from %s to %s\n", name, from, to)
		return
	}
	fmt.Fprintf(cmd.OutOrStdout(), "%s is already up to date (%s)\n", name, to)
}

// upgrader carries the state shared by every per-plugin upgrade in one command
// invocation: the loaded manifest (saved after each successful upgrade) and a
// memoized catalog index loader so each catalog is fetched at most once.
type upgrader struct {
	cmd            *cobra.Command
	pluginsDir     string
	manifest       *pluginstore.Manifest
	currentVersion string
	getIndex       func(catalogName string) (*pluginstore.CachedIndex, error)
}

func newUpgrader(cmd *cobra.Command, pluginsDir string, manifest *pluginstore.Manifest, currentVersion string) (*upgrader, error) {
	reg, err := pluginstore.LoadRegistry(pluginsDir)
	if err != nil {
		return nil, fmt.Errorf("load catalog registry: %w", err)
	}
	warnDisabledCatalogs(cmd.ErrOrStderr(), reg)
	return &upgrader{
		cmd:            cmd,
		pluginsDir:     pluginsDir,
		manifest:       manifest,
		currentVersion: currentVersion,
		getIndex:       catalogIndexLoader(cmd, pluginsDir, reg),
	}, nil
}

// upgrade brings one installed plugin to the latest version offered by the
// source it was installed from and records the result in plugins.json. It
// returns the latest version and whether a new version was installed; when the
// plugin was already current nothing is downloaded or written.
func (u *upgrader) upgrade(name string, entry *pluginstore.InstalledPlugin) (latest string, upgraded bool, err error) {
	ctx := u.cmd.Context()

	var (
		newEntry   *pluginstore.InstalledPlugin
		binaryPath string
	)

	// Direct GitHub source (owner/repo format) — use the GitHub release flow.
	if entry.Catalog == "" && strings.Contains(entry.Source, "/") {
		owner, repo, _, parseErr := parseSource(entry.Source)
		if parseErr != nil {
			return "", false, fmt.Errorf("invalid source %q: %w", entry.Source, parseErr)
		}
		latest, err = fetchLatestTag(ctx, owner, repo)
		if err != nil {
			return "", false, fmt.Errorf("resolve latest version for %s/%s: %w", owner, repo, err)
		}
		if latest == entry.Version {
			return latest, false, nil
		}
		newEntry, _, binaryPath, err = installPluginFromGitHub(ctx, u.pluginsDir, owner, repo, latest, u.currentVersion)
		if err != nil {
			return "", false, err
		}
	} else {
		// Catalog source. An empty Catalog on a non-slash source is a legacy
		// curated install, which upgrades from the official catalog. A recorded
		// legacy "default" name is canonicalized to "datum".
		catalogName := pluginstore.CanonicalCatalogName(entry.Catalog)
		if catalogName == "" {
			catalogName = pluginstore.OfficialCatalogName
		}
		idx, idxErr := u.getIndex(catalogName)
		if idxErr != nil {
			return "", false, idxErr
		}
		plugin := pluginstore.FindInIndex(idx, name)
		if plugin == nil {
			return "", false, fmt.Errorf("plugin %q not found in catalog %q", name, catalogName)
		}
		latest = plugin.Spec.Version
		if latest == entry.Version {
			return latest, false, nil
		}
		newEntry, _, binaryPath, err = installPlugin(ctx, u.pluginsDir, name, "", u.currentVersion, idx)
		if err != nil {
			return "", false, err
		}
		newEntry.Catalog = catalogName
	}

	prev := u.manifest.Plugins[name]
	u.manifest.Plugins[name] = newEntry
	if saveErr := pluginstore.Save(u.pluginsDir, u.manifest); saveErr != nil {
		u.manifest.Plugins[name] = prev
		// L1: binary is on disk but unrecorded — attempt cleanup.
		if binaryPath != "" {
			if removeErr := os.Remove(binaryPath); removeErr != nil && !os.IsNotExist(removeErr) {
				return "", false, fmt.Errorf("save plugins manifest: %w; also failed to remove orphaned binary %s: %v", saveErr, binaryPath, removeErr)
			}
		}
		return "", false, fmt.Errorf("save plugins manifest: %w", saveErr)
	}
	return newEntry.Version, true, nil
}
