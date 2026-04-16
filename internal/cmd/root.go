package cmd

import (
	"context"

	"github.com/spf13/cobra"
	"go.datum.net/datumctl/internal/client"
	"go.datum.net/datumctl/internal/cmd/auth"
	"go.datum.net/datumctl/internal/cmd/create"
	datumctx "go.datum.net/datumctl/internal/cmd/ctx"
	"go.datum.net/datumctl/internal/cmd/docs"
	"go.datum.net/datumctl/internal/cmd/login"
	"go.datum.net/datumctl/internal/cmd/logout"
	"go.datum.net/datumctl/internal/cmd/whoami"
	activity "go.miloapis.com/activity/pkg/cmd"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/kubectl/pkg/cmd/apiresources"
	"k8s.io/kubectl/pkg/cmd/apply"
	kubeauth "k8s.io/kubectl/pkg/cmd/auth"
	delcmd "k8s.io/kubectl/pkg/cmd/delete"
	"k8s.io/kubectl/pkg/cmd/describe"
	"k8s.io/kubectl/pkg/cmd/diff"
	"k8s.io/kubectl/pkg/cmd/edit"
	"k8s.io/kubectl/pkg/cmd/explain"
	"k8s.io/kubectl/pkg/cmd/get"
	"k8s.io/kubectl/pkg/cmd/version"
)

// hideFlags hides the named flags from a command's flag set. Flags that do not
// exist on the command are silently skipped (MarkHidden returns an error which
// is discarded via _ =).
func hideFlags(cmd *cobra.Command, flags ...string) {
	for _, f := range flags {
		_ = cmd.Flags().MarkHidden(f)
	}
}

// hidePersistentFlags hides the named flags from a command's persistent flag
// set. Flags that do not exist are silently skipped.
func hidePersistentFlags(cmd *cobra.Command, flags ...string) {
	for _, f := range flags {
		_ = cmd.PersistentFlags().MarkHidden(f)
	}
}

func RootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "datumctl",
		Short: "The official CLI for Datum Cloud",
		Long: `datumctl is the official command-line interface for Datum Cloud, the connectivity
infrastructure platform for developers and forward-thinking companies.

Use datumctl to authenticate and manage all your Datum Cloud resources —
projects, organizations, networking, compute, and more — directly from the
terminal. No knowledge of Kubernetes or kubectl required.

Get started:
  datumctl login
  datumctl get organizations
  datumctl get dnszones`,
	}
	// kubectl version expects this flag to exist; add it here to avoid nil deref.
	rootCmd.PersistentFlags().Bool("warnings-as-errors", false, "Treat warnings as errors")
	ioStreams := genericclioptions.IOStreams{
		In:     rootCmd.InOrStdin(),
		Out:    rootCmd.OutOrStdout(),
		ErrOut: rootCmd.ErrOrStderr(),
	}

	ctx := context.Background()
	factory, err := client.NewDatumFactory(ctx)
	if err != nil {
		panic(err)
	}
	factory.AddFlags(rootCmd.PersistentFlags())
	factory.AddFlagMutualExclusions(rootCmd)

	// Hide kubectl auth/TLS/internal flags that are not relevant to Datum Cloud
	// users. The flags still work if someone passes them explicitly; they just
	// won't appear in --help output.
	// Update flag descriptions to reflect Datum Cloud context.
	if f := rootCmd.PersistentFlags().Lookup("namespace"); f != nil {
		f.Usage = "Namespace scope for this request (currently only 'default' is supported)"
	}

	hidePersistentFlags(rootCmd,
		"as", "as-group", "as-uid", "as-user-extra",
		"certificate-authority", "insecure-skip-tls-verify", "tls-server-name",
		"server", "token", "user",
		"log-flush-frequency", "v", "vmodule", "disable-compression",
		"warnings-as-errors",
	)

	rootCmd.AddGroup(&cobra.Group{ID: "auth", Title: "Authentication"})
	rootCmd.AddGroup(&cobra.Group{ID: "context", Title: "Context"})
	rootCmd.AddGroup(&cobra.Group{ID: "other", Title: "Other Commands"})
	rootCmd.AddGroup(&cobra.Group{ID: "resource", Title: "Resource Management"})

	// Top-level auth entry points. Promoted out of the 'auth' subgroup so that
	// new users find 'datumctl login' at the root of the CLI, while experienced
	// users can still reach 'datumctl auth login' for advanced options
	// (machine-account, device flow).
	loginCmd := login.Command()
	loginCmd.GroupID = "auth"
	rootCmd.AddCommand(loginCmd)

	logoutCmd := logout.Command()
	logoutCmd.GroupID = "auth"
	rootCmd.AddCommand(logoutCmd)

	whoamiCmd := whoami.Command()
	whoamiCmd.GroupID = "auth"
	rootCmd.AddCommand(whoamiCmd)

	ctxCmd := datumctx.Command()
	ctxCmd.GroupID = "context"
	rootCmd.AddCommand(ctxCmd)

	authCommand := auth.Command()
	whoami := kubeauth.NewCmdWhoAmI(factory, ioStreams)
	whoami.Short = "Show your identity on a Datum Cloud control plane (kubectl users only)"
	whoami.Long = `For kubectl users only. Requires a control plane context configured via
'datumctl auth update-kubeconfig'.

Queries the Datum Cloud API server to display the user identity and group
memberships it has resolved from your current credentials. Useful for
confirming which account kubectl is using and troubleshooting access-denied
errors against the control plane.

To see which datumctl account is currently active (without needing a
control plane context), use 'datumctl auth list' instead.`
	whoami.Example = `  # Show your identity on the configured control plane
  datumctl auth whoami

  # Show your identity in JSON format
  datumctl auth whoami -o json`
	authCommand.AddCommand(whoami)
	cani := kubeauth.NewCmdCanI(factory, ioStreams)
	cani.Short = "Check permissions on a Datum Cloud control plane (kubectl users only)"
	cani.Long = `For kubectl users only. Requires a control plane context configured via
'datumctl auth update-kubeconfig'.

Verify whether the active user has permission to perform a specific action
against a Datum Cloud resource type on the configured control plane.

VERB is an API verb: get, list, watch, create, update, patch, delete, or '*'.
TYPE is a Datum Cloud resource type (e.g., projects, dnszones, domains).

Use 'datumctl api-resources' to see all available resource types.`
	cani.Example = `  # Check if you can list projects on the control plane
  datumctl auth can-i list projects

  # Check if you can create DNS zones
  datumctl auth can-i create dnszones

  # List all your permitted actions
  datumctl auth can-i --list

  # List permitted actions in a specific namespace
  datumctl auth can-i --list --namespace default`
	authCommand.AddCommand(cani)
	authCommand.GroupID = "auth"
	rootCmd.AddCommand(authCommand)

	getCmd := get.NewCmdGet("datumctl", factory, ioStreams)
	getCmd.Short = "List or retrieve Datum Cloud resources"
	getCmd.Long = `Display one or more Datum Cloud resources in a formatted table, or in JSON
or YAML for scripting and inspection.

Use the --organization or --project flags to target a specific context.
Use 'datumctl api-resources' to see all available resource types.

Tip: 'datumctl get organizations' lists your organization memberships and
does not require an --organization or --project flag. All other resource
types require one of these flags to specify the target context.
The 'organizations' shorthand also works with 'datumctl delete', 'datumctl edit',
and 'datumctl describe'.`
	getCmd.Example = `  # List your organization memberships (no context required)
  datumctl get organizations

  # List all projects in an organization
  datumctl get projects --organization <org-id>

  # Get a specific project by name
  datumctl get project my-project-id --organization <org-id>

  # List DNS zones in a project namespace
  datumctl get dnszones --project <project-id> --namespace default

  # List all resources of a type across all namespaces
  datumctl get dnszones --organization <org-id> --all-namespaces

  # Output as YAML
  datumctl get project my-project-id --organization <org-id> -o yaml

  # Watch for changes
  datumctl get projects --organization <org-id> --watch`
	hideFlags(getCmd,
		"allow-missing-template-keys", "chunk-size", "kustomize",
		"output-watch-events", "raw", "server-print", "show-managed-fields",
		"subresource", "template",
	)
	rootCmd.AddCommand(WrapResourceCommand(getCmd))

	deleteCmd := delcmd.NewCmdDelete(factory, ioStreams)
	deleteCmd.Short = "Delete Datum Cloud resources"
	deleteCmd.Long = `Delete one or more Datum Cloud resources by name, label selector, or
by providing a resource manifest file.

Resources can be specified as TYPE NAME pairs, or from a YAML/JSON file
with -f. JSON and YAML formats are accepted.

Note: this command does not perform a version check before deletion. If
someone has updated a resource between when you fetched it and when you
delete it, the deletion still proceeds. Use --dry-run=client to preview
what would be deleted before committing.`
	deleteCmd.Example = `  # Delete a project by name
  datumctl delete project my-project-id --organization <org-id>

  # Delete a DNS zone by name
  datumctl delete dnszone my-zone --project <project-id> --namespace default

  # Delete resources defined in a manifest file
  datumctl delete -f ./my-resource.yaml --organization <org-id>

  # Delete resources matching a label selector
  datumctl delete dnszones -l app=my-app --project <project-id>

  # Preview what would be deleted without actually deleting
  datumctl delete project my-project-id --organization <org-id> --dry-run=client`
	hideFlags(deleteCmd,
		"allow-missing-template-keys", "chunk-size", "kustomize",
		"output-watch-events", "raw", "server-print", "show-managed-fields",
		"subresource", "template",
	)
	rootCmd.AddCommand(WrapResourceCommand(deleteCmd))

	createCmd := create.NewCmdCreate(factory, ioStreams)
	hideFlags(createCmd,
		"allow-missing-template-keys", "kustomize", "template", "save-config",
	)
	createCmd.GroupID = "resource"
	rootCmd.AddCommand(createCmd)

	applyCmd := apply.NewCmdApply("datumctl", factory, ioStreams)
	applyCmd.Short = "Apply a Datum Cloud resource manifest (create or update)"
	applyCmd.Long = `Create or update Datum Cloud resources by applying a manifest file or
reading from stdin. If the resource does not exist it is created; if it
already exists it is updated to match the desired state in the manifest.

This is the recommended way to manage Datum Cloud resources declaratively.
Store your manifests in source control and apply them to keep your
infrastructure in sync.

JSON and YAML formats are accepted. Multiple resources can be placed in a
single file using YAML document separators (---).

Use --dry-run=server to validate your manifests against the API server
without persisting any changes.`
	applyCmd.Example = `  # Apply a project manifest
  datumctl apply -f ./project.yaml --organization <org-id>

  # Apply all manifests in a directory
  datumctl apply -f ./infra/ --organization <org-id>

  # Apply from stdin
  cat dnszone.yaml | datumctl apply -f - --project <project-id>

  # Preview changes without applying them
  datumctl apply -f ./project.yaml --organization <org-id> --dry-run=server

  # Diff then apply
  datumctl diff -f ./project.yaml --organization <org-id> && datumctl apply -f ./project.yaml --organization <org-id>`
	hideFlags(applyCmd,
		"allow-missing-template-keys", "kustomize", "template",
		"server-dry-run", "prune-allowlist",
	)
	applyCmd.GroupID = "resource"
	rootCmd.AddCommand(applyCmd)

	editCmd := edit.NewCmdEdit(factory, ioStreams)
	editCmd.Short = "Open a Datum Cloud resource in your editor and apply the changes"
	editCmd.Long = `Fetch a Datum Cloud resource, open it in your local text editor, and
apply any changes you save back to the platform.

The editor is determined by the EDITOR environment variable (also supports
KUBE_EDITOR for compatibility with kubectl workflows), falling back to
'vi' on Linux/macOS or 'notepad' on Windows.

The resource is displayed in YAML by default. Use -o json to edit in JSON
format instead.

Changes are applied when you save and close the file. If a conflict is
detected (the resource was modified server-side while your editor was open),
datumctl saves your changes to a temporary file so you can reconcile them.`
	editCmd.Example = `  # Edit a project
  datumctl edit project my-project-id --organization <org-id>

  # Edit a DNS zone, opening in a specific editor
  EDITOR="code --wait" datumctl edit dnszone my-zone --project <project-id> --namespace default

  # Edit a resource in JSON format
  datumctl edit project my-project-id --organization <org-id> -o json`
	hideFlags(editCmd,
		"allow-missing-template-keys", "chunk-size", "kustomize",
		"output-watch-events", "raw", "server-print", "show-managed-fields",
		"subresource", "template",
	)
	rootCmd.AddCommand(WrapResourceCommand(editCmd))

	describeCmd := describe.NewCmdDescribe("datumctl", factory, ioStreams)
	describeCmd.Short = "Show detailed information about a Datum Cloud resource"
	describeCmd.Long = `Print a detailed, human-readable description of one or more Datum Cloud
resources, including status conditions and related events where available.

You can select resources by name, by label selector (-l), or from a
manifest file (-f). If you provide a name prefix, datumctl will show
details for all resources whose names start with that prefix.

Use 'datumctl get' for a concise list, and 'datumctl describe' when you
need full status information, such as when troubleshooting a resource that
is not reaching a ready state.`
	describeCmd.Example = `  # Describe a specific project
  datumctl describe project my-project-id --organization <org-id>

  # Describe all DNS zones in a namespace
  datumctl describe dnszones --project <project-id> --namespace default

  # Describe resources matching a label selector
  datumctl describe dnszones -l app=my-app --project <project-id>

  # Describe a resource from a manifest file
  datumctl describe -f ./project.yaml --organization <org-id>`
	hideFlags(describeCmd,
		"allow-missing-template-keys", "chunk-size", "kustomize",
		"output-watch-events", "raw", "server-print", "show-managed-fields",
		"subresource", "template",
	)
	rootCmd.AddCommand(WrapResourceCommand(describeCmd))

	diffCmd := diff.NewCmdDiff(factory, ioStreams)
	diffCmd.Short = "Preview changes a manifest would make to live resources"
	diffCmd.Long = `Show the difference between what is currently deployed on the Datum Cloud
platform and what would be applied from a given manifest file.

The output is always YAML. The diff uses the 'diff' tool in your PATH
with -u (unified) and -N (treat absent files as empty) flags by default.

Set the DATUMCTL_EXTERNAL_DIFF environment variable to use a different diff
tool (KUBECTL_EXTERNAL_DIFF is also supported for kubectl workflow compatibility).
Example: DATUMCTL_EXTERNAL_DIFF="colordiff -N -u"

Exit codes:
  0   No differences were found.
  1   Differences were found.
  >1  An error occurred.`
	diffCmd.Example = `  # Preview changes from a manifest file before applying
  datumctl diff -f ./project.yaml --organization <org-id>

  # Diff from stdin
  cat dnszone.yaml | datumctl diff -f - --project <project-id>

  # Use a color diff tool
  DATUMCTL_EXTERNAL_DIFF="colordiff -N -u" datumctl diff -f ./project.yaml --organization <org-id>`
	hideFlags(diffCmd,
		"allow-missing-template-keys", "kustomize", "template",
	)
	diffCmd.GroupID = "resource"
	rootCmd.AddCommand(diffCmd)

	explainCmd := explain.NewCmdExplain("datumctl", factory, ioStreams)
	explainCmd.Short = "Show the schema and field documentation for a Datum Cloud resource type"
	explainCmd.Long = `Display the schema definition and field-level documentation for any
Datum Cloud resource type supported by the current control plane.

Fields are referenced using dot notation: TYPE.fieldName.subFieldName.
Information is retrieved from the API server in OpenAPI format, so it
always reflects the exact version of the platform you are connected to.

Use 'datumctl api-resources' to see all available resource types.`
	explainCmd.Example = `  # Show the schema for the Project resource type
  datumctl explain projects

  # Show all fields recursively
  datumctl explain projects --recursive

  # Show documentation for a specific field
  datumctl explain projects.spec

  # Show documentation using the OpenAPI v2 format
  datumctl explain projects --output=plaintext-openapiv2`
	hideFlags(explainCmd, "api-version")
	explainCmd.GroupID = "other"
	rootCmd.AddCommand(explainCmd)

	apiVersionCmd := apiresources.NewCmdAPIVersions(factory, ioStreams)
	apiVersionCmd.Short = "List all API group/version pairs supported by the current Datum Cloud context"
	apiVersionCmd.Long = `Print all API group and version combinations available from the Datum Cloud
API server for the currently configured control plane, one per line in the
form group/version (e.g., networking.datumapis.com/v1alpha).

Use 'datumctl api-resources' to also see the individual resource types
within each group.`
	apiVersionCmd.Example = `  # List all API versions
  datumctl api-versions`
	apiVersionCmd.GroupID = "other"
	rootCmd.AddCommand(apiVersionCmd)

	apiResourceCmd := apiresources.NewCmdAPIResources(factory, ioStreams)
	apiResourceCmd.Short = "List all resource types available in the current Datum Cloud context"
	apiResourceCmd.Long = `Print a table of all resource types available from the Datum Cloud API
server for the currently configured control plane.

This is the starting point for discovering what you can manage with
'datumctl get', 'datumctl apply', and 'datumctl explain'. The output
includes short names, API group, whether the resource is namespaced,
and the kind name.

The list is fetched fresh from the server on each invocation. To use a
cached copy, pass --cached.`
	apiResourceCmd.Example = `  # List all available resource types
  datumctl api-resources

  # List resource types with additional detail (verbs, short names)
  datumctl api-resources -o wide

  # List resource types sorted by name
  datumctl api-resources --sort-by=name

  # List only namespaced resource types
  datumctl api-resources --namespaced=true

  # List resource types for a specific API group
  datumctl api-resources --api-group=networking.datumapis.com`
	apiResourceCmd.GroupID = "other"
	rootCmd.AddCommand(apiResourceCmd)

	versionCmd := version.NewCmdVersion(factory, ioStreams)
	versionCmd.Short = "Print the datumctl client and API server version"
	versionCmd.Long = `Print version information for the datumctl client binary and, when you
are logged in and have an organization or project configured, for the
Datum Cloud API server.

Use --client to print only the local client version without contacting
the server.`
	versionCmd.Example = `  # Print client and server versions
  datumctl version

  # Print client version only (no server connection required)
  datumctl version --client

  # Print version in JSON format
  datumctl version -o json`
	versionCmd.GroupID = "other"
	rootCmd.AddCommand(versionCmd)

	activityCmd := activity.NewActivityCommand(activity.ActivityCommandOptions{
		Factory:   factory,
		IOStreams: ioStreams,
	})
	activityCmd.Short = "View activity logs for Datum Cloud resources"
	activityCmd.Long = `The activity group provides commands to query audit logs, events, and
change history for Datum Cloud resources.

Subcommands:
  audit     View audit log entries for actions taken on your resources
  events    View events emitted by Datum Cloud resources
  feed      View a combined activity feed across resource types
  history   View the change history for a specific resource`
	activityCmd.Example = `  # View recent audit events
  datumctl activity audit --start-time "now-7d"

  # View warning events
  datumctl activity events --type Warning --start-time "now-7d"

  # View a combined activity feed filtered by human-initiated changes
  datumctl activity feed --change-source human`
	for _, subCmd := range activityCmd.Commands() {
		switch subCmd.Name() {
		case "audit":
			subCmd.Short = "View audit log entries for Datum Cloud resources"
			subCmd.Long = `Display audit log entries recording actions taken on your Datum Cloud
resources. Audit events capture who did what, when, and to which resource.

Use --start-time and --end-time to filter by time range.`
			subCmd.Example = `  # View audit events from the last 7 days
  datumctl activity audit --start-time "now-7d"

  # View audit events for a specific resource type
  datumctl activity audit --resource projects`
		case "events":
			subCmd.Short = "View events emitted by Datum Cloud resources"
			subCmd.Long = `Display events emitted by Datum Cloud resources. Events record
state changes and noteworthy occurrences during the lifecycle of resources.

Use --type to filter by event severity (Normal or Warning).
Use --start-time to limit results to recent events.`
			subCmd.Example = `  # View all recent events
  datumctl activity events --start-time "now-7d"

  # View only warning events
  datumctl activity events --type Warning --start-time "now-7d"`
		case "feed":
			subCmd.Short = "View a combined activity feed across Datum Cloud resource types"
			subCmd.Long = `Display a combined stream of activity across all resource types,
merging audit events, resource events, and change history into a single feed.

Use --change-source to filter by the origin of changes (e.g., human, system).`
			subCmd.Example = `  # View the combined activity feed
  datumctl activity feed

  # Filter to human-initiated changes only
  datumctl activity feed --change-source human`
		case "history":
			subCmd.Short = "View the change history for a specific Datum Cloud resource"
			subCmd.Long = `Display the change history for a specific Datum Cloud resource,
showing a chronological log of modifications made to its spec over time.

Specify the resource type and name to view its history.`
			subCmd.Example = `  # View change history for a project
  datumctl activity history projects my-project-id

  # View change history with diffs between versions
  datumctl activity history projects my-project-id --diff`
		}
	}
	rootCmd.AddCommand(activityCmd)

	docsCmd := docs.Command(rootCmd)
	docsCmd.GroupID = "other"
	rootCmd.AddCommand(docsCmd)

	return rootCmd
}
