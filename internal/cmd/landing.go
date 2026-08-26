package cmd

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"go.datum.net/datumctl/internal/authutil"
	"go.datum.net/datumctl/internal/datumconfig"
	"go.datum.net/datumctl/internal/onboarding"
	"go.datum.net/datumctl/internal/pluginstore"
)

// runLanding prints a contextual welcome when `datumctl` is invoked with no
// subcommand. If config can't be loaded for any reason, it falls through to
// the logged-out landing rather than a hard error.
func runLanding(cmd *cobra.Command, _ []string) {
	out := cmd.OutOrStdout()

	cfg, err := datumconfig.LoadAuto()
	if err != nil || cfg == nil {
		printLoggedOutLanding(out)
		return
	}

	session := cfg.ActiveSessionEntry()
	if session == nil {
		printLoggedOutLanding(out)
		return
	}

	printLoggedInLanding(cmd.Context(), out, cfg, session)
}

func printLoggedOutLanding(out io.Writer) {
	fmt.Fprintln(out, `Welcome to Datum Cloud.

datumctl manages Datum Cloud resources from your terminal — DNS zones,
projects, workloads, IAM, and more.

You're not signed in yet. Pick the login style that fits your situation:

  datumctl login
    Opens a browser for OAuth sign-in, then walks you through picking
    a default context. The easiest path for everyday use.

  datumctl login --no-browser
    Prints a short code and a URL to visit on any device — no browser
    needed on this machine. Good for SSH sessions, remote servers, or
    any environment where a browser can't be launched locally.

  datumctl login --credentials ./key.json
    Authenticates as a service account for CI/CD and automation.
    Non-interactive — no browser, no prompts.
    Requires: a service account created in the Datum Cloud portal and
    its credentials JSON file downloaded from there. Human accounts
    cannot use this path.

Run 'datumctl --help' for the full command reference.`)
}

func printLoggedInLanding(ctx context.Context, out io.Writer, cfg *datumconfig.ConfigV1Beta1, session *datumconfig.Session) {
	name := firstName(session.UserName)
	greeting := timeOfDayGreeting(time.Now())
	if name == "" {
		fmt.Fprintf(out, "%s.\n", greeting)
	} else {
		fmt.Fprintf(out, "%s, %s.\n", greeting, name)
	}
	fmt.Fprintln(out)

	// Identity + context block
	fmt.Fprintf(out, "  Signed in as   %s\n", session.UserEmail)

	ctxEntry := cfg.CurrentContextEntry()
	if ctxEntry != nil {
		var ctxLine string
		if ctxEntry.ProjectID != "" {
			projName := cfg.ProjectDisplayName(ctxEntry.Session, ctxEntry.ProjectID)
			ctxLine = fmt.Sprintf("%q project (%s)", projName, ctxEntry.Ref())
		} else {
			orgName := cfg.OrgDisplayName(ctxEntry.Session, ctxEntry.OrganizationID)
			ctxLine = fmt.Sprintf("%q org (%s)", orgName, ctxEntry.OrganizationID)
		}
		fmt.Fprintf(out, "  Context        %s\n", ctxLine)
	} else {
		fmt.Fprintln(out, "  Context        (none — run 'datumctl ctx use' to pick one)")
	}

	// Show access breadth if we have cache data
	orgs := cfg.Cache.Organizations
	projects := cfg.Cache.Projects
	if len(orgs) > 0 {
		if len(projects) > 0 {
			fmt.Fprintf(out, "  Access         %d org(s), %d project(s)\n", len(orgs), len(projects))
		} else {
			fmt.Fprintf(out, "  Access         %d org(s)\n", len(orgs))
		}
	} else if len(cfg.ContextsForSession(session.Name)) == 0 {
		if portalBase, err := onboarding.DerivePortalURL(session.Endpoint.Server); err == nil {
			fmt.Fprintf(out, "  Next step      %s\n", portalBase)
			fmt.Fprintln(out)
			fmt.Fprintln(out, "You'll need an organization before datumctl can do much. Create one in the portal to get going.")
			fmt.Fprintln(out)
			fmt.Fprintln(out, "  datumctl login           Sign in and set up your account")
			fmt.Fprintln(out, "  datumctl auth switch     Switch to another account")
			fmt.Fprintln(out)
			fmt.Fprintf(out, "Tip: %s\n", pickTip(time.Now().UnixNano()))
			fmt.Fprintln(out)
			fmt.Fprintln(out, "Run 'datumctl --help' for the full command reference.")
			return
		}
	}
	fmt.Fprintln(out)

	onboardingResult, onboardingChecked := checkOnboardingStatus(ctx, cfg, session)
	if onboardingChecked && onboardingResult.State != onboarding.Complete {
		fmt.Fprintf(out, "  Onboarding     %s\n", onboarding.StatusLabel(onboardingResult))
		fmt.Fprintf(out, "  Next step      %s\n", onboardingResult.ActionURL)
		fmt.Fprintln(out)
		fmt.Fprintln(out, "This organization still needs a little setup in the portal before you can use it here.")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "  datumctl whoami        Check onboarding status")
		fmt.Fprintln(out, "  datumctl auth switch   Switch to another account")
		fmt.Fprintln(out)
		fmt.Fprintf(out, "Tip: %s\n", pickTip(time.Now().UnixNano()))
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Run 'datumctl --help' for the full command reference.")
		return
	}

	// Contextual next-step suggestions
	if ctxEntry == nil {
		fmt.Fprintln(out, "First, pick a working context:")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "  datumctl ctx use       Interactive context picker")
		fmt.Fprintln(out, "  datumctl ctx           List all available contexts")
	} else {
		fmt.Fprintln(out, "What would you like to do?")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "  Explore what's there   datumctl get <resource>")
		fmt.Fprintln(out, "                         datumctl describe <resource> <name>")
		fmt.Fprintln(out, "  Ship something         datumctl apply -f file.yaml")
		fmt.Fprintln(out, "                         datumctl create <resource> ...")
		fmt.Fprintln(out, "  Extend datumctl        datumctl plugin browse")
		fmt.Fprintln(out, "  Explore the API        datumctl api-resources")
		fmt.Fprintln(out, "  Follow the audit trail datumctl activity")
		fmt.Fprintln(out, "  Switch context         datumctl ctx")
	}
	fmt.Fprintln(out, "  Switch account         datumctl auth switch")
	fmt.Fprintln(out)

	// Installed plugins become `datumctl <command>` verbs, so reflect the user's
	// own setup on the landing. Best-effort and local-only — never blocks the
	// landing on a missing or unreadable plugin store.
	installed := printInstalledPlugins(out)

	// Plugins the user could add, drawn from whatever catalog cache is already on
	// disk. Same best-effort contract as the installed block: never fetches,
	// never blocks, prints nothing when there is nothing to show.
	printAvailablePlugins(out, installed)

	fmt.Fprintf(out, "Tip: %s\n", pickTip(time.Now().UnixNano()))
	fmt.Fprintln(out)

	fmt.Fprintln(out, "Run 'datumctl --help' for the full command reference.")
}

// printInstalledPlugins renders an "Installed Plugins" block listing each
// plugin as a runnable `datumctl <command>` verb with the catalog it came from.
// It is best-effort and local-only: any error reading the plugin store, or no
// installed plugins, simply prints nothing.
//
// It returns the set of installed plugin names so the "Available plugins" block
// below can subtract them without re-reading the store.
func printInstalledPlugins(out io.Writer) map[string]bool {
	plugins := collectInstalledPlugins()
	if len(plugins) == 0 {
		return nil
	}

	// Pad the command column so the source labels line up.
	cmdWidth := 0
	for _, p := range plugins {
		if w := len("datumctl " + p.name); w > cmdWidth {
			cmdWidth = w
		}
	}

	installed := make(map[string]bool, len(plugins))
	for i, p := range plugins {
		label := ""
		if i == 0 {
			label = "Installed Plugins"
		}
		fmt.Fprintf(out, "  %-22s %-*s (%s)\n", label, cmdWidth, "datumctl "+p.name, p.source)
		installed[p.name] = true
	}
	fmt.Fprintln(out)
	return installed
}

// landingAvailableLimit caps how many not-yet-installed plugins the landing
// names before deferring to the browser. The landing is a signpost, not a
// catalog listing: a long block would crowd out the sections above it.
const landingAvailableLimit = 3

// landingLineWidth is the widest line the available-plugins block will emit;
// descriptions are elided to whatever the label and name columns leave, so a row
// still fits an 80-column terminal.
const landingLineWidth = 78

// printAvailablePlugins renders an "Available plugins" block naming plugins the
// user could install but has not. Like the installed block it is best-effort
// and local-only: it reads the catalog cache already on disk and never fetches,
// so a cold cache, an unreadable store, or a fully-installed catalog all print
// nothing rather than delaying the landing on a network call.
func printAvailablePlugins(out io.Writer, installed map[string]bool) {
	plugins := collectAvailablePlugins(installed)
	if len(plugins) == 0 {
		return
	}

	shown := plugins
	if len(shown) > landingAvailableLimit {
		shown = shown[:landingAvailableLimit]
	}

	nameWidth := 0
	for _, p := range shown {
		if len(p.name) > nameWidth {
			nameWidth = len(p.name)
		}
	}

	for i, p := range shown {
		label := ""
		if i == 0 {
			label = "Available plugins"
		}
		row := fmt.Sprintf("  %-22s %-*s", label, nameWidth, p.name)
		if desc := elide(p.desc, landingLineWidth-len(row)-2); desc != "" {
			row += "  " + desc
		}
		fmt.Fprintln(out, strings.TrimRight(row, " "))
	}

	if remaining := len(plugins) - len(shown); remaining > 0 {
		fmt.Fprintf(out, "  %-22s +%d more — datumctl plugin browse\n", "", remaining)
	}
	fmt.Fprintf(out, "  %-22s Install with 'datumctl plugin install <name>'\n", "")
	fmt.Fprintln(out)
}

// elide truncates s to at most max runes, marking the cut with an ellipsis.
func elide(s string, max int) string {
	s = strings.TrimSpace(s)
	if max <= 1 {
		return ""
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return strings.TrimRight(string(r[:max-1]), " ") + "…"
}

// landingPluginSource returns a short catalog label for an installed plugin,
// mirroring how `plugin list` labels provenance.
func landingPluginSource(entry *pluginstore.InstalledPlugin) string {
	if entry == nil {
		return ""
	}
	if entry.Catalog != "" {
		return pluginstore.CanonicalCatalogName(entry.Catalog)
	}
	// Legacy records: a slash in Source means a direct GitHub install; otherwise
	// it came from the curated official catalog.
	if strings.Contains(entry.Source, "/") {
		return "direct"
	}
	return pluginstore.OfficialCatalogName
}

// firstName returns the first whitespace-delimited token of full. Returns ""
// for empty/whitespace input.
func firstName(full string) string {
	full = strings.TrimSpace(full)
	if full == "" {
		return ""
	}
	if i := strings.IndexAny(full, " \t"); i > 0 {
		return full[:i]
	}
	return full
}

func timeOfDayGreeting(now time.Time) string {
	switch h := now.Hour(); {
	case h >= 4 && h < 12:
		return "Good morning"
	case h >= 12 && h < 17:
		return "Good afternoon"
	case h >= 17 && h < 22:
		return "Good evening"
	default:
		return "Welcome back"
	}
}

var landingTips = []string{
	// Workflow tips
	"'datumctl apply -f -' reads YAML from stdin — great for piping from scripts.",
	"'datumctl diff -f file.yaml' previews changes before you commit them.",
	"'datumctl get <resource> -o yaml' dumps the raw object, pipe-ready.",
	"'datumctl get <resource> --watch' streams live updates as resources change.",
	// Discovery tips
	"'datumctl explain <resource>' prints the full field schema — no docs tab needed.",
	"'datumctl explain <resource>.spec' drills into a specific field tree.",
	"'datumctl api-resources' is the fastest way to see everything you can manage.",
	// Auth / context tips
	"'datumctl ctx' with no arguments opens an interactive context picker.",
	"'datumctl auth switch' jumps between accounts and restores each one's last context.",
	"Multiple accounts? 'datumctl auth list' shows every session on this machine.",
	// Audit tips
	"'datumctl activity' tails the audit trail across your whole control plane.",
	"'datumctl activity --start-time now-1h' scopes the feed to the last hour.",
	// Plugin / marketplace tips
	"'datumctl plugin browse' explores plugins across every registered catalog.",
	"'datumctl plugin search <keyword>' finds plugins to install from any catalog.",
	"'datumctl plugin index add <name> <url>' registers a team or community catalog.",
	// Power-user tips
	"Set DATUM_PROJECT or DATUM_ORGANIZATION to override context for a single command.",
	"'datumctl describe <resource> <name>' shows status conditions — handy for debugging.",
	"JSON and YAML both work with -f; mix them freely in a single directory.",
	"'datumctl version --client' prints the local version without hitting the server.",
	"Append '-o json | jq .' to any get command for pretty-printed, filterable output.",
}

func pickTip(seed int64) string {
	r := rand.New(rand.NewSource(seed))
	return landingTips[r.Intn(len(landingTips))]
}

func checkOnboardingStatus(ctx context.Context, cfg *datumconfig.ConfigV1Beta1, session *datumconfig.Session) (onboarding.Result, bool) {
	orgID := onboarding.ResolveEffectiveOrgID(cfg, os.Getenv("DATUM_PROJECT"), os.Getenv("DATUM_ORGANIZATION"))
	if orgID == "" {
		return onboarding.Result{}, false
	}

	tknSrc, err := authutil.GetTokenSourceForUser(ctx, session.UserKey)
	if err != nil {
		return onboarding.Result{}, false
	}
	userID, err := authutil.GetUserIDFromTokenForUser(session.UserKey)
	if err != nil {
		return onboarding.Result{}, false
	}
	apiHostname, err := authutil.GetAPIHostnameForUser(session.UserKey)
	if err != nil {
		return onboarding.Result{}, false
	}
	result, err := onboarding.CheckOrg(ctx, apiHostname, tknSrc, userID, orgID, cfg.OrgDisplayName(session.Name, orgID))
	if err != nil {
		return onboarding.Result{}, false
	}
	return result, true
}
