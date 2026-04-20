package tui

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"go.datum.net/datumctl/internal/client"
	"go.datum.net/datumctl/internal/datumconfig"
	"go.datum.net/datumctl/internal/tui/components"
	tuictx "go.datum.net/datumctl/internal/tui/context"
	"go.datum.net/datumctl/internal/tui/data"
	"go.datum.net/datumctl/internal/tui/layout"
	"go.datum.net/datumctl/internal/tui/styles"
)

type PaneID int

const (
	NavPane PaneID = iota
	TablePane
	DetailPane
	QuotaDashboardPane
	ActivityPane
	HistoryPane
	ActivityDashboardPane // FB-016 — project-scope human-activity rollup
	DiffPane
)

type OverlayID int

const (
	NoOverlay OverlayID = iota
	CtxSwitcherOverlay
	HelpOverlayID
	DeleteConfirmationOverlay // FB-017 — delete with confirmation dialog
)

// DashboardOrigin stashes the pane and welcome-dashboard state at the moment
// a dashboard (QuotaDashboardPane / ActivityDashboardPane) is opened.
// Two-slot: one slot per dashboard. Cross-dashboard presses (3 from ActivityDashboardPane,
// 4 from QuotaDashboardPane) skip the stash-overwrite so the original non-dashboard entry
// pane is always the Esc destination, preventing cyclic bounce loops.
type DashboardOrigin struct {
	Pane          PaneID
	ShowDashboard bool // FB-041 welcome state; restore on exit
}

// dashboardOriginLabel converts a DashboardOrigin to a human-readable label for the [3]/[4] back hint.
func dashboardOriginLabel(origin DashboardOrigin) string {
	switch origin.Pane {
	case NavPane:
		if origin.ShowDashboard {
			return "welcome panel"
		}
		return "navigation"
	case TablePane:
		return "resource list"
	case DetailPane:
		return "detail view"
	case QuotaDashboardPane:
		return "quota dashboard"
	case ActivityDashboardPane:
		return "activity dashboard"
	default:
		return ""
	}
}

type AppModel struct {
	width, height    int
	activePane       PaneID
	overlay          OverlayID
	refreshing       bool
	preserveCursor   bool
	detailReturnPane PaneID // pane to return to on Esc from DetailPane
	tuiCtx           tuictx.TUIContext
	header           components.HeaderModel
	sidebar          components.NavSidebarModel
	table            components.ResourceTableModel
	banner           components.QuotaBannerModel
	detail           components.DetailViewModel
	quota            components.QuotaDashboardModel
	activity         components.ActivityViewModel
	history          components.HistoryViewModel
	diff             components.DiffViewModel
	ctxOverlay       components.CtxSwitcherModel
	filterBar        components.FilterBarModel
	statusBar        components.StatusBarModel
	helpOverlay      components.HelpOverlayModel
	activityDashboard            components.ActivityDashboardModel
	activityCRDAbsentThisSession bool
	activityRollupFetchedAt      time.Time
	loadState                    data.LoadState
	resourceTypes                []data.ResourceType
	resources        []data.ResourceRow
	tableColumns     []string
	tableTypeName    string
	buckets          []data.AllowanceBucket
	bucketLoading    bool
	describeContent  string // raw describe text; S3 quota block appended at display time
	describeRaw      *unstructured.Unstructured // raw object for YAML toggle (FB-009)
	yamlMode         bool                       // true = show raw YAML instead of formatted describe
	conditionsMode   bool                       // true = show conditions table instead of describe/yaml (FB-018)
	eventsMode       bool                       // true = show events table instead of describe/yaml/conditions (FB-019)
	events           []data.EventRow            // last fetched events for current resource
	eventsLoading    bool                       // true while LoadEventsCmd is in flight
	eventsErr        error                      // last error from ListEvents; nil on success
	describeRT       data.ResourceType
	// activity state
	activityAPIGroup  string
	activityKind      string   // rt.Kind ("Project") — used in CEL filter
	activityRTName    string   // rt.Name ("projects") — matches detail.ResourceKind()
	activityName      string
	activityNamespace string
	bucketSiblingsRestricted bool
	bucketErr                error
	bucketUnauthorized       bool
	registrations            []data.ResourceRegistration
	registrationsLoading     bool
	// history state
	historyRows             []data.HistoryRow
	currentHistoryManifests []map[string]any
	selectedRevIdx          int // 0-based index into currentHistoryManifests for DiffPane
	historyRT               data.ResourceType
	historyName             string
	historyNamespace        string
	// FB-005: error recovery state.
	showDashboard           bool // FB-041: show welcome panel while preserving loaded tableTypeName
	lastEntryViaQuickJump  bool // FB-072: set when quick-jump dispatches TablePane; cleared on sidebar interaction
	bucketsFetchedAt    time.Time // FB-043: time of last successful bucket fetch; zero until first success
	loadErr             error     // last error from LoadErrorMsg; used for in-pane card
	lastFailedFetchKind string // "tableList" | "describe"; determines redispatchLastFetch target
	statusErrToken      int    // bumped per LoadErrorMsg to expire stale ClearStatusErrCmd ticks

	factory                 *client.DatumCloudFactory
	rc                      data.ResourceClient
	bc                      data.BucketClient // nil when rc doesn't implement BucketClient
	rrc                     data.ResourceRegistrationClient // nil in tests
	ac                      *data.ActivityClient
	hc                      *data.HistoryClient
	ctx                     context.Context

	// FB-017 delete confirmation state.
	deleteConfirmation   components.DeleteConfirmationModel
	pendingCursorAdvance bool // set on delete success; consumed by next ResourcesLoadedMsg
	deletedRowCursor     int  // cursor index at delete time; used by pendingCursorAdvance logic

	// FB-047 quota-loading key-queue state.
	pendingQuotaOpen bool // set when '3' pressed during loading; cleared on load complete/error/ctx switch

	quotaOriginPane    DashboardOrigin // FB-048/FB-087: stash before opening QuotaDashboard
	activityOriginPane DashboardOrigin // FB-048/FB-087: stash before opening ActivityDashboard
}

func NewAppModel(ctx context.Context, factory *client.DatumCloudFactory, tuiCtx tuictx.TUIContext) AppModel {
	krc := data.NewKubeResourceClient(factory)
	rrc := data.NewKubeResourceRegistrationClient(factory)
	var rc data.ResourceClient = krc
	var bc data.BucketClient = krc
	ac := data.NewActivityClient(factory)
	hc := data.NewHistoryClient(factory)
	m := AppModel{
		ctx:         ctx,
		factory:     factory,
		rc:          rc,
		bc:          bc,
		rrc:         rrc,
		ac:          ac,
		hc:          hc,
		tuiCtx:      tuiCtx,
		header:      components.NewHeaderModel(tuiCtx),
		sidebar:     components.NewNavSidebarModel(styles.SidebarWidth, 20),
		table:       components.NewResourceTableModel(40, 20),
		banner:      components.NewQuotaBannerModel(40),
		detail:            components.NewDetailViewModel(80, 20),
		quota:             components.NewQuotaDashboardModel(40, 20, tuiCtx.ProjectName+" (proj)"),
		activity:          components.NewActivityViewModel(80, 20),
		activityDashboard: components.NewActivityDashboardModel(40, 20, tuiCtx.ProjectName),
		history:     components.NewHistoryViewModel(80, 20),
		diff:        components.NewDiffViewModel(80, 20),
		ctxOverlay:  components.NewCtxSwitcherModel(tuiCtx.Config, 80, 24),
		filterBar:   components.NewFilterBarModel(),
		helpOverlay: components.NewHelpOverlayModel(),
		activePane:  NavPane,
	}
	m.updatePaneFocus()
	return m
}

func (m AppModel) Init() tea.Cmd {
	cmds := []tea.Cmd{
		data.LoadResourceTypesCmd(m.ctx, m.rc),
		data.TickCmd(),
		m.table.Init(),
	}
	if m.bc != nil {
		cmds = append(cmds, data.LoadBucketsCmd(m.ctx, m.bc))
	}
	if m.rrc != nil {
		cmds = append(cmds, data.LoadResourceRegistrationsCmd(m.ctx, m.rrc))
	}
	// FB-067: dispatch activity rollup on startup when a project context is active.
	// FB-082: gate SetActivityLoading(true) inside the same condition so first View() shows loading state.
	if m.ac != nil && m.tuiCtx.ActiveCtx != nil && m.tuiCtx.ActiveCtx.ProjectID != "" {
		m.table.SetActivityLoading(true)
		cmds = append(cmds, data.LoadRecentProjectActivityCmd(m.ctx, m.ac, 24*time.Hour, 10))
	}
	return tea.Batch(cmds...)
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m = m.recalcLayout()
		return m, nil

	case tea.KeyMsg:
		if m.overlay != NoOverlay {
			return m.handleOverlayKey(msg, &cmds)
		}
		if m.filterBar.Focused() {
			return m.handleFilterKey(msg, &cmds)
		}
		return m.handleNormalKey(msg, &cmds)

	case data.ResourceTypesLoadedMsg:
		m.resourceTypes = msg.Types
		m.loadState = data.LoadStateIdle
		m.statusBar.Err = nil
		m = m.recalcLayout() // re-sizes sidebar to terminal width with types now known

	case data.OpenDeleteConfirmationMsg:
		m.deleteConfirmation = components.NewDeleteConfirmationModel(msg.Target, m.registrations)
		m.overlay = DeleteConfirmationOverlay
		m.statusBar.Mode = components.ModeOverlay
		return m, m.deleteConfirmation.Init()

	case data.DeleteResourceSucceededMsg:
		// Success: invalidate cache, set cursor-advance flag, close dialog, re-fetch.
		// This is also handled when the dialog was already dismissed (late arrival).
		m.rc.InvalidateResourceListCache(msg.Target.RT.Kind)
		m.pendingCursorAdvance = true
		m.deletedRowCursor = m.table.Cursor()
		if m.overlay == DeleteConfirmationOverlay {
			m.overlay = NoOverlay
			m.statusBar.Mode = components.ModeNormal
		}
		if rt, ok := m.sidebar.SelectedType(); ok {
			ns := ""
			if rt.Namespaced {
				ns = m.tuiCtx.Namespace
			}
			cmds = append(cmds, data.LoadResourcesCmd(m.ctx, m.rc, rt, ns))
		}

	case data.DeleteResourceFailedMsg:
		if msg.NotFound {
			// 404 is treated as success-equivalent (resource already gone).
			m.rc.InvalidateResourceListCache(msg.Target.RT.Kind)
			m.pendingCursorAdvance = true
			m.deletedRowCursor = m.table.Cursor()
			if m.overlay == DeleteConfirmationOverlay {
				m.overlay = NoOverlay
				m.statusBar.Mode = components.ModeNormal
			}
			if rt, ok := m.sidebar.SelectedType(); ok {
				ns := ""
				if rt.Namespaced {
					ns = m.tuiCtx.Namespace
				}
				cmds = append(cmds, data.LoadResourcesCmd(m.ctx, m.rc, rt, ns))
			}
		} else if m.overlay == DeleteConfirmationOverlay {
			// Update dialog state for the specific error type.
			switch {
			case msg.Forbidden:
				m.deleteConfirmation.SetState(components.DeleteStateForbidden)
			case msg.Conflict:
				m.deleteConfirmation.SetState(components.DeleteStateConflict)
			default:
				m.deleteConfirmation.SetState(components.DeleteStateTransientError)
				m.deleteConfirmation.SetErrorDetail(msg.Err.Error())
			}
		}
		// If overlay != DeleteConfirmationOverlay: discard error (late arrival, operator moved on).

	case data.ResourcesLoadedMsg:
		shouldPreserveCursor := m.preserveCursor
		m.refreshing = false
		m.preserveCursor = false
		m.tuiCtx.Refreshing = false
		m.resources = msg.Rows
		m.tableColumns = msg.Columns
		m.loadState = data.LoadStateIdle
		m.loadErr = nil
		m.table.SetLoadErr(nil, data.ErrorSeverityWarning)
		m.statusBar.Err = nil
		m.tuiCtx.LastRefresh = time.Now()
		m.tuiCtx.ResourceCount = len(msg.Rows)
		m.sidebar.SetCount(msg.ResourceType.Name, len(msg.Rows))
		m.header.Ctx = m.tuiCtx
		tableW := layout.TableWidth(m.width, layout.SidebarWidth(m.width))
		m.table.SetColumns(msg.Columns, tableW)
		if shouldPreserveCursor {
			m.table.RefreshRows(msg.Rows)
		} else {
			m.table.SetRows(msg.Rows)
			m.table.SetFilter("")
			m.filterBar.Blur()
		}
		// Cursor-advance after delete success: advance or clamp per spec §3e.
		// deletedRowCursor is the cursor index captured at delete-success time,
		// before this re-fetch's SetRows reset the cursor.
		if m.pendingCursorAdvance {
			priorCursor := m.deletedRowCursor
			newLen := len(msg.Rows)
			var newCursor int
			switch {
			case newLen == 0:
				newCursor = 0
			case priorCursor >= newLen:
				newCursor = newLen - 1
			default:
				newCursor = priorCursor
			}
			m.table.SetCursor(newCursor)
			m.pendingCursorAdvance = false
		}
		m.table.SetLoadState(data.LoadStateIdle)
		m.updateBanner()

	case data.DescribeResultMsg:
		m.describeContent = msg.Content
		m.describeRaw = msg.Raw
		m.detail.SetDescribeAvailable(true)
		// FB-005: clear describe-error state on successful describe.
		if m.loadState == data.LoadStateError && m.lastFailedFetchKind == "describe" {
			m.loadState = data.LoadStateIdle
			m.loadErr = nil
			m.statusBar.Err = nil
			m.statusBar.ErrSeverity = data.ErrorSeverityWarning
		}
		m.detail.SetMode(m.detailModeLabel())
		m.detail.SetContent(m.buildDetailContent())
		m.detail.SetLoading(false)

	case data.EventsLoadedMsg: // FB-019 // AC#24
		m.eventsLoading = false
		m.detail.SetEventsLoading(false) // FB-122
		if msg.Err != nil {
			m.eventsErr = msg.Err
			m.events = nil
		} else {
			m.eventsErr = nil
			m.events = msg.Events
			m.detail.SetEventsFetchedAt(time.Now()) // FB-025
		}
		var hintCmd tea.Cmd
		if m.eventsMode || m.loadState == data.LoadStateError || m.describeRaw == nil {
			m.detail.SetMode(m.detailModeLabel())
			m.detail.SetContent(m.buildDetailContent())
			// FB-053: signal the error→placeholder transition with a transient hint.
			if m.loadState == data.LoadStateError && m.lastFailedFetchKind == "describe" && msg.Err == nil {
				hintCmd = m.postHint("Events loaded — [E] to view")
			}
		}
		return m, hintCmd

	case data.HintClearMsg:
		m.statusBar.ClearHintIfToken(msg.Token)

	case data.ResourceRegistrationsLoadedMsg:
		m.registrationsLoading = false
		if msg.Err != nil {
			// Silent degradation — 403 or other error; leave m.registrations nil.
			break
		}
		m.registrations = msg.Registrations
		m.banner.SetRegistrations(m.registrations)
		m.quota.SetRegistrations(m.registrations)
		m.activityDashboard.SetRegistrations(m.registrations)
		// S3 (RenderQuotaBlock) picks up registrations at render time via buildDetailContent.
		if m.activePane == DetailPane {
			m.detail.SetMode(m.detailModeLabel())
			m.detail.SetContent(m.buildDetailContent())
		}

	case data.BucketsLoadedMsg:
		m.bucketLoading = false
		m.bucketErr = nil
		m.bucketUnauthorized = false
		// FB-071: clear status bar error set by BucketsErrorMsg on successful reload.
		m.statusBar.Err = nil
		m.statusBar.ErrSeverity = data.ErrorSeverityWarning
		m.refreshing = false
		m.tuiCtx.Refreshing = false
		m.header.Ctx = m.tuiCtx
		m.buckets = msg.Buckets
		m.bucketSiblingsRestricted = msg.SiblingEnumerationRestricted
		m.bucketsFetchedAt = time.Now()
		m.quota.SetLoading(false)
		m.quota.SetBuckets(msg.Buckets)
		m.quota.SetBucketFetchedAt(m.bucketsFetchedAt)
		ak, an := m.activeConsumer()
		m.quota.SetActiveConsumer(ak, an)
		m.quota.SetSiblingRestricted(m.bucketSiblingsRestricted)
		m.banner.SetActiveConsumer(ak, an)
		m.banner.SetSiblingRestricted(m.bucketSiblingsRestricted)
		m.updateBanner()
		m.table.SetAttentionItems(computeAttentionItems(m.buckets, ak, an, m.registrations)) // FB-042
		// If currently in DetailPane, re-render content with S3 quota block.
		if m.activePane == DetailPane {
			m.detail.SetMode(m.detailModeLabel())
			m.detail.SetContent(m.buildDetailContent())
		}
		// FB-078: no force-transition — operator confirms manually via '3'.
		if m.pendingQuotaOpen {
			m.pendingQuotaOpen = false
			m.table.SetPendingQuotaOpen(false)                               // FB-099: reset strip label
			m.statusBar.PostHint("Quota dashboard ready — press [3]")        // FB-097: persistent (no HintClearCmd)
			return m, nil
		}

	case data.BucketsErrorMsg:
		m.bucketLoading = false
		m.bucketErr = msg.Err
		m.bucketUnauthorized = msg.Unauthorized
		m.refreshing = false
		m.tuiCtx.Refreshing = false
		m.header.Ctx = m.tuiCtx
		// FB-071: propagate to global status bar so operator sees the error regardless of active pane.
		sev := data.ErrorSeverityWarning
		if msg.Unauthorized {
			sev = data.ErrorSeverityError
		}
		m.statusBar.Err = msg.Err
		m.statusBar.ErrSeverity = sev
		// FB-107: drop activePane gate — clear refreshing/loading unconditionally so
		// off-pane errors don't leave quota.refreshing=true on return to QuotaDashboardPane.
		m.quota.SetLoading(false)
		m.quota.SetLoadErr(msg.Err)
		// FB-047: clear pending open on error — no transition, error shown separately.
		// FB-077: also clear quota loading state to prevent re-queue loop on next '3' press.
		if m.pendingQuotaOpen {
			m.pendingQuotaOpen = false
			m.table.SetPendingQuotaOpen(false) // FB-099: reset strip label
			m.statusBar.Hint = ""
			m.quota.SetLoading(false)
			m.quota.SetLoadErr(msg.Err)
		}

	case data.HistoryLoadedMsg:
		// Validate the response matches the current describe target.
		if msg.Name != m.historyName || msg.Namespace != m.historyNamespace ||
			msg.APIGroup != m.historyRT.Group || msg.Kind != m.historyRT.Kind {
			break // stale response; discard
		}
		m.refreshing = false
		m.tuiCtx.Refreshing = false
		m.header.Ctx = m.tuiCtx
		if msg.Err != nil {
			m.history.SetError(msg.Err, msg.Unauthorized)
			break
		}
		m.historyRows = msg.Rows
		m.currentHistoryManifests = msg.Manifests
		m.history.SetRows(msg.Rows, msg.Truncated)

	case data.ActivityLoadedMsg:
		if msg.Err != nil {
			m.activity.SetError(msg.Err, msg.Unauthorized)
		} else if msg.IsFirstPage {
			m.activity.SetRows(msg.Rows, msg.NextContinue)
		} else {
			m.activity.AppendRows(msg.Rows, msg.NextContinue)
		}
		m.refreshing = false
		m.tuiCtx.Refreshing = false
		m.header.Ctx = m.tuiCtx

	case data.ProjectActivityLoadedMsg:
		m.activityDashboard.SetLoading(false)
		m.activityDashboard.SetRows(msg.Rows)
		m.table.SetActivityRows(msg.Rows) // FB-042: feed welcome teaser
		m.activityRollupFetchedAt = time.Now()

	case data.ProjectActivityErrorMsg:
		m.activityDashboard.SetLoading(false)
		isCRDAbsent := errors.Is(msg.Err, data.ErrActivityCRDAbsent) || errors.Is(msg.Err, data.ErrActivityCRDPartial)
		if isCRDAbsent {
			m.activityCRDAbsentThisSession = true
		}
		m.activityDashboard.SetLoadErr(msg.Err, msg.Unauthorized, isCRDAbsent)
		// FB-082: resolve S3 teaser out of loading state.
		// FB-100: only mark fetch failed when no stale rows exist — preserves data operator already had.
		m.table.SetActivityLoading(false)
		if m.table.ActivityRowCount() == 0 {
			m.table.SetActivityFetchFailed(true)
			m.table.SetActivityCRDAbsent(isCRDAbsent) // FB-102: gate recovery hint on transient-only
		}

	case components.NeedNextActivityPageMsg:
		if m.ac != nil && m.activity.NextContinue() != "" {
			m.activity.SetLoadingMore(true)
			return m, data.LoadActivityCmd(
				m.ctx, m.ac,
				m.activityAPIGroup, m.activityKind, m.activityName, m.activityNamespace,
				m.activity.NextContinue(),
			)
		}

	case data.LoadErrorMsg:
		m.refreshing = false
		m.preserveCursor = false
		m.bucketLoading = false
		m.tuiCtx.Refreshing = false
		m.header.Ctx = m.tuiCtx
		m.loadState = data.LoadStateError
		m.loadErr = msg.Err
		if m.activePane == DetailPane {
			m.lastFailedFetchKind = "describe"
		} else {
			m.lastFailedFetchKind = "tableList"
		}
		m.statusBar.Err = msg.Err
		m.statusBar.ErrSeverity = msg.Severity
		m.statusErrToken++
		cmds = append(cmds, data.ClearStatusErrCmd(m.statusErrToken, 10*time.Second))
		m.table.SetLoadErr(msg.Err, msg.Severity)
		m.table.SetLoadState(data.LoadStateError)
		if m.activePane == DetailPane {
			// Ensure Esc from describe-error card returns to TABLE (not NavPane zero-value).
			if m.detailReturnPane == NavPane {
				m.detailReturnPane = TablePane
			}
			// Populate viewport with error card so it renders immediately on LoadErrorMsg.
			m.detail.SetMode(m.detailModeLabel())
			m.detail.SetContent(m.buildDetailContent())
			m.detail.SetLoading(false)
		}
		// FB-107: drop activePane gate — clear refreshing/loading unconditionally.
		m.quota.SetLoading(false)
		m.quota.SetLoadErr(msg.Err)
		// FB-047: clear pending open on load error.
		// FB-077: also clear quota loading state to prevent re-queue loop on next '3' press.
		if m.pendingQuotaOpen {
			m.pendingQuotaOpen = false
			m.table.SetPendingQuotaOpen(false) // FB-099: reset strip label
			m.statusBar.Hint = ""
			m.quota.SetLoading(false)
			m.quota.SetLoadErr(msg.Err)
		}

	case data.ClearStatusErrMsg:
		if msg.Token == m.statusErrToken {
			// FB-135: if BucketsErrorMsg is still active, restore it instead of blanking —
			// a transient list-error clear must not erase the persistent bucket-health signal.
			if m.bucketErr != nil {
				sev := data.ErrorSeverityWarning
				if m.bucketUnauthorized {
					sev = data.ErrorSeverityError
				}
				m.statusBar.Err = m.bucketErr
				m.statusBar.ErrSeverity = sev
			} else {
				m.statusBar.Err = nil
				m.statusBar.ErrSeverity = data.ErrorSeverityWarning
			}
		}

	case data.TickMsg:
		if m.overlay == NoOverlay {
			switch m.activePane {
			case NavPane, TablePane:
				if rt, ok := m.sidebar.SelectedType(); ok {
					ns := ""
					if rt.Namespaced {
						ns = m.tuiCtx.Namespace
					}
					m.preserveCursor = true
					cmds = append(cmds, data.LoadResourcesCmd(m.ctx, m.rc, rt, ns))
				} else if m.tableTypeName == "" && m.bc != nil && !m.bucketLoading {
					// Welcome panel is showing — refresh platform-health summary (AC#20).
					cmds = append(cmds, data.LoadBucketsCmd(m.ctx, m.bc))
				}
			case QuotaDashboardPane:
				// Ticker refresh is intentionally silent (no SetRefreshing call).
				// Background cadence is ambient; operator-facing indicator fires only on [r] keypress.
				// See FB-108 for the design rationale.
				if m.bc != nil && !m.bucketLoading {
					cmds = append(cmds, data.LoadBucketsCmd(m.ctx, m.bc))
				}
			}
		}
		cmds = append(cmds, data.TickCmd())

	case components.ContextSwitchedMsg:
		m.refreshing = false
		m.preserveCursor = false
		m.bucketLoading = false
		m.bucketErr = nil
		m.bucketUnauthorized = false
		m.tuiCtx = msg.Ctx
		m.tuiCtx.Refreshing = false
		// AC#19: dismiss any open delete dialog on context switch.
		if m.overlay == DeleteConfirmationOverlay {
			m.overlay = NoOverlay
			m.deleteConfirmation = components.DeleteConfirmationModel{}
		}
		m.resources = nil
		m.tableColumns = nil
		m.tableTypeName = ""
		m.buckets = nil
		m.bucketSiblingsRestricted = false
		m.registrations = nil
		m.registrationsLoading = false
		m.statusBar.Hint = ""
		m.statusBar.BumpHintToken()
		m.pendingQuotaOpen = false              // FB-047: discard queued open on context switch
		m.table.SetPendingQuotaOpen(false)      // FB-099: reset strip label
		m.quotaOriginPane    = DashboardOrigin{} // FB-087: clear both stash slots on context switch
		m.activityOriginPane = DashboardOrigin{}
		m.quota.SetOriginLabel("")
		m.activityDashboard.SetOriginLabel("")
		m.describeContent = ""
		m.describeRaw = nil
		m.detail.SetDescribeAvailable(false)
		m.yamlMode = false
		m.conditionsMode = false // AC#5
		m.eventsMode = false     // AC#5 FB-019
		m.events = nil
		m.eventsLoading = false
		m.detail.SetEventsLoading(false) // FB-122
		m.eventsErr = nil
		m.detail.SetEventsFetchedAt(time.Time{}) // FB-025
		m.detail.SetMode("")
		m.activityAPIGroup = ""
		m.activityKind = ""
		m.activityRTName = ""
		m.activityName = ""
		m.activityNamespace = ""
		if m.bc != nil {
			m.bc.InvalidateBucketCache()
		}
		if m.rrc != nil {
			m.rrc.InvalidateRegistrationCache()
		}
		if m.ac != nil {
			m.ac.Invalidate() // clears both per-resource and project-activity caches
		}
		m.activityCRDAbsentThisSession = false
		m.activityRollupFetchedAt = time.Time{}
		m.activityDashboard.ClearCRDAbsentFlag()
		if m.hc != nil {
			m.hc.Invalidate()
		}
		m.activity.Reset()
		m.history.Reset()
		m.diff.Reset()
		m.historyRows = nil
		m.currentHistoryManifests = nil
		m.selectedRevIdx = 0
		m.historyRT = data.ResourceType{}
		m.historyName = ""
		m.historyNamespace = ""
		m.banner.SetBuckets(nil)
		m.banner.SetActiveConsumer("", "")
		m.banner.SetSiblingRestricted(false)
		m.banner.SetRegistrations(nil)
		m.header = components.NewHeaderModel(msg.Ctx)
		m.quota.SetBuckets(nil)
		m.quota.SetActiveConsumer("", "")
		m.quota.SetSiblingRestricted(false)
		m.quota.SetRegistrations(nil)
		m.quota.SetRefreshing(false) // FB-107: clear in-flight refresh indicator on context switch
		m.quota.ResetGrouping()
		m.ctxOverlay = components.NewCtxSwitcherModel(msg.Ctx.Config, m.width, m.height)
		m.overlay = NoOverlay
		m.activePane = NavPane
		m.statusBar.Mode = components.ModeNormal
		// FB-005: clear error state on context switch (AC#13).
		m.loadState = data.LoadStateIdle
		m.loadErr = nil
		m.lastFailedFetchKind = ""
		m.statusBar.Err = nil
		m.statusBar.ErrSeverity = data.ErrorSeverityWarning
		m.table.SetLoadErr(nil, data.ErrorSeverityWarning)
		m.table.SetLoadState(data.LoadStateIdle)
		m.showDashboard = false
		m.table.SetForceDashboard(false)
		m.bucketsFetchedAt = time.Time{}
		m.quota.SetBucketFetchedAt(time.Time{})
		m.table.SetActivityRows(nil)    // FB-042: clear welcome teaser on context switch
		// FB-082: SetActivityLoading moved into dispatch gate below (Bug #2 fix).
		m.table.SetAttentionItems(nil)  // FB-042
		m.updatePaneFocus()
		cmds = append(cmds, data.LoadResourceTypesCmd(m.ctx, m.rc))
		if m.bc != nil {
			m.bucketLoading = true
			cmds = append(cmds, data.LoadBucketsCmd(m.ctx, m.bc))
		}
		if m.rrc != nil {
			m.registrationsLoading = true
			cmds = append(cmds, data.LoadResourceRegistrationsCmd(m.ctx, m.rrc))
		}
		// FB-067: dispatch activity rollup on context switch when a project context is active.
		// FB-082: gate SetActivityLoading(true) inside dispatch; else-branch resolves to no-data (org scope).
		if m.ac != nil && m.tuiCtx.ActiveCtx != nil && m.tuiCtx.ActiveCtx.ProjectID != "" {
			m.table.SetActivityLoading(true)
			cmds = append(cmds, data.LoadRecentProjectActivityCmd(m.ctx, m.ac, 24*time.Hour, 10))
		} else {
			m.table.SetActivityLoading(false)
			m.table.SetActivityRows([]data.ActivityRow{})
		}
	}

	var cmd tea.Cmd
	m.history, cmd = m.history.Update(msg)
	cmds = append(cmds, cmd)
	m.diff, cmd = m.diff.Update(msg)
	cmds = append(cmds, cmd)
	m.sidebar, cmd = m.sidebar.Update(msg)
	if rt, ok := m.sidebar.SelectedType(); ok {
		m.table.SetHoveredType(rt)
	}
	m.table, cmd = m.table.Update(msg)
	cmds = append(cmds, cmd)
	m.quota, cmd = m.quota.Update(msg)
	cmds = append(cmds, cmd)
	m.activity, cmd = m.activity.Update(msg)
	cmds = append(cmds, cmd)
	m.activityDashboard, cmd = m.activityDashboard.Update(msg)
	cmds = append(cmds, cmd)
	if m.refreshing {
		frame := m.table.SpinnerFrame()
		if m.activePane == QuotaDashboardPane {
			frame = m.quota.SpinnerFrame()
		}
		m.tuiCtx.SpinnerFrame = frame
		m.header.Ctx = m.tuiCtx
	}
	m.detail, cmd = m.detail.Update(msg)
	cmds = append(cmds, cmd)
	m.filterBar, cmd = m.filterBar.Update(msg)
	cmds = append(cmds, cmd)
	if m.overlay == DeleteConfirmationOverlay {
		m.deleteConfirmation, cmd = m.deleteConfirmation.Update(msg)
		cmds = append(cmds, cmd)
	}

	m.refreshLandingInputs()

	return m, tea.Batch(cmds...)
}

// refreshLandingInputs pushes AppModel's landing-screen state onto the
// ResourceTableModel so welcomePanel() renders from current data. Safe to call
// unconditionally; ResourceTableModel only consults these fields when no type
// is selected.
func (m *AppModel) refreshLandingInputs() {
	m.table.SetTUIContext(m.tuiCtx)
	m.table.SetBuckets(m.buckets)
	m.table.SetBucketLoading(m.bucketLoading)
	m.table.SetBucketErr(m.bucketErr, m.bucketUnauthorized)
	m.table.SetBucketConfigured(m.bc != nil) // FB-074: disambiguate unconfigured vs. zero-governed
	m.table.SetRegistrations(m.registrations)
	show, age := staleContextAgeDisplay(m.tuiCtx.Config, time.Now())
	m.table.SetStaleCacheAge(show, age)
}

// staleContextAgeDisplay reports whether the landing banner should warn about
// a stale context cache and returns a human-readable age (e.g. "3d", "31h").
// Returns (false, "") when the cache is fresh, never refreshed, or config is
// unavailable.
func staleContextAgeDisplay(cfg *datumconfig.ConfigV1Beta1, now time.Time) (bool, string) {
	if cfg == nil || cfg.Cache.LastRefreshed == nil {
		return false, ""
	}
	age := now.Sub(*cfg.Cache.LastRefreshed)
	if age <= 24*time.Hour {
		return false, ""
	}
	if days := int(age / (24 * time.Hour)); days >= 1 {
		return true, fmt.Sprintf("%dd", days)
	}
	return true, fmt.Sprintf("%dh", int(age/time.Hour))
}

// openDiffForRow transitions to DiffPane showing the diff for the given row
// at manifest index manifestIdx (0-based, oldest-first).
func (m *AppModel) openDiffForRow(row data.HistoryRow, manifestIdx int) {
	if len(m.currentHistoryManifests) == 0 {
		return
	}
	body, isCreation, predMissing, err := m.hc.ComputeDiff(m.currentHistoryManifests, manifestIdx)
	if err != nil {
		body = "— could not compute diff: " + err.Error() + " —"
	}
	colorized := styles.ColorizeDiff(body)

	var prev *data.HistoryRow
	if !isCreation && !predMissing && manifestIdx > 0 && manifestIdx-1 < len(m.historyRows) {
		p := m.historyRows[manifestIdx-1]
		prev = &p
	}

	m.diff.SetRevision(row, prev, colorized, isCreation, predMissing)
	m.activePane = DiffPane
	m.updatePaneFocus()
}

// openDiffAtSelected recomputes the diff at selectedRevIdx and updates DiffPane.
// Called by [ and ] handlers.
func (m *AppModel) openDiffAtSelected() {
	if len(m.currentHistoryManifests) == 0 || m.selectedRevIdx >= len(m.historyRows) {
		return
	}
	body, isCreation, predMissing, err := m.hc.ComputeDiff(m.currentHistoryManifests, m.selectedRevIdx)
	if err != nil {
		body = "— could not compute diff: " + err.Error() + " —"
	}
	colorized := styles.ColorizeDiff(body)

	row := m.historyRows[m.selectedRevIdx]
	var prev *data.HistoryRow
	if !isCreation && !predMissing && m.selectedRevIdx > 0 {
		p := m.historyRows[m.selectedRevIdx-1]
		prev = &p
	}

	m.diff.SetRevision(row, prev, colorized, isCreation, predMissing)
}

// redispatchLastFetch re-issues the fetch that most recently produced a LoadErrorMsg.
// Called from the FB-005 error-state retry handler.
func (m *AppModel) redispatchLastFetch() tea.Cmd {
	switch m.lastFailedFetchKind {
	case "describe":
		ns := ""
		if m.describeRT.Namespaced {
			ns = m.tuiCtx.Namespace
		}
		m.detail.SetLoading(true)
		return data.DescribeResourceCmd(m.ctx, m.rc, m.describeRT, m.detail.ResourceName(), ns)
	default: // "tableList"
		rt, ok := m.sidebar.SelectedType()
		if !ok {
			return nil
		}
		ns := ""
		if rt.Namespaced {
			ns = m.tuiCtx.Namespace
		}
		m.table.SetLoadState(data.LoadStateLoading)
		return data.LoadResourcesCmd(m.ctx, m.rc, rt, ns)
	}
}

// postHint stamps the status bar with a transient hint and returns a Cmd that
// fires HintClearMsg{Token} after 3 seconds.
func (m *AppModel) postHint(text string) tea.Cmd {
	token := m.statusBar.PostHint(text)
	return data.HintClearCmd(token, 3*time.Second)
}

func (m *AppModel) updatePaneFocus() {
	m.sidebar.SetFocused(m.activePane == NavPane)
	m.table.SetFocused(m.activePane == TablePane)
	m.table.SetNavPaneFocused(m.activePane == NavPane)
	m.detail.SetFocused(m.activePane == DetailPane)
	m.quota.SetFocused(m.activePane == QuotaDashboardPane)
	m.activity.SetFocused(m.activePane == ActivityPane)
	m.history.SetFocused(m.activePane == HistoryPane)
	m.activityDashboard.SetFocused(m.activePane == ActivityDashboardPane)
	m.diff.SetFocused(m.activePane == DiffPane)
	// AC#17: show x — delete hint only on panes where x is active (per AC#10).
	m.helpOverlay.ShowDeleteHint = m.activePane == TablePane || m.activePane == DetailPane
	m.helpOverlay.ShowConditionsHint = m.activePane == DetailPane // AC#22
	m.helpOverlay.ShowEventsHint = m.activePane == DetailPane     // AC#26 FB-019

	switch m.activePane {
	case NavPane:
		if m.showDashboard {
			m.statusBar.Pane = "NAV_DASHBOARD"
		} else {
			m.statusBar.Pane = "NAV"
		}
		m.tuiCtx.ActivePaneLabel = "NAV"
	case TablePane:
		m.statusBar.Pane = "TABLE"
		m.tuiCtx.ActivePaneLabel = "TABLE"
	case DetailPane:
		m.statusBar.Pane = "DETAIL"
		m.tuiCtx.ActivePaneLabel = "DETAIL"
	case QuotaDashboardPane:
		m.statusBar.Pane = "QUOTA"
		m.tuiCtx.ActivePaneLabel = "QUOTA"
	case ActivityPane:
		m.statusBar.Pane = "ACTIVITY"
		m.tuiCtx.ActivePaneLabel = "ACTIVITY"
	case HistoryPane:
		m.statusBar.Pane = "HISTORY"
		m.tuiCtx.ActivePaneLabel = "HISTORY"
		m.statusBar.Mode = components.ModeDetail
	case ActivityDashboardPane:
		m.statusBar.Pane = "ACTIVITY"
		m.tuiCtx.ActivePaneLabel = "ACTIVITY"
	case DiffPane:
		m.statusBar.Pane = "DIFF"
		m.tuiCtx.ActivePaneLabel = "DIFF"
		m.statusBar.Mode = components.ModeDetail
	}
	// FB-078 Option B: cancel pending quota open when operator navigates away from origin pane.
	if m.pendingQuotaOpen && m.activePane != m.quotaOriginPane.Pane {
		m.pendingQuotaOpen = false
		m.table.SetPendingQuotaOpen(false)                // FB-099: reset strip label
		m.statusBar.PostHint("Quota dashboard cancelled") // FB-096: acknowledge nav-cancel
		m.quotaOriginPane = DashboardOrigin{}             // FB-095: clear stale origin on nav-cancel
	}
	m.header.Ctx = m.tuiCtx
}

// activityRollupLoaded reports whether project-activity data has been fetched
// for the current session (and the CRD-absent one-shot hasn't been triggered).
func (m *AppModel) activityRollupLoaded() bool {
	return m.activityDashboard.HasRows() || m.activityCRDAbsentThisSession
}

func (m AppModel) handleOverlayKey(msg tea.KeyMsg, _ *[]tea.Cmd) (tea.Model, tea.Cmd) {
	if m.statusBar.Hint != "" {
		m.statusBar.Hint = ""
		m.statusBar.BumpHintToken()
	}

	switch msg.String() {
	case "esc":
		// Esc universally dismisses all overlays. For delete-confirmation in InFlight
		// state the API call continues; a late-arriving success still invalidates cache.
		m.overlay = NoOverlay
		m.statusBar.Mode = components.ModeNormal
		return m, nil
	case "?":
		// `?` closes CtxSwitcher/Help, but NOT DeleteConfirmation — the operator must
		// make an explicit Y/N/Esc decision on a pending delete.
		if m.overlay != DeleteConfirmationOverlay {
			m.overlay = NoOverlay
			m.statusBar.Mode = components.ModeNormal
			return m, nil
		}
		// Fall through: treat `?` as "any other key" on delete confirmation (dismisses).
	case "q", "ctrl+c":
		return m, tea.Quit
	}

	var cmd tea.Cmd
	switch m.overlay {
	case CtxSwitcherOverlay:
		m.ctxOverlay, cmd = m.ctxOverlay.Update(msg)
	case DeleteConfirmationOverlay:
		m, cmd = m.handleDeleteConfirmationKey(msg)
	}
	return m, cmd
}

// handleDeleteConfirmationKey routes a key event to the delete-confirmation state
// machine and returns the resulting model + any command to dispatch.
func (m AppModel) handleDeleteConfirmationKey(msg tea.KeyMsg) (AppModel, tea.Cmd) {
	state := m.deleteConfirmation.State()

	switch msg.String() {
	case "y", "Y":
		if state == components.DeleteStatePrompt {
			m.deleteConfirmation.SetState(components.DeleteStateInFlight)
			return m, data.DeleteResourceCmd(m.ctx, m.rc, m.deleteConfirmation.Target())
		}
		// InFlight/Forbidden/Conflict/TransientError: no-op
		return m, nil

	case "n", "N":
		if state == components.DeleteStateInFlight {
			// Captive during in-flight: no-op.
			return m, nil
		}
		m.overlay = NoOverlay
		m.statusBar.Mode = components.ModeNormal
		return m, nil

	case "r":
		switch state {
		case components.DeleteStateConflict:
			// Close dialog and refresh the resource list so operator can re-inspect.
			m.overlay = NoOverlay
			m.statusBar.Mode = components.ModeNormal
			if rt, ok := m.sidebar.SelectedType(); ok {
				ns := ""
				if rt.Namespaced {
					ns = m.tuiCtx.Namespace
				}
				return m, data.LoadResourcesCmd(m.ctx, m.rc, rt, ns)
			}
			return m, nil
		case components.DeleteStateTransientError:
			// Retry: re-dispatch the same delete call.
			m.deleteConfirmation.SetState(components.DeleteStateInFlight)
			return m, data.DeleteResourceCmd(m.ctx, m.rc, m.deleteConfirmation.Target())
		}
		// Prompt/InFlight/Forbidden: no-op
		return m, nil

	case "x":
		// AC#20: repeat x is a no-op while dialog is open (stays in current state).
		return m, nil
	}

	// Any other key: dismiss (except InFlight which is captive).
	if state != components.DeleteStateInFlight {
		m.overlay = NoOverlay
		m.statusBar.Mode = components.ModeNormal
	}
	return m, nil
}

func (m AppModel) handleFilterKey(msg tea.KeyMsg, _ *[]tea.Cmd) (tea.Model, tea.Cmd) {
	if m.statusBar.Hint != "" {
		m.statusBar.Hint = ""
		m.statusBar.BumpHintToken()
	}
	applyFilter := func(v string) {
		m.table.SetFilter(v)
		m.quota.SetFilter(v)
	}
	switch msg.String() {
	case "enter":
		applyFilter(m.filterBar.Value())
		m.filterBar.Blur()
		m.statusBar.Mode = components.ModeNormal
		return m, nil
	case "esc":
		m.filterBar.Blur()
		applyFilter("")
		m.statusBar.Mode = components.ModeNormal
		return m, nil
	}
	var cmd tea.Cmd
	m.filterBar, cmd = m.filterBar.Update(msg)
	applyFilter(m.filterBar.Value())
	return m, cmd
}

func (m AppModel) handleNormalKey(msg tea.KeyMsg, _ *[]tea.Cmd) (tea.Model, tea.Cmd) {
	// Every keypress clears any transient hint; explicit clears below are defensive documentation only.
	if m.statusBar.Hint != "" {
		m.statusBar.Hint = ""
		m.statusBar.BumpHintToken()
	}

	// FB-042: quick-jump keys — active when welcome panel is visible and NavPane is not focused.
	// FB-073: NavPane has no quick-jump semantics; gate prevents accidental fires during sidebar scroll.
	if (m.tableTypeName == "" || m.showDashboard) && m.activePane != NavPane {
		var matchSubstrs []string
		switch msg.String() {
		case "n":
			matchSubstrs = []string{"namespaces"}
		case "b":
			matchSubstrs = []string{"backends"}
		case "w":
			matchSubstrs = []string{"workloads"}
		case "p":
			matchSubstrs = []string{"projects"}
		case "g":
			matchSubstrs = []string{"gateways"}
		case "v":
			matchSubstrs = []string{"services"}
		case "i":
			matchSubstrs = []string{"ingresses"}
		case "z":
			matchSubstrs = []string{"dnsrecordsets", "dnszones"}
		}
		if len(matchSubstrs) > 0 {
			for _, rt := range m.resourceTypes {
				name := strings.ToLower(rt.Name)
				for _, sub := range matchSubstrs {
					if strings.Contains(name, sub) {
						m.showDashboard = false
						m.table.SetForceDashboard(false)
						m.activePane = TablePane
						m.lastEntryViaQuickJump = true // FB-072: single Esc returns to welcome
						m.tableTypeName = rt.Name
						m.loadState = data.LoadStateLoading
						m.loadErr = nil
						m.lastFailedFetchKind = ""
						m.table.SetLoadErr(nil, data.ErrorSeverityWarning)
						m.table.SetTypeContext(rt.Name, true)
						m.table.SetLoadState(data.LoadStateLoading)
						m.updatePaneFocus()
						ns := ""
						if rt.Namespaced {
							ns = m.tuiCtx.Namespace
						}
						return m, data.LoadResourcesCmd(m.ctx, m.rc, rt, ns)
					}
				}
			}
			// No matching resource type — no-op (key not shown in S4).
			return m, nil
		}
	}

	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "tab":
		if m.activePane == NavPane {
			m.showDashboard = false
			m.table.SetForceDashboard(false)
			m.activePane = TablePane
		} else if m.activePane == TablePane {
			m.activePane = NavPane
		}
		m.updatePaneFocus()
		return m, nil
	case "shift+tab":
		if m.activePane == TablePane {
			m.activePane = NavPane
		} else if m.activePane == NavPane {
			m.showDashboard = false
			m.table.SetForceDashboard(false)
			m.activePane = TablePane
		}
		m.updatePaneFocus()
		return m, nil
	case "t":
		// Toggle between QuotaDashboardPane (S2) and TablePane (S1) for allowancebuckets.
		switch m.activePane {
		case QuotaDashboardPane:
			if !m.quota.IsLoading() {
				m.activePane = TablePane
				m.updatePaneFocus()
			}
		case TablePane:
			if m.tableTypeName == "allowancebuckets" {
				m.activePane = QuotaDashboardPane
				m.updatePaneFocus()
			}
		}
		return m, nil
	case "s":
		if m.activePane == QuotaDashboardPane {
			m.quota.ToggleGrouping()
		}
		return m, nil
	case "y":
		if m.activePane == DetailPane && m.describeRaw != nil {
			m.yamlMode = !m.yamlMode
			if m.yamlMode {
				m.conditionsMode = false // quad-state exclusivity (FB-018/FB-019) // AC#20
				m.eventsMode = false     // AC#20
				m.detail.SetMode("yaml")
			} else {
				m.detail.SetMode("describe")
			}
			m.detail.SetMode(m.detailModeLabel())
			m.detail.SetContent(m.buildDetailContent())
			m.detail.ScrollToTop()
		}
		return m, nil
	case "C": // AC#1 — pane-local conditions toggle
		if m.activePane == DetailPane && m.describeRaw != nil {
			m.conditionsMode = !m.conditionsMode
			if m.conditionsMode {
				m.yamlMode = false    // quad-state exclusivity // AC#20
				m.eventsMode = false  // AC#20
				m.detail.SetMode("conditions")
			} else {
				m.detail.SetMode("describe")
			}
			m.detail.SetMode(m.detailModeLabel())
			m.detail.SetContent(m.buildDetailContent())
			m.detail.ScrollToTop()
		}
		return m, nil
	case "E": // AC#1 FB-019 — pane-local events toggle
		if m.activePane == DetailPane && (m.describeRaw != nil || m.events != nil || m.eventsLoading || m.eventsErr != nil) { // FB-024 D1: admit entry when events attempted; FB-086: also admit on double-failure
			m.eventsMode = !m.eventsMode
			if m.eventsMode {
				m.yamlMode = false       // quad-state exclusivity // AC#23
				m.conditionsMode = false // quad-state exclusivity // AC#23
				m.detail.SetMode("events")
				// Re-dispatch if no events cached or prior fetch failed, and no fetch in flight. // AC#3, FB-024 D4
				if (m.events == nil || m.eventsErr != nil) && !m.eventsLoading {
					m.eventsLoading = true
					m.detail.SetEventsLoading(true) // FB-122
					ns := ""
					if m.describeRT.Namespaced {
						ns = m.tuiCtx.Namespace
					}
					m.detail.SetMode(m.detailModeLabel())
					m.detail.SetContent(m.buildDetailContent())
					m.detail.ScrollToTop()
					return m, data.LoadEventsCmd(m.ctx, m.rc, m.describeRT.Kind, m.detail.ResourceName(), ns)
				}
			} else {
				m.detail.SetMode("describe")
			}
			m.detail.SetMode(m.detailModeLabel())
			m.detail.SetContent(m.buildDetailContent())
			m.detail.ScrollToTop()
		}
		return m, nil
	case "?":
		if m.overlay == HelpOverlayID {
			m.overlay = NoOverlay
			m.statusBar.Mode = components.ModeNormal
		} else {
			m.helpOverlay.ShowDeleteHint = m.activePane == TablePane || m.activePane == DetailPane
			m.helpOverlay.ShowConditionsHint = m.activePane == DetailPane // AC#22 FB-018
			m.helpOverlay.ShowEventsHint = m.activePane == DetailPane     // FB-019
			m.overlay = HelpOverlayID
			m.statusBar.Mode = components.ModeOverlay
		}
		return m, nil
	case "3": // FB-035 — app-global QuotaDashboard keybind
		if m.activePane == QuotaDashboardPane {
			// Toggle-back (FB-050): restore quota origin.
			m.activePane = m.quotaOriginPane.Pane
			m.showDashboard = m.quotaOriginPane.ShowDashboard
			m.table.SetForceDashboard(m.quotaOriginPane.ShowDashboard)
			m.quota.SetOriginLabel("")
			m.updatePaneFocus()
			return m, nil
		}
		// Guard (FB-094): cross-dashboard press from ActivityDash preserves entry stash.
		// FB-080: also skip stash on second press (cancel path) — stash was written on first press.
		if m.activePane != ActivityDashboardPane && !m.pendingQuotaOpen {
			m.quotaOriginPane = DashboardOrigin{
				Pane:          m.activePane,
				ShowDashboard: m.showDashboard,
			}
			m.quota.SetOriginLabel(dashboardOriginLabel(m.quotaOriginPane))
		}
		if !m.quota.IsLoading() {
			m.showDashboard = false
			m.table.SetForceDashboard(false)
			m.activePane = QuotaDashboardPane
			m.statusBar.Hint = "" // FB-112: clear FB-097 ready-prompt on confirm
			m.updatePaneFocus()
		} else if !m.pendingQuotaOpen {
			// First press during loading: queue the open + show hint.
			// No HintClearCmd — hint clears on load completion or error (FB-047).
			m.pendingQuotaOpen = true
			m.table.SetPendingQuotaOpen(true) // FB-099: show cancel label in strip
			m.statusBar.PostHint("Quota dashboard loading… press [3] to cancel")
		} else {
			// FB-080: second press cancels the queued open.
			m.pendingQuotaOpen = false
			m.table.SetPendingQuotaOpen(false) // FB-099: reset strip label
			m.statusBar.Hint = ""
			m.quotaOriginPane = DashboardOrigin{} // FB-095: clear stale origin on second-press cancel
			return m, m.postHint("Quota dashboard cancelled")
		}

	case "4":
		if m.activePane == ActivityDashboardPane {
			// Toggle-back (FB-050): restore activity origin.
			m.activePane = m.activityOriginPane.Pane
			m.showDashboard = m.activityOriginPane.ShowDashboard
			m.table.SetForceDashboard(m.activityOriginPane.ShowDashboard)
			m.activityDashboard.SetOriginLabel("")
			m.updatePaneFocus()
			return m, nil
		}
		// Guard (FB-094): cross-dashboard press from QuotaDash preserves entry stash.
		if m.activePane != QuotaDashboardPane {
			m.activityOriginPane = DashboardOrigin{
				Pane:          m.activePane,
				ShowDashboard: m.showDashboard,
			}
			m.activityDashboard.SetOriginLabel(dashboardOriginLabel(m.activityOriginPane))
		}
		m.showDashboard = false
		m.table.SetForceDashboard(false)
		orgScope := m.tuiCtx.ActiveCtx == nil || m.tuiCtx.ActiveCtx.ProjectID == ""
		m.activityDashboard.SetOrgScope(orgScope)
		m.activePane = ActivityDashboardPane
		m.updatePaneFocus()
		if !orgScope && !m.activityRollupLoaded() {
			m.activityDashboard.SetLoading(true)
			return m, data.LoadRecentProjectActivityCmd(m.ctx, m.ac, 24*time.Hour, 10)
		}
		return m, nil

	case "c":
		if m.activePane == HistoryPane {
			m.history.ToggleHumanFilter()
			return m, nil
		}
		if m.activePane == DiffPane {
			return m, nil
		}
		m.overlay = CtxSwitcherOverlay
		m.statusBar.Mode = components.ModeOverlay
		return m, nil
	case "r":
		// FB-005 §4: error-state branch MUST run before the normal-state refresh branch.
		if m.loadState == data.LoadStateError {
			sev := components.ErrorSeverityOf(m.loadErr, m.rc)
			if sev == data.ErrorSeverityError {
				// P1: clear statusBar.Err so hint wins the statusbar switch (Err branch
				// would otherwise shadow the hint). In-pane card stays via loadState+table.loadErr.
				m.statusBar.Err = nil
				m.statusBar.ErrSeverity = data.ErrorSeverityWarning
				return m, m.postHint("No retry available for this error")
			}
			m.loadState = data.LoadStateLoading
			m.refreshing = true // P2: guard repeat r via normal-state refreshing check
			m.statusBar.Err = nil
			m.statusBar.ErrSeverity = data.ErrorSeverityWarning
			m.table.SetLoadErr(nil, data.ErrorSeverityWarning)
			return m, m.redispatchLastFetch()
		}
		switch m.activePane {
		case NavPane:
			m.loadState = data.LoadStateLoading
			rCmds := []tea.Cmd{data.LoadResourceTypesCmd(m.ctx, m.rc)}
			// FB-076: refresh activity teaser on welcome panel when project-scoped.
			if m.ac != nil && m.tuiCtx.ActiveCtx != nil && m.tuiCtx.ActiveCtx.ProjectID != "" {
				// FB-103: show spinner only when no rows exist; stale rows stay visible (FB-076).
				if m.table.ActivityRowCount() == 0 {
					m.table.SetActivityLoading(true)
				}
				rCmds = append(rCmds, data.LoadRecentProjectActivityCmd(m.ctx, m.ac, 24*time.Hour, 10))
			}
			return m, tea.Batch(rCmds...)
		case TablePane, DetailPane:
			rt, ok := m.sidebar.SelectedType()
			if !ok || m.tableTypeName == "" {
				return m, nil
			}
			if m.refreshing {
				return m, nil
			}
			m.refreshing = true
			m.preserveCursor = true
			m.tuiCtx.Refreshing = true
			m.header.Ctx = m.tuiCtx
			ns := ""
			if rt.Namespaced {
				ns = m.tuiCtx.Namespace
			}
			cmds := []tea.Cmd{data.LoadResourcesCmd(m.ctx, m.rc, rt, ns)}
			if m.banner.HasBuckets() && m.bc != nil && !m.bucketLoading {
				m.bc.InvalidateBucketCache()
				m.bucketLoading = true
				cmds = append(cmds, data.LoadBucketsCmd(m.ctx, m.bc))
			}
			if m.activePane == DetailPane && (m.eventsMode || m.events != nil) && !m.eventsLoading {
				m.eventsLoading = true
				m.detail.SetEventsLoading(true) // FB-122
				evNS := ""
				if m.describeRT.Namespaced {
					evNS = m.tuiCtx.Namespace
				}
				cmds = append(cmds, data.LoadEventsCmd(m.ctx, m.rc, m.describeRT.Kind, m.detail.ResourceName(), evNS))
			}
			return m, tea.Batch(cmds...)
		case QuotaDashboardPane:
			if m.refreshing || m.bc == nil {
				return m, nil
			}
			rt, ok := m.sidebar.SelectedType()
			if !ok {
				return m, nil
			}
			m.bc.InvalidateBucketCache()
			m.refreshing = true
			m.bucketLoading = true
			m.tuiCtx.Refreshing = true
			m.header.Ctx = m.tuiCtx
			m.quota.SetRefreshing(true) // FB-063: preserve bucket data during refresh
			m.quota.SetLoadErr(nil)
			ns := ""
			if rt.Namespaced {
				ns = m.tuiCtx.Namespace
			}
			return m, tea.Batch(
				data.LoadBucketsCmd(m.ctx, m.bc),
				data.LoadResourcesCmd(m.ctx, m.rc, rt, ns),
			)
		case ActivityPane:
			if m.ac == nil || m.activityKind == "" {
				return m, nil
			}
			m.ac.ForceRefresh(m.activityAPIGroup, m.activityKind, m.activityName, m.activityNamespace)
			m.activity.Reset()
			m.activity.SetResourceContext(m.detail.ResourceKind(), m.detail.ResourceName())
			m.activity.SetLoading(true)
			m.refreshing = true
			m.tuiCtx.Refreshing = true
			m.header.Ctx = m.tuiCtx
			return m, data.LoadActivityCmd(m.ctx, m.ac, m.activityAPIGroup, m.activityKind, m.activityName, m.activityNamespace, "")
		case ActivityDashboardPane:
			if m.ac == nil || m.activityDashboard.CRDAbsent() {
				return m, nil
			}
			orgScope := m.tuiCtx.ActiveCtx == nil || m.tuiCtx.ActiveCtx.ProjectID == ""
			if orgScope {
				return m, nil
			}
			m.ac.ForceRefreshProject(24*time.Hour, 10)
			m.activityDashboard.SetLoading(true)
			return m, data.LoadRecentProjectActivityCmd(m.ctx, m.ac, 24*time.Hour, 10)
		case HistoryPane, DiffPane:
			if m.hc == nil || m.historyName == "" {
				return m, nil
			}
			m.hc.ForceRefresh(m.historyRT, m.historyName, m.historyNamespace)
			m.history.Reset()
			m.history.SetResourceContext(m.historyRT.Kind, m.historyName)
			m.history.SetLoading(true)
			m.currentHistoryManifests = nil
			m.refreshing = true
			m.tuiCtx.Refreshing = true
			m.header.Ctx = m.tuiCtx
			if m.activePane == DiffPane {
				m.activePane = HistoryPane
				m.updatePaneFocus()
			}
			return m, data.LoadHistoryCmd(m.ctx, m.hc, m.historyRT, m.historyName, m.historyNamespace)
		}
		return m, nil
	case "/":
		switch m.activePane {
		case TablePane:
			if m.tableTypeName == "" {
				return m, m.postHint("Select a resource type first, then press / to filter")
			}
			focusCmd := m.filterBar.Focus()
			m.statusBar.Mode = components.ModeFilter
			return m, focusCmd
		case NavPane:
			return m, m.postHint("Select a resource type first, then press / to filter")
		// QuotaDashboardPane: silent no-op — '/' not advertised in this pane (FB-049)
		default:
			// DetailPane, ActivityPane: inert — no hint.
			return m, nil
		}
	case "H":
		switch m.activePane {
		case DetailPane:
			if m.detail.Loading() {
				return m, nil // inert while describe is loading
			}
			kind := m.detail.ResourceKind()
			name := m.detail.ResourceName()
			ns := ""
			if m.describeRT.Namespaced {
				ns = m.tuiCtx.Namespace
			}
			// Detect resource-target change and invalidate history.
			if name != m.historyName || m.describeRT.Group != m.historyRT.Group || m.describeRT.Kind != m.historyRT.Kind {
				if m.hc != nil {
					m.hc.ForceRefresh(m.describeRT, name, ns)
				}
				m.history.Reset()
				m.currentHistoryManifests = nil
			}
			m.historyRT = m.describeRT
			m.historyName = name
			m.historyNamespace = ns
			m.history.SetResourceContext(kind, name)
			m.activePane = HistoryPane
			m.updatePaneFocus()
			// Only dispatch if no rows loaded yet or cache is stale (no-rows indicates not yet fetched).
			if !m.history.HasRows() && m.hc != nil {
				m.history.SetLoading(true)
				m.refreshing = true
				m.tuiCtx.Refreshing = true
				m.header.Ctx = m.tuiCtx
				return m, data.LoadHistoryCmd(m.ctx, m.hc, m.historyRT, name, ns)
			}
			return m, nil
		case HistoryPane:
			m.history.ResetFilter()
			m.yamlMode = false
			m.conditionsMode = false // AC#4
			m.eventsMode = false     // AC#4 FB-019
			m.events = nil
			m.eventsLoading = false
			m.detail.SetEventsLoading(false) // FB-122
			m.eventsErr = nil
			m.detail.SetEventsFetchedAt(time.Time{}) // FB-025
			m.detail.SetMode("")
			m.activePane = DetailPane
			m.statusBar.Mode = components.ModeDetail
			m.updatePaneFocus()
			return m, nil
		case DiffPane:
			m.yamlMode = false
			m.conditionsMode = false // AC#4
			m.eventsMode = false     // AC#4 FB-019
			m.events = nil
			m.eventsLoading = false
			m.detail.SetEventsLoading(false) // FB-122
			m.eventsErr = nil
			m.detail.SetEventsFetchedAt(time.Time{}) // FB-025
			m.detail.SetMode("")
			m.activePane = DetailPane
			m.statusBar.Mode = components.ModeDetail
			m.updatePaneFocus()
			return m, nil
		}
		return m, nil

	case "[":
		if m.activePane == DiffPane {
			if m.selectedRevIdx > 0 {
				m.selectedRevIdx--
				m.openDiffAtSelected()
			}
		}
		return m, nil

	case "]":
		if m.activePane == DiffPane {
			if m.selectedRevIdx < len(m.currentHistoryManifests)-1 {
				m.selectedRevIdx++
				m.openDiffAtSelected()
			}
		}
		return m, nil

	case "esc":
		switch m.activePane {
		case ActivityDashboardPane:
			m.activePane = m.activityOriginPane.Pane
			m.showDashboard = m.activityOriginPane.ShowDashboard
			m.table.SetForceDashboard(m.activityOriginPane.ShowDashboard)
			m.activityDashboard.SetOriginLabel("")
			m.updatePaneFocus()
			return m, nil
		case DiffPane:
			m.activePane = HistoryPane
			m.updatePaneFocus()
			return m, nil
		case HistoryPane:
			m.history.ResetFilter()
			m.yamlMode = false
			m.conditionsMode = false // AC#4
			m.eventsMode = false     // AC#4 FB-019
			m.events = nil
			m.eventsLoading = false
			m.detail.SetEventsLoading(false) // FB-122
			m.eventsErr = nil
			m.detail.SetEventsFetchedAt(time.Time{}) // FB-025
			m.detail.SetMode("")
			m.activePane = DetailPane
			m.statusBar.Mode = components.ModeDetail
			m.updatePaneFocus()
			return m, nil
		case ActivityPane:
			m.activePane = DetailPane
			m.updatePaneFocus()
			return m, nil
		case DetailPane:
			m.yamlMode = false
			m.conditionsMode = false // AC#4
			m.eventsMode = false     // AC#4 FB-019
			m.events = nil
			m.eventsLoading = false
			m.detail.SetEventsLoading(false) // FB-122
			m.eventsErr = nil
			m.detail.SetEventsFetchedAt(time.Time{}) // FB-025
			m.describeRaw = nil
			m.detail.SetDescribeAvailable(false)
			m.detail.SetMode("")
			// FB-005 AC#17: describe-error state always returns to TABLE, not NAV.
			if m.loadState == data.LoadStateError && m.lastFailedFetchKind == "describe" {
				m.activePane = TablePane
			} else {
				m.activePane = m.detailReturnPane
			}
			if m.activePane == QuotaDashboardPane {
				m.statusBar.Mode = components.ModeNormal
			} else {
				m.statusBar.Mode = components.ModeNormal
				m.detailReturnPane = TablePane // reset
			}
			m.updatePaneFocus()
			return m, nil
		case TablePane:
			if m.lastEntryViaQuickJump {
				// FB-072: quick-jump is a 1-key entry so Esc returns to welcome in one press.
				m.lastEntryViaQuickJump = false
				m.showDashboard = true
				m.table.SetForceDashboard(true)
				m.activePane = NavPane
				m.updatePaneFocus()
				return m, nil
			}
			m.activePane = NavPane
			m.updatePaneFocus()
			return m, nil
		case QuotaDashboardPane:
			m.activePane = m.quotaOriginPane.Pane
			m.showDashboard = m.quotaOriginPane.ShowDashboard
			m.table.SetForceDashboard(m.quotaOriginPane.ShowDashboard)
			m.quota.ResetGrouping()
			m.quota.SetOriginLabel("")
			m.updatePaneFocus()
			return m, nil
		case NavPane:
			if m.pendingQuotaOpen {
				m.pendingQuotaOpen = false
				m.table.SetPendingQuotaOpen(false) // FB-099: reset strip label
				m.statusBar.Hint = ""
				m.quotaOriginPane = DashboardOrigin{} // FB-096: clear stale origin on Esc cancel
				return m, m.postHint("Quota dashboard cancelled")
			}
			if !m.showDashboard && m.tableTypeName != "" {
				m.showDashboard = true
				m.table.SetForceDashboard(true)
				return m, m.postHint("Returned to welcome panel")
			}
			return m, nil
		}
	case "enter":
		if m.activePane == HistoryPane {
			if row, manifestIdx, ok := m.history.SelectedRow(); ok {
				m.selectedRevIdx = manifestIdx
				m.openDiffForRow(row, manifestIdx)
			}
			return m, nil
		}
		switch m.activePane {
		case NavPane:
			m.showDashboard = false
			m.table.SetForceDashboard(false)
			m.lastEntryViaQuickJump = false // FB-072: sidebar Enter clears quick-jump flag
			if rt, ok := m.sidebar.SelectedType(); ok {
				if rt.Name == "allowancebuckets" && m.bc != nil {
					// S2 dashboard is the default entry point for allowancebuckets.
					// Also fire LoadResourcesCmd so S1 raw table is populated for `t` toggle.
					m.activePane = QuotaDashboardPane
					m.tableTypeName = rt.Name
					m.table.SetTypeContext(rt.Name, true)
					m.table.SetLoadState(data.LoadStateLoading)
					m.quota.SetLoading(true)
					m.quota.SetLoadErr(nil)
					m.bucketLoading = true
					m.describeRT = rt
					m.updatePaneFocus()
					ns := ""
					if rt.Namespaced {
						ns = m.tuiCtx.Namespace
					}
					batch := []tea.Cmd{
						data.LoadBucketsCmd(m.ctx, m.bc),
						data.LoadResourcesCmd(m.ctx, m.rc, rt, ns),
					}
					if m.rrc != nil && m.registrations == nil && !m.registrationsLoading {
						m.registrationsLoading = true
						batch = append(batch, data.LoadResourceRegistrationsCmd(m.ctx, m.rrc))
					}
					return m, tea.Batch(batch...)
				}
				m.activePane = TablePane
				m.tableTypeName = rt.Name
				m.loadState = data.LoadStateLoading
				m.loadErr = nil
				m.lastFailedFetchKind = ""
				m.table.SetLoadErr(nil, data.ErrorSeverityWarning)
				m.table.SetTypeContext(rt.Name, true)
				m.table.SetLoadState(data.LoadStateLoading)
				m.updatePaneFocus()
				ns := ""
				if rt.Namespaced {
					ns = m.tuiCtx.Namespace
				}
				// Also fetch buckets for the banner when cache is empty.
				if m.bc != nil && len(m.buckets) == 0 && !m.bucketLoading {
					m.bucketLoading = true
					batch := []tea.Cmd{
						data.LoadResourcesCmd(m.ctx, m.rc, rt, ns),
						data.LoadBucketsCmd(m.ctx, m.bc),
					}
					if m.rrc != nil && m.registrations == nil && !m.registrationsLoading {
						m.registrationsLoading = true
						batch = append(batch, data.LoadResourceRegistrationsCmd(m.ctx, m.rrc))
					}
					return m, tea.Batch(batch...)
				}
				return m, data.LoadResourcesCmd(m.ctx, m.rc, rt, ns)
			}
		case TablePane:
			if row, ok := m.table.SelectedRow(); ok {
				if rt, ok2 := m.sidebar.SelectedType(); ok2 {
					ns := ""
					if rt.Namespaced {
						ns = m.tuiCtx.Namespace
					}
					m.detail.SetResourceContext(rt.Name, row.Name)
					m.detail.SetLoading(true)
					m.detailReturnPane = TablePane
					m.activePane = DetailPane
					m.statusBar.Mode = components.ModeDetail
					m.describeRT = rt
					m.describeRaw = nil
					m.detail.SetDescribeAvailable(false)
					m.yamlMode = false
					m.conditionsMode = false // AC#4
					m.eventsMode = false     // AC#4 FB-019
					m.events = nil
					m.eventsLoading = true
					m.detail.SetEventsLoading(true) // FB-122
					m.eventsErr = nil
					m.detail.SetMode("")
					m.detail.ScrollToTop()
					m.updatePaneFocus()
					var cmds2 []tea.Cmd
					cmds2 = append(cmds2, data.DescribeResourceCmd(m.ctx, m.rc, rt, row.Name, ns))
					cmds2 = append(cmds2, data.LoadEventsCmd(m.ctx, m.rc, rt.Kind, row.Name, ns)) // FB-019
					if m.bc != nil && !m.bucketLoading {
						cmds2 = append(cmds2, data.LoadBucketsCmd(m.ctx, m.bc))
					}
					if m.rrc != nil && m.registrations == nil && !m.registrationsLoading {
						m.registrationsLoading = true
						cmds2 = append(cmds2, data.LoadResourceRegistrationsCmd(m.ctx, m.rrc))
					}
					return m, tea.Batch(cmds2...)
				}
			}
		case QuotaDashboardPane:
			if !m.quota.IsLoading() {
				if b, ok := m.quota.SelectedBucket(); ok {
					if rt, ok2 := m.sidebar.SelectedType(); ok2 {
						ns := ""
						if rt.Namespaced {
							ns = m.tuiCtx.Namespace
						}
						m.detail.SetResourceContext(rt.Name, b.Name)
						m.detail.SetLoading(true)
						m.detailReturnPane = QuotaDashboardPane
						m.activePane = DetailPane
						m.statusBar.Mode = components.ModeDetail
						m.describeRT = rt
						m.describeRaw = nil
						m.detail.SetDescribeAvailable(false)
						m.yamlMode = false
						m.conditionsMode = false // AC#4
						m.eventsMode = false     // AC#4 FB-019
						m.events = nil
						m.eventsLoading = true
						m.detail.SetEventsLoading(true) // FB-122
						m.eventsErr = nil
						m.detail.SetMode("")
						m.detail.ScrollToTop()
						m.updatePaneFocus()
						return m, tea.Batch(
							data.DescribeResourceCmd(m.ctx, m.rc, rt, b.Name, ns),
							data.LoadEventsCmd(m.ctx, m.rc, rt.Kind, b.Name, ns), // FB-019
						)
					}
				}
			}
		}
	case "a":
		switch m.activePane {
		case DetailPane:
			if !m.detail.Loading() {
				// Capture the resource context and enter ActivityPane.
				kind := m.detail.ResourceKind()
				name := m.detail.ResourceName()
				ns := ""
				if m.describeRT.Namespaced {
					ns = m.tuiCtx.Namespace
				}
				// Reset view modes when leaving DetailPane via activity.
				m.yamlMode = false
				m.conditionsMode = false // AC#4
				m.eventsMode = false     // AC#4 FB-019
				m.events = nil
				m.eventsLoading = false
				m.detail.SetEventsLoading(false) // FB-122
				m.eventsErr = nil
				m.detail.SetMode("")
				// Capture previous values before reassigning.
				prevRTName := m.activityRTName
				prevName := m.activityName
				m.activityAPIGroup = m.describeRT.Group
				m.activityKind = m.describeRT.Kind
				m.activityRTName = m.describeRT.Name
				m.activityName = name
				m.activityNamespace = ns
				// Reset when the resource changed or no rows are loaded yet.
				if !m.activity.HasRows() || m.describeRT.Name != prevRTName || name != prevName {
					m.activity.Reset()
					m.activity.SetResourceContext(kind, name)
					m.activity.SetLoading(true)
					m.activePane = ActivityPane
					m.updatePaneFocus()
					return m, data.LoadActivityCmd(m.ctx, m.ac, m.activityAPIGroup, m.activityKind, name, ns, "")
				}
				m.activity.SetResourceContext(kind, name)
				m.activePane = ActivityPane
				m.updatePaneFocus()
			}
		case ActivityPane:
			m.activePane = DetailPane
			m.updatePaneFocus()
		}
		return m, nil
	case "x":
		switch m.activePane {
		case TablePane:
			if row, ok := m.table.SelectedRow(); ok {
				if rt, ok2 := m.sidebar.SelectedType(); ok2 {
					ns := ""
					if rt.Namespaced {
						ns = m.tuiCtx.Namespace
					}
					return m, data.OpenDeleteConfirmationCmd(data.DeleteTarget{
						RT:        rt,
						Name:      row.Name,
						Namespace: ns,
					})
				}
			}
		case DetailPane:
			if !m.detail.Loading() && m.detail.ResourceName() != "" {
				ns := ""
				if m.describeRT.Namespaced {
					ns = m.tuiCtx.Namespace
				}
				return m, data.OpenDeleteConfirmationCmd(data.DeleteTarget{
					RT:        m.describeRT,
					Name:      m.detail.ResourceName(),
					Namespace: ns,
				})
			}
		}
		// All other panes: no-op (AC#10).
		return m, nil

	case "d":
		if m.activePane == TablePane {
			if row, ok := m.table.SelectedRow(); ok {
				if rt, ok2 := m.sidebar.SelectedType(); ok2 {
					ns := ""
					if rt.Namespaced {
						ns = m.tuiCtx.Namespace
					}
					m.detail.SetResourceContext(rt.Name, row.Name)
					m.detail.SetLoading(true)
					m.detailReturnPane = TablePane
					m.activePane = DetailPane
					m.statusBar.Mode = components.ModeDetail
					m.describeRT = rt
					m.describeRaw = nil
					m.detail.SetDescribeAvailable(false)
					m.yamlMode = false
					m.conditionsMode = false // AC#4
					m.eventsMode = false     // AC#4 FB-019
					m.events = nil
					m.eventsLoading = true
					m.detail.SetEventsLoading(true) // FB-122
					m.eventsErr = nil
					m.detail.SetEventsFetchedAt(time.Time{}) // FB-025
					m.detail.SetMode("")
					m.detail.ScrollToTop()
					m.updatePaneFocus()
					var cmds2 []tea.Cmd
					cmds2 = append(cmds2, data.DescribeResourceCmd(m.ctx, m.rc, rt, row.Name, ns))
					cmds2 = append(cmds2, data.LoadEventsCmd(m.ctx, m.rc, rt.Kind, row.Name, ns)) // FB-019
					if m.bc != nil && !m.bucketLoading {
						cmds2 = append(cmds2, data.LoadBucketsCmd(m.ctx, m.bc))
					}
					return m, tea.Batch(cmds2...)
				}
			}
		}
	case "j", "down":
		switch m.activePane {
		case NavPane:
			m.lastEntryViaQuickJump = false // FB-072: sidebar interaction clears quick-jump flag
			var cmd tea.Cmd
			m.sidebar, cmd = m.sidebar.Update(msg)
			if rt, ok := m.sidebar.SelectedType(); ok {
				m.table.SetHoveredType(rt)
			}
			return m, cmd
		case TablePane:
			var cmd tea.Cmd
			m.table, cmd = m.table.Update(msg)
			return m, cmd
		case DetailPane:
			var cmd tea.Cmd
			m.detail, cmd = m.detail.Update(msg)
			return m, cmd
		case QuotaDashboardPane:
			var cmd tea.Cmd
			m.quota, cmd = m.quota.Update(msg)
			return m, cmd
		case ActivityPane:
			var cmd tea.Cmd
			m.activity, cmd = m.activity.Update(msg)
			return m, cmd
		case ActivityDashboardPane:
			var cmd tea.Cmd
			m.activityDashboard, cmd = m.activityDashboard.Update(msg)
			return m, cmd
		case HistoryPane:
			m.history.CursorDown()
			return m, nil
		case DiffPane:
			var cmd tea.Cmd
			m.diff, cmd = m.diff.Update(msg)
			return m, cmd
		}
	case "k", "up":
		switch m.activePane {
		case NavPane:
			m.lastEntryViaQuickJump = false // FB-072: sidebar interaction clears quick-jump flag
			var cmd tea.Cmd
			m.sidebar, cmd = m.sidebar.Update(msg)
			if rt, ok := m.sidebar.SelectedType(); ok {
				m.table.SetHoveredType(rt)
			}
			return m, cmd
		case TablePane:
			var cmd tea.Cmd
			m.table, cmd = m.table.Update(msg)
			return m, cmd
		case DetailPane:
			var cmd tea.Cmd
			m.detail, cmd = m.detail.Update(msg)
			return m, cmd
		case QuotaDashboardPane:
			var cmd tea.Cmd
			m.quota, cmd = m.quota.Update(msg)
			return m, cmd
		case ActivityPane:
			var cmd tea.Cmd
			m.activity, cmd = m.activity.Update(msg)
			return m, cmd
		case ActivityDashboardPane:
			var cmd tea.Cmd
			m.activityDashboard, cmd = m.activityDashboard.Update(msg)
			return m, cmd
		case HistoryPane:
			m.history.CursorUp()
			return m, nil
		case DiffPane:
			var cmd tea.Cmd
			m.diff, cmd = m.diff.Update(msg)
			return m, cmd
		}
	case "pgup", "pgdown", "ctrl+u", "ctrl+d", "g", "G":
		if m.activePane == DetailPane {
			var cmd tea.Cmd
			m.detail, cmd = m.detail.Update(msg)
			return m, cmd
		}
		if m.activePane == ActivityPane {
			var cmd tea.Cmd
			m.activity, cmd = m.activity.Update(msg)
			return m, cmd
		}
		if m.activePane == HistoryPane {
			switch msg.String() {
			case "g":
				m.history.CursorTop()
			case "G":
				m.history.CursorBottom()
			default:
				var cmd tea.Cmd
				m.history, cmd = m.history.Update(msg)
				return m, cmd
			}
			return m, nil
		}
		if m.activePane == DiffPane {
			var cmd tea.Cmd
			m.diff, cmd = m.diff.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

// tableView composes the right-column view for NavPane/TablePane, stacking the
// quota banner above the resource table when matches are present.
func (m AppModel) tableView() string {
	if !m.banner.HasBuckets() {
		return m.table.View()
	}
	return lipgloss.JoinVertical(lipgloss.Left, m.banner.View(), m.table.View())
}

// updateBanner recomputes the quota banner for the current resource type using
// the cached bucket list.
func (m *AppModel) updateBanner() {
	rt, ok := m.sidebar.SelectedType()
	if !ok || m.tableTypeName == "" {
		m.banner.SetBuckets(nil)
		return
	}
	matches := data.FindBucketsForResource(m.buckets, rt.Group, rt.Name)
	sort.Slice(matches, func(i, j int) bool {
		pi, pj := float64(0), float64(0)
		if matches[i].Limit > 0 {
			pi = float64(matches[i].Allocated) / float64(matches[i].Limit)
		}
		if matches[j].Limit > 0 {
			pj = float64(matches[j].Allocated) / float64(matches[j].Limit)
		}
		return pi > pj
	})
	m.banner.SetBuckets(matches)
}

// buildQuotaTopHint returns a one-line "[3] quota dashboard ───" hint for the
// top of the DetailPane viewport when matching buckets exist. Returns "" when
// the pane is too narrow. FB-064.
func (m AppModel) buildQuotaTopHint() string {
	innerW := max(1, m.detail.Width()-3)
	if innerW < 30 {
		return ""
	}
	accentBold := lipgloss.NewStyle().Foreground(styles.Accent).Bold(true)
	muted := lipgloss.NewStyle().Foreground(styles.Muted)
	prefix := accentBold.Render("[3]") + muted.Render(" quota dashboard  ")
	const prefixPlainW = 21 // plain: "[3] quota dashboard  " = 3 + 18
	ruleLen := max(0, innerW-prefixPlainW)
	return prefix + muted.Render(strings.Repeat("─", ruleLen))
}

// buildDetailContent returns the content to display in the detail pane.
// In YAML mode it returns the raw manifest only (no quota block). In describe
// buildQuotaSectionHeader returns the "── Quota" separator with an optional
// freshness signal embedded on the right when width and fetch-time allow (FB-043).
func (m AppModel) buildQuotaSectionHeader() string {
	innerW := max(1, m.detail.Width()-3)
	muted := lipgloss.NewStyle().Foreground(styles.Muted)

	if innerW < 30 {
		// Very narrow — plain separator, [3] and freshness both dropped.
		const plainPrefix = "── Quota "
		ruleLen := max(0, innerW-len(plainPrefix))
		return "\n" + plainPrefix + strings.Repeat("─", ruleLen) + "\n"
	}

	// [3] affordance fits (innerW >= 30). FB-044, copy aligned FB-109.
	accentBold := lipgloss.NewStyle().Foreground(styles.Accent).Bold(true)
	prefixRendered := muted.Render("── ") + accentBold.Render("[3]") + muted.Render(" quota dashboard  ")
	const prefixWidth = 24 // plain: "── [3] quota dashboard  " = 3 + 3 + 18

	if m.bucketsFetchedAt.IsZero() || innerW < 60 {
		ruleLen := max(0, innerW-prefixWidth)
		return "\n" + prefixRendered + muted.Render(strings.Repeat("─", ruleLen)) + "\n"
	}

	fresh := "  updated " + components.HumanizeSince(m.bucketsFetchedAt)
	ruleLen := max(0, innerW-prefixWidth-lipgloss.Width(fresh))
	return "\n" + prefixRendered + muted.Render(strings.Repeat("─", ruleLen)) + muted.Render(fresh) + "\n"
}

// detailModeLabel returns the mode string for m.detail.SetMode when the
// placeholder is active; otherwise returns the existing mode (unchanged). FB-051.
func (m AppModel) detailModeLabel() string {
	if m.describeRaw == nil && m.events != nil && !m.yamlMode && !m.conditionsMode && !m.eventsMode {
		if m.detail.Loading() {
			return "" // spinner is the active signal; suppress settled-state label
		}
		return "describe [unavailable]"
	}
	return m.detail.Mode()
}

// placeholderActionRow renders the inline action row for the placeholder body.
// FB-084: retryable gates [r]; copy qualified to "retry describe". FB-052.
// FB-106: contentW<40 renders short form "[r] retry" to avoid overflow at narrow widths.
func placeholderActionRow(errMode bool, retryable bool, contentW int, accentBold, muted lipgloss.Style) string {
	eKey   := "  " + accentBold.Render("[E]") + muted.Render(" events")
	escKey := "  " + accentBold.Render("[Esc]") + muted.Render(" back")
	if errMode && retryable {
		retryCopy := " retry describe"
		if contentW < 40 {
			retryCopy = " retry"
		}
		rKey := "  " + accentBold.Render("[r]") + muted.Render(retryCopy)
		return eKey + rKey + escKey
	}
	return eKey + escKey
}

// buildDetailContent builds the content for the detail pane.
// In describe mode it appends the S3 quota block when matching buckets are available.
func (m AppModel) buildDetailContent() string {
	// FB-038: stable loading placeholder when both fetches in-flight and not in events mode.
	if m.describeRaw == nil && m.events == nil && !m.eventsMode && m.eventsLoading {
		return lipgloss.NewStyle().Foreground(styles.Muted).Render("Loading…")
	}
	// FB-024/FB-051/FB-052: placeholder when describe unavailable but events are loaded.
	if m.describeRaw == nil && m.events != nil && !m.yamlMode && !m.conditionsMode && !m.eventsMode {
		muted      := lipgloss.NewStyle().Foreground(styles.Muted)
		accentBold := lipgloss.NewStyle().Foreground(styles.Accent).Bold(true)

		errMode := m.loadState == data.LoadStateError &&
			m.lastFailedFetchKind == "describe" &&
			m.loadErr != nil

		lines := []string{muted.Render("Describe unavailable — only events loaded.")}
		if errMode {
			lines = append(lines, muted.Render("  (describe failed: "+components.SanitizeErrMsg(m.loadErr)+")"))
		}
		retryable := errMode && components.ErrorSeverityOf(m.loadErr, m.rc) != data.ErrorSeverityError
		lines = append(lines, placeholderActionRow(errMode, retryable, m.detail.Width(), accentBold, muted))
		return strings.Join(lines, "\n")
	}
	// FB-005: inline error card when a describe fetch failed.
	if m.loadState == data.LoadStateError && m.lastFailedFetchKind == "describe" && !m.eventsMode {
		innerW := max(1, m.detail.Width()-3)
		sev := components.ErrorSeverityOf(m.loadErr, m.rc)
		rt := m.describeRT.Name
		if name := m.detail.ResourceName(); name != "" {
			if m.describeRT.Namespaced && m.tuiCtx.Namespace != "" {
				rt = rt + "/" + m.tuiCtx.Namespace + "/" + name
			} else {
				rt = rt + "/" + name
			}
		}
		return components.RenderErrorBlock(components.ErrorBlock{
			Title:    components.SanitizedTitleForError(m.loadErr, "Could not describe "+rt),
			Detail:   components.SanitizeErrMsg(m.loadErr),
			Actions:  components.ActionsForSeverity(sev, "back to table"),
			Severity: sev,
			Width:    innerW,
		})
	}
	if m.yamlMode {
		yamlStr, err := data.RawYAML(m.describeRaw)
		if err != nil {
			return components.RenderErrorBlock(components.ErrorBlock{
				Title:    "Could not render YAML",
				Detail:   components.SanitizeErrMsg(err),
				Actions:  []components.ActionHint{{Key: "y", Label: "toggle yaml"}, {Key: "Esc", Label: "back"}},
				Severity: data.ErrorSeverityError,
				Width:    m.detail.Width() - 4,
			})
		}
		return yamlStr
	}
	if m.conditionsMode { // AC#1
		return components.RenderConditionsTable(m.describeRaw, m.detail.Width())
	}
	if m.eventsMode { // AC#1 FB-019
		return components.RenderEventsTable(m.events, m.eventsLoading, m.eventsErr, m.rc, m.detail.Width(), m.detail.Spinner(), m.detail.EventsFetchedAt())
	}

	content := m.describeContent
	if len(m.buckets) == 0 || content == "" {
		return content
	}
	matching := data.FindBucketsForResource(m.buckets, m.describeRT.Group, m.describeRT.Name)
	if len(matching) == 0 {
		return content
	}
	var sb strings.Builder
	if topHint := m.buildQuotaTopHint(); topHint != "" {
		sb.WriteString(topHint)
		sb.WriteString("\n")
	}
	sb.WriteString(content)
	sb.WriteString(m.buildQuotaSectionHeader())
	ak, an := m.activeConsumer()
	tree := data.ClassifyTreeBuckets(matching, ak, an)
	if tree.HasTree {
		sb.WriteString(components.RenderQuotaTree(tree, m.detail.Width(), m.bucketSiblingsRestricted))
		sb.WriteString("\n")
	} else {
		for _, b := range matching {
			sb.WriteString(components.RenderQuotaBlock(b, m.detail.Width(), m.registrations))
			sb.WriteString("\n\n")
		}
	}
	return sb.String()
}

// computeAttentionItems returns up to 3 quota-based AttentionItems for the welcome spotlight (FB-042).
func computeAttentionItems(buckets []data.AllowanceBucket, activeKind, activeName string, regs []data.ResourceRegistration) []components.AttentionItem {
	summary := data.ComputePlatformHealthSummary(buckets, activeKind, activeName, regs)
	var items []components.AttentionItem
	for _, r := range summary.TopThree {
		if r.PercentInt < 80 {
			continue
		}
		items = append(items, components.AttentionItem{
			Kind:    "quota",
			Label:   r.Label + " quota",
			Detail:  fmt.Sprintf("%d%% allocated", r.PercentInt),
			NavKey:  "[3]",
			NavHint: "quota dashboard",
		})
	}
	return items
}

// activeConsumer returns the (kind, name) of the active consumer derived from
// the TUI context. Kind is canonical-cased ("Project", "Organization").
func (m AppModel) activeConsumer() (kind, name string) {
	if m.tuiCtx.ActiveCtx == nil {
		return "", ""
	}
	if m.tuiCtx.ActiveCtx.ProjectID != "" {
		return "Project", m.tuiCtx.ActiveCtx.ProjectID
	}
	if m.tuiCtx.ActiveCtx.OrganizationID != "" {
		return "Organization", m.tuiCtx.ActiveCtx.OrganizationID
	}
	return "", ""
}

func (m AppModel) recalcLayout() AppModel {
	sidebarWidth := layout.SidebarWidth(m.width)

	filterVisible := m.filterBar.Focused()
	mainH := layout.MainAreaWithFilter(m.height, filterVisible)
	tableW := layout.TableWidth(m.width, sidebarWidth)

	m.header.Width = m.width
	m.statusBar.Width = m.width

	newSidebar := components.NewNavSidebarModel(sidebarWidth, mainH)
	newSidebar.SetItems(m.resourceTypes)
	newSidebar.SetFocused(m.activePane == NavPane)
	m.sidebar = newSidebar

	m.banner.SetSize(tableW)

	tableH := max(1, mainH-m.banner.Height())
	newTable := components.NewResourceTableModel(tableW, tableH)
	newTable.SetColumns(m.tableColumns, tableW)
	newTable.SetRows(m.resources)
	newTable.SetTypeContext(m.tableTypeName, m.tableTypeName != "")
	newTable.SetFocused(m.activePane == TablePane)
	m.table = newTable

	prevDetail := m.detail
	m.detail = components.NewDetailViewModel(tableW, mainH)
	m.detail.SetResourceContext(prevDetail.ResourceKind(), prevDetail.ResourceName())
	m.detail.SetLoading(prevDetail.Loading())
	m.detail.SetFocused(m.activePane == DetailPane)
	m.detail.SetMode(prevDetail.Mode())
	m.detail.SetDescribeAvailable(prevDetail.DescribeAvailable())

	m.quota.SetSize(tableW, mainH)
	m.quota.SetFocused(m.activePane == QuotaDashboardPane)

	m.activity.SetSize(tableW, mainH)
	m.activity.SetFocused(m.activePane == ActivityPane)

	m.activityDashboard.SetSize(tableW, mainH)
	m.activityDashboard.SetFocused(m.activePane == ActivityDashboardPane)

	m.history.SetSize(tableW, mainH)
	m.history.SetFocused(m.activePane == HistoryPane)

	m.diff.SetSize(tableW, mainH)
	m.diff.SetFocused(m.activePane == DiffPane)

	m.ctxOverlay = components.NewCtxSwitcherModel(m.tuiCtx.Config, m.width, m.height)
	m.helpOverlay.Width = m.width
	m.helpOverlay.Height = m.height
	m.helpOverlay.ShowDeleteHint = m.activePane == TablePane || m.activePane == DetailPane
	m.helpOverlay.ShowConditionsHint = m.activePane == DetailPane // AC#22

	if rt, ok := m.sidebar.SelectedType(); ok {
		m.table.SetHoveredType(rt)
	}
	m.refreshLandingInputs()

	return m
}

func (m AppModel) View() string {
	header := m.header.View()

	var mainContent string
	switch m.activePane {
	case DetailPane:
		mainContent = lipgloss.JoinHorizontal(lipgloss.Top,
			m.sidebar.View(),
			m.detail.View(),
		)
	case QuotaDashboardPane:
		mainContent = lipgloss.JoinHorizontal(lipgloss.Top,
			m.sidebar.View(),
			m.quota.View(),
		)
	case ActivityPane:
		mainContent = lipgloss.JoinHorizontal(lipgloss.Top,
			m.sidebar.View(),
			m.activity.View(),
		)
	case ActivityDashboardPane:
		mainContent = lipgloss.JoinHorizontal(lipgloss.Top,
			m.sidebar.View(),
			m.activityDashboard.View(),
		)
	case HistoryPane:
		mainContent = lipgloss.JoinHorizontal(lipgloss.Top,
			m.sidebar.View(),
			m.history.View(),
		)
	case DiffPane:
		mainContent = lipgloss.JoinHorizontal(lipgloss.Top,
			m.sidebar.View(),
			m.diff.View(),
		)
	default:
		rightCol := m.tableView()
		mainContent = lipgloss.JoinHorizontal(lipgloss.Top,
			m.sidebar.View(),
			rightCol,
		)
	}

	status := m.statusBar.View()

	rows := []string{header, mainContent}
	if m.filterBar.Focused() {
		rows = append(rows, m.filterBar.View())
	}
	rows = append(rows, status)
	base := lipgloss.JoinVertical(lipgloss.Left, rows...)

	switch m.overlay {
	case CtxSwitcherOverlay:
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
			m.ctxOverlay.View(),
			lipgloss.WithWhitespaceBackground(styles.OverlayBackdrop),
		)
	case HelpOverlayID:
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
			m.helpOverlay.View(),
			lipgloss.WithWhitespaceBackground(styles.OverlayBackdrop),
		)
	case DeleteConfirmationOverlay:
		return m.deleteConfirmation.View(m.width, m.height)
	}

	return base
}
