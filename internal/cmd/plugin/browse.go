package plugin

import (
	"fmt"
	"os"
	"strconv"

	"charm.land/huh/v2"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	componentversion "k8s.io/component-base/version"
	"k8s.io/kubectl/pkg/util/templates"

	customerrors "go.datum.net/datumctl/internal/errors"
	"go.datum.net/datumctl/internal/pluginstore"
)

// browseEntry is one selectable plugin in the interactive browser, paired with
// the catalog it came from.
type browseEntry struct {
	catalog pluginstore.Catalog
	plugin  pluginstore.Plugin
}

func browseCmd() *cobra.Command {
	var indexName string
	cmd := &cobra.Command{
		Use:   "browse",
		Short: "Browse and install plugins interactively",
		Long: templates.LongDesc(`
			Open an interactive browser over every registered plugin catalog.

			Type to filter, inspect a plugin's description, version, source catalog, and
			trust badge ("official" for Datum's curated datum catalog, "third-party" for
			catalogs you added), and install the selected plugin in place. Use --index to
			scope the browser to a single catalog.`),
		Example: templates.Examples(`
			# Browse all catalogs
			datumctl plugin browse

			# Browse a single catalog
			datumctl plugin browse --index acme`),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !browseIsTerminal() {
				return customerrors.NewUserErrorWithHint(
					"The interactive browser requires a terminal.",
					"Use 'datumctl plugin search' to list plugins non-interactively.",
				)
			}

			pluginsDir, err := resolvePluginsDir(cmd)
			if err != nil {
				return err
			}
			reg, err := pluginstore.LoadRegistry(pluginsDir)
			if err != nil {
				return fmt.Errorf("load catalog registry: %w", err)
			}

			warnDisabledCatalogs(cmd.ErrOrStderr(), reg)
			catalogs := reg.Active()
			if indexName != "" {
				cat := reg.Find(indexName)
				if cat == nil {
					return customerrors.NewUserError(fmt.Sprintf("catalog %q is not registered", indexName))
				}
				if cat.Disabled {
					return customerrors.NewUserError(fmt.Sprintf("catalog %q is disabled: %s", indexName, cat.DisabledReason))
				}
				catalogs = []pluginstore.Catalog{*cat}
			}

			var entries []browseEntry
			for i := range catalogs {
				cat := catalogs[i]
				idx, idxErr := loadOrRefreshCatalog(cmd, pluginsDir, cat)
				if idxErr != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "warning: skipping catalog %q: %v\n", cat.Name, idxErr)
					continue
				}
				for j := range idx.Plugins {
					entries = append(entries, browseEntry{catalog: cat, plugin: idx.Plugins[j]})
				}
			}

			if len(entries) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No plugins available. Add a catalog with 'datumctl plugin index add'.")
				return nil
			}

			selected, ok, err := runBrowser(entries)
			if err != nil {
				return err
			}
			if !ok {
				fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
				return nil
			}

			currentVersion := componentversion.Get().GitVersion
			// Empty version installs the catalog's recommended release.
			return installFromCatalog(cmd, pluginsDir, reg, selected.catalog.Name, selected.plugin.Name, "", currentVersion)
		},
	}
	cmd.Flags().StringVar(&indexName, "index", "", "Scope the browser to a single catalog")
	return cmd
}

// runBrowser presents the filterable plugin picker followed by an install
// confirmation. It returns the chosen entry and whether the user confirmed the
// install.
func runBrowser(entries []browseEntry) (browseEntry, bool, error) {
	options := browseOptions(entries)

	var picked string
	selectForm := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Datum Plugin Catalog").
				Description("Type to filter. Enter to inspect and install.").
				Options(options...).
				Value(&picked).
				Filtering(true),
		),
	)
	if err := selectForm.Run(); err != nil {
		return browseEntry{}, false, fmt.Errorf("plugin selection: %w", err)
	}

	i, err := strconv.Atoi(picked)
	if err != nil || i < 0 || i >= len(entries) {
		return browseEntry{}, false, fmt.Errorf("invalid selection")
	}
	entry := entries[i]

	var confirm bool
	confirmForm := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(fmt.Sprintf("Install %s from %s?", entry.plugin.Name, entry.catalog.Name)).
				Description(browseDetails(entry)).
				Affirmative("Install").
				Negative("Cancel").
				Value(&confirm),
		),
	)
	if err := confirmForm.Run(); err != nil {
		return browseEntry{}, false, fmt.Errorf("install confirmation: %w", err)
	}
	return entry, confirm, nil
}

// browseOptions builds the picker options. Each option's value is the entry's
// index so the selection maps back unambiguously even across catalogs that
// share a plugin name.
func browseOptions(entries []browseEntry) []huh.Option[string] {
	options := make([]huh.Option[string], len(entries))
	for i, e := range entries {
		label := fmt.Sprintf("%s  [%s]  %s  — %s",
			e.plugin.Name, e.catalog.Trust(), e.catalog.Name, e.plugin.Spec.ShortDescription)
		options[i] = huh.NewOption(label, strconv.Itoa(i))
	}
	return options
}

// browseDetails renders the inspect panel shown on the confirmation step.
func browseDetails(e browseEntry) string {
	desc := e.plugin.Spec.ShortDescription
	if e.plugin.Spec.Description != "" {
		desc = e.plugin.Spec.Description
	}
	out := fmt.Sprintf(
		"Catalog:   %s (%s)\nVersion:   %s\nTrust:     %s",
		e.catalog.Name, e.catalog.Type, e.plugin.Spec.Version, e.catalog.Trust(),
	)
	if e.plugin.Spec.Homepage != "" {
		out += "\nHomepage:  " + e.plugin.Spec.Homepage
	}
	if desc != "" {
		out += "\n\n" + desc
	}
	return out
}

// browseIsTerminal reports whether stdin is an interactive terminal.
func browseIsTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}
