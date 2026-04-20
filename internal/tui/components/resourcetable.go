package components

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	tuictx "go.datum.net/datumctl/internal/tui/context"
	"go.datum.net/datumctl/internal/tui/data"
	"go.datum.net/datumctl/internal/tui/styles"
)

// AttentionItem is a pre-computed "needs attention" entry for the welcome dashboard spotlight (FB-042).
type AttentionItem struct {
	Kind    string // "quota" | "condition"
	Label   string // e.g. "dnszones quota" | "backend/api-gw"
	Detail  string // e.g. "91% allocated" | "condition: Degraded"
	NavKey  string // e.g. "[3]" | "[Enter]"
	NavHint string // e.g. "quota dashboard" | "view"
}

// quickJumpEntry maps a single-key shortcut to a resource type name match.
type quickJumpEntry struct {
	key          string
	matchSubstrs []string // any match shows the entry
	label        string
}

// quickJumpLabel returns the curated display label for a resource type name,
// falling back to the raw typeName when no quickJumpTable entry matches. FB-090.
func quickJumpLabel(typeName string) string {
	lower := strings.ToLower(typeName)
	for _, e := range quickJumpTable {
		for _, sub := range e.matchSubstrs {
			if strings.Contains(lower, sub) {
				return e.label
			}
		}
	}
	return typeName
}

// quickJumpTable is the ordered set of quick-jump entries for the welcome dashboard (§6 of FB-042 spec).
var quickJumpTable = []quickJumpEntry{
	{"n", []string{"namespaces"}, "namespaces"},
	{"b", []string{"backends"}, "backends"},
	{"w", []string{"workloads"}, "workloads"},
	{"p", []string{"projects"}, "projects"},
	{"g", []string{"gateways"}, "gateways"},
	{"v", []string{"services"}, "services"},
	{"i", []string{"ingresses"}, "ingresses"},
	{"z", []string{"dnsrecordsets", "dnszones"}, "dns"},
}

type ResourceTableModel struct {
	table        table.Model
	spinner      spinner.Model
	allRows      []data.ResourceRow
	filteredRows []data.ResourceRow
	filter       string
	loadState    data.LoadState
	typeName     string
	hoveredType  data.ResourceType
	focused      bool
	columns      []string
	tableWidth   int
	tableHeight  int

	// FB-005: in-pane error card state.
	loadErr     error
	errSeverity data.ErrorSeverity

	// Landing-screen inputs (FB-015). These power welcomePanel() when
	// typeName == "". Zero-values yield a safe degraded rendering.
	tuiCtx             tuictx.TUIContext
	buckets            []data.AllowanceBucket
	bucketLoading      bool
	bucketErr          error
	bucketUnauthorized bool
	registrations      []data.ResourceRegistration
	staleBanner        bool
	staleAge           string
	forceDashboard     bool // FB-041: show welcome panel even when typeName != ""

	// FB-042: enhanced welcome dashboard inputs.
	activityRows       []data.ActivityRow // nil = not yet loaded; empty = no activity
	activityLoading    bool               // true while first fetch is in flight
	activityFetchFailed bool              // FB-082: true when last fetch returned an error
	pendingQuotaOpen   bool              // FB-099: substitutes [3] strip label to "cancel"
	attentionItems     []AttentionItem    // pre-computed by model.go
}

func NewResourceTableModel(tableWidth, totalHeight int) ResourceTableModel {
	t := table.New(
		table.WithFocused(true),
		table.WithHeight(totalHeight),
		table.WithStyles(table.Styles{
			Header:   styles.TableStyle,
			Cell:     lipgloss.NewStyle().Padding(0, 1),
			Selected: styles.SelectedRowStyle,
		}),
	)
	s := spinner.New()
	s.Spinner = spinner.Dot
	return ResourceTableModel{
		table:       t,
		spinner:     s,
		tableWidth:  tableWidth,
		tableHeight: totalHeight,
	}
}

func (m ResourceTableModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m ResourceTableModel) Update(msg tea.Msg) (ResourceTableModel, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	default:
		m.table, cmd = m.table.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m ResourceTableModel) View() string {
	var content string

	switch {
	case m.loadState == data.LoadStateLoading:
		content = m.spinner.View() + fmt.Sprintf(" Loading %s...", m.typeName)
	case m.loadState == data.LoadStateError && m.loadErr != nil:
		innerW := max(1, m.tableWidth-3)
		innerH := max(1, m.tableHeight-2)
		card := RenderErrorBlock(ErrorBlock{
			Title:    sanitizedTitleForError(m.loadErr, "Could not load "+m.typeName),
			Detail:   SanitizeErrMsg(m.loadErr),
			Actions:  actionsForSeverity(m.errSeverity, "back to navigation"),
			Severity: m.errSeverity,
			Width:    innerW,
		})
		content = lipgloss.Place(innerW, innerH, lipgloss.Center, lipgloss.Center, card)
	case m.typeName == "" || m.forceDashboard:
		content = m.welcomePanel()
	case len(m.filteredRows) == 0 && m.filter != "":
		noResults := lipgloss.NewStyle().Foreground(styles.Muted).
			Render(fmt.Sprintf("No results for %q", m.filter))
		escHint := lipgloss.NewStyle().Foreground(styles.Accent).
			Render("[Esc] clear filter")
		block := lipgloss.JoinVertical(lipgloss.Center, noResults, "", escHint)
		content = lipgloss.NewStyle().
			Width(m.tableWidth).Align(lipgloss.Center).
			Render(block)
	case len(m.filteredRows) == 0:
		content = lipgloss.NewStyle().Foreground(styles.Muted).
			Width(m.tableWidth).Align(lipgloss.Center).
			Render(fmt.Sprintf("No %s found", m.typeName))
	default:
		content = m.table.View()
	}

	return styles.PaneBorder(m.focused).Render(content)
}

func (m ResourceTableModel) welcomePanel() string {
	// Effective content width (subtract 2-char horizontal pane padding × 2).
	contentW := max(1, m.tableWidth-4)
	contentH := max(1, m.tableHeight-4)

	muted := lipgloss.NewStyle().Foreground(styles.Muted)
	sep := muted.Render(strings.Repeat("─", max(1, contentW)))

	// Width/height gates (§9 of FB-042 spec).
	barsMode := contentW >= 80
	wideEnough := contentW >= 50 // gates S3/S4/S5

	showS5 := contentH >= 30 && contentW >= 60
	showS3 := contentH >= 24 && wideEnough
	showS4 := contentH >= 18 && wideEnough
	showS2List := contentH >= 18
	showS6 := contentH >= 12
	showStaleBanner := m.staleBanner && contentH >= 12

	var regions []string

	// Stale-context banner.
	if showStaleBanner {
		warnBold := lipgloss.NewStyle().Foreground(styles.Warning).Bold(true)
		warn := lipgloss.NewStyle().Foreground(styles.Warning)
		regions = append(regions,
			warnBold.Render("⚠ ")+warn.Render(
				fmt.Sprintf("Context cache last refreshed %s ago — press `c` to switch or `r` to refresh", m.staleAge)),
			"")
	}

	// S1: identity + greeting.
	regions = append(regions, m.renderHeaderBand(contentW), "")

	// S1/S2 separator.
	regions = append(regions, sep, "")

	// S2: platform health (enhanced, full-width).
	regions = append(regions, m.renderPlatformHealthSection(contentW, !barsMode, showS2List))

	// S3: recent activity teaser.
	if showS3 {
		regions = append(regions, "", m.renderActivitySection(contentW))
	}

	// S4: quick-jump shortcuts.
	if showS4 {
		// FB-105: all-clear flavor line when no attention items and no activity.
		if len(m.attentionItems) == 0 && !m.activityLoading && len(m.activityRows) == 0 {
			regions = append(regions, muted.Render("all clear · no issues detected"))
		}
		if qj := m.renderQuickJumpSection(contentW); qj != "" {
			regions = append(regions, "", qj)
		}
	}

	// S5: needs attention spotlight.
	if showS5 {
		if att := m.renderAttentionSection(contentW); att != "" {
			regions = append(regions, "", att)
		}
	}

	// S7: hovered resource type documentation (FB-104).
	if m.hoveredType.Kind != "" {
		regions = append(regions, "", m.renderHoveredTypeSection(contentW))
	}

	// S6: keybind strip.
	if showS6 {
		regions = append(regions, "", sep, "", m.renderKeybindStrip(contentW))
	}

	body := lipgloss.JoinVertical(lipgloss.Left, regions...)
	return lipgloss.NewStyle().Padding(2, 2).Render(body)
}

// renderPlatformHealthSection renders the S2 platform health block (full-width, single-column).
// textOnly=true suppresses progress bars. showList=true renders the top-3 rows.
func (m ResourceTableModel) renderPlatformHealthSection(contentW int, textOnly, showList bool) string {
	accent := lipgloss.NewStyle().Foreground(styles.Accent).Bold(true)
	muted := lipgloss.NewStyle().Foreground(styles.Muted)
	success := lipgloss.NewStyle().Foreground(styles.Success).Bold(true)
	warn := lipgloss.NewStyle().Foreground(styles.Warning).Bold(true)

	leftHeader := accent.Render("Platform health")

	if m.bucketLoading && m.buckets == nil {
		return leftHeader + "\n\n" + muted.Render("⟳ loading platform health…")
	}
	if m.bucketErr != nil {
		if m.bucketUnauthorized {
			return leftHeader + "\n\n" + muted.Render("Platform health unavailable")
		}
		return leftHeader + "\n\n" + muted.Render("Platform health temporarily unavailable")
	}

	ak, an := m.activeConsumer()
	summary := data.ComputePlatformHealthSummary(m.buckets, ak, an, m.registrations)

	if summary.TotalGovernedTypes == 0 {
		return leftHeader + "\n\n" + muted.Render("No governed resource types in this project")
	}

	// Narrow: no status line, just a summary text below header.
	if contentW < 50 {
		summaryText := fmt.Sprintf("%d of %d governed types ≥80%% allocated", summary.ConstrainedTypes, summary.TotalGovernedTypes)
		return leftHeader + "\n" + muted.Render(summaryText)
	}

	// Build status line (right-aligned).
	var statusLine string
	if summary.ConstrainedTypes == 0 {
		statusLine = success.Render("✓ All clear")
	} else {
		statusLine = warn.Render(fmt.Sprintf("⚠ %d resource type(s) need attention", summary.ConstrainedTypes))
	}
	gap := max(1, contentW-lipgloss.Width(leftHeader)-lipgloss.Width(statusLine))
	headerRow := leftHeader + strings.Repeat(" ", gap) + statusLine

	if !showList {
		return headerRow
	}

	// Full list with top-3 rows.
	rows := []string{headerRow}
	if len(summary.TopThree) > 0 {
		rows = append(rows, "")
		for _, r := range summary.TopThree {
			rows = append(rows, "  "+m.renderQuotaRow(r, contentW-2, textOnly))
		}
		rows = append(rows, "", muted.Render("  (press [3] for full dashboard)"))
	}
	return strings.Join(rows, "\n")
}

// renderActivitySection renders the S3 recent activity teaser block.
func (m ResourceTableModel) renderActivitySection(contentW int) string {
	accentBold := lipgloss.NewStyle().Foreground(styles.Accent).Bold(true)
	muted := lipgloss.NewStyle().Foreground(styles.Muted)
	secondary := lipgloss.NewStyle().Foreground(styles.Secondary)

	left := accentBold.Render("Recent activity")
	var hint string
	if len(m.activityRows) > 0 && !m.activityFetchFailed {
		hint = muted.Render("[4] full dashboard")
	}
	gap := max(1, contentW-lipgloss.Width(left)-lipgloss.Width(hint))
	header := left + strings.Repeat(" ", gap) + hint
	rule := muted.Render(strings.Repeat("─", max(1, contentW)))

	var body string
	switch {
	case m.activityLoading && m.activityRows == nil:
		body = muted.Render("⟳ loading…")
	case m.activityFetchFailed:
		body = muted.Render("activity unavailable")
	case len(m.activityRows) == 0:
		body = muted.Render("no recent activity")
	default:
		rows := m.activityRows
		if len(rows) > 3 {
			rows = rows[:3]
		}
		// FB-082: three-tier narrow-width column-drop contract.
		showResource := contentW >= 65
		var actorW, summaryW int
		if contentW >= 65 {
			actorW = 24
			summaryW = max(8, contentW-65)
		} else if contentW >= 45 {
			actorW = 24
			summaryW = max(8, contentW-37)
		} else {
			actorW = max(1, min(16, contentW-22)) // FB-101: guard against contentW ≤ 22
			summaryW = max(8, contentW-11-actorW)
		}
		var lines []string
		for _, row := range rows {
			age := HumanizeSince(row.Timestamp)
			ageRunes := []rune(age)
			if len(ageRunes) < 7 {
				age = age + strings.Repeat(" ", 7-len(ageRunes))
			} else if len(ageRunes) > 7 {
				age = string(ageRunes[:7])
			}

			actor := row.ActorDisplay
			if actor == "" {
				actor = "system"
			}
			actorRunes := []rune(actor)
			if len(actorRunes) > actorW {
				actor = string(actorRunes[:actorW-1]) + "…"
			} else {
				actor = actor + strings.Repeat(" ", actorW-len(actorRunes))
			}

			summary := row.Summary
			summaryRunes := []rune(summary)
			if len(summaryRunes) > summaryW {
				summary = string(summaryRunes[:summaryW-1]) + "…"
			}

			var line string
			if showResource {
				var resource string
				if row.ResourceRef != nil {
					resource = strings.ToLower(row.ResourceRef.Kind) + "/" + row.ResourceRef.Name
				}
				resourceRunes := []rune(resource)
				if len(resourceRunes) > 28 {
					resource = string(resourceRunes[:27]) + "…"
				} else {
					resource = resource + strings.Repeat(" ", 28-len(resourceRunes))
				}
				line = secondary.Render(age) + "  " + muted.Render(actor) + "  " + muted.Render(resource) + "  " + muted.Render(summary)
			} else {
				line = secondary.Render(age) + "  " + muted.Render(actor) + "  " + muted.Render(summary)
			}
			lines = append(lines, line)
		}
		body = strings.Join(lines, "\n")
	}

	return header + "\n" + rule + "\n" + body
}

// renderQuickJumpSection renders the S4 quick-jump shortcuts row.
// Returns "" when no matching registrations are present.
func (m ResourceTableModel) renderQuickJumpSection(contentW int) string {
	key := lipgloss.NewStyle().Foreground(styles.Accent).Bold(true)
	muted := lipgloss.NewStyle().Foreground(styles.Muted)

	var entryStrings []string
	for _, e := range quickJumpTable {
		if m.hasRegistrationMatch(e.matchSubstrs) {
			entryStrings = append(entryStrings, key.Render("["+e.key+"]")+" "+muted.Render(e.label))
		}
	}
	if len(entryStrings) == 0 {
		return ""
	}

	prefix := muted.Render("jump to:  ")
	full := prefix + strings.Join(entryStrings, "  ")
	if lipgloss.Width(full) <= contentW {
		return full
	}
	for i := len(entryStrings) - 1; i > 0; i-- {
		candidate := prefix + strings.Join(entryStrings[:i], "  ") + " …"
		if lipgloss.Width(candidate) <= contentW {
			return candidate
		}
	}
	return prefix + entryStrings[0]
}

// renderAttentionSection renders the S5 "Needs attention" spotlight block.
// Returns "" when attentionItems is empty.
func (m ResourceTableModel) renderAttentionSection(contentW int) string {
	if len(m.attentionItems) == 0 {
		return ""
	}

	accentBold := lipgloss.NewStyle().Foreground(styles.Accent).Bold(true)
	muted := lipgloss.NewStyle().Foreground(styles.Muted)
	warn := lipgloss.NewStyle().Foreground(styles.Warning)
	secondary := lipgloss.NewStyle().Foreground(styles.Secondary)

	header := accentBold.Render("Needs attention")
	rule := muted.Render(strings.Repeat("─", max(1, contentW)))

	items := make([]AttentionItem, len(m.attentionItems))
	copy(items, m.attentionItems)
	if len(items) > 3 {
		items = items[:3]
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Kind != items[j].Kind {
			return items[i].Kind == "quota"
		}
		return items[i].Detail < items[j].Detail
	})

	var lines []string
	lines = append(lines, header)
	lines = append(lines, rule)

	for _, item := range items {
		var icon string
		if item.Kind == "quota" {
			icon = warn.Render("▲")
		} else {
			icon = warn.Render("⚠")
		}

		label := item.Label
		labelRunes := []rune(label)
		if len(labelRunes) < 30 {
			label = label + strings.Repeat(" ", 30-len(labelRunes))
		} else if len(labelRunes) > 30 {
			label = string(labelRunes[:29]) + "…"
		}

		detail := item.Detail
		detailRunes := []rune(detail)
		if len(detailRunes) < 24 {
			detail = detail + strings.Repeat(" ", 24-len(detailRunes))
		} else if len(detailRunes) > 24 {
			detail = string(detailRunes[:23]) + "…"
		}

		nav := accentBold.Render(item.NavKey) + " " + muted.Render(item.NavHint)
		line := icon + "  " + secondary.Render(label) + "  " + muted.Render(detail) + "  " + nav
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// wrapDescription word-wraps desc to contentW, returning at most 2 lines.
// The second line is truncated with … if text remains. Returns nil for empty desc.
func wrapDescription(desc string, contentW int) []string {
	if desc == "" {
		return nil
	}
	runes := []rune(desc)
	var lines []string
	for len(runes) > 0 && len(lines) < 2 {
		if len(runes) <= contentW {
			lines = append(lines, string(runes))
			break
		}
		end := contentW
		for end > 0 && runes[end-1] != ' ' {
			end--
		}
		if end == 0 {
			end = contentW
		}
		chunk := strings.TrimRight(string(runes[:end]), " ")
		runes = []rune(strings.TrimLeft(string(runes[end:]), " "))
		if len(lines) == 1 && len(runes) > 0 {
			chunkRunes := []rune(chunk)
			maxLen := max(1, contentW-1)
			if len(chunkRunes) > maxLen {
				chunkRunes = chunkRunes[:maxLen]
			}
			chunk = string(chunkRunes) + "…"
			runes = nil
		}
		lines = append(lines, chunk)
	}
	return lines
}

// renderHoveredTypeSection renders the S7 hovered resource type documentation block (FB-104).
func (m ResourceTableModel) renderHoveredTypeSection(contentW int) string {
	accentBold := lipgloss.NewStyle().Foreground(styles.Accent).Bold(true)
	secondary := lipgloss.NewStyle().Foreground(styles.Secondary)
	muted := lipgloss.NewStyle().Foreground(styles.Muted)
	rt := m.hoveredType

	scopeLabel := "Cluster"
	if rt.Namespaced {
		scopeLabel = "Namespaced"
	}
	versionGroup := rt.Group
	if rt.Version != "" {
		versionGroup += " · " + rt.Version
	}
	meta := accentBold.Render(rt.Kind) + "  " +
		secondary.Render(versionGroup) + "  " +
		muted.Render(scopeLabel)
	rule := muted.Render(strings.Repeat("─", max(1, contentW)))

	parts := []string{meta, rule}
	for _, line := range wrapDescription(rt.Description, contentW) {
		parts = append(parts, muted.Render(line))
	}
	return strings.Join(parts, "\n")
}

// hasRegistrationMatch returns true if any registration's Name contains any of the given substrs.
func (m ResourceTableModel) hasRegistrationMatch(substrs []string) bool {
	for _, r := range m.registrations {
		name := strings.ToLower(r.Name)
		for _, s := range substrs {
			if strings.Contains(name, s) {
				return true
			}
		}
	}
	return false
}

// renderHeaderBand renders the "Welcome, <user>" / "<org> / <project>" band
// with optional [READ-ONLY] badge right-aligned on line 2.
func (m ResourceTableModel) renderHeaderBand(contentW int) string {
	muted := lipgloss.NewStyle().Foreground(styles.Muted)
	accentBold := lipgloss.NewStyle().Foreground(styles.Accent).Bold(true)
	secondary := lipgloss.NewStyle().Foreground(styles.Secondary)
	warnBold := lipgloss.NewStyle().Foreground(styles.Warning).Bold(true)

	who := m.tuiCtx.UserName
	if who == "" {
		who = m.tuiCtx.UserEmail
	}
	var line1 string
	if who == "" {
		line1 = muted.Render("Welcome")
	} else {
		line1 = muted.Render("Welcome, ") + accentBold.Render(who)
	}

	// Line 2: <org> [/ <project>]  [READ-ONLY]
	leftParts := []string{}
	if m.tuiCtx.OrgName != "" {
		leftParts = append(leftParts, secondary.Render(m.tuiCtx.OrgName))
	}
	if m.tuiCtx.ProjectName != "" {
		if len(leftParts) > 0 {
			leftParts = append(leftParts, muted.Render(" / "))
		}
		leftParts = append(leftParts, secondary.Render(m.tuiCtx.ProjectName))
	}
	left := strings.Join(leftParts, "")
	var line2 string
	if m.tuiCtx.ReadOnly && contentW >= 60 {
		badge := warnBold.Render("[READ-ONLY]")
		gap := contentW - lipgloss.Width(left) - lipgloss.Width(badge)
		if gap < 1 {
			gap = 1
		}
		line2 = left + strings.Repeat(" ", gap) + badge
	} else {
		line2 = left
	}

	// FB-054: Tab-to-resume hint when a cached table exists.
	// FB-105: orientation hint when no resource types loaded yet (project-scoped, registrations empty).
	var line3 string
	if m.forceDashboard && m.typeName != "" {
		accentBold2 := lipgloss.NewStyle().Foreground(styles.Accent).Bold(true)
		muted2 := lipgloss.NewStyle().Foreground(styles.Muted)

		// FB-089: label-idiom copy at all widths; FB-090: curated label via quickJumpLabel.
		displayName := quickJumpLabel(m.typeName)
		tabKey := accentBold2.Render("[Tab]")
		full := tabKey + muted2.Render(" resume "+displayName+" (cached)")

		if lipgloss.Width(full) <= contentW {
			line3 = full
		} else {
			short := tabKey + muted2.Render(" resume "+displayName)
			if lipgloss.Width(short) <= contentW {
				line3 = short
			} else {
				maxName := max(1, contentW-lipgloss.Width(tabKey)-lipgloss.Width(muted2.Render(" resume …")))
				name := displayName
				if len([]rune(name)) > maxName {
					name = string([]rune(name)[:maxName-1]) + "…"
				}
				line3 = tabKey + muted2.Render(" resume "+name)
			}
		}
	} else if m.tuiCtx.ActiveCtx != nil && m.tuiCtx.ActiveCtx.ProjectID != "" && len(m.registrations) == 0 {
		line3 = muted.Render("→  select a resource type from the sidebar to get started")
	}

	switch {
	case line2 == "" && line3 == "":
		return line1
	case line3 == "":
		return line1 + "\n" + line2
	case line2 == "":
		return line1 + "\n" + line3
	default:
		return line1 + "\n" + line2 + "\n" + line3
	}
}


func (m ResourceTableModel) renderQuotaRow(r data.ConstrainedRow, width int, textOnly bool) string {
	barStyle, suffixStyle, suffix := QuotaBarStyling(r.PercentInt)

	// Label column width: about 40% of available width, min 12.
	labelW := max(12, min(24, width*40/100))
	label := r.Label
	if lipgloss.Width(label) > labelW {
		if labelW >= 2 {
			label = string([]rune(label)[:labelW-1]) + "…"
		} else {
			label = "…"
		}
	}
	labelCell := lipgloss.NewStyle().Width(labelW).Render(lipgloss.NewStyle().Foreground(styles.Secondary).Render(label))

	pctText := fmt.Sprintf(" %3d%%", r.PercentInt)
	styledPct := barStyle.Render(pctText)

	if textOnly {
		line := labelCell + "  " + styledPct
		if suffix != "" {
			line += suffixStyle.Render(suffix)
		}
		return line
	}

	// Bar column: remaining width after label + percent (5 chars) + suffix gutter (8).
	// overhead = labelW + 2 (gutter) + 5 (pct) + 8 (suffix)
	barWidth := max(6, width-labelW-2-5-8)
	var barStr string
	if r.PercentInt >= 100 {
		barStr = strings.Repeat("█", barWidth)
	} else {
		filled := max(0, min(barWidth, barWidth*r.PercentInt/100))
		barStr = strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
	}
	styledBar := barStyle.Render(barStr)

	line := labelCell + "  " + styledBar + styledPct
	if suffix != "" {
		line += suffixStyle.Render(suffix)
	}
	return line
}

// renderKeybindStrip renders a single-line condensed keybind strip. Truncates
// from the right with `…` when natural width exceeds contentW.
func (m ResourceTableModel) renderKeybindStrip(contentW int) string {
	key := lipgloss.NewStyle().Foreground(styles.Primary).Bold(true)
	muted := lipgloss.NewStyle().Foreground(styles.Muted)

	pair := func(k, v string) string {
		if v == "" {
			return key.Render(k)
		}
		return key.Render(k) + " " + muted.Render(v)
	}

	var parts []string
	if m.forceDashboard || m.typeName == "" {
		// Welcome/dashboard context.
		// Tab is omitted when the S1 band already owns the Tab hint (cached-table resume).
		hasCachedTable := m.forceDashboard && m.typeName != ""
		parts = []string{pair("j/k", "move")}
		if !hasCachedTable {
			parts = append(parts, pair("Tab", "next pane"))
		}
		threeLabel := "quota"
		if m.pendingQuotaOpen {
			threeLabel = "cancel"
		}
		parts = append(parts,
			pair("Enter", "select"),
			pair("3", threeLabel),
			pair("4", "activity"),
			pair("c", "ctx"),
			pair("?", "help"),
			pair("q", "quit"),
		)
	} else {
		// Typed-table context: full set including x delete, / filter, and cross-context keys.
		parts = []string{
			pair("j/k", "move"),
			pair("Tab", "next pane"),
			pair("Enter", "select"),
			pair("x", "delete"),
			pair("/", "filter"),
			pair("3", "quota"),
			pair("4", "activity"),
			pair("c", "ctx"),
			pair("?", "help"),
			pair("q", "quit"),
		}
	}
	strip := strings.Join(parts, "  ")
	if lipgloss.Width(strip) <= contentW {
		return strip
	}
	// Truncate from the right.
	// Work in rune/width-safe chunks: shorten by dropping trailing parts.
	for i := len(parts) - 1; i > 0; i-- {
		candidate := strings.Join(parts[:i], "  ") + " …"
		if lipgloss.Width(candidate) <= contentW {
			return candidate
		}
	}
	// Extreme narrow: truncate to just the keys.
	bareParts := []string{"j/k", "Tab", "Ent", "/", "c", "?", "q"}
	for i := len(bareParts); i > 0; i-- {
		candidate := strings.Join(bareParts[:i], "  ")
		if lipgloss.Width(candidate) <= contentW {
			return key.Render(candidate)
		}
	}
	return "…"
}

// activeConsumer returns the (kind, name) pair derived from the TUI context.
func (m ResourceTableModel) activeConsumer() (kind, name string) {
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

// RefreshRows replaces the row set while attempting to keep the cursor on the
// same logical row (by name). Use SetRows for initial load and type-switch.
func (m *ResourceTableModel) RefreshRows(rows []data.ResourceRow) {
	prevCursor := m.table.Cursor()
	prevName := ""
	if prevCursor >= 0 && prevCursor < len(m.filteredRows) {
		prevName = m.filteredRows[prevCursor].Name
	}

	m.allRows = rows
	m.applyFilter() // resets internal cursor to 0

	if len(m.filteredRows) == 0 {
		m.table.SetCursor(0)
		return
	}

	if prevName != "" {
		for i, row := range m.filteredRows {
			if row.Name == prevName {
				m.table.SetCursor(i)
				return
			}
		}
	}

	// Prev row gone. Filter-active: fall to row 0. Otherwise clamp.
	if m.filter != "" {
		m.table.SetCursor(0)
		return
	}
	newIdx := prevCursor
	if newIdx > len(m.filteredRows)-1 {
		newIdx = len(m.filteredRows) - 1
	}
	if newIdx < 0 {
		newIdx = 0
	}
	m.table.SetCursor(newIdx)
}

func (m *ResourceTableModel) SetRows(rows []data.ResourceRow) {
	m.allRows = rows
	m.applyFilter()
}

func (m *ResourceTableModel) SetFilter(filter string) {
	m.filter = filter
	m.applyFilter()
}

func (m *ResourceTableModel) SetFocused(b bool) {
	m.focused = b
	if b {
		m.table.Focus()
	} else {
		m.table.Blur()
	}
}

func (m ResourceTableModel) SpinnerFrame() string {
	return m.spinner.View()
}

func (m *ResourceTableModel) SetLoadState(s data.LoadState) {
	m.loadState = s
}

// SetLoadErr stores the in-pane error card state for FB-005.
// Pass nil to clear (e.g. on retry).
func (m *ResourceTableModel) SetLoadErr(err error, sev data.ErrorSeverity) {
	m.loadErr = err
	m.errSeverity = sev
}

func (m *ResourceTableModel) SetHoveredType(rt data.ResourceType) {
	m.hoveredType = rt
}

// SetTUIContext plumbs the active TUI context for the welcome header band.
func (m *ResourceTableModel) SetTUIContext(tc tuictx.TUIContext) {
	m.tuiCtx = tc
}

// SetBuckets plumbs the latest AllowanceBucket snapshot for the platform-health
// region.
func (m *ResourceTableModel) SetBuckets(b []data.AllowanceBucket) {
	m.buckets = b
}

// SetBucketLoading flags the platform-health region as loading.
func (m *ResourceTableModel) SetBucketLoading(loading bool) {
	m.bucketLoading = loading
}

// SetBucketErr flips the platform-health region into its error placeholder.
func (m *ResourceTableModel) SetBucketErr(err error, unauthorized bool) {
	m.bucketErr = err
	m.bucketUnauthorized = unauthorized
}

// SetRegistrations plumbs the registration snapshot for label resolution in
// the platform-health top-3 rows.
func (m *ResourceTableModel) SetRegistrations(r []data.ResourceRegistration) {
	m.registrations = r
}

// SetStaleCacheAge gates the stale-context banner. showBanner==false hides it
// regardless of ageText.
func (m *ResourceTableModel) SetStaleCacheAge(showBanner bool, ageText string) {
	m.staleBanner = showBanner
	m.staleAge = ageText
}

func (m *ResourceTableModel) SetForceDashboard(show bool) {
	m.forceDashboard = show
}

func (m *ResourceTableModel) SetActivityRows(rows []data.ActivityRow) {
	m.activityFetchFailed = false // FB-082: successful data arrival clears error state
	m.activityRows = rows
	m.activityLoading = false
}

func (m *ResourceTableModel) SetActivityLoading(b bool) {
	m.activityLoading = b
}

func (m *ResourceTableModel) SetActivityFetchFailed(failed bool) {
	m.activityFetchFailed = failed
}

// ActivityRowCount returns the number of activity rows currently held in the model.
func (m ResourceTableModel) ActivityRowCount() int {
	return len(m.activityRows)
}

func (m *ResourceTableModel) SetPendingQuotaOpen(v bool) { m.pendingQuotaOpen = v }

func (m *ResourceTableModel) SetAttentionItems(items []AttentionItem) {
	m.attentionItems = items
}

func (m *ResourceTableModel) SetTypeContext(name string, selected bool) {
	if selected {
		m.typeName = name
	} else {
		m.typeName = ""
	}
}

func (m *ResourceTableModel) SetColumns(cols []string, tableWidth int) {
	m.columns = cols
	m.tableWidth = tableWidth
	// Each column cell has Padding(0,1) applied outside the col.Width measurement,
	// adding 2 chars per column. Subtract that before distributing content widths.
	innerWidth := tableWidth - 2*len(cols)
	if innerWidth < 0 {
		innerWidth = 0
	}
	widths := dynamicColumnWidths(cols, innerWidth)
	tableCols := make([]table.Column, len(cols))
	for i, name := range cols {
		tableCols[i] = table.Column{Title: name, Width: widths[i]}
	}
	// Clear rows before changing column schema: bubbles/table.SetColumns calls
	// UpdateViewport which renders existing rows against the new column widths.
	// Rows from a prior resource type have a different cell count and will panic.
	m.table.SetRows(nil)
	m.table.SetColumns(tableCols)
}

// dynamicColumnWidths distributes the total table width across columns.
// "Name" always gets the bulk of the space; other columns get fixed widths
// capped so Name is never starved even with many printer columns.
func dynamicColumnWidths(cols []string, totalWidth int) []int {
	if len(cols) == 0 {
		return nil
	}

	widths := make([]int, len(cols))
	nameIdx := -1

	// First pass: assign fixed widths to well-known columns (excluding Name).
	fixedTotal := 0
	for i, col := range cols {
		switch strings.ToLower(col) {
		case "name":
			nameIdx = i
		case "age":
			widths[i] = 8
			fixedTotal += 8
		case "namespace":
			widths[i] = 20
			fixedTotal += 20
		}
	}

	// Collect indices of columns that still need a width (not Name/Age/Namespace).
	var otherIdxs []int
	for i, col := range cols {
		lower := strings.ToLower(col)
		if i == nameIdx || lower == "age" || lower == "namespace" {
			continue
		}
		otherIdxs = append(otherIdxs, i)
	}

	// Each "other" column gets at most 16 chars so Name is never starved.
	// If the equal share is less than 8, clamp to 8 (truncation is acceptable).
	otherW := 0
	if len(otherIdxs) > 0 {
		available := totalWidth - fixedTotal
		if nameIdx >= 0 {
			// Reserve at least 35% of total for Name before splitting the rest.
			available = totalWidth - fixedTotal - (totalWidth*35/100)
		}
		if available < 0 {
			available = 0
		}
		otherW = available / len(otherIdxs)
		if otherW < 8 {
			otherW = 8
		}
		for _, i := range otherIdxs {
			widths[i] = otherW
		}
	}

	// Name gets all remaining space.
	if nameIdx >= 0 {
		used := 0
		for i, w := range widths {
			if i != nameIdx {
				used += w
			}
		}
		nameW := totalWidth - used
		if nameW < 10 {
			nameW = 10
		}
		widths[nameIdx] = nameW
	} else if len(cols) > 0 && totalWidth > 0 {
		// No Name column: give any leftover to the first column.
		used := 0
		for _, w := range widths {
			used += w
		}
		if totalWidth > used {
			widths[0] += totalWidth - used
		}
	}

	return widths
}

func (m *ResourceTableModel) applyFilter() {
	lower := strings.ToLower(m.filter)
	m.filteredRows = nil
	var tableRows []table.Row
	for _, row := range m.allRows {
		if lower != "" && !strings.Contains(strings.ToLower(row.Name), lower) {
			continue
		}
		m.filteredRows = append(m.filteredRows, row)
		cells := make(table.Row, len(row.Cells))
		copy(cells, row.Cells)
		for i, col := range m.columns {
			if i >= len(cells) {
				break
			}
			switch strings.ToLower(col) {
			case "status", "phase", "state", "ready":
				dot := lipgloss.NewStyle().Foreground(styles.StatusColorFor(cells[i])).Render("● ")
				cells[i] = dot + cells[i]
			}
		}
		tableRows = append(tableRows, cells)
	}
	m.table.SetRows(tableRows)
	// SetColumns clears the charmbracelet table via SetRows(nil), driving cursor to -1.
	// After repopulating rows, ensure cursor is non-negative so SelectedRow() works.
	if len(tableRows) > 0 && m.table.Cursor() < 0 {
		m.table.SetCursor(0)
	}
}


func (m ResourceTableModel) SelectedRow() (data.ResourceRow, bool) {
	if len(m.filteredRows) == 0 {
		return data.ResourceRow{}, false
	}
	idx := m.table.Cursor()
	if idx < 0 || idx >= len(m.filteredRows) {
		return data.ResourceRow{}, false
	}
	return m.filteredRows[idx], true
}

// Cursor returns the current table cursor index (0-based, relative to filtered rows).
func (m ResourceTableModel) Cursor() int { return m.table.Cursor() }

// SetCursor moves the table cursor to idx, clamped to valid range.
func (m *ResourceTableModel) SetCursor(idx int) { m.table.SetCursor(idx) }

