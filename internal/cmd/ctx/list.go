package ctx

import (
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/rodaine/table"
	"github.com/spf13/cobra"
	"go.datum.net/datumctl/internal/datumconfig"
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
	cfg, err := datumconfig.LoadAuto()
	if err != nil {
		return err
	}

	if len(cfg.Contexts) == 0 {
		fmt.Println("No contexts available. Run 'datumctl login' to get started.")
		return nil
	}

	all, _ := cmd.Flags().GetBool("all")
	if all {
		printAllContexts(os.Stdout, cfg)
		return nil
	}

	activeSession := ""
	if s := cfg.ActiveSessionEntry(); s != nil {
		activeSession = s.Name
	}
	printContextTree(os.Stdout, cfg, activeSession)
	return nil
}

type orgGroup struct {
	orgID    string
	orgCtx   *datumconfig.DiscoveredContext
	projects []*datumconfig.DiscoveredContext
}

// printAllContexts lists every session's contexts, grouped by account and
// endpoint so overlapping refs across environments stay distinguishable.
func printAllContexts(w io.Writer, cfg *datumconfig.ConfigV1Beta1) {
	for i := range cfg.Sessions {
		s := &cfg.Sessions[i]
		if len(cfg.ContextsForSession(s.Name)) == 0 {
			continue
		}
		if i > 0 {
			fmt.Fprintln(w)
		}
		fmt.Fprintf(w, "%s  (%s)\n", s.UserEmail, datumconfig.StripScheme(s.Endpoint.Server))
		printContextTree(w, cfg, s.Name)
	}
}

// printContextTree prints the contexts owned by sessionName as an org/project
// tree. Display names are resolved within that session.
func printContextTree(w io.Writer, cfg *datumconfig.ConfigV1Beta1, sessionName string) {
	// Group contexts by org.
	groups := make(map[string]*orgGroup)
	var orgOrder []string

	for i := range cfg.Contexts {
		ctx := &cfg.Contexts[i]
		if ctx.Session != sessionName {
			continue
		}
		orgID := ctx.OrganizationID

		g, ok := groups[orgID]
		if !ok {
			g = &orgGroup{orgID: orgID}
			groups[orgID] = g
			orgOrder = append(orgOrder, orgID)
		}

		if ctx.ProjectID == "" {
			g.orgCtx = ctx
		} else {
			g.projects = append(g.projects, ctx)
		}
	}

	// Sort projects within each group.
	for _, g := range groups {
		sort.Slice(g.projects, func(i, j int) bool {
			return g.projects[i].ProjectID < g.projects[j].ProjectID
		})
	}

	tbl := table.New("Display Name", "Name", "Type", "Current")
	tbl.WithWriter(w)

	for _, orgID := range orgOrder {
		g := groups[orgID]

		if g.orgCtx != nil {
			current := ""
			if cfg.CurrentContext == g.orgCtx.Name {
				current = "*"
			}
			tbl.AddRow(cfg.OrgDisplayName(sessionName, orgID), orgID, "org", current)
		}

		for _, p := range g.projects {
			current := ""
			if cfg.CurrentContext == p.Name {
				current = "*"
			}
			tbl.AddRow("  "+cfg.ProjectDisplayName(sessionName, p.ProjectID), p.Ref(), "project", current)
		}
	}

	tbl.Print()
}
