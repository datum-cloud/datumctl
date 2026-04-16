package ctx

import (
	"fmt"
	"io"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"go.datum.net/datumctl/internal/datumconfig"
)

func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Short:   "List available contexts",
		Aliases: []string{"ls"},
		Args:    cobra.NoArgs,
		RunE:    runList,
	}
}

func runList(_ *cobra.Command, _ []string) error {
	cfg, err := datumconfig.LoadAuto()
	if err != nil {
		return err
	}

	if len(cfg.Contexts) == 0 {
		fmt.Println("No contexts available. Run 'datumctl login' to get started.")
		return nil
	}

	printContextTree(os.Stdout, cfg)
	return nil
}

type orgGroup struct {
	orgID    string
	orgCtx   *datumconfig.DiscoveredContext
	projects []*datumconfig.DiscoveredContext
}

func printContextTree(w io.Writer, cfg *datumconfig.ConfigV1Beta1) {
	// Group contexts by org.
	groups := make(map[string]*orgGroup)
	var orgOrder []string

	for i := range cfg.Contexts {
		ctx := &cfg.Contexts[i]
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

	tw := tabwriter.NewWriter(w, 0, 4, 3, ' ', 0)
	fmt.Fprintln(tw, "  DISPLAY NAME\tNAME\tTYPE\tCURRENT")

	for _, orgID := range orgOrder {
		g := groups[orgID]

		// Org row.
		if g.orgCtx != nil {
			current := ""
			if cfg.CurrentContext == g.orgCtx.Name {
				current = "*"
			}
			fmt.Fprintf(tw, "  %s\t%s\t%s\t%s\n", cfg.OrgDisplayName(orgID), orgID, "org", current)
		}

		// Project rows, indented.
		for _, p := range g.projects {
			current := ""
			if cfg.CurrentContext == p.Name {
				current = "*"
			}
			fmt.Fprintf(tw, "    %s\t%s\t%s\t%s\n", cfg.ProjectDisplayName(p.ProjectID), p.Ref(), "project", current)
		}
	}

	tw.Flush()
}
