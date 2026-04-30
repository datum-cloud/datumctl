package data

import (
	"context"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ActivityLoadedMsg carries the result of a ListActivity call.
type ActivityLoadedMsg struct {
	Rows         []ActivityRow
	NextContinue string
	IsFirstPage  bool // true → replace existing rows; false → append
	Err          error
	Unauthorized bool
}

// ProjectActivityLoadedMsg carries the result of a ListRecentProjectActivity call.
type ProjectActivityLoadedMsg struct {
	Rows []ActivityRow
}

// ProjectActivityErrorMsg carries a project-activity load failure.
type ProjectActivityErrorMsg struct {
	Err          error
	Unauthorized bool
}

type ResourceTypesLoadedMsg struct{ Types []ResourceType }
type ResourcesLoadedMsg struct {
	Rows         []ResourceRow
	ResourceType ResourceType
	Columns      []string // display column names from table API
}
type DescribeResultMsg struct {
	Content string
	Raw     *unstructured.Unstructured
}
type LoadErrorMsg struct {
	Err      error
	Severity ErrorSeverity // set by the cmd that produced the error; zero-value is Warning
}
type TickMsg struct{}

// HintClearMsg signals the status bar to clear any pending transient hint.
// The Token must match StatusBarModel.hintToken; mismatched tokens (stale ticks)
// are silently ignored so rapid hint re-posts don't clear each other prematurely.
type HintClearMsg struct {
	Token int
}

// HintClearCmd fires HintClearMsg{Token: token} after delay.
func HintClearCmd(token int, delay time.Duration) tea.Cmd {
	return tea.Tick(delay, func(time.Time) tea.Msg {
		return HintClearMsg{Token: token}
	})
}

// ClearStatusErrMsg signals the status bar to clear the error indicator.
// Token must match AppModel.statusErrToken; stale ticks are silently ignored.
type ClearStatusErrMsg struct{ Token int }

// ClearStatusErrCmd fires ClearStatusErrMsg{Token: token} after delay.
func ClearStatusErrCmd(token int, delay time.Duration) tea.Cmd {
	return tea.Tick(delay, func(time.Time) tea.Msg {
		return ClearStatusErrMsg{Token: token}
	})
}
type BucketsLoadedMsg struct {
	Buckets                      []AllowanceBucket
	SiblingEnumerationRestricted bool
}

// BucketsErrorMsg carries a bucket-load failure so landing and quota UIs can
// render a specific error state without ambiguating against other LoadErrorMsg
// sources.
type BucketsErrorMsg struct {
	Err          error
	Unauthorized bool
}

// BucketClient is implemented by KubeResourceClient and used by AppModel for
// quota dashboard operations.
type BucketClient interface {
	ListAllowanceBuckets(ctx context.Context) ([]AllowanceBucket, error)
	InvalidateBucketCache()
}

func LoadResourceTypesCmd(ctx context.Context, rc ResourceClient) tea.Cmd {
	return func() tea.Msg {
		types, err := rc.ListResourceTypes(ctx)
		if err != nil {
			return LoadErrorMsg{Err: err, Severity: SeverityOf(err, rc)}
		}
		return ResourceTypesLoadedMsg{Types: types}
	}
}

func LoadResourcesCmd(ctx context.Context, rc ResourceClient, rt ResourceType, ns string) tea.Cmd {
	return func() tea.Msg {
		rows, cols, err := rc.ListResources(ctx, rt, ns)
		if err != nil {
			return LoadErrorMsg{Err: err, Severity: SeverityOf(err, rc)}
		}
		return ResourcesLoadedMsg{Rows: rows, ResourceType: rt, Columns: cols}
	}
}

func DescribeResourceCmd(ctx context.Context, rc ResourceClient, rt ResourceType, name, ns string) tea.Cmd {
	return func() tea.Msg {
		result, err := rc.DescribeResource(ctx, rt, name, ns)
		if err != nil {
			return LoadErrorMsg{Err: err, Severity: SeverityOf(err, rc)}
		}
		return DescribeResultMsg{Content: result.Content, Raw: result.Raw}
	}
}

func TickCmd() tea.Cmd {
	return tea.Tick(15*time.Second, func(t time.Time) tea.Msg {
		return TickMsg{}
	})
}

func LoadBucketsCmd(ctx context.Context, bc BucketClient) tea.Cmd {
	return func() tea.Msg {
		buckets, err := bc.ListAllowanceBuckets(ctx)
		if err != nil {
			return BucketsErrorMsg{
				Err:          err,
				Unauthorized: k8serrors.IsForbidden(err) || k8serrors.IsUnauthorized(err),
			}
		}
		return BucketsLoadedMsg{Buckets: buckets}
	}
}

// LoadResourceRegistrationsCmd fetches ResourceRegistration objects from the platform API.
// On error, surfaces fall back silently to short-name labels.
func LoadResourceRegistrationsCmd(ctx context.Context, rrc ResourceRegistrationClient) tea.Cmd {
	return func() tea.Msg {
		regs, err := rrc.ListResourceRegistrations(ctx)
		if err != nil {
			return ResourceRegistrationsLoadedMsg{Err: err, Unauthorized: isRegistrationUnauthorized(err)}
		}
		return ResourceRegistrationsLoadedMsg{Registrations: regs}
	}
}

// LoadRecentProjectActivityCmd fetches project-scoped human activity within window.
func LoadRecentProjectActivityCmd(ctx context.Context, ac *ActivityClient, window time.Duration, limit int) tea.Cmd {
	return func() tea.Msg {
		rows, err := ac.ListRecentProjectActivity(ctx, window, limit)
		if err != nil {
			return ProjectActivityErrorMsg{Err: err, Unauthorized: ac.IsUnauthorized(err)}
		}
		return ProjectActivityLoadedMsg{Rows: rows}
	}
}

// LoadActivityCmd fetches a page of activity for a resource. continueToken
// empty = first page (cache eligible); non-empty = next page (cache bypass).
func LoadActivityCmd(ctx context.Context, ac *ActivityClient, apiGroup, kind, name, namespace, continueToken string) tea.Cmd {
	isFirst := continueToken == ""
	return func() tea.Msg {
		rows, next, err := ac.ListActivity(ctx, apiGroup, kind, name, namespace, continueToken)
		if err != nil {
			return ActivityLoadedMsg{
				IsFirstPage:  isFirst,
				Err:          err,
				Unauthorized: ac.IsUnauthorized(err),
			}
		}
		return ActivityLoadedMsg{
			Rows:         rows,
			NextContinue: next,
			IsFirstPage:  isFirst,
		}
	}
}

// DeleteTarget identifies a resource to delete, carrying enough information to
// construct the Kubernetes GVR and display the confirmation dialog.
type DeleteTarget struct {
	RT        ResourceType
	Name      string
	Namespace string
}

// OpenDeleteConfirmationMsg is returned by OpenDeleteConfirmationCmd to ask
// AppModel to open the delete confirmation dialog for the given target.
type OpenDeleteConfirmationMsg struct {
	Target DeleteTarget
}

// OpenDeleteConfirmationCmd asks AppModel to open the delete confirmation dialog.
func OpenDeleteConfirmationCmd(target DeleteTarget) tea.Cmd {
	return func() tea.Msg { return OpenDeleteConfirmationMsg{Target: target} }
}

// DeleteResourceSucceededMsg is returned when DeleteResource completes with no error.
type DeleteResourceSucceededMsg struct {
	Target DeleteTarget
}

// DeleteResourceFailedMsg is returned when DeleteResource returns an error.
// The Forbidden / NotFound / Conflict flags are pre-classified for dialog routing.
type DeleteResourceFailedMsg struct {
	Target    DeleteTarget
	Err       error
	Forbidden bool
	NotFound  bool
	Conflict  bool
}

// DeleteResourceCmd calls ResourceClient.DeleteResource and returns a success or
// failure message. The returned message is always handled by AppModel.Update.
func DeleteResourceCmd(ctx context.Context, rc ResourceClient, target DeleteTarget) tea.Cmd {
	return func() tea.Msg {
		err := rc.DeleteResource(ctx, target.RT, target.Name, target.Namespace)
		if err == nil {
			return DeleteResourceSucceededMsg{Target: target}
		}
		return DeleteResourceFailedMsg{
			Target:    target,
			Err:       err,
			Forbidden: rc.IsForbidden(err),
			NotFound:  rc.IsNotFound(err),
			Conflict:  rc.IsConflict(err),
		}
	}
}

// TreeBuckets holds the classified result of a parent+child bucket pair for
// a single resource type. HasTree is true only when both parent (org-scoped)
// and activeChild (project-scoped matching active consumer) are present.
type TreeBuckets struct {
	Parent      *AllowanceBucket
	ActiveChild *AllowanceBucket
	Siblings    []AllowanceBucket // children whose consumer != active consumer
	HasTree     bool
}

// ClassifyTreeBuckets classifies a pre-filtered (same ResourceType) slice of
// AllowanceBuckets into parent (org-scoped), active child, and sibling children.
// HasTree is true only when both parent AND activeChild are identified.
func ClassifyTreeBuckets(buckets []AllowanceBucket, activeConsumerKind, activeConsumerName string) TreeBuckets {
	var result TreeBuckets
	for i := range buckets {
		b := &buckets[i]
		if strings.EqualFold(b.ConsumerKind, "Organization") {
			if result.Parent == nil {
				result.Parent = b
			}
		} else if activeConsumerKind != "" &&
			strings.EqualFold(b.ConsumerKind, activeConsumerKind) &&
			b.ConsumerName == activeConsumerName {
			result.ActiveChild = b
		}
	}
	if result.Parent != nil && result.ActiveChild != nil {
		result.HasTree = true
		result.Siblings = FindSiblingBuckets(buckets, *result.Parent, activeConsumerKind, activeConsumerName)
	}
	return result
}

// FindSiblingBuckets returns buckets governing the same resource type as parent
// whose ConsumerKind differs from parent's ConsumerKind AND whose
// (ConsumerKind, ConsumerName) is not the active consumer.
//
// Returns an empty (non-nil) slice when no siblings are found or parent is zero.
func FindSiblingBuckets(
	buckets []AllowanceBucket,
	parent AllowanceBucket,
	activeConsumerKind, activeConsumerName string,
) []AllowanceBucket {
	if parent.ResourceType == "" {
		return []AllowanceBucket{}
	}
	out := []AllowanceBucket{}
	for _, b := range buckets {
		if b.ResourceType != parent.ResourceType {
			continue
		}
		if strings.EqualFold(b.ConsumerKind, parent.ConsumerKind) {
			continue
		}
		if strings.EqualFold(b.ConsumerKind, activeConsumerKind) && b.ConsumerName == activeConsumerName {
			continue
		}
		out = append(out, b)
	}
	return out
}

// FindBucketsForResource returns buckets whose spec.resourceType matches
// "<group>/<resource>" (or just "<resource>" for the core group).
// Multiple buckets may apply (e.g. project + org scoped).
func FindBucketsForResource(buckets []AllowanceBucket, group, resource string) []AllowanceBucket {
	rt := group + "/" + resource
	if group == "" {
		rt = resource
	}
	var out []AllowanceBucket
	for _, b := range buckets {
		if b.ResourceType == rt {
			out = append(out, b)
		}
	}
	return out
}
