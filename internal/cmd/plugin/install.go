package plugin

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
	componentversion "k8s.io/component-base/version"

	"go.datum.net/datumctl/internal/client"
	customerrors "go.datum.net/datumctl/internal/errors"
	"go.datum.net/datumctl/internal/plugindispatch"
	"go.datum.net/datumctl/internal/pluginstore"
)

func installCmd(factory *client.DatumCloudFactory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install [name | owner/repo[@version]]",
		Short: "Install a datumctl plugin",
		Long: `Install a datumctl plugin from the curated plugin index or a GitHub Release.

With no arguments, restores all plugins recorded in plugins.json to their
recorded versions. Use this to reproduce a plugin set on a new machine.

With a plugin name argument, installs from the curated index:
  - name             installs the latest indexed version

With an owner/repo argument, installs directly from a GitHub Release:
  - owner/repo         installs the latest release
  - owner/repo@v1.2.0  installs a specific version

The plugin binary is written to the managed plugins directory
(~/.datumctl/plugins/ by default).`,
		Example: `  # Install the dns plugin from the curated index
  datumctl plugin install dns

  # Install directly from a GitHub Release
  datumctl plugin install datum-cloud/datumctl-dns

  # Install a specific version from GitHub
  datumctl plugin install datum-cloud/datumctl-dns@v1.2.0

  # Restore all plugins from plugins.json
  datumctl plugin install`,
		RunE: func(cmd *cobra.Command, args []string) error {
			pluginsDir, err := resolvePluginsDir(cmd)
			if err != nil {
				return err
			}

			currentVersion := componentversion.Get().GitVersion

			if len(args) == 0 {
				return installAllFromManifest(cmd, pluginsDir, currentVersion)
			}

			arg := args[0]

			// Third-party GitHub release path.
			if strings.Contains(arg, "/") {
				owner, repo, version, parseErr := parseSource(arg)
				if parseErr != nil {
					return customerrors.NewUserError(parseErr.Error())
				}
				entry, pluginName, binaryPath, installErr := installPluginFromGitHub(cmd.Context(), pluginsDir, owner, repo, version, currentVersion)
				if installErr != nil {
					return customerrors.NewUserError(fmt.Sprintf("install plugin %s/%s: %v", owner, repo, installErr))
				}
				return saveAndReport(cmd, pluginsDir, pluginName, entry, binaryPath)
			}

			// Curated index path.
			idx, idxErr := loadOrRefreshIndex(cmd)
			if idxErr != nil {
				return customerrors.NewUserError("could not fetch plugin index: " + idxErr.Error())
			}
			entry, pluginName, binaryPath, installErr := installPlugin(cmd.Context(), pluginsDir, arg, "", currentVersion, idx)
			if installErr != nil {
				return customerrors.NewUserError(fmt.Sprintf("install plugin %s: %v", arg, installErr))
			}
			return saveAndReport(cmd, pluginsDir, pluginName, entry, binaryPath)
		},
	}
	cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		idx, _ := pluginstore.LoadIndex()
		var names []string
		if idx != nil {
			for _, p := range idx.Plugins {
				names = append(names, p.Name+"\t"+p.Spec.ShortDescription)
			}
		}
		return names, cobra.ShellCompDirectiveNoFileComp
	}
	return cmd
}

// saveAndReport upserts plugins.json and prints the install confirmation.
// binaryPath is the path of the binary that was just written; if Save fails,
// saveAndReport attempts to remove the orphaned binary (L1).
func saveAndReport(cmd *cobra.Command, pluginsDir, pluginName string, entry *pluginstore.InstalledPlugin, binaryPath string) error {
	manifest, err := pluginstore.Load(pluginsDir)
	if err != nil {
		return fmt.Errorf("load plugins manifest: %w", err)
	}
	if manifest.Plugins == nil {
		manifest.Plugins = make(map[string]*pluginstore.InstalledPlugin)
	}
	manifest.Plugins[pluginName] = entry
	if saveErr := pluginstore.Save(pluginsDir, manifest); saveErr != nil {
		// L1: binary is on disk but unrecorded — attempt cleanup.
		if binaryPath != "" {
			if removeErr := os.Remove(binaryPath); removeErr != nil && !os.IsNotExist(removeErr) {
				return fmt.Errorf("save plugins manifest: %w; also failed to remove orphaned binary %s: %v", saveErr, binaryPath, removeErr)
			}
		}
		return fmt.Errorf("save plugins manifest: %w", saveErr)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Installed %s %s\n", pluginName, entry.Version)
	return nil
}

// loadOrRefreshIndex loads the cached index; if stale, refreshes it.
// Returns (nil, err) only if both load and refresh fail without a cache.
func loadOrRefreshIndex(cmd *cobra.Command) (*pluginstore.CachedIndex, error) {
	idx, _ := pluginstore.LoadIndex()
	if pluginstore.IsStale(idx) {
		fresh, refreshErr := pluginstore.RefreshIndex(cmd.Context())
		if refreshErr != nil {
			if fresh == nil {
				return nil, refreshErr
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "warning: index refresh failed (%v), using cached results\n", refreshErr)
			return fresh, nil
		}
		return fresh, nil
	}
	return idx, nil
}

// installAllFromManifest restores all plugins recorded in plugins.json.
func installAllFromManifest(cmd *cobra.Command, pluginsDir, currentVersion string) error {
	manifest, err := pluginstore.Load(pluginsDir)
	if err != nil {
		return fmt.Errorf("load plugins manifest: %w", err)
	}

	if len(manifest.Plugins) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No plugins recorded in plugins.json.")
		return nil
	}

	// Load/refresh the index up front for index-based entries.
	idx, _ := pluginstore.LoadIndex()
	if pluginstore.IsStale(idx) {
		fresh, refreshErr := pluginstore.RefreshIndex(cmd.Context())
		if refreshErr != nil {
			if fresh == nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: index refresh failed (%v); index-based plugins may fail\n", refreshErr)
			} else {
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: index refresh failed (%v), using cached results\n", refreshErr)
				idx = fresh
			}
		} else {
			idx = fresh
		}
	}

	var errs []string
	for name, entry := range manifest.Plugins {
		// Third-party source (owner/repo format).
		if strings.Contains(entry.Source, "/") {
			owner, repo, _, parseErr := parseSource(entry.Source)
			if parseErr != nil {
				errs = append(errs, fmt.Sprintf("%s: invalid source %q: %v", name, entry.Source, parseErr))
				continue
			}
			newEntry, _, _, installErr := installPluginFromGitHub(cmd.Context(), pluginsDir, owner, repo, entry.Version, currentVersion)
			if installErr != nil {
				errs = append(errs, fmt.Sprintf("%s: %v", name, installErr))
				continue
			}
			manifest.Plugins[name] = newEntry
			fmt.Fprintf(cmd.OutOrStdout(), "Installed %s %s\n", name, newEntry.Version)
			continue
		}

		// Curated index source (short name).
		newEntry, _, _, installErr := installPlugin(cmd.Context(), pluginsDir, name, entry.Version, currentVersion, idx)
		if installErr != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", name, installErr))
			continue
		}
		manifest.Plugins[name] = newEntry
		fmt.Fprintf(cmd.OutOrStdout(), "Installed %s %s\n", name, newEntry.Version)
	}

	if saveErr := pluginstore.Save(pluginsDir, manifest); saveErr != nil {
		errs = append(errs, fmt.Sprintf("save plugins manifest: %v", saveErr))
	}

	if len(errs) > 0 {
		return customerrors.NewUserError("some plugins failed to install:\n  " + strings.Join(errs, "\n  "))
	}
	return nil
}

// installPlugin installs a named plugin from the curated index.
// Plugin.Name from the index is used as the binary name component; it is
// validated against rePluginName (C2 — index-install path).
// Returns (entry, pluginName, writtenBinaryPath, error). writtenBinaryPath is
// non-empty when the binary was successfully written; callers should remove it
// if the subsequent manifest Save fails (L1).
func installPlugin(ctx context.Context, pluginsDir, pluginName, version, currentVersion string, idx *pluginstore.CachedIndex) (*pluginstore.InstalledPlugin, string, string, error) {
	if idx == nil {
		return nil, pluginName, "", fmt.Errorf("plugin index is not available; run 'datumctl plugin search' to fetch it")
	}

	plugin := pluginstore.FindInIndex(idx, pluginName)
	if plugin == nil {
		return nil, pluginName, "", fmt.Errorf("plugin %q not found in index; run 'datumctl plugin search' to list available plugins", pluginName)
	}

	// Validate the plugin name from the index before using it in a file path (C2).
	if !rePluginName.MatchString(plugin.Name) {
		return nil, pluginName, "", fmt.Errorf("index plugin name %q is invalid: must start with a lowercase letter or digit and contain only lowercase letters, digits, and hyphens", plugin.Name)
	}

	// Determine target version.
	if version == "" {
		version = plugin.Spec.Version
	}

	// Select the matching platform.
	platform, err := pluginstore.GetMatchingPlatform(plugin, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return nil, pluginName, "", fmt.Errorf("select platform: %w", err)
	}

	// Download and verify the archive.
	archiveBytes, err := downloadAndVerifyURI(ctx, platform.URI, platform.SHA256)
	if err != nil {
		return nil, pluginName, "", fmt.Errorf("download plugin: %w", err)
	}

	// Extract the binary.
	binaryName := "datumctl-" + pluginName
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}

	binaryBytes, err := extractFromArchive(archiveBytes, platform, pluginName, platform.URI)
	if err != nil {
		return nil, pluginName, "", fmt.Errorf("extract binary: %w", err)
	}

	// Read and check the manifest before writing.
	pluginManifest, err := readPluginManifestFromBytes(binaryBytes)
	if err != nil {
		return nil, pluginName, "", fmt.Errorf("read plugin manifest: %w", err)
	}
	if pluginManifest != nil {
		if _, err := checkCompatibility(pluginManifest, currentVersion, plugindispatch.PluginAPIVersion); err != nil {
			return nil, pluginName, "", err
		}
	}

	// Write the binary to disk.
	writtenPath, err := writeBinary(pluginsDir, binaryName, binaryBytes)
	if err != nil {
		return nil, pluginName, "", err
	}

	// Compute SHA256 of the extracted binary (not the archive) so that
	// verifyManagedPluginIntegrity can compare the on-disk binary against the
	// recorded hash. The archive SHA256 was already verified against the index
	// above; only the binary hash is stored for runtime integrity checks.
	binarySHA256 := sha256HexOf(binaryBytes)

	entry := &pluginstore.InstalledPlugin{
		Source:      pluginName,
		Version:     version,
		SHA256:      binarySHA256,
		InstalledAt: time.Now().UTC(),
		Manifest:    pluginManifest,
	}
	return entry, pluginName, writtenPath, nil
}

// extractFromArchive extracts the plugin binary from the archive bytes.
// If platform.Files is non-empty, the first matching FileOperation is used.
// Otherwise the binary is auto-detected by matching datumctl-<pluginName>[.exe].
func extractFromArchive(archiveBytes []byte, platform *pluginstore.Platform, pluginName, archiveURI string) ([]byte, error) {
	isTarGz := strings.HasSuffix(archiveURI, ".tar.gz") || strings.HasSuffix(archiveURI, ".tgz")
	isZip := strings.HasSuffix(archiveURI, ".zip")

	if len(platform.Files) > 0 {
		// Use the first FileOperation that specifies the binary.
		op := platform.Files[0]
		if isTarGz {
			return extractBinaryFromTarGz(newBytesReader(archiveBytes), op.From)
		} else if isZip {
			return extractBinaryFromZip(archiveBytes, op.From)
		}
		return nil, fmt.Errorf("unsupported archive format for URI: %s", archiveURI)
	}

	// Auto-detect: find the file named datumctl-<pluginName>[.exe].
	binName := "datumctl-" + pluginName
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}

	if isTarGz {
		return extractBinaryFromTarGz(newBytesReader(archiveBytes), binName)
	} else if isZip {
		return extractBinaryFromZip(archiveBytes, binName)
	}
	return nil, fmt.Errorf("unsupported archive format for URI: %s", archiveURI)
}

// installPluginFromGitHub is the third-party GitHub Release install path.
// It mirrors the original installPlugin logic for owner/repo[@version] sources.
// Returns (entry, pluginName, writtenBinaryPath, error). writtenBinaryPath is
// non-empty when the binary was successfully written; callers should remove it
// if the subsequent manifest Save fails (L1).
func installPluginFromGitHub(ctx context.Context, pluginsDir, owner, repo, version, currentVersion string) (*pluginstore.InstalledPlugin, string, string, error) {
	pluginName, err := pluginNameFromRepo(repo)
	if err != nil {
		return nil, "", "", err
	}

	if version == "" {
		version, err = fetchLatestTag(ctx, owner, repo)
		if err != nil {
			return nil, pluginName, "", fmt.Errorf("resolve latest version for %s/%s: %w", owner, repo, err)
		}
	}

	assetName, err := pluginAssetName("datumctl-"+pluginName, version)
	if err != nil {
		return nil, pluginName, "", err
	}

	checksums, err := fetchChecksums(ctx, owner, repo, version)
	if err != nil {
		return nil, pluginName, "", err
	}

	binaryBytes, _, err := downloadAndVerify(ctx, owner, repo, version, assetName, checksums)
	if err != nil {
		return nil, pluginName, "", err
	}

	pluginManifest, err := readPluginManifestFromBytes(binaryBytes)
	if err != nil {
		return nil, pluginName, "", fmt.Errorf("read plugin manifest: %w", err)
	}
	if pluginManifest != nil {
		if _, err := checkCompatibility(pluginManifest, currentVersion, plugindispatch.PluginAPIVersion); err != nil {
			return nil, pluginName, "", err
		}
	}

	binaryName := "datumctl-" + pluginName
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	writtenPath, err := writeBinary(pluginsDir, binaryName, binaryBytes)
	if err != nil {
		return nil, pluginName, "", err
	}

	// Compute SHA256 of the extracted binary (not the archive) so that
	// verifyManagedPluginIntegrity can compare the on-disk binary against the
	// recorded hash. archiveSHA256 was already verified against checksums.txt
	// above; only the binary hash is stored for runtime integrity checks.
	binarySHA256 := sha256HexOf(binaryBytes)

	entry := &pluginstore.InstalledPlugin{
		Source:      "github.com/" + owner + "/" + repo,
		Version:     version,
		SHA256:      binarySHA256,
		InstalledAt: time.Now().UTC(),
		Manifest:    pluginManifest,
	}
	return entry, pluginName, writtenPath, nil
}
