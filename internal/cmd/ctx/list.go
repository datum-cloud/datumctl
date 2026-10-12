package ctx

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/rodaine/table"
	"github.com/spf13/cobra"
	"go.datum.net/datumctl/internal/datumconfig"
	"go.datum.net/datumctl/internal/discovery"
)

func listCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List available contexts",
		Long: `List the contexts for the active session.

By default only the active session's contexts are shown. Use --all to list
every session's contexts grouped by account and endpoint — useful when the
same organization or project name exists in more than one environment.`,
		Aliases: []string{"ls"},
		Args:    cobra.NoArgs,
		RunE:    runList,
	}
	cmd.Flags().Bool("all", false, "List contexts from every session, grouped by account and endpoint")
	return cmd
}

func runList(cmd *cobra.Command, _ []string) error {
	ctx := cmdContext(cmd)

	cfg, err := datumconfig.LoadAuto()
	if err != nil {
		return err
	}

	all, _ := cmd.Flags().GetBool("all")
	if all {
		return printAllContexts(ctx, os.Stdout, cfg)
	}

	session, err := cfg.ActiveSessionEntryE()
	if err != nil {
		return err
	}
	if session == nil {
		fmt.Println("No contexts available. Run 'datumctl login' to get started.")
		return nil
	}

	dir, err := listSession(ctx, session)
	if err != nil {
		return err
	}
	dir.WarnFailures(os.Stderr)
	if len(dir.Orgs) == 0 {
		fmt.Println("No contexts available. Create an organization in the Datum Cloud portal to get started.")
		return nil
	}
	printContextTree(os.Stdout, cfg, dir)
	return nil
}

func listSession(ctx context.Context, session *datumconfig.Session) (*discovery.Directory, error) {
	api, err := discovery.ForSession(ctx, session)
	if err != nil {
		return nil, err
	}
	dir, err := discovery.List(ctx, api, session.Name)
	if err != nil {
		return nil, fmt.Errorf("list contexts for %s: %w", session.UserEmail, err)
	}
	return dir, nil
}

// printAllContexts lists every session's contexts, grouped by account and
// endpoint so overlapping refs across environments stay distinguishable. A
// session whose contexts cannot be listed is skipped with a warning.
func printAllContexts(ctx context.Context, w io.Writer, cfg *datumconfig.ConfigV1Beta1) error {
	if len(cfg.Sessions) == 0 {
		fmt.Fprintln(w, "No contexts available. Run 'datumctl login' to get started.")
		return nil
	}
	printed := 0
	for i := range cfg.Sessions {
		s := &cfg.Sessions[i]
		dir, err := listSession(ctx, s)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Warning:", err)
			continue
		}
		dir.WarnFailures(os.Stderr)
		if len(dir.Orgs) == 0 {
			continue
		}
		if printed > 0 {
			fmt.Fprintln(w)
		}
		printed++
		fmt.Fprintf(w, "%s  (%s)\n", s.UserEmail, datumconfig.StripScheme(s.Endpoint.Server))
		printContextTree(w, cfg, dir)
	}
	return nil
}

// printContextTree prints a session's contexts as an org/project tree.
func printContextTree(w io.Writer, cfg *datumconfig.ConfigV1Beta1, dir *discovery.Directory) {
	tbl := table.New("Display Name", "Name", "Type", "Current")
	tbl.WithWriter(w)

	current := func(c *datumconfig.DiscoveredContext) string {
		if cfg.CurrentContextName() == c.Name {
			return "*"
		}
		return ""
	}

	contexts := dir.Contexts()
	for i := range contexts {
		org := &contexts[i]
		if org.ProjectID != "" {
			continue
		}
		tbl.AddRow(dir.OrgDisplayName(org.OrganizationID), org.OrganizationID, "org", current(org))
		for j := range contexts {
			p := &contexts[j]
			if p.ProjectID == "" || p.OrganizationID != org.OrganizationID {
				continue
			}
			tbl.AddRow("  "+dir.ProjectDisplayName(p.ProjectID), p.Ref(), "project", current(p))
		}
	}

	tbl.Print()
}
