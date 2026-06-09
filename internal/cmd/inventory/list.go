package inventory

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/kubectl/pkg/util/templates"

	"go.datum.net/datumctl/internal/client"
	customerrors "go.datum.net/datumctl/internal/errors"
	"go.datum.net/datumctl/internal/output"
)

const (
	apiGroup   = "inventory.miloapis.com"
	apiVersion = "v1alpha1"

	labelRegion  = "topology.inventory.miloapis.com/region"
	labelSite    = "topology.inventory.miloapis.com/site"
	labelCluster = "topology.inventory.miloapis.com/cluster"
)

func gvr(resource string) schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: apiGroup, Version: apiVersion, Resource: resource}
}

// filterFlag declares one --flag on a list subcommand. A flag either narrows
// the server-side label selector (labelKey set) or filters client-side on a
// spec field (predicate set); never both.
type filterFlag struct {
	name      string
	usage     string
	labelKey  string
	predicate func(u unstructured.Unstructured, value string) bool
}

// resourceView describes how to list and render one inventory kind.
type resourceView struct {
	resource string
	use      string
	short    string
	headers  []any
	row      func(u unstructured.Unstructured) []any
	filters  []filterFlag
}

var providersView = resourceView{
	resource: "providers",
	use:      "providers",
	short:    "List inventory providers",
	headers:  []any{"NAME", "DISPLAY", "TYPE", "READY"},
	row: func(u unstructured.Unstructured) []any {
		return []any{u.GetName(), str(u, "spec", "displayName"), str(u, "spec", "type"), ready(u)}
	},
}

var regionsView = resourceView{
	resource: "regions",
	use:      "regions",
	short:    "List inventory regions",
	headers:  []any{"NAME", "DISPLAY", "READY"},
	row: func(u unstructured.Unstructured) []any {
		return []any{u.GetName(), str(u, "spec", "displayName"), ready(u)}
	},
}

var sitesView = resourceView{
	resource: "sites",
	use:      "sites",
	short:    "List inventory sites",
	headers:  []any{"NAME", "REGION", "PROVIDER", "TYPE", "READY"},
	row: func(u unstructured.Unstructured) []any {
		return []any{
			u.GetName(),
			str(u, "spec", "regionRef", "name"),
			str(u, "spec", "providerRef", "name"),
			str(u, "spec", "type"),
			ready(u),
		}
	},
	filters: []filterFlag{
		{name: "region", usage: "Filter by region name", labelKey: labelRegion},
		{name: "provider", usage: "Filter by provider name", predicate: func(u unstructured.Unstructured, v string) bool {
			return str(u, "spec", "providerRef", "name") == v
		}},
	},
}

var clustersView = resourceView{
	resource: "clusters",
	use:      "clusters",
	short:    "List inventory clusters",
	headers:  []any{"NAME", "REGION", "CP-SITE", "ROLE", "PROVIDER", "READY"},
	row: func(u unstructured.Unstructured) []any {
		return []any{
			u.GetName(),
			u.GetLabels()[labelRegion],
			str(u, "spec", "controlPlaneSiteRef", "name"),
			str(u, "spec", "role"),
			str(u, "spec", "provider"),
			ready(u),
		}
	},
	filters: []filterFlag{
		{name: "region", usage: "Filter by region name", labelKey: labelRegion},
		{name: "site", usage: "Filter by control-plane site name", predicate: func(u unstructured.Unstructured, v string) bool {
			return str(u, "spec", "controlPlaneSiteRef", "name") == v
		}},
	},
}

var nodesView = resourceView{
	resource: "nodes",
	use:      "nodes",
	short:    "List inventory nodes",
	headers:  []any{"NAME", "SITE", "CLUSTER", "ROLE", "ARCH", "CPU", "PHASE", "READY"},
	row: func(u unstructured.Unstructured) []any {
		return []any{
			u.GetName(),
			str(u, "spec", "siteRef", "name"),
			str(u, "spec", "assignment", "clusterRef", "name"),
			str(u, "spec", "assignment", "role"),
			str(u, "spec", "hardware", "cpuArchitecture"),
			intStr(u, "spec", "hardware", "cpuCores"),
			str(u, "status", "phase"),
			ready(u),
		}
	},
	filters: []filterFlag{
		{name: "region", usage: "Filter by region name", labelKey: labelRegion},
		{name: "site", usage: "Filter by site name", labelKey: labelSite},
		{name: "cluster", usage: "Filter by cluster name", labelKey: labelCluster},
	},
}

func newListCmd(factory *client.DatumCloudFactory, view resourceView) *cobra.Command {
	values := make(map[string]*string, len(view.filters))
	cmd := &cobra.Command{
		Use:   view.use,
		Short: view.short,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			format, _ := cmd.Flags().GetString("output")
			applyInventoryScope(cmd, factory)

			selector := labels.Set{}
			var predicates []func(u unstructured.Unstructured) bool
			for _, f := range view.filters {
				v := *values[f.name]
				if v == "" {
					continue
				}
				switch {
				case f.labelKey != "":
					selector[f.labelKey] = v
				case f.predicate != nil:
					fn, val := f.predicate, v
					predicates = append(predicates, func(u unstructured.Unstructured) bool { return fn(u, val) })
				}
			}

			list, err := listResources(cmd.Context(), factory, view.resource, selector.String())
			if err != nil {
				return err
			}
			filterItems(list, predicates)
			return render(cmd, format, list, view.headers, view.row)
		},
	}
	for i := range view.filters {
		f := view.filters[i]
		values[f.name] = cmd.Flags().String(f.name, "", f.usage)
	}
	cmd.Example = listExample(view)
	return cmd
}

func listExample(view resourceView) string {
	lines := []string{fmt.Sprintf("# List all %s", view.resource), "datumctl inventory " + view.use}
	for _, f := range view.filters {
		lines = append(lines,
			"",
			fmt.Sprintf("# Filter by %s", f.name),
			fmt.Sprintf("datumctl inventory %s --%s <%s>", view.use, f.name, f.name))
	}
	return templates.Examples(strings.Join(lines, "\n"))
}

// applyInventoryScope defaults to the platform root, where inventory lives,
// unless the user explicitly selected an organization or project scope.
func applyInventoryScope(cmd *cobra.Command, factory *client.DatumCloudFactory) {
	if cmd.Flags().Changed("platform-wide") ||
		cmd.Flags().Changed("organization") ||
		cmd.Flags().Changed("project") {
		return
	}
	*factory.ConfigFlags.PlatformWide = true
}

func listResources(ctx context.Context, factory *client.DatumCloudFactory, resource, selector string) (*unstructured.UnstructuredList, error) {
	dc, err := factory.DynamicClient()
	if err != nil {
		return nil, customerrors.NewUserError(fmt.Sprintf("could not reach Datum Cloud: %v", err))
	}
	list, err := dc.Resource(gvr(resource)).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return nil, customerrors.NewUserError(fmt.Sprintf("could not list %s: %v", resource, err))
	}
	sort.Slice(list.Items, func(i, j int) bool { return list.Items[i].GetName() < list.Items[j].GetName() })
	return list, nil
}

func filterItems(list *unstructured.UnstructuredList, predicates []func(u unstructured.Unstructured) bool) {
	if len(predicates) == 0 {
		return
	}
	kept := list.Items[:0]
	for _, item := range list.Items {
		match := true
		for _, p := range predicates {
			if !p(item) {
				match = false
				break
			}
		}
		if match {
			kept = append(kept, item)
		}
	}
	list.Items = kept
}

func render(cmd *cobra.Command, format string, list *unstructured.UnstructuredList, headers []any, row func(u unstructured.Unstructured) []any) error {
	switch format {
	case "json", "yaml":
		return output.CLIPrint(cmd.OutOrStdout(), format, list, nil)
	case "", "table":
		rows := make([][]any, 0, len(list.Items))
		for _, item := range list.Items {
			rows = append(rows, row(item))
		}
		if len(rows) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No matching inventory found.")
			return nil
		}
		return output.CLIPrint(cmd.OutOrStdout(), "table", list, func() (output.ColumnFormatter, output.RowFormatterFunc) {
			return output.ColumnFormatter(headers), func() output.RowFormatter { return rows }
		})
	default:
		return customerrors.NewUserErrorWithHint(
			fmt.Sprintf("invalid value %q for --output", format),
			"Allowed values: table, json, yaml.")
	}
}
