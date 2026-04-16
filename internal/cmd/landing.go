package cmd

import (
	"fmt"
	"io"
	"math/rand"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"go.datum.net/datumctl/internal/datumconfig"
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

	printLoggedInLanding(out, cfg, session)
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
    Authenticates as a machine account for CI/CD and automation.
    Non-interactive — no browser, no prompts.
    Requires: a machine account created in the Datum Cloud portal and
    its credentials JSON file downloaded from there. Human accounts
    cannot use this path.

Run 'datumctl --help' for the full command reference.`)
}

func printLoggedInLanding(out io.Writer, cfg *datumconfig.ConfigV1Beta1, session *datumconfig.Session) {
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
		displayRef := cfg.DisplayRef(ctxEntry)
		fmt.Fprintf(out, "  Context        %s\n", displayRef)
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
	}
	fmt.Fprintln(out)

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
		fmt.Fprintln(out, "  Explore the API        datumctl api-resources")
		fmt.Fprintln(out, "  Follow the audit trail datumctl activity")
		fmt.Fprintln(out, "  Switch context         datumctl ctx")
	}
	fmt.Fprintln(out, "  Switch account         datumctl auth switch")
	fmt.Fprintln(out)

	fmt.Fprintf(out, "Tip: %s\n", pickTip(time.Now().UnixNano()))
	fmt.Fprintln(out)

	fmt.Fprintln(out, "Run 'datumctl --help' for the full command reference.")
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
