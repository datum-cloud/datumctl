package plugin

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"k8s.io/kubectl/pkg/util/templates"

	customerrors "go.datum.net/datumctl/internal/errors"
	"go.datum.net/datumctl/internal/pluginstore"
)

// indexListRefreshTimeout bounds the best-effort metadata refresh that
// `plugin index list` performs per catalog so an offline catalog cannot stall
// the listing.
const indexListRefreshTimeout = 3 * time.Second

// indexCmd is the 'plugin index' command group for managing plugin catalogs.
func indexCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "index",
		Short: "Manage plugin catalogs (indexes)",
		Long: templates.LongDesc(`
			Manage the plugin catalogs datumctl searches and installs from.

			A catalog (called an index in command syntax) is a published list of
			datumctl plugins. Datum's curated "datum" catalog is always present and
			trusted with no setup. You can register additional catalogs — a company's
			internal catalog or a community one — and their plugins then appear in
			search, browse, and install alongside Datum's.

			Every catalog carries a trust badge: "official" for Datum's curated datum
			catalog, "third-party" for any catalog you add.`),
		Example: templates.Examples(`
			# Register a community catalog from a GitHub repository
			datumctl plugin index add community datum-community/datumctl-plugins

			# List registered catalogs
			datumctl plugin index list

			# Refresh catalog metadata
			datumctl plugin index update

			# Remove a catalog
			datumctl plugin index remove community`),
	}
	cmd.AddCommand(
		indexAddCmd(),
		indexListCmd(),
		indexRemoveCmd(),
		indexUpdateCmd(),
		indexValidateCmd(),
	)
	return cmd
}

func indexAddCmd() *cobra.Command {
	var assumeYes bool
	cmd := &cobra.Command{
		Use:   "add <name> <source>",
		Short: "Register a third-party plugin catalog",
		Long: templates.LongDesc(`
			Register a plugin catalog under a short name.

			The source may be an HTTPS manifest URL, a GitHub owner/repo shorthand, or
			a local path (for development or air-gapped use). Adding a third-party
			catalog is a one-time trust decision: its plugins are programs that run on
			your machine with your Datum credentials, and Datum does not review them.

			Downloads remain HTTPS-only and checksum-verified regardless of which
			catalog a plugin comes from.`),
		Example: templates.Examples(`
			# From an HTTPS manifest URL
			datumctl plugin index add acme https://plugins.acme.example/index.yaml

			# From a GitHub repository (owner/repo)
			datumctl plugin index add community datum-community/datumctl-plugins

			# From a local path
			datumctl plugin index add local ./my-catalog

			# Skip the trust prompt (for scripts/CI)
			datumctl plugin index add acme https://plugins.acme.example/index.yaml --yes`),
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name, source := args[0], args[1]

			if err := pluginstore.ValidateCatalogName(name); err != nil {
				return customerrors.NewUserError(err.Error())
			}
			// Validate the source format up front for a clear early error.
			if _, err := pluginstore.ResolveCatalogSource(source); err != nil {
				return customerrors.NewUserError(err.Error())
			}

			pluginsDir, err := resolvePluginsDir(cmd)
			if err != nil {
				return err
			}

			reg, err := pluginstore.LoadRegistry(pluginsDir)
			if err != nil {
				return fmt.Errorf("load catalog registry: %w", err)
			}
			if existing := reg.Find(name); existing != nil {
				if existing.Managed {
					return customerrors.NewUserError(fmt.Sprintf("catalog %q is managed by your organization and cannot be modified", name))
				}
				return customerrors.NewUserError(fmt.Sprintf("catalog %q is already registered (source: %s)", name, existing.Source))
			}

			// Enterprise allow-list enforcement.
			managed, err := pluginstore.LoadManagedConfig()
			if err != nil {
				return err
			}
			if !managed.IsAllowed(name, source) {
				return customerrors.NewUserErrorWithHint(
					fmt.Sprintf("catalog %q is not permitted by your organization's plugin catalog allow-list", name),
					"Contact your platform team, or ask them to add it to the managed allow-list.",
				)
			}

			// One-time trust decision for third-party catalogs.
			if !assumeYes {
				answered, ok, promptErr := confirmCatalogTrust(cmd, name)
				if promptErr != nil {
					return promptErr
				}
				if !answered {
					// No one answered the prompt (e.g. non-interactive shell, EOF).
					// Fail loudly (non-zero) rather than silently succeeding without
					// adding the catalog, so a CI script that forgot --yes is not misled.
					return customerrors.NewUserErrorWithHint(
						fmt.Sprintf("adding third-party catalog %q requires confirmation", name),
						"Re-run with --yes to confirm without a prompt (for scripts/CI).",
					)
				}
				if !ok {
					fmt.Fprintln(cmd.OutOrStdout(), "Aborted. No catalog was added.")
					return nil
				}
			}

			cat := pluginstore.Catalog{
				Name:      name,
				Source:    source,
				Type:      pluginstore.CatalogTypeCustom,
				TrustedAt: time.Now().UTC(),
			}
			reg.Catalogs = append(reg.Catalogs, cat)
			if err := pluginstore.SaveRegistry(pluginsDir, reg); err != nil {
				return fmt.Errorf("save catalog registry: %w", err)
			}

			// Best-effort initial refresh to capture the catalog header and warm the
			// cache. A failure here does not undo the registration.
			if idx, refreshErr := pluginstore.RefreshCatalog(cmd.Context(), pluginsDir, cat); refreshErr != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: could not fetch catalog %q yet (%v); it is registered and will be retried on use\n", name, refreshErr)
			} else if idx != nil {
				updateCatalogHeader(pluginsDir, reg, name, idx)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Added catalog %s (%s)  [%s]\n", name, source, pluginstore.TrustThirdParty)
			return nil
		},
	}
	cmd.Flags().BoolVar(&assumeYes, "yes", false, "Bypass the third-party trust prompt")
	return cmd
}

func indexListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List registered plugin catalogs",
		Long: templates.LongDesc(`
			List every registered plugin catalog with its type, plugin count, trust
			badge, and description. The official datum catalog is always shown first.`),
		Example: templates.Examples(`
			# List registered catalogs
			datumctl plugin index list`),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			pluginsDir, err := resolvePluginsDir(cmd)
			if err != nil {
				return err
			}
			reg, err := pluginstore.LoadRegistry(pluginsDir)
			if err != nil {
				return fmt.Errorf("load catalog registry: %w", err)
			}

			// Resolve each catalog's index so plugin counts populate consistently.
			// Already-fresh caches are used as-is; stale or uncached catalogs are
			// refreshed concurrently with a short per-catalog timeout. This is
			// best-effort — an offline catalog degrades to a "—" marker and never
			// fails the listing.
			indexes := make([]*pluginstore.CachedIndex, len(reg.Catalogs))
			var wg sync.WaitGroup
			for i := range reg.Catalogs {
				cat := reg.Catalogs[i]
				cached, _ := pluginstore.LoadCatalogIndex(pluginsDir, cat.Name)
				if !pluginstore.IsStale(cached) {
					indexes[i] = cached
					continue
				}
				wg.Add(1)
				go func(i int, cat pluginstore.Catalog) {
					defer wg.Done()
					ctx, cancel := context.WithTimeout(cmd.Context(), indexListRefreshTimeout)
					defer cancel()
					// RefreshCatalog degrades to a stale cache on failure, so a
					// non-nil result is still usable for a count.
					if idx, _ := pluginstore.RefreshCatalog(ctx, pluginsDir, cat); idx != nil {
						indexes[i] = idx
					}
				}(i, cat)
			}
			wg.Wait()

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 3, ' ', 0)
			fmt.Fprintln(w, "NAME\tTYPE\tPLUGINS\tTRUST\tDESCRIPTION")
			for i := range reg.Catalogs {
				cat := &reg.Catalogs[i]
				desc := catalogDescription(cat)
				count := "—"
				if idx := indexes[i]; idx != nil && !idx.RefreshedAt.IsZero() {
					count = fmt.Sprintf("%d", len(idx.Plugins))
					if desc == "" && idx.Header.Description != "" {
						desc = idx.Header.Description
					}
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", cat.Name, cat.Type, count, cat.Trust(), desc)
			}
			return w.Flush()
		},
	}
}

func indexRemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove a registered plugin catalog",
		Long: templates.LongDesc(`
			Remove a registered plugin catalog and its cached metadata. The default
			catalog cannot be removed, and catalogs pre-seeded by your organization's
			managed configuration cannot be removed here.

			Removing a catalog does not uninstall plugins you already installed from it.`),
		Example: templates.Examples(`
			# Remove the community catalog
			datumctl plugin index remove community`),
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: registeredCatalogNames,
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if pluginstore.CanonicalCatalogName(name) == pluginstore.OfficialCatalogName {
				return customerrors.NewUserError("the official datum catalog cannot be removed")
			}
			pluginsDir, err := resolvePluginsDir(cmd)
			if err != nil {
				return err
			}
			reg, err := pluginstore.LoadRegistry(pluginsDir)
			if err != nil {
				return fmt.Errorf("load catalog registry: %w", err)
			}
			cat := reg.Find(name)
			if cat == nil {
				return customerrors.NewUserError(fmt.Sprintf("catalog %q is not registered", name))
			}
			if cat.Managed {
				return customerrors.NewUserError(fmt.Sprintf("catalog %q is managed by your organization and cannot be removed", name))
			}

			kept := reg.Catalogs[:0]
			for _, c := range reg.Catalogs {
				if c.Name != name {
					kept = append(kept, c)
				}
			}
			reg.Catalogs = kept
			if err := pluginstore.SaveRegistry(pluginsDir, reg); err != nil {
				return fmt.Errorf("save catalog registry: %w", err)
			}

			// Best-effort cache cleanup.
			if dir, dErr := pluginstore.CatalogCacheDir(pluginsDir, name); dErr == nil {
				_ = os.RemoveAll(dir)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Removed catalog %s\n", name)
			return nil
		},
	}
	return cmd
}

func indexUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update [name]",
		Short: "Refresh catalog metadata",
		Long: templates.LongDesc(`
			Refresh one or all registered catalogs by re-fetching their manifests.
			With no argument, every catalog is refreshed; a failure for one catalog
			does not stop the others.`),
		Example: templates.Examples(`
			# Refresh all catalogs
			datumctl plugin index update

			# Refresh a single catalog
			datumctl plugin index update community`),
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: registeredCatalogNames,
		RunE: func(cmd *cobra.Command, args []string) error {
			pluginsDir, err := resolvePluginsDir(cmd)
			if err != nil {
				return err
			}
			reg, err := pluginstore.LoadRegistry(pluginsDir)
			if err != nil {
				return fmt.Errorf("load catalog registry: %w", err)
			}

			var targets []pluginstore.Catalog
			if len(args) == 1 {
				cat := reg.Find(args[0])
				if cat == nil {
					return customerrors.NewUserError(fmt.Sprintf("catalog %q is not registered", args[0]))
				}
				targets = []pluginstore.Catalog{*cat}
			} else {
				targets = reg.Catalogs
			}

			var failures int
			for _, cat := range targets {
				idx, refreshErr := pluginstore.RefreshCatalog(cmd.Context(), pluginsDir, cat)
				if refreshErr != nil {
					failures++
					fmt.Fprintf(cmd.ErrOrStderr(), "warning: %s: %v\n", cat.Name, refreshErr)
					continue
				}
				updateCatalogHeader(pluginsDir, reg, cat.Name, idx)
				fmt.Fprintf(cmd.OutOrStdout(), "Updated %s (%d plugins)\n", cat.Name, len(idx.Plugins))
			}
			if failures > 0 && len(targets) == 1 {
				return customerrors.NewUserError(fmt.Sprintf("could not update catalog %q", targets[0].Name))
			}
			return nil
		},
	}
}

func indexValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate <path | url>",
		Short: "Validate a catalog manifest before publishing",
		Long: templates.LongDesc(`
			Lint a catalog manifest before you publish it. The validator reports
			missing plugin names or versions, missing or non-HTTPS download URIs,
			missing checksums, and invalid platform selectors.

			The source may be a local path, an HTTPS URL, or a GitHub owner/repo.`),
		Example: templates.Examples(`
			# Validate a local manifest
			datumctl plugin index validate ./index.yaml

			# Validate a published manifest
			datumctl plugin index validate https://plugins.acme.example/index.yaml`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			list, err := pluginstore.FetchAndParseCatalog(cmd.Context(), args[0])
			if err != nil {
				return customerrors.NewUserError(fmt.Sprintf("could not read catalog manifest: %v", err))
			}
			problems := pluginstore.LintCatalog(list)
			if len(problems) == 0 {
				n := len(list.Items)
				fmt.Fprintf(cmd.OutOrStdout(), "OK: manifest is valid (%d plugin(s)).\n", n)
				return nil
			}
			var b strings.Builder
			fmt.Fprintf(&b, "catalog manifest has %d problem(s):", len(problems))
			for _, p := range problems {
				b.WriteString("\n  - " + p)
			}
			return customerrors.NewUserError(b.String())
		},
	}
}

// confirmCatalogTrust shows the one-time third-party trust explanation and reads
// a y/N answer from stdin. answered reports whether the user actually responded;
// it is false when the prompt receives no input (EOF / non-interactive), letting
// the caller distinguish an unanswered prompt from an explicit "no".
func confirmCatalogTrust(cmd *cobra.Command, name string) (answered, confirmed bool, err error) {
	// Interactive prompts go to stderr so stdout stays clean for redirection.
	out := cmd.ErrOrStderr()
	fmt.Fprintf(out, "\n  You're adding a third-party plugin catalog: %q\n\n", name)
	fmt.Fprintln(out, "  Datum does not review, verify, or endorse plugins from this catalog.")
	fmt.Fprintln(out, "  Plugins are programs that run on your machine with your Datum credentials.")
	fmt.Fprintln(out, "  Only add catalogs you trust.")
	fmt.Fprint(out, "\n  Add this catalog? [y/N] ")

	reader := bufio.NewReader(cmd.InOrStdin())
	line, readErr := reader.ReadString('\n')
	if readErr != nil && line == "" {
		// EOF with no input (e.g. non-interactive shell): the prompt went
		// unanswered. Leave the decision to the caller.
		return false, false, nil
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return true, answer == "y" || answer == "yes", nil
}

// updateCatalogHeader copies the freshly fetched catalog header into the
// registry entry and persists it. Best-effort: errors are ignored so a metadata
// refresh never blocks the primary operation.
func updateCatalogHeader(pluginsDir string, reg *pluginstore.Registry, name string, idx *pluginstore.CachedIndex) {
	cat := reg.Find(name)
	if cat == nil || idx == nil {
		return
	}
	if idx.Header.Description != "" {
		cat.Description = idx.Header.Description
	}
	if idx.Header.Owner != "" {
		cat.Owner = idx.Header.Owner
	}
	if idx.Header.Homepage != "" {
		cat.Homepage = idx.Header.Homepage
	}
	cat.LastUpdated = time.Now().UTC()
	_ = pluginstore.SaveRegistry(pluginsDir, reg)
}

// catalogDescription returns the best available description for a catalog.
func catalogDescription(cat *pluginstore.Catalog) string {
	if cat.Description != "" {
		return cat.Description
	}
	if cat.Name == pluginstore.OfficialCatalogName {
		return "Datum-curated plugins"
	}
	return ""
}

// registeredCatalogNames is a completion function returning registered catalog
// names.
func registeredCatalogNames(cmd *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	pluginsDir, err := resolvePluginsDir(cmd)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	reg, err := pluginstore.LoadRegistry(pluginsDir)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	var names []string
	for i := range reg.Catalogs {
		names = append(names, reg.Catalogs[i].Name)
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}
