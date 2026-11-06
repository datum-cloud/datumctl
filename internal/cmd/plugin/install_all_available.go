package plugin

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	customerrors "go.datum.net/datumctl/internal/errors"
	"go.datum.net/datumctl/internal/pluginstore"
)

// catalogInstallPlan is what 'datumctl plugin install --all-available' does
// with each plugin in a catalog.
type catalogInstallPlan struct {
	// install lists plugins that are missing or behind the catalog's version.
	install []string
	// upToDate lists plugins already installed from this catalog at its version.
	upToDate []string
	// skipped lists plugins installed from another catalog or from GitHub,
	// which --all-available leaves alone rather than replace.
	skipped []skippedPlugin
}

type skippedPlugin struct {
	name   string
	reason string
}

// planCatalogInstall decides, for each plugin in idx, whether to install it,
// leave it because it is up to date, or skip it because a plugin of the same
// name came from somewhere else. It does no I/O.
func planCatalogInstall(catalogName string, idx *pluginstore.CachedIndex, manifest *pluginstore.Manifest) catalogInstallPlan {
	var plan catalogInstallPlan
	for i := range idx.Plugins {
		p := &idx.Plugins[i]
		entry := manifest.Plugins[p.Name]
		if entry == nil {
			plan.install = append(plan.install, p.Name)
			continue
		}
		if from, _ := installedCatalogLabel(entry); from != catalogName {
			plan.skipped = append(plan.skipped, skippedPlugin{
				name:   p.Name,
				reason: fmt.Sprintf("already installed from %s", from),
			})
			continue
		}
		if entry.Version == p.Spec.Version {
			plan.upToDate = append(plan.upToDate, p.Name)
			continue
		}
		plan.install = append(plan.install, p.Name)
	}
	return plan
}

// installAllAvailable installs every plugin in one catalog (the official datum
// catalog unless catalogName names another), upgrading any that are behind and
// leaving the rest alone. It carries on past a failed plugin and reports every
// failure at the end.
//
// A third-party catalog needs confirmation first, because it installs plugins
// nobody named one by one: a y/N prompt, or assumeYes (--yes) for scripts.
func installAllAvailable(cmd *cobra.Command, pluginsDir, catalogName, currentVersion string, assumeYes bool) error {
	reg, err := pluginstore.LoadRegistry(pluginsDir)
	if err != nil {
		return fmt.Errorf("load catalog registry: %w", err)
	}
	warnDisabledCatalogs(cmd.ErrOrStderr(), reg)

	if catalogName == "" {
		catalogName = pluginstore.OfficialCatalogName
	}
	cat := reg.Find(catalogName)
	if cat == nil {
		return customerrors.NewUserError(fmt.Sprintf("catalog %q is not registered; run 'datumctl plugin index list' to see registered catalogs", catalogName))
	}
	if cat.Disabled {
		return customerrors.NewUserError(fmt.Sprintf("catalog %q is disabled: %s", cat.Name, cat.DisabledReason))
	}
	idx, err := loadOrRefreshCatalog(cmd, pluginsDir, *cat)
	if err != nil {
		return indexFetchUserError(err)
	}

	manifest, err := pluginstore.Load(pluginsDir)
	if err != nil {
		return fmt.Errorf("load plugins manifest: %w", err)
	}
	if manifest.Plugins == nil {
		manifest.Plugins = make(map[string]*pluginstore.InstalledPlugin)
	}

	plan := planCatalogInstall(cat.Name, idx, manifest)
	out := cmd.OutOrStdout()
	if len(idx.Plugins) == 0 {
		fmt.Fprintf(out, "Catalog %s has no plugins.\n", cat.Name)
		return nil
	}
	for _, name := range plan.upToDate {
		fmt.Fprintf(out, "%s %s is up to date\n", name, manifest.Plugins[name].Version)
	}
	for _, s := range plan.skipped {
		fmt.Fprintf(out, "Skipped %s: %s\n", s.name, s.reason)
	}

	if len(plan.install) > 0 && !cat.IsOfficial() && !assumeYes {
		answered, ok := confirmThirdPartyInstallAll(cmd, cat.Name, plan.install)
		if !answered {
			// Nobody answered (a script, CI, a coding agent's shell). Fail
			// rather than install third-party code nobody agreed to.
			return customerrors.NewUserErrorWithHint(
				fmt.Sprintf("installing every plugin from third-party catalog %q requires confirmation", cat.Name),
				"Re-run with --yes to confirm without a prompt, or install the plugins you want one at a time.",
			)
		}
		if !ok {
			fmt.Fprintln(out, "Aborted. No plugins were installed.")
			return nil
		}
	}

	var errs []string
	installed := 0
	for _, name := range plan.install {
		entry, _, _, installErr := installPlugin(cmd.Context(), pluginsDir, name, "", currentVersion, idx)
		if installErr != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", name, installErr))
			continue
		}
		entry.Catalog = cat.Name
		manifest.Plugins[name] = entry
		installed++
		fmt.Fprintf(out, "Installed %s %s from %s  [%s]\n", name, entry.Version, cat.Name, installBadge(cat.Name))
	}

	if installed > 0 {
		if saveErr := pluginstore.Save(pluginsDir, manifest); saveErr != nil {
			errs = append(errs, fmt.Sprintf("save plugins manifest: %v", saveErr))
		}
	}
	if len(errs) > 0 {
		return customerrors.NewUserError("some plugins failed to install:\n  " + strings.Join(errs, "\n  "))
	}
	return nil
}

// confirmThirdPartyInstallAll lists the plugins a third-party --all-available
// would install and reads a y/N answer from stdin. answered is false when no
// one responds (EOF, non-interactive), so the caller can tell that apart from
// an explicit "no". The prompt goes to stderr to keep stdout clean.
func confirmThirdPartyInstallAll(cmd *cobra.Command, catalogName string, plugins []string) (answered, confirmed bool) {
	out := cmd.ErrOrStderr()
	fmt.Fprintf(out, "\n  You're installing every plugin from the third-party catalog %q:\n\n", catalogName)
	for _, name := range plugins {
		fmt.Fprintf(out, "    %s\n", name)
	}
	fmt.Fprintln(out, "\n  Datum does not review, verify, or endorse plugins from this catalog.")
	fmt.Fprintln(out, "  Plugins are programs that run on your machine with your Datum credentials.")
	fmt.Fprint(out, "\n  Install all of them? [y/N] ")

	line, readErr := bufio.NewReader(cmd.InOrStdin()).ReadString('\n')
	if readErr != nil && line == "" {
		return false, false
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return true, answer == "y" || answer == "yes"
}
