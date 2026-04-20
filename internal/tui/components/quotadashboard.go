package components

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"go.datum.net/datumctl/internal/tui/data"
	"go.datum.net/datumctl/internal/tui/styles"
)

type Grouping int

const (
	GroupingGrouped Grouping = iota
	GroupingFlat
)

type QuotaDashboardModel struct {
	width, height        int
	focused              bool
	grouping             Grouping
	loading              bool
	refreshing           bool      // FB-063: true when re-fetching with existing bucket data still visible
	loadErr              error
	spinner              spinner.Model
	buckets              []data.AllowanceBucket
	filter               string
	cursor               int
	vp                   viewport.Model
	ctxLabel             string
	activeConsumerKind string
	activeConsumerName string
	siblingRestricted  bool
	registrations      []data.ResourceRegistration // nil until fetch completes
	fetchedAt          time.Time                   // FB-043: time of last successful bucket fetch
	originLabel        string                      // FB-088: human-readable return destination for [3] hint
	refreshFailed      bool                        // FB-060: true after a failed refresh, cleared by SetBuckets
}

func (m *QuotaDashboardModel) SetOriginLabel(label string) { m.originLabel = label }

func NewQuotaDashboardModel(width, height int, ctxLabel string) QuotaDashboardModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	m := QuotaDashboardModel{
		spinner:  s,
		ctxLabel: ctxLabel,
		grouping: GroupingGrouped,
	}
	m.SetSize(width, height)
	return m
}

func (m QuotaDashboardModel) Init() tea.Cmd { return m.spinner.Tick }

func (m QuotaDashboardModel) Update(msg tea.Msg) (QuotaDashboardModel, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			m.moveCursor(1)
		case "k", "up":
			m.moveCursor(-1)
		case "s":
			m.ToggleGrouping()
		}
		return m, nil
	}
	m.vp, cmd = m.vp.Update(msg)
	return m, cmd
}

func (m *QuotaDashboardModel) moveCursor(delta int) {
	items := m.orderedItems()
	bucketCount := countBucketItems(items)
	if bucketCount == 0 {
		return
	}
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= bucketCount {
		m.cursor = bucketCount - 1
	}
	m.scrollToCursor(items)
}

// scrollToCursor adjusts the viewport so the current cursor block is visible.
func (m *QuotaDashboardModel) scrollToCursor(items []viewItem) {
	lineOffset := 0
	bucketIdx := 0
	for _, item := range items {
		switch {
		case item.isDivider:
			lineOffset += 2
		case item.isGroupHeader:
			lineOffset += 1
		case item.isSiblingConsume:
			lineOffset += 1
		case item.isTreeRow:
			if bucketIdx == m.cursor {
				// Scroll to group header (1 line back from parent row start, or at child).
				m.vp.SetYOffset(max(0, lineOffset))
				return
			}
			bucketIdx++
			lineOffset += 1
			if item.connector == "└─ " {
				lineOffset += 1 // blank after group
			}
		default:
			if bucketIdx == m.cursor {
				m.vp.SetYOffset(lineOffset)
				return
			}
			bucketIdx++
			lineOffset += 4 // 3 block lines + 1 blank
		}
	}
}

func (m QuotaDashboardModel) View() string {
	if m.height < 6 {
		return styles.PaneBorder(m.focused).Render(m.buildMainContent())
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		m.titleBar(),
		m.titleRule(),
		m.vp.View(),
		m.footerRule(),
		m.scrollFooter(),
	)
	return styles.PaneBorder(m.focused).Render(content)
}

func (m QuotaDashboardModel) buildMainContent() string {
	switch {
	case m.loading && len(m.buckets) == 0:
		return m.spinner.View() + " Loading quota data…"
	case m.loadErr != nil:
		title, detail := titleAndDetailForError(m.loadErr, "Could not load allowance buckets")
		sev := data.SeverityOfClassified(m.loadErr)
		return RenderErrorBlock(ErrorBlock{
			Title:    title,
			Detail:   detail,
			Actions:  actionsForSeverity(sev, "back"),
			Severity: sev,
			Width:    m.width,
		})
	}

	filtered := m.filteredBuckets()
	if len(filtered) == 0 {
		muted := lipgloss.NewStyle().Foreground(styles.Muted)
		backLabel := m.originLabel
		if backLabel == "" {
			backLabel = "navigation"
		}
		lines := []string{
			"",
			muted.Render("  No allowance buckets configured for this context."),
			"",
			muted.Render(fmt.Sprintf("  [Esc] back to %s", backLabel)),
		}
		return strings.Join(lines, "\n")
	}

	items := m.orderedItemsFrom(filtered)
	return m.renderItems(items)
}

func (m *QuotaDashboardModel) refreshViewport() {
	if (m.loading && len(m.buckets) == 0) || m.loadErr != nil {
		return
	}
	filtered := m.filteredBuckets()
	items := m.orderedItemsFrom(filtered)
	m.vp.SetContent(m.renderItems(items))
	m.scrollToCursor(items)
}

func (m QuotaDashboardModel) renderItems(items []viewItem) string {
	muted := lipgloss.NewStyle().Foreground(styles.Muted)
	accentBold := lipgloss.NewStyle().Foreground(styles.Accent).Bold(true)
	secondary := lipgloss.NewStyle().Foreground(styles.Secondary).Bold(true)

	var sb strings.Builder
	bucketIdx := 0
	for i, item := range items {
		switch {
		case item.isDivider:
			sb.WriteString(muted.Render("  ── org ──"))
			sb.WriteString("\n\n")

		case item.isGroupHeader:
			sb.WriteString(accentBold.Render("  " + item.groupLabel + " • Quota"))
			sb.WriteString("\n")

		case item.isSiblingConsume:
			sb.WriteString(m.renderSiblingConsumeRow(item, muted))
			sb.WriteString("\n")

		case item.isTreeRow:
			selected := bucketIdx == m.cursor
			sb.WriteString(m.renderDashboardTreeRow(item, selected, muted, secondary))
			sb.WriteString("\n")
			// Blank line after the last item in the tree group (connector "└─").
			if item.connector == "└─ " {
				sb.WriteString("\n")
			}
			bucketIdx++

		default:
			block := m.renderBucketBlock(item.bucket, bucketIdx == m.cursor, accentBold)
			sb.WriteString(block)
			sb.WriteString("\n\n")
			bucketIdx++
		}
		_ = i
	}
	return sb.String()
}

// renderDashboardTreeRow renders a single compact tree row for S2.
// Format: <connector><consumerLabel padded>  [bar] NNN/NNN
func (m QuotaDashboardModel) renderDashboardTreeRow(
	item viewItem, selected bool,
	muted, secondary lipgloss.Style,
) string {
	b := item.bucket
	// overhead: connector(3) + consumerLabel(20) + bar+counts(same as renderBarLine overhead=26)
	barWidth := max(10, m.width-3-20-26)

	connector := muted.Render(item.connector)
	labelStr := consumerLabel(b.ConsumerKind, b.ConsumerName)
	var label string
	if item.isActiveConsumer {
		label = secondary.Render(labelStr)
	} else {
		label = muted.Render(labelStr)
	}
	if selected {
		connector = lipgloss.NewStyle().Foreground(styles.Accent).Render(item.connector)
	}

	sibNote := ""
	if item.sibRestricted {
		sibNote = muted.Render("  (other projects' usage hidden)")
	}

	bar := renderTreeBarCounts(b, barWidth)
	return connector + label + sibNote + bar
}

// renderSiblingConsumeRow renders the │    sibling-consume row for S2.
func (m QuotaDashboardModel) renderSiblingConsumeRow(item viewItem, muted lipgloss.Style) string {
	barWidth := max(10, m.width-3-20-26)
	pct := float64(0)
	if item.sibLimit > 0 {
		pct = float64(item.sibAllocated) / float64(item.sibLimit)
	}
	filled := min(barWidth, max(0, int(float64(barWidth)*pct)))
	empty := barWidth - filled
	barStr := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	countStr := fmt.Sprintf(" %3d of %-3d by %d %s", item.sibAllocated, item.sibLimit, item.sibCount, item.sibKind)
	return muted.Render("│    sibling-consume  ") +
		muted.Render("["+barStr+"]") +
		muted.Render(countStr)
}

func (m QuotaDashboardModel) renderBucketBlock(b data.AllowanceBucket, selected bool, accentBold lipgloss.Style) string {
	group, name := data.SplitResourceType(b.ResourceType)
	label := name
	if dn := data.ResolveDescription(m.registrations, group, name); dn != "" {
		label = dn
	}
	header := accentBold.Render("  " + label)
	if selected {
		accent := lipgloss.NewStyle().Foreground(styles.Accent)
		header = accent.Render("▶ ") + accentBold.Render(label)
	}

	barLine := renderBarLine(b, m.width)

	return header + "\n" + barLine
}

func (m QuotaDashboardModel) titleBar() string {
	w := m.width
	accentBold := lipgloss.NewStyle().Foreground(styles.Accent).Bold(true)
	muted := lipgloss.NewStyle().Foreground(styles.Muted)

	baseLeft := accentBold.Render("quota usage")
	if m.ctxLabel != "" {
		baseLeft += muted.Render(" — " + m.ctxLabel)
	}

	var hint string
	var freshPrefix string
	var refreshingLabel string
	var refreshFailedLabel string
	if w < 80 {
		if m.originLabel != "" {
			backHint := accentBold.Render("[3]") + muted.Render(" back to "+m.originLabel+"  ")
			hint = backHint + muted.Render("[↑/↓] [t] [s] [r]")
		} else {
			hint = muted.Render("[↑/↓] [t] [s] [r]")
		}
		freshPrefix = " · "
		refreshingLabel = " ↻"
		refreshFailedLabel = " ✗"
	} else {
		if m.originLabel != "" {
			backHint := accentBold.Render("[3]") + muted.Render(" back to "+m.originLabel+"  ")
			hint = backHint + muted.Render("[↑/↓] move  [t] table  [s] group  [r] refresh")
		} else {
			hint = muted.Render("[↑/↓] move  [t] table  [s] group  [r] refresh")
		}
		freshPrefix = "  updated "
		refreshingLabel = "  ⟳ refreshing…"
		refreshFailedLabel = "  ✗ refresh failed"
	}

	warning := lipgloss.NewStyle().Foreground(styles.Warning)

	left := baseLeft
	if m.refreshing {
		refresh := muted.Render(refreshingLabel)
		candidate := baseLeft + refresh
		if w-lipgloss.Width(candidate)-lipgloss.Width(hint) >= 2 {
			left = candidate
		}
	} else if m.refreshFailed {
		fail := warning.Render(refreshFailedLabel)
		candidate := baseLeft + fail
		if w-lipgloss.Width(candidate)-lipgloss.Width(hint) >= 2 {
			left = candidate
		}
	} else if !m.fetchedAt.IsZero() {
		fresh := muted.Render(freshPrefix + HumanizeSince(m.fetchedAt))
		candidate := baseLeft + fresh
		if w-lipgloss.Width(candidate)-lipgloss.Width(hint) >= 2 {
			left = candidate
		}
	}

	gap := max(1, w-lipgloss.Width(left)-lipgloss.Width(hint))
	return left + strings.Repeat(" ", gap) + hint
}

func (m QuotaDashboardModel) titleRule() string {
	return lipgloss.NewStyle().Foreground(styles.InactiveBorderColor).
		Render(strings.Repeat("─", m.width))
}

func (m QuotaDashboardModel) footerRule() string {
	return lipgloss.NewStyle().Foreground(styles.InactiveBorderColor).
		Render(strings.Repeat("─", m.width))
}

func (m QuotaDashboardModel) scrollFooter() string {
	pct := m.vp.ScrollPercent()
	var label string
	switch pct {
	case 0.0:
		label = "top"
	case 1.0:
		label = "100%"
	default:
		label = fmt.Sprintf("%d%%", int(pct*100))
	}

	secondary := lipgloss.NewStyle().Foreground(styles.Secondary).Bold(true)
	muted := lipgloss.NewStyle().Foreground(styles.Muted)

	styledLabel := secondary.Render(label)
	labelWidth := lipgloss.Width(styledLabel)
	ruleWidth := max(m.width-labelWidth-2, 0)

	return muted.Render(strings.Repeat("─", ruleWidth)) + "  " + styledLabel
}

// SetSize updates the component dimensions and rebuilds the viewport.
func (m *QuotaDashboardModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	vpH := h
	if h >= 6 {
		vpH = max(h-4, 1)
	}
	m.vp.Width = w
	m.vp.Height = vpH
	m.refreshViewport()
}

func (m *QuotaDashboardModel) SetFocused(focused bool) {
	m.focused = focused
}

func (m *QuotaDashboardModel) SetLoading(loading bool) {
	m.loading = loading
	if !loading {
		m.refreshing = false // FB-063: BucketsLoadedMsg clears refreshing state automatically
	}
}

func (m *QuotaDashboardModel) SetRefreshing(b bool) {
	m.refreshing = b
}

func (m *QuotaDashboardModel) SetLoadErr(err error) {
	m.loadErr = err
}

func (m *QuotaDashboardModel) SetBuckets(buckets []data.AllowanceBucket) {
	prevName := m.selectedBucketName()
	m.buckets = buckets
	m.refreshFailed = false // FB-060: success clears failure indicator
	m.snapCursorToName(prevName)
	m.refreshViewport()
}

func (m *QuotaDashboardModel) SetFilter(filter string) {
	m.filter = filter
	m.cursor = 0
	m.refreshViewport()
}

func (m *QuotaDashboardModel) ToggleGrouping() {
	prevName := m.selectedBucketName()
	if m.grouping == GroupingGrouped {
		m.grouping = GroupingFlat
	} else {
		m.grouping = GroupingGrouped
	}
	m.snapCursorToName(prevName)
	m.refreshViewport()
}

func (m *QuotaDashboardModel) ResetGrouping() {
	m.grouping = GroupingGrouped
}

func (m *QuotaDashboardModel) SetCtxLabel(label string) {
	m.ctxLabel = label
}

func (m *QuotaDashboardModel) SetActiveConsumer(kind, name string) {
	m.activeConsumerKind = kind
	m.activeConsumerName = name
	m.refreshViewport()
}

func (m *QuotaDashboardModel) SetSiblingRestricted(v bool) {
	m.siblingRestricted = v
	m.refreshViewport()
}

// SetRegistrations sets the ResourceRegistration slice used to resolve display
// names in dashboard labels. Pass nil to fall back to short-name labels.
func (m *QuotaDashboardModel) SetRegistrations(regs []data.ResourceRegistration) {
	m.registrations = regs
	m.refreshViewport()
}

func (m *QuotaDashboardModel) SetBucketFetchedAt(t time.Time) {
	m.fetchedAt = t
}

// SetRefreshFailed marks or clears the failed-refresh indicator (FB-060).
func (m *QuotaDashboardModel) SetRefreshFailed(failed bool) {
	m.refreshFailed = failed
}

func (m QuotaDashboardModel) SpinnerFrame() string { return m.spinner.View() }
func (m QuotaDashboardModel) IsLoading() bool      { return m.loading }
func (m QuotaDashboardModel) IsRefreshing() bool   { return m.refreshing }
func (m QuotaDashboardModel) HasBuckets() bool     { return len(m.buckets) > 0 }

func (m QuotaDashboardModel) SelectedBucket() (data.AllowanceBucket, bool) {
	filtered := m.filteredBuckets()
	ordered := m.orderedItemsFrom(filtered)
	bucketIdx := 0
	for _, item := range ordered {
		if item.isDivider {
			continue
		}
		if bucketIdx == m.cursor {
			return item.bucket, true
		}
		bucketIdx++
	}
	return data.AllowanceBucket{}, false
}

func (m QuotaDashboardModel) selectedBucketName() string {
	if b, ok := m.SelectedBucket(); ok {
		return b.Name
	}
	return ""
}

func (m *QuotaDashboardModel) snapCursorToName(name string) {
	if name == "" {
		m.cursor = 0
		return
	}
	filtered := m.filteredBuckets()
	ordered := m.orderedItemsFrom(filtered)
	bucketIdx := 0
	for _, item := range ordered {
		if item.isDivider || item.isGroupHeader || item.isSiblingConsume {
			continue
		}
		if item.bucket.Name == name {
			m.cursor = bucketIdx
			return
		}
		bucketIdx++
	}
	// Name not found: clamp
	count := countBucketItems(ordered)
	if m.cursor >= count {
		m.cursor = max(0, count-1)
	}
}

func (m QuotaDashboardModel) filteredBuckets() []data.AllowanceBucket {
	if m.filter == "" {
		return m.buckets
	}
	lower := strings.ToLower(m.filter)
	var out []data.AllowanceBucket
	for _, b := range m.buckets {
		if strings.Contains(strings.ToLower(b.ResourceType), lower) {
			out = append(out, b)
		}
	}
	return out
}

// viewItem is an element in the ordered display list.
type viewItem struct {
	isDivider bool
	bucket    data.AllowanceBucket

	// Tree rendering fields (zero values = standalone flat bucket)
	isGroupHeader    bool   // resource-type header row (no cursor position)
	groupLabel       string // label for group header
	isSiblingConsume bool   // sibling-consume aggregation row (no cursor position)
	sibAllocated int64
	sibLimit     int64
	sibCount     int
	sibKind      string // pluralized kind ("projects")
	connector    string // "├─ " / "└─ " / "" (standalone)
	isTreeRow        bool   // true → single-line tree row; false → 3-line standalone block
	isActiveConsumer bool
	sibRestricted    bool // parent row: append "(sibling data unavailable)"
}

func (m QuotaDashboardModel) orderedItems() []viewItem {
	return m.orderedItemsFrom(m.filteredBuckets())
}

func (m QuotaDashboardModel) orderedItemsFrom(buckets []data.AllowanceBucket) []viewItem {
	if m.grouping == GroupingFlat {
		sorted := make([]data.AllowanceBucket, len(buckets))
		copy(sorted, buckets)
		sortByPct(sorted)
		items := make([]viewItem, len(sorted))
		for i, b := range sorted {
			items[i] = viewItem{bucket: b}
		}
		return items
	}

	// Group buckets by ResourceType, then build tree-aware items per group.
	groups := groupBucketsByResourceType(buckets)
	// Sort groups by max child percentage (active child pct), tiebreak by parent pct.
	sort.SliceStable(groups, func(i, j int) bool {
		pi := maxChildPct(groups[i], m.activeConsumerKind, m.activeConsumerName)
		pj := maxChildPct(groups[j], m.activeConsumerKind, m.activeConsumerName)
		if pi != pj {
			return pi > pj
		}
		return parentPct(groups[i]) > parentPct(groups[j])
	})

	var items []viewItem
	for _, group := range groups {
		items = append(items, m.buildTreeItems(group)...)
	}
	return items
}

// groupBucketsByResourceType groups buckets into slices by ResourceType, preserving
// within-group order.
func groupBucketsByResourceType(buckets []data.AllowanceBucket) [][]data.AllowanceBucket {
	seen := map[string]int{}
	var groups [][]data.AllowanceBucket
	for _, b := range buckets {
		if idx, ok := seen[b.ResourceType]; ok {
			groups[idx] = append(groups[idx], b)
		} else {
			seen[b.ResourceType] = len(groups)
			groups = append(groups, []data.AllowanceBucket{b})
		}
	}
	return groups
}

// buildTreeItems converts a single resource-type group into tree-aware viewItems.
func (m QuotaDashboardModel) buildTreeItems(group []data.AllowanceBucket) []viewItem {
	tree := data.ClassifyTreeBuckets(group, m.activeConsumerKind, m.activeConsumerName)
	if !tree.HasTree {
		// Standalone: render each bucket as a classic 3-line block.
		sortByPct(group)
		var items []viewItem
		for _, b := range group {
			items = append(items, viewItem{bucket: b})
		}
		return items
	}

	rtGroup, rtName := data.SplitResourceType(tree.Parent.ResourceType)
	label := rtName
	if dn := data.ResolveDescription(m.registrations, rtGroup, rtName); dn != "" {
		label = dn
	}
	var items []viewItem
	// Group header
	items = append(items, viewItem{isGroupHeader: true, groupLabel: label})

	// Parent row
	parentItem := viewItem{
		bucket:        *tree.Parent,
		connector:     "├─ ",
		isTreeRow:     true,
		sibRestricted: m.siblingRestricted,
	}
	items = append(items, parentItem)

	// Sibling-consume row
	hasSibRow := len(tree.Siblings) > 0 && !m.siblingRestricted
	if hasSibRow {
		var sibAllocated int64
		for _, s := range tree.Siblings {
			sibAllocated += s.Allocated
		}
		sibKind := ""
		if len(tree.Siblings) > 0 {
			sibKind = pluralizeKind(tree.Siblings[0].ConsumerKind, len(tree.Siblings))
		}
		items = append(items, viewItem{
			isSiblingConsume: true,
			sibAllocated:     sibAllocated,
			sibLimit:         tree.Parent.Limit,
			sibCount:         len(tree.Siblings),
			sibKind:          sibKind,
		})
	}

	// Child row
	items = append(items, viewItem{
		bucket:           *tree.ActiveChild,
		connector:        "└─ ",
		isTreeRow:        true,
		isActiveConsumer: true,
	})

	return items
}

func maxChildPct(group []data.AllowanceBucket, activeKind, activeName string) float64 {
	for _, b := range group {
		if strings.EqualFold(b.ConsumerKind, activeKind) && b.ConsumerName == activeName {
			return bucketPct(b)
		}
	}
	// No active child — use max project pct
	var max float64
	for _, b := range group {
		if strings.EqualFold(b.ConsumerKind, "project") {
			if p := bucketPct(b); p > max {
				max = p
			}
		}
	}
	return max
}

func parentPct(group []data.AllowanceBucket) float64 {
	for _, b := range group {
		if strings.EqualFold(b.ConsumerKind, "organization") {
			return bucketPct(b)
		}
	}
	return 0
}


func sortByPct(buckets []data.AllowanceBucket) {
	sort.SliceStable(buckets, func(i, j int) bool {
		pi := bucketPct(buckets[i])
		pj := bucketPct(buckets[j])
		if pi != pj {
			return pi > pj
		}
		return buckets[i].ResourceType < buckets[j].ResourceType
	})
}

func bucketPct(b data.AllowanceBucket) float64 {
	if b.Limit == 0 {
		return 0
	}
	return float64(b.Allocated) / float64(b.Limit)
}

func countBucketItems(items []viewItem) int {
	n := 0
	for _, item := range items {
		if !item.isDivider && !item.isGroupHeader && !item.isSiblingConsume {
			n++
		}
	}
	return n
}
