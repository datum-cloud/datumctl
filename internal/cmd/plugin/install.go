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
		Long: `Install a datumctl plugin from a plugin catalog or a GitHub Release.

With no arguments, restores all plugins recorded in plugins.json to their
recorded versions. Use this to reproduce a plugin set on a new machine.

With a plugin name argument, installs from a registered catalog:
  - name             resolves against the default catalog first, then any other
                     registered catalog; a name found in more than one catalog
                     prints the options instead of guessing
  - catalog/name     installs from a specific registered catalog

With an owner/repo argument, installs directly from a GitHub Release:
  - owner/repo         installs the latest release
  - owner/repo@v1.2.0  installs a specific version

The plugin binary is written to the managed plugins directory
(~/.datumctl/plugins/ by default).`,
		Example: `  # Install the dns plugin from the default catalog
  datumctl plugin install dns

  # Install from a specific catalog
  datumctl plugin install acme/deploy

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

			reg, err := pluginstore.LoadRegistry(pluginsDir)
			if err != nil {
				return fmt.Errorf("load catalog registry: %w", err)
			}
			return installArg(cmd, pluginsDir, reg, args[0], currentVersion)
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

// installArg dispatches a single install argument to the right path based on
// catalog-aware addressing:
//   - "owner/repo@version"  -> direct GitHub release (the '@' is unambiguous)
//   - "catalog/name"        -> catalog-qualified, when the prefix is a
//     registered catalog name
//   - "owner/repo"          -> direct GitHub release otherwise
//   - "name"                -> bare name, resolved across catalogs (default
//     first); a collision prints the qualified options instead of guessing
func installArg(cmd *cobra.Command, pluginsDir string, reg *pluginstore.Registry, arg, currentVersion string) error {
	if strings.Contains(arg, "@") {
		return installGitHubArg(cmd, pluginsDir, arg, currentVersion)
	}
	if strings.Contains(arg, "/") {
		prefix, rest, _ := strings.Cut(arg, "/")
		if reg.Find(prefix) != nil {
			return installFromCatalog(cmd, pluginsDir, reg, prefix, rest, currentVersion)
		}
		return installGitHubArg(cmd, pluginsDir, arg, currentVersion)
	}
	return installBareName(cmd, pluginsDir, reg, arg, currentVersion)
}

// installGitHubArg installs directly from a GitHub release (owner/repo[@version]).
func installGitHubArg(cmd *cobra.Command, pluginsDir, arg, currentVersion string) error {
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

// installFromCatalog installs a named plugin from a specific registered catalog.
func installFromCatalog(cmd *cobra.Command, pluginsDir string, reg *pluginstore.Registry, catalogName, pluginName, currentVersion string) error {
	cat := reg.Find(catalogName)
	if cat == nil {
		return customerrors.NewUserError(fmt.Sprintf("catalog %q is not registered; run 'datumctl plugin index add %s <source>' first", catalogName, catalogName))
	}
	idx, err := loadOrRefreshCatalog(cmd, pluginsDir, *cat)
	if err != nil {
		return indexFetchUserError(err)
	}
	entry, name, binaryPath, installErr := installPlugin(cmd.Context(), pluginsDir, pluginName, "", currentVersion, idx)
	if installErr != nil {
		return customerrors.NewUserError(fmt.Sprintf("install plugin %s/%s: %v", catalogName, pluginName, installErr))
	}
	entry.Catalog = catalogName
	return saveAndReport(cmd, pluginsDir, name, entry, binaryPath)
}

// installBareName resolves a bare plugin name across all catalogs (default
// first) and installs the unique match, or surfaces the options on a collision.
func installBareName(cmd *cobra.Command, pluginsDir string, reg *pluginstore.Registry, name, currentVersion string) error {
	matches := resolveBareName(cmd, pluginsDir, reg, name)
	switch len(matches) {
	case 0:
		return customerrors.NewUserError(fmt.Sprintf("plugin %q not found in any registered catalog; run 'datumctl plugin search %s' to look for it", name, name))
	case 1:
		return installFromCatalog(cmd, pluginsDir, reg, matches[0].catalog.Name, name, currentVersion)
	default:
		return customerrors.NewUserError(collisionError(name, matches).Error())
	}
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
	if entry.Catalog != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Installed %s %s from %s  [%s]\n",
			pluginName, entry.Version, entry.Catalog, installBadge(entry.Catalog))
	} else {
		// Direct GitHub install: not from a curated catalog.
		fmt.Fprintf(cmd.OutOrStdout(), "Installed %s %s  [%s]\n",
			pluginName, entry.Version, pluginstore.TrustThirdParty)
	}
	return nil
}

// installAllFromManifest restores all plugins recorded in plugins.json. Each
// recorded plugin is restored from the catalog it was originally installed from
// (or directly from GitHub for owner/repo sources). Catalog indexes are loaded
// lazily and memoized so each catalog is fetched at most once.
func installAllFromManifest(cmd *cobra.Command, pluginsDir, currentVersion string) error {
	manifest, err := pluginstore.Load(pluginsDir)
	if err != nil {
		return fmt.Errorf("load plugins manifest: %w", err)
	}

	if len(manifest.Plugins) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No plugins recorded in plugins.json.")
		return nil
	}

	reg, err := pluginstore.LoadRegistry(pluginsDir)
	if err != nil {
		return fmt.Errorf("load catalog registry: %w", err)
	}

	// Lazily load+refresh each catalog index, memoized by catalog name.
	indexCache := map[string]*pluginstore.CachedIndex{}
	getIndex := func(catalogName string) (*pluginstore.CachedIndex, error) {
		if idx, ok := indexCache[catalogName]; ok {
			return idx, nil
		}
		cat := reg.Find(catalogName)
		if cat == nil {
			return nil, fmt.Errorf("catalog %q is no longer registered", catalogName)
		}
		idx, loadErr := loadOrRefreshCatalog(cmd, pluginsDir, *cat)
		if loadErr != nil {
			return nil, loadErr
		}
		indexCache[catalogName] = idx
		return idx, nil
	}

	var errs []string
	for name, entry := range manifest.Plugins {
		// Direct GitHub source (owner/repo format).
		if entry.Catalog == "" && strings.Contains(entry.Source, "/") {
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

		// Catalog source. An empty Catalog on a non-slash source is a legacy
		// curated install, which restores from the default catalog.
		catalogName := entry.Catalog
		if catalogName == "" {
			catalogName = pluginstore.DefaultCatalogName
		}
		idx, idxErr := getIndex(catalogName)
		if idxErr != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", name, idxErr))
			continue
		}
		newEntry, _, _, installErr := installPlugin(cmd.Context(), pluginsDir, name, entry.Version, currentVersion, idx)
		if installErr != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", name, installErr))
			continue
		}
		newEntry.Catalog = catalogName
		manifest.Plugins[name] = newEntry
		fmt.Fprintf(cmd.OutOrStdout(), "Installed %s %s from %s\n", name, newEntry.Version, catalogName)
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

	// Extract the binary, then write it under its GENERIC name (e.g. "ipam").
	// Catalog/managed plugins are not datumctl-branded on disk; the install
	// record in plugins.json — not a filename prefix — marks them as plugins.
	binaryName := pluginName
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
// If platform.Files is non-empty, the first FileOperation's From path is used
// verbatim (this is how a catalog points at a generically named binary such as
// "ipam"). Otherwise the binary is auto-detected, accepting EITHER the generic
// name "<pluginName>[.exe]" or the legacy "datumctl-<pluginName>[.exe]".
func extractFromArchive(archiveBytes []byte, platform *pluginstore.Platform, pluginName, archiveURI string) ([]byte, error) {
	isTarGz := strings.HasSuffix(archiveURI, ".tar.gz") || strings.HasSuffix(archiveURI, ".tgz")
	isZip := strings.HasSuffix(archiveURI, ".zip")

	var candidates []string
	if len(platform.Files) > 0 {
		// Explicit binary directive from the catalog manifest.
		candidates = []string{platform.Files[0].From}
	} else {
		// Auto-detect: generic name first, then the legacy datumctl- prefix.
		suffix := ""
		if runtime.GOOS == "windows" {
			suffix = ".exe"
		}
		candidates = []string{pluginName + suffix, "datumctl-" + pluginName + suffix}
	}

	if !isTarGz && !isZip {
		return nil, fmt.Errorf("unsupported archive format for URI: %s", archiveURI)
	}

	var lastErr error
	for _, name := range candidates {
		var (
			b   []byte
			err error
		)
		if isTarGz {
			b, err = extractBinaryFromTarGz(newBytesReader(archiveBytes), name)
		} else {
			b, err = extractBinaryFromZip(archiveBytes, name)
		}
		if err == nil {
			return b, nil
		}
		lastErr = err
	}
	return nil, fmt.Errorf("no plugin binary found in archive (tried %v): %w", candidates, lastErr)
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

	// Write under the generic name; the install record marks it as a plugin.
	binaryName := pluginName
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
