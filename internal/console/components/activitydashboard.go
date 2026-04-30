package components

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"go.datum.net/datumctl/internal/console/data"
	"go.datum.net/datumctl/internal/console/styles"
)

// ActivityDashboardModel renders the project-wide human-activity rollup pane (FB-016).
type ActivityDashboardModel struct {
	width, height int
	focused       bool
	loading       bool
	rows          []data.ActivityRow
	loadErr       error
	unauthorized  bool
	crdAbsent     bool      // one-shot per-session; cleared on ContextSwitchedMsg
	staleRefresh  bool      // last refresh failed but we have cached rows
	staleAt       time.Time // timestamp of last successful fetch (for humanized-age)
	cursor        int
	ctxLabel      string
	registrations []data.ResourceRegistration
	spinner       spinner.Model
	orgScope      bool   // true when tuiCtx.Project == "" — render hint, skip fetch
	originLabel   string // FB-088: human-readable return destination for [4] hint
}

func (m *ActivityDashboardModel) SetOriginLabel(label string) { m.originLabel = label }

// NewActivityDashboardModel constructs the model for the activity dashboard pane.
func NewActivityDashboardModel(width, height int, ctxLabel string) ActivityDashboardModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	m := ActivityDashboardModel{
		spinner:  s,
		ctxLabel: ctxLabel,
	}
	m.SetSize(width, height)
	return m
}

func (m ActivityDashboardModel) Init() tea.Cmd {
	return func() tea.Msg { return m.spinner.Tick() }
}

func (m ActivityDashboardModel) Update(msg tea.Msg) (ActivityDashboardModel, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if m.cursor < len(m.rows)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		}
	}
	return m, nil
}

func (m *ActivityDashboardModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *ActivityDashboardModel) SetFocused(b bool) { m.focused = b }
func (m *ActivityDashboardModel) SetLoading(b bool) { m.loading = b }

func (m *ActivityDashboardModel) SetRows(rows []data.ActivityRow) {
	m.rows = rows
	m.loadErr = nil
	m.unauthorized = false
	m.staleRefresh = false
	m.staleAt = time.Now()
	if m.cursor >= len(rows) {
		m.cursor = max(0, len(rows)-1)
	}
}

func (m *ActivityDashboardModel) SetLoadErr(err error, unauthorized, crdAbsent bool) {
	if len(m.rows) > 0 {
		// Keep stale rows; raise stale strip.
		m.staleRefresh = true
		m.crdAbsent = m.crdAbsent || crdAbsent
	} else {
		m.loadErr = err
		m.unauthorized = unauthorized
		m.crdAbsent = m.crdAbsent || crdAbsent
	}
}

func (m *ActivityDashboardModel) SetRegistrations(regs []data.ResourceRegistration) {
	m.registrations = regs
}

func (m *ActivityDashboardModel) SetOrgScope(b bool) { m.orgScope = b }

// ClearCRDAbsentFlag resets the one-shot session flag on ContextSwitchedMsg.
func (m *ActivityDashboardModel) ClearCRDAbsentFlag() {
	m.crdAbsent = false
	m.staleRefresh = false
	m.staleAt = time.Time{}
}

func (m ActivityDashboardModel) SpinnerFrame() string { return m.spinner.View() }

// HasRows reports whether any activity rows are currently cached.
func (m ActivityDashboardModel) HasRows() bool { return len(m.rows) > 0 }

// CRDAbsent reports the one-shot CRD-absent session flag.
func (m ActivityDashboardModel) CRDAbsent() bool { return m.crdAbsent }

func (m ActivityDashboardModel) View() string {
	inner := m.renderInner()
	return styles.PaneBorder(m.focused).Render(styles.SurfaceFill(inner, m.width, m.height))
}

func (m ActivityDashboardModel) renderInner() string {
	contentW := m.width
	contentH := m.height

	muted := lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Muted)
	bold := lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Secondary).Bold(true)

	rule := func() string {
		return muted.Render(strings.Repeat("─", max(0, contentW)))
	}

	// Unusable band.
	if contentW < 40 {
		return muted.Render("Terminal too narrow for activity dashboard")
	}

	// Org-scope gate — no project selected.
	if m.orgScope {
		header := bold.Render("Recent activity")
		hint := muted.Render("Select a project to see recent activity")
		keybinds := muted.Render("[esc] back · [?] help")
		return strings.Join([]string{header, "", rule(), "", center(hint, contentW), "", rule(), keybinds}, "\n")
	}

	// Height-based footer drop.
	showFooter := contentH >= 12
	showSeparators := contentH >= 8

	var lines []string

	// Stale-refresh strip (prepended above header when rows are cached but last refresh failed).
	if m.staleRefresh && len(m.rows) > 0 {
		warnStyle := lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Warning)
		age := humanizedAge(time.Since(m.staleAt))
		staleLine := warnStyle.Render("⚠ refresh failed") + muted.Render(" — showing rows from "+age)
		lines = append(lines, staleLine)
		lines = append(lines, "")
	}

	// Header band.
	header := m.titleBar(bold, muted, contentW)
	lines = append(lines, header)

	// Separator + column headers.
	if showSeparators {
		lines = append(lines, rule())
	}

	switch {
	case m.loading:
		lines = append(lines, "")
		lines = append(lines, center(m.spinner.View()+" loading recent activity…", contentW))
		lines = append(lines, "")

	case m.crdAbsent:
		lines = append(lines, "")
		lines = append(lines, center(muted.Render("Recent activity not available on this cluster"), contentW))
		lines = append(lines, "")

	case m.unauthorized:
		lines = append(lines, "")
		lines = append(lines, center(muted.Render("Recent activity unavailable — insufficient permissions"), contentW))
		lines = append(lines, "")

	case m.loadErr != nil && len(m.rows) == 0:
		sev := data.SeverityOfClassified(m.loadErr)
		lines = append(lines, RenderErrorBlock(ErrorBlock{
			Title:    "Recent activity temporarily unavailable",
			Detail:   SanitizeErrMsg(m.loadErr),
			Actions:  actionsForSeverity(sev, "back"),
			Severity: sev,
			Width:    contentW,
		}))

	case len(m.rows) == 0:
		lines = append(lines, "")
		lines = append(lines, center(muted.Render("No recent human activity in the last 24 hours."), contentW))
		lines = append(lines, "")

	default:
		lines = append(lines, m.renderColumnHeader(bold, muted, contentW))
		lines = append(lines, "")
		maxRows := 10
		if contentH < 18 {
			maxRows = max(1, contentH-6)
		}
		shown := min(maxRows, len(m.rows))
		for i := 0; i < shown; i++ {
			lines = append(lines, m.renderRow(m.rows[i], i == m.cursor, contentW, muted))
		}
		lines = append(lines, "")
	}

	if showSeparators {
		lines = append(lines, rule())
	}
	if showFooter {
		lines = append(lines, m.keybindStrip(muted, bold))
	}

	return strings.Join(lines, "\n")
}

func (m ActivityDashboardModel) titleBar(bold, muted lipgloss.Style, w int) string {
	left := m.renderHeader(bold, muted)
	if m.originLabel == "" {
		return left
	}
	accentBold := lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Accent).Bold(true)
	backHint := accentBold.Render("[4]") + muted.Render(" back to "+m.originLabel+"  ")
	hint := backHint + muted.Render("[↑/↓] move  [r] refresh")
	gap := max(1, w-lipgloss.Width(left)-lipgloss.Width(hint))
	return left + strings.Repeat(" ", gap) + hint
}

func (m ActivityDashboardModel) renderHeader(bold, muted lipgloss.Style) string {
	title := bold.Render("Recent activity")
	if m.orgScope {
		return title
	}
	suffix := " — last 24 hours"
	if len(m.rows) > 0 && !m.loading {
		suffix += muted.Render(fmt.Sprintf(" · %d events", len(m.rows)))
	}
	return title + muted.Render(suffix)
}

func (m ActivityDashboardModel) renderColumnHeader(bold, muted lipgloss.Style, contentW int) string {
	switch widthBand(contentW) {
	case bandWide:
		ts, res, actor, summary := wideColumnWidths(contentW)
		return bold.Render(
			padRight("TIMESTAMP", ts) + " " +
				padRight("RESOURCE", res) + " " +
				padRight("ACTOR", actor) + " " +
				truncate("SUMMARY", summary),
		)
	case bandStandard:
		ts, res, summary := standardColumnWidths(contentW)
		return bold.Render(
			padRight("TIMESTAMP", ts) + " " +
				padRight("RESOURCE", res) + " " +
				truncate("SUMMARY", summary),
		)
	case bandNarrow:
		return bold.Render("TIMESTAMP RESOURCE") + "\n  " + bold.Render("SUMMARY")
	default:
		return ""
	}
}

func (m ActivityDashboardModel) renderRow(row data.ActivityRow, selected bool, contentW int, muted lipgloss.Style) string {
	ref := row.ResourceRef
	resourceLabel := ""
	if ref != nil {
		kind := ref.Kind
		if dn := data.ResolveDescription(m.registrations, ref.APIGroup, ref.Kind); dn != "" {
			kind = dn
		}
		resourceLabel = kind + "/" + ref.Name
	}
	actor := row.ActorDisplay
	if actor == "" {
		actor = "—"
	}

	prefix := "  "
	if selected {
		prefix = lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Accent).Render("▶ ")
	}

	switch widthBand(contentW) {
	case bandWide:
		ts, res, actorW, summaryW := wideColumnWidths(contentW)
		tsStr := formatActivityDashboardTimestamp(row.Timestamp, false)
		return prefix + padRight(tsStr, ts-2) + " " +
			padRight(truncate(resourceLabel, res), res) + " " +
			padRight(truncate(actor, actorW), actorW) + " " +
			muted.Render(truncate(row.Summary, summaryW))

	case bandStandard:
		ts, res, summaryW := standardColumnWidths(contentW)
		tsStr := formatActivityDashboardTimestamp(row.Timestamp, true)
		return prefix + padRight(tsStr, ts-2) + " " +
			padRight(truncate(resourceLabel, res), res) + " " +
			muted.Render(truncate(row.Summary, summaryW))

	case bandNarrow:
		tsStr := formatActivityDashboardTimestamp(row.Timestamp, true)
		line1 := prefix + tsStr + " " + resourceLabel
		line2 := "  " + muted.Render(truncate(row.Summary, contentW-2))
		return line1 + "\n" + line2

	default:
		return ""
	}
}

func (m ActivityDashboardModel) keybindStrip(muted, bold lipgloss.Style) string {
	parts := []string{}
	if !m.loading && !m.crdAbsent && !m.unauthorized {
		parts = append(parts, bold.Render("[r]")+" "+muted.Render("refresh"))
	}
	parts = append(parts, bold.Render("[esc]")+" "+muted.Render("back"))
	parts = append(parts, bold.Render("[?]")+" "+muted.Render("help"))
	return strings.Join(parts, " · ")
}

// widthBand returns the render band for the given content width.
type band int

const (
	bandUnusable band = iota
	bandNarrow
	bandStandard
	bandWide
)

func widthBand(w int) band {
	switch {
	case w < 40:
		return bandUnusable
	case w < 60:
		return bandNarrow
	case w < 80:
		return bandStandard
	default:
		return bandWide
	}
}

func wideColumnWidths(contentW int) (ts, res, actor, summary int) {
	ts = 16
	res = max(20, contentW/4)
	actor = max(18, contentW/4)
	summary = max(1, contentW-ts-res-actor-3)
	return
}

func standardColumnWidths(contentW int) (ts, res, summary int) {
	ts = 11
	res = max(18, contentW*30/100)
	summary = max(1, contentW-ts-res-2)
	return
}

// formatActivityDashboardTimestamp formats a row timestamp for the given band.
// compact=true uses HH:MM for <48h rows, MM-DD for ≥48h rows (narrow/standard).
// compact=false uses YYYY-MM-DD HH:MM for all rows (wide band).
func formatActivityDashboardTimestamp(t time.Time, compact bool) string {
	age := time.Since(t)
	if !compact {
		return t.Local().Format("2006-01-02 15:04")
	}
	if age < 48*time.Hour {
		return t.Local().Format("15:04")
	}
	return t.Local().Format("01-02")
}

func humanizedAge(d time.Duration) string {
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%d minutes ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

func center(s string, width int) string {
	visLen := lipgloss.Width(s)
	if visLen >= width {
		return s
	}
	pad := (width - visLen) / 2
	return strings.Repeat(" ", pad) + s
}

