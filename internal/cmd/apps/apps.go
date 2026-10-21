// Package apps provides project template history and conditional authoring operations.
package apps

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/pmezard/go-difflib/difflib"
	"github.com/spf13/cobra"
	customerrors "go.datum.net/datumctl/internal/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/rest"
	"k8s.io/kubectl/pkg/util/templates"
)

const apiPath = "/apis/apps.miloapis.com/v1alpha1/"
const maxResponseBytes = 8 << 20
const maxTemplateRevisions = 1000
const maxContinueTokenBytes = 4096

type Options struct {
	// RESTConfig must resolve an authenticated project scope, never the platform or an organization.
	RESTConfig func() (*rest.Config, error)
	Out        io.Writer
}

type template struct {
	metav1.TypeMeta `json:",inline"`
	Metadata        metav1.ObjectMeta `json:"metadata"`
	Spec            templateSpec      `json:"spec"`
}

// templateSpec is the native exact-definition envelope. DefinitionJSON is
// treated as opaque bytes after JSON and digest validation; decoding the
// definition into interface{} would lose large integers and decimal spelling.
type templateSpec struct {
	DefinitionJSON string `json:"definitionJSON"`
	Digest         string `json:"digest"`
}

type revision struct {
	Metadata metav1.ObjectMeta `json:"metadata"`
	Spec     struct {
		TemplateUID       string `json:"templateUID"`
		Sequence          uint64 `json:"sequence"`
		Digest            string `json:"digest"`
		EffectiveActorUID string `json:"effectiveActorUID"`
		AcceptedAt        string `json:"acceptedAt"`
		DefinitionJSON    string `json:"definitionJSON"`
	} `json:"spec"`
}

type apiClient struct {
	base *url.URL
	http *http.Client
}

func newClient(o Options) (*apiClient, error) {
	cfg, err := o.RESTConfig()
	if err != nil {
		return nil, err
	}
	cfg = rest.CopyConfig(cfg)
	if cfg.Timeout == 0 || cfg.Timeout > 30*time.Second {
		cfg.Timeout = 30 * time.Second
	}
	base, err := url.Parse(cfg.Host)
	if err != nil || base.Host == "" || base.RawQuery != "" || base.Fragment != "" {
		return nil, customerrors.NewUserError("The project API address is invalid.")
	}
	h, err := rest.HTTPClientFor(cfg)
	if err != nil {
		return nil, customerrors.NewUserError("Could not initialize the authenticated project client.")
	}
	// A redirected request must not carry the configured authentication to another endpoint.
	h.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &apiClient{base: base, http: h}, nil
}

func (c *apiClient) request(ctx context.Context, method, resource, name string, body any, dry bool, out any) error {
	return c.requestWithQuery(ctx, method, resource, name, body, dry, nil, out)
}

func (c *apiClient) requestWithQuery(ctx context.Context, method, resource, name string, body any, dry bool, query url.Values, out any) error {
	u := *c.base
	u.Path = strings.TrimRight(u.Path, "/") + apiPath + resource
	if name != "" {
		u.Path += "/" + name
	}
	if dry || len(query) != 0 {
		values := make(url.Values, len(query)+1)
		for key, entries := range query {
			values[key] = append([]string(nil), entries...)
		}
		if dry {
			values.Set("dryRun", "All")
		}
		u.RawQuery = values.Encode()
	}
	var input io.Reader
	if body != nil {
		b, e := json.Marshal(body)
		if e != nil {
			return customerrors.NewUserError("Could not encode the template request.")
		}
		input = bytes.NewReader(b)
	}
	req, e := http.NewRequestWithContext(ctx, method, u.String(), input)
	if e != nil {
		return customerrors.NewUserError("Could not build the template request.")
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, e := c.http.Do(req)
	if e != nil {
		return customerrors.NewUserErrorWithHint("The template request could not be completed.", "Check project connectivity and permissions. A write may have completed; inspect the template before retrying.")
	}
	defer resp.Body.Close()
	raw, e := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if e != nil || len(raw) > maxResponseBytes {
		return responseFailure(method, "The template response could not be read within the supported size limit.")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		switch resp.StatusCode {
		case http.StatusConflict:
			return customerrors.NewUserErrorWithHint("The template changed or its name was reused.", "Inspect the current UID and resource version, then review the operation again. No automatic retry was made.")
		case http.StatusNotFound:
			return customerrors.NewUserErrorWithHint("The template API or selected resource is unavailable in this project.", "Check the project, template name and revision. For deleted template history, use --uid.")
		case http.StatusUnauthorized, http.StatusForbidden:
			return customerrors.NewUserErrorWithHint("Access to this template operation was denied.", "Check your signed-in account and project permissions.")
		case http.StatusGone:
			return customerrors.NewUserErrorWithHint("The selected history page expired.", "Run the history command again to start a fresh read.")
		default:
			if resp.StatusCode >= 500 {
				return responseFailure(method, fmt.Sprintf("The template API returned HTTP %d.", resp.StatusCode))
			}
			return customerrors.NewUserError(fmt.Sprintf("The template API rejected the request (HTTP %d).", resp.StatusCode))
		}
	}
	if e = json.Unmarshal(raw, out); e != nil {
		return responseFailure(method, "The template API returned an invalid response.")
	}
	return nil
}

type historyEntry struct {
	Name       string
	Sequence   uint64
	Digest     string
	ActorUID   string
	AcceptedAt string
}

func readRevisionHistory(ctx context.Context, c *apiClient, templateUID string) ([]historyEntry, error) {
	query := url.Values{"limit": {"1"}, "labelSelector": {"apps.miloapis.com/template-uid=" + templateUID}}
	seenTokens := make(map[string]struct{})
	seenNames := make(map[string]struct{})
	seenUIDs := make(map[string]struct{})
	seenSequences := make(map[uint64]struct{})
	items := make([]historyEntry, 0)
	for page := 0; ; page++ {
		if page >= maxTemplateRevisions {
			return nil, customerrors.NewUserError("Template history exceeds the supported 1000 revisions.")
		}
		var list struct {
			Items    []revision      `json:"items"`
			Metadata metav1.ListMeta `json:"metadata"`
		}
		if err := c.requestWithQuery(ctx, http.MethodGet, "apptemplaterevisions", "", nil, false, query, &list); err != nil {
			return nil, err
		}
		if len(list.Items) > 1 {
			return nil, customerrors.NewUserError("The history API exceeded the requested page size.")
		}
		for _, item := range list.Items {
			name, uid := item.Metadata.Name, string(item.Metadata.UID)
			if item.Spec.TemplateUID != templateUID || !validName(name) || !validUID(uid) ||
				item.Spec.Sequence == 0 || item.Spec.Sequence > maxTemplateRevisions ||
				!validDefinition(item.Spec.DefinitionJSON, item.Spec.Digest) {
				return nil, customerrors.NewUserError("The history API returned an invalid revision identity or definition.")
			}
			if _, ok := seenNames[name]; ok {
				return nil, customerrors.NewUserError("The history API returned a duplicate revision identity.")
			}
			if _, ok := seenUIDs[uid]; ok {
				return nil, customerrors.NewUserError("The history API returned a duplicate revision identity.")
			}
			if _, ok := seenSequences[item.Spec.Sequence]; ok {
				return nil, customerrors.NewUserError("The history API returned a duplicate revision sequence.")
			}
			seenNames[name], seenUIDs[uid], seenSequences[item.Spec.Sequence] = struct{}{}, struct{}{}, struct{}{}
			// Keep only the fields that the history output needs. In particular,
			// do not retain definition strings, annotations, labels or managed fields
			// while reading up to 1000 pages.
			items = append(items, historyEntry{
				Name: name, Sequence: item.Spec.Sequence, Digest: item.Spec.Digest,
				ActorUID: item.Spec.EffectiveActorUID, AcceptedAt: item.Spec.AcceptedAt,
			})
		}
		if list.Metadata.Continue == "" {
			break
		}
		if len(list.Metadata.Continue) > maxContinueTokenBytes {
			return nil, customerrors.NewUserError("The history API returned an oversized continuation token.")
		}
		if _, ok := seenTokens[list.Metadata.Continue]; ok {
			return nil, customerrors.NewUserError("The history API repeated a continuation token.")
		}
		seenTokens[list.Metadata.Continue] = struct{}{}
		query.Set("continue", list.Metadata.Continue)
	}
	return items, nil
}

func responseFailure(method, message string) error {
	if method == http.MethodPut || method == http.MethodDelete {
		return customerrors.NewUserErrorWithHint("The template write outcome is unknown.", "Inspect the current template and history before retrying; the write may have completed. No automatic retry was made.")
	}
	return customerrors.NewUserError(message)
}
func writeResult(method string, result template, name, uid string) error {
	if result.Metadata.Name != name || string(result.Metadata.UID) != uid || result.Metadata.ResourceVersion == "" {
		return responseFailure(method, "The template response has incomplete identity.")
	}
	return nil
}

func validName(s string) bool { return len(validation.IsDNS1123Subdomain(s)) == 0 }
func validUID(s string) bool {
	return s != "" && len(validation.IsValidLabelValue(s)) == 0
}
func validResourceVersion(s string) bool {
	return s != "" && len(s) <= 256 && !strings.ContainsAny(s, "\x00\r\n")
}
func dryRun(s string) (bool, error) {
	if s != "none" && s != "server" {
		return false, customerrors.NewUserError("Use --dry-run=server or --dry-run=none.")
	}
	return s == "server", nil
}
func readTemplate(ctx context.Context, c *apiClient, name string) (template, error) {
	var t template
	if !validName(name) {
		return t, customerrors.NewUserError("Use a valid template name.")
	}
	if err := c.request(ctx, http.MethodGet, "apptemplates", name, nil, false, &t); err != nil {
		return t, err
	}
	if t.Metadata.Name != name || !validUID(string(t.Metadata.UID)) || t.Metadata.ResourceVersion == "" || !validDefinition(t.Spec.DefinitionJSON, t.Spec.Digest) {
		return t, customerrors.NewUserError("The template response has incomplete identity or definition.")
	}
	return t, nil
}
func readRevision(ctx context.Context, c *apiClient, name, uid string) (revision, error) {
	var r revision
	if !validName(name) {
		return r, customerrors.NewUserError("Use the revision name shown by template history.")
	}
	if err := c.request(ctx, http.MethodGet, "apptemplaterevisions", name, nil, false, &r); err != nil {
		return r, err
	}
	if r.Metadata.Name != name || r.Spec.TemplateUID != uid || !validDefinition(r.Spec.DefinitionJSON, r.Spec.Digest) {
		return r, customerrors.NewUserError("The revision identity, definition or digest is invalid for the current template UID.")
	}
	return r, nil
}

// validDefinition checks the exact bytes served by the API. It never parses
// numbers into float64 and deliberately exposes no response content in errors.
func validDefinition(definitionJSON, digest string) bool {
	if len(definitionJSON) == 0 || len(definitionJSON) > maxResponseBytes || !json.Valid([]byte(definitionJSON)) {
		return false
	}
	if len(digest) != len("sha256:")+sha256.Size*2 || !strings.HasPrefix(digest, "sha256:") {
		return false
	}
	encoded := digest[len("sha256:"):]
	if _, err := hex.DecodeString(encoded); err != nil || strings.ToLower(encoded) != encoded {
		return false
	}
	sum := sha256.Sum256([]byte(definitionJSON))
	return digest == "sha256:"+hex.EncodeToString(sum[:])
}

// Command constructs commands without touching credentials or the network until execution.
func Command(o Options) *cobra.Command {
	root := &cobra.Command{Use: "apps", Short: "Manage project app templates", Long: templates.LongDesc(`Inspect project template history and restore or delete an authoring template. App installation and publication are not available through these commands.`)}
	group := &cobra.Command{Use: "templates", Short: "Inspect and change project template authoring"}
	root.AddCommand(group)
	group.AddCommand(historyCommand(o), revisionCommand(o, false), revisionCommand(o, true), deleteCommand(o))
	return root
}
func historyCommand(o Options) *cobra.Command {
	var uid string
	cmd := &cobra.Command{Use: "history [NAME]", Short: "List immutable revisions of one template UID", Args: cobra.MaximumNArgs(1), Example: templates.Examples(`datumctl apps templates history web-api --project my-project
 datumctl apps templates history --uid TEMPLATE_UID --project my-project`), RunE: func(cmd *cobra.Command, args []string) error {
		if (len(args) == 0) == (uid == "") {
			return customerrors.NewUserError("Provide a template name or --uid, but not both.")
		}
		if uid != "" && !validUID(uid) {
			return customerrors.NewUserError("Use a valid template UID.")
		}
		c, e := newClient(o)
		if e != nil {
			return e
		}
		selected := uid
		if len(args) == 1 {
			t, e := readTemplate(cmd.Context(), c, args[0])
			if e != nil {
				return e
			}
			selected = string(t.Metadata.UID)
		}
		list, e := readRevisionHistory(cmd.Context(), c, selected)
		if e != nil {
			return e
		}
		sort.Slice(list, func(i, j int) bool { return list[i].Sequence < list[j].Sequence })
		rows := make([]map[string]any, 0)
		for _, r := range list {
			rows = append(rows, map[string]any{"name": r.Name, "sequence": r.Sequence, "digest": r.Digest, "actorUID": r.ActorUID, "acceptedAt": r.AcceptedAt})
		}
		return json.NewEncoder(o.Out).Encode(map[string]any{"templateUID": selected, "revisions": rows})
	}}
	cmd.Flags().StringVar(&uid, "uid", "", "Read retained history by template UID, including after deletion")
	return cmd
}
func revisionCommand(o Options, restore bool) *cobra.Command {
	var revisionName, dry string
	verb, short := "diff", "Compare the current template with an immutable revision"
	if restore {
		verb, short = "restore", "Restore a revision as a new authoring revision"
	}
	cmd := &cobra.Command{Use: verb + " NAME", Short: short, Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		preview, e := dryRun(dry)
		if e != nil {
			return e
		}
		if revisionName == "" {
			return customerrors.NewUserError("Choose --revision from the template history.")
		}
		c, e := newClient(o)
		if e != nil {
			return e
		}
		current, e := readTemplate(cmd.Context(), c, args[0])
		if e != nil {
			return e
		}
		r, e := readRevision(cmd.Context(), c, revisionName, string(current.Metadata.UID))
		if e != nil {
			return e
		}
		if !restore {
			var a, b bytes.Buffer
			if e = json.Indent(&a, []byte(current.Spec.DefinitionJSON), "", "  "); e != nil {
				return e
			}
			if e = json.Indent(&b, []byte(r.Spec.DefinitionJSON), "", "  "); e != nil {
				return e
			}
			diff, e := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{A: difflib.SplitLines(a.String() + "\n"), B: difflib.SplitLines(b.String() + "\n"), FromFile: "current", ToFile: "selected revision", Context: 3})
			if e != nil {
				return e
			}
			_, e = io.WriteString(o.Out, diff)
			return e
		}
		// Preserve the exact current UID/version. A concurrent update or name reuse fails once.
		current.Spec = templateSpec{DefinitionJSON: r.Spec.DefinitionJSON, Digest: r.Spec.Digest}
		var result template
		if e = c.request(cmd.Context(), http.MethodPut, "apptemplates", args[0], current, preview, &result); e != nil {
			return e
		}
		if e = writeResult(http.MethodPut, result, args[0], string(current.Metadata.UID)); e != nil {
			return e
		}
		return summary(o.Out, result, preview, "restored")
	}}
	cmd.Flags().StringVar(&revisionName, "revision", "", "Exact immutable revision name from history")
	dry = "none"
	if restore {
		cmd.Flags().StringVar(&dry, "dry-run", "none", "Use server to validate without saving a new revision")
	}
	cmd.Example = templates.Examples("datumctl apps templates " + verb + " web-api --revision TEMPLATE_UID.REVISION_UID --project my-project")
	return cmd
}
func deleteCommand(o Options) *cobra.Command {
	var uid, version, dry string
	cmd := &cobra.Command{Use: "delete NAME", Short: "Delete authoring while retaining immutable revision history", Args: cobra.ExactArgs(1), Long: templates.LongDesc(`Delete the exact template UID and resource version you inspected. The name becomes reusable; immutable history remains. A server dry-run validates the request without reserving the template against later changes.`), RunE: func(cmd *cobra.Command, args []string) error {
		preview, e := dryRun(dry)
		if e != nil {
			return e
		}
		if !validName(args[0]) || !validUID(uid) || !validResourceVersion(version) {
			return customerrors.NewUserError("Provide the inspected template name, --uid and --resource-version.")
		}
		c, e := newClient(o)
		if e != nil {
			return e
		}
		typedUID := types.UID(uid)
		options := metav1.DeleteOptions{TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "DeleteOptions"}, Preconditions: &metav1.Preconditions{UID: &typedUID, ResourceVersion: &version}}
		if preview {
			options.DryRun = []string{metav1.DryRunAll}
		}
		var result template
		// DELETE dry-run belongs in the body. This API rejects query options.
		if e = c.request(cmd.Context(), http.MethodDelete, "apptemplates", args[0], options, false, &result); e != nil {
			return e
		}
		if e = writeResult(http.MethodDelete, result, args[0], uid); e != nil {
			return e
		}
		return summary(o.Out, result, preview, "deleted")
	}}
	cmd.Flags().StringVar(&uid, "uid", "", "Exact UID of the inspected template")
	cmd.Flags().StringVar(&version, "resource-version", "", "Exact resource version of the inspected template")
	cmd.Flags().StringVar(&dry, "dry-run", "none", "Use server to validate without deleting authoring")
	cmd.Example = templates.Examples(`datumctl apps templates delete web-api --uid TEMPLATE_UID --resource-version VERSION --dry-run=server --project my-project`)
	return cmd
}
func summary(w io.Writer, t template, dry bool, action string) error {
	return json.NewEncoder(w).Encode(map[string]any{"name": t.Metadata.Name, "uid": t.Metadata.UID, "resourceVersion": t.Metadata.ResourceVersion, "generation": t.Metadata.Generation, "dryRun": dry, "action": action})
}
