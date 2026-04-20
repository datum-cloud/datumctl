package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"go.datum.net/datumctl/internal/tui/data"
	"go.datum.net/datumctl/internal/tui/styles"
)

// HistoryViewModel renders the revision list for a single resource.
type HistoryViewModel struct {
	width, height int
	focused       bool
	vp            viewport.Model
	spinner       spinner.Model

	loading      bool
	err          error
	unauthorized bool
	truncated    bool

	rows         []data.HistoryRow // all rows, index 0 = REV 1 (oldest)
	filterHuman  bool
	visibleIdx   []int // indices into rows after filter
	cursor       int   // position within visibleIdx
	preFilterCur int   // saved cursor position before filter toggled on

	resourceKind string
	resourceName string
}

func NewHistoryViewModel(width, height int) HistoryViewModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	m := HistoryViewModel{spinner: s}
	m.SetSize(width, height)
	return m
}

func (m HistoryViewModel) Init() tea.Cmd { return m.spinner.Tick }

func (m HistoryViewModel) Update(msg tea.Msg) (HistoryViewModel, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	default:
		m.vp, cmd = m.vp.Update(msg)
	}
	return m, cmd
}

func (m HistoryViewModel) View() string {
	var content string
	if m.height < 6 {
		content = m.vp.View()
	} else {
		parts := []string{
			m.titleBar(),
			m.titleRule(),
			m.columnHeader(),
		}
		if m.filterHuman {
			parts = append(parts, m.filterBanner())
		}
		if m.truncated {
			parts = append(parts, m.truncationBanner())
		}
		parts = append(parts,
			m.vp.View(),
			m.footerRule(),
			m.scrollFooter(),
		)
		content = lipgloss.JoinVertical(lipgloss.Left, parts...)
	}
	return styles.PaneBorder(m.focused).Render(content)
}

func (m *HistoryViewModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.rebuildViewport()
}

func (m *HistoryViewModel) SetFocused(focused bool) { m.focused = focused }

func (m *HistoryViewModel) SetResourceContext(kind, name string) {
	m.resourceKind = kind
	m.resourceName = name
}

func (m *HistoryViewModel) SetLoading(loading bool) {
	m.loading = loading
	m.refreshContent()
}

func (m *HistoryViewModel) SetRows(rows []data.HistoryRow, truncated bool) {
	m.rows = rows
	m.truncated = truncated
	m.err = nil
	m.unauthorized = false
	m.rebuildVisibleIdx()
	m.cursor = 0 // top = newest (visibleIdx is reversed: newest at index 0)
	m.refreshContent()
}

func (m *HistoryViewModel) SetError(err error, unauthorized bool) {
	m.err = err
	m.unauthorized = unauthorized
	m.loading = false
	m.refreshContent()
}

func (m *HistoryViewModel) SetTruncated(t bool) {
	m.truncated = t
	m.refreshContent()
}

// ToggleHumanFilter toggles the human-only filter on or off.
func (m *HistoryViewModel) ToggleHumanFilter() {
	if m.filterHuman {
		// Turning off: restore pre-filter cursor.
		m.filterHuman = false
		m.rebuildVisibleIdx()
		// Snap cursor to preFilterCur, clamped.
		m.cursor = m.preFilterCur
		if m.cursor >= len(m.visibleIdx) {
			m.cursor = max(0, len(m.visibleIdx)-1)
		}
	} else {
		// Turning on: save cursor, rebuild filtered list.
		m.preFilterCur = m.cursor
		m.filterHuman = true
		m.rebuildVisibleIdx()
		m.cursor = 0 // snap to newest visible human row
	}
	m.refreshContent()
}

// ResetFilter clears the human-only filter without restoring cursor (for pane exit).
func (m *HistoryViewModel) ResetFilter() {
	m.filterHuman = false
	m.rebuildVisibleIdx()
}

// CursorUp moves the cursor toward newer revisions (up in the displayed list).
func (m *HistoryViewModel) CursorUp() {
	if m.cursor > 0 {
		m.cursor--
		m.refreshContent()
	}
}

// CursorDown moves the cursor toward older revisions (down in the displayed list).
func (m *HistoryViewModel) CursorDown() {
	if m.cursor < len(m.visibleIdx)-1 {
		m.cursor++
		m.refreshContent()
	}
}

// CursorTop moves cursor to newest (top of displayed list).
func (m *HistoryViewModel) CursorTop() {
	m.cursor = 0
	m.refreshContent()
}

// CursorBottom moves cursor to oldest (bottom of displayed list).
func (m *HistoryViewModel) CursorBottom() {
	if len(m.visibleIdx) > 0 {
		m.cursor = len(m.visibleIdx) - 1
	}
	m.refreshContent()
}

// SelectedRow returns the currently selected HistoryRow and its 0-based manifest index.
func (m HistoryViewModel) SelectedRow() (data.HistoryRow, int, bool) {
	if len(m.visibleIdx) == 0 {
		return data.HistoryRow{}, 0, false
	}
	if m.cursor < 0 || m.cursor >= len(m.visibleIdx) {
		return data.HistoryRow{}, 0, false
	}
	// visibleIdx stores oldest-first indices into m.rows; rows and manifests are parallel.
	idx := m.visibleIdx[m.cursor]
	return m.rows[idx], idx, true
}

// HasRows returns true when at least one row is loaded (regardless of filter).
func (m HistoryViewModel) HasRows() bool { return len(m.rows) > 0 }

// Reset clears all state for a new resource.
func (m *HistoryViewModel) Reset() {
	m.rows = nil
	m.visibleIdx = nil
	m.cursor = 0
	m.preFilterCur = 0
	m.filterHuman = false
	m.err = nil
	m.unauthorized = false
	m.loading = false
	m.truncated = false
	m.vp.GotoTop()
	m.refreshContent()
}

func (m *HistoryViewModel) rebuildVisibleIdx() {
	m.visibleIdx = nil
	for i, row := range m.rows {
		if m.filterHuman && row.Source != "human" {
			continue
		}
		m.visibleIdx = append(m.visibleIdx, i)
	}
	// visibleIdx is oldest-first (ascending REV). The display shows newest first,
	// so we reverse it so cursor=0 → newest visible row.
	for i, j := 0, len(m.visibleIdx)-1; i < j; i, j = i+1, j-1 {
		m.visibleIdx[i], m.visibleIdx[j] = m.visibleIdx[j], m.visibleIdx[i]
	}
}

func (m *HistoryViewModel) rebuildViewport() {
	chrome := 0
	if m.height >= 6 {
		// titleBar + titleRule + columnHeader + footerRule + scrollFooter = 5 fixed lines
		chrome = 5
		if m.filterHuman {
			chrome++ // filter banner
		}
		if m.truncated {
			chrome++ // truncation banner
		}
	}
	vpH := max(m.height-chrome, 1)
	m.vp.Width = m.width
	m.vp.Height = vpH
}

func (m *HistoryViewModel) refreshContent() {
	m.rebuildViewport()
	m.vp.SetContent(m.buildContent())
}

func (m HistoryViewModel) buildContent() string {
	switch {
	case m.loading:
		return "\n  " + m.spinner.View() + "  loading history…\n"
	case m.unauthorized:
		muted := lipgloss.NewStyle().Foreground(styles.Muted)
		return "\n" +
			muted.Render("  Audit history is not enabled for this project.") + "\n" +
			muted.Render("  Enable the activity service in project settings to collect") + "\n" +
			muted.Render("  audit events and change history.") + "\n\n" +
			muted.Render("  [Esc] back to describe") + "\n"
	case m.err != nil:
		title, detail := titleAndDetailForError(m.err, "Could not load history")
		sev := data.SeverityOfClassified(m.err)
		return RenderErrorBlock(ErrorBlock{
			Title:    title,
			Detail:   detail,
			Actions:  actionsForSeverity(sev, "back to describe"),
			Severity: sev,
			Width:    m.vp.Width,
		})
	case len(m.rows) == 0:
		muted := lipgloss.NewStyle().Foreground(styles.Muted)
		return "\n" +
			muted.Render("  No change history recorded for this resource.") + "\n" +
			muted.Render("  (Audit log covers the last 30 days; very recent changes may take ~60s to appear.)") + "\n"
	case m.filterHuman && len(m.visibleIdx) == 0:
		muted := lipgloss.NewStyle().Foreground(styles.Muted)
		return "\n" +
			muted.Render("  No human-source revisions in this window.") + "\n"
	}

	return m.renderRows()
}

func (m HistoryViewModel) renderRows() string {
	revW, timeW, userW, srcW, summaryW := m.columnWidths()

	var sb strings.Builder

	for i, idx := range m.visibleIdx {
		row := m.rows[idx]
		sb.WriteString(m.renderRow(row, i == m.cursor, revW, timeW, userW, srcW, summaryW))
		sb.WriteByte('\n')
	}

	muted := lipgloss.NewStyle().Foreground(styles.Muted)
	sb.WriteByte('\n')
	centered := strings.Repeat(" ", max(0, (m.width-22)/2))
	sb.WriteString(muted.Render(centered + "— end of history —"))
	sb.WriteByte('\n')
	return sb.String()
}

// columnWidths returns column widths based on terminal width.
// Ladder: ≥100 full; 80-99 compressed time+user; 60-79 narrow user; 40-59 drop user; <40 warn.
func (m HistoryViewModel) columnWidths() (revW, timeW, userW, srcW, summaryW int) {
	w := m.width
	const sep = 2 // spaces between columns

	revW = 3 // "▸ N" prefix always 2 chars + right-justified rev number up to 3 digits

	switch {
	case w < 40:
		return revW, 0, 0, 0, 0
	case w < 60:
		// REV, TIME(8), SRC, SUMMARY
		timeW = 8
		srcW = 6
		summaryW = max(1, w-revW-timeW-srcW-sep*3)
		return revW, timeW, 0, srcW, summaryW
	case w < 80:
		// REV, TIME(8), USER(10), SRC, SUMMARY
		timeW = 8
		userW = 10
		srcW = 6
		summaryW = max(1, w-revW-timeW-userW-srcW-sep*4)
		return revW, timeW, userW, srcW, summaryW
	case w < 100:
		// REV, TIME(8), USER(14), SRC(6), SUMMARY
		timeW = 8
		userW = 14
		srcW = 6
		summaryW = max(1, w-revW-timeW-userW-srcW-sep*4)
		return revW, timeW, userW, srcW, summaryW
	default:
		// Full: REV(3), TIME(19), USER(22), SOURCE(6), SUMMARY(remainder)
		timeW = 19
		userW = 22
		srcW = 6
		summaryW = max(1, w-revW-timeW-userW-srcW-sep*4)
		return revW, timeW, userW, srcW, summaryW
	}
}

func (m HistoryViewModel) renderRow(
	row data.HistoryRow,
	selected bool,
	revW, timeW, userW, srcW, summaryW int,
) string {
	muted := lipgloss.NewStyle().Foreground(styles.Muted)
	accent := lipgloss.NewStyle().Foreground(styles.Accent)
	def := lipgloss.NewStyle()

	if m.width < 40 {
		return muted.Render("— terminal too narrow —")
	}

	// Cursor glyph occupies 2 chars (▸ + space or 2 spaces).
	cursor := "  "
	if selected {
		cursor = lipgloss.NewStyle().Foreground(styles.Accent).Render("▸ ")
	}

	// REV: right-justified within revW.
	revStr := fmt.Sprintf("%*d", revW, row.Rev)

	timeStr := ""
	if timeW > 0 {
		timeStr = formatHistoryTime(row.Timestamp, timeW)
		timeStr = padRight(timeStr, timeW)
	}

	userStr := ""
	if userW > 0 {
		disp := row.UserDisp
		if disp == "" {
			disp = "(anonymous)"
		}
		userStr = truncate(disp, userW)
		userStr = padRight(userStr, userW)
	}

	srcStr := ""
	if srcW > 0 {
		src := row.Source
		if len(src) > srcW {
			src = src[:srcW]
		}
		switch row.Source {
		case "human":
			srcStr = accent.Render(padRight(src, srcW))
		default:
			srcStr = muted.Render(padRight(src, srcW))
		}
	}

	summaryStr := ""
	if summaryW > 0 {
		summary := truncate(row.Summary, summaryW)
		summaryStr = def.Render(summary)
	}

	parts := []string{cursor + revStr}
	if timeW > 0 {
		parts = append(parts, timeStr)
	}
	if userW > 0 {
		parts = append(parts, userStr)
	}
	if srcW > 0 {
		parts = append(parts, srcStr)
	}
	if summaryW > 0 {
		parts = append(parts, summaryStr)
	}

	return strings.Join(parts, "  ")
}

// formatHistoryTime formats a timestamp per the spec §4 ladder.
func formatHistoryTime(t time.Time, budget int) string {
	local := t.Local()
	now := time.Now().Local()

	if budget >= 19 {
		return local.Format("2006-01-02 15:04:05")
	}

	sameDay := local.Year() == now.Year() && local.Month() == now.Month() && local.Day() == now.Day()
	if sameDay {
		return local.Format("15:04:05")
	}
	sameYear := local.Year() == now.Year()
	if sameYear {
		if budget >= 12 {
			return local.Format("Jan 02 15:04")
		}
		return local.Format("Jan 02")
	}
	return local.Format("2006-01-02")
}

func (m HistoryViewModel) columnHeader() string {
	revW, timeW, userW, srcW, summaryW := m.columnWidths()
	style := lipgloss.NewStyle().Foreground(styles.Secondary).Bold(true)

	if m.width < 40 {
		return style.Render("— terminal too narrow —")
	}

	// Leading cursor glyph width = 2.
	parts := []string{"  " + padRight("REV", revW)}
	if timeW > 0 {
		label := "TIMESTAMP"
		if timeW < 19 {
			label = "TIME"
		}
		parts = append(parts, padRight(label, timeW))
	}
	if userW > 0 {
		parts = append(parts, padRight("USER", userW))
	}
	if srcW > 0 {
		parts = append(parts, padRight("SRC", srcW))
	}
	if summaryW > 0 {
		parts = append(parts, padRight("SUMMARY", summaryW))
	}
	return style.Render(strings.Join(parts, "  "))
}

func (m HistoryViewModel) filterBanner() string {
	muted := lipgloss.NewStyle().Foreground(styles.Muted)
	visible := len(m.visibleIdx)
	total := len(m.rows)
	label := fmt.Sprintf(" filter: human only (press c to clear) ")
	countStr := fmt.Sprintf(" %d of %d ", visible, total)
	ruleW := max(0, m.width-len(label)-len(countStr))
	return muted.Render("──" + label + strings.Repeat("─", ruleW) + countStr + "──")
}

func (m HistoryViewModel) truncationBanner() string {
	muted := lipgloss.NewStyle().Foreground(styles.Muted)
	msg := " showing latest 100 changes; use `datumctl activity history --all-pages` for full "
	ruleW := max(0, m.width-len(msg)-4)
	return muted.Render("── " + msg + strings.Repeat("─", ruleW))
}

func (m HistoryViewModel) titleBar() string {
	w := m.width
	accentBold := lipgloss.NewStyle().Foreground(styles.Accent).Bold(true)
	muted := lipgloss.NewStyle().Foreground(styles.Muted)

	leftText := accentBold.Render(m.resourceKind) +
		muted.Render(" / ") +
		accentBold.Render(m.resourceName)

	rightText := muted.Render("[H] describe  [Esc] back  [Enter] diff")

	gap := max(1, w-lipgloss.Width(leftText)-lipgloss.Width(rightText))
	return leftText + strings.Repeat(" ", gap) + rightText
}

func (m HistoryViewModel) titleRule() string {
	return lipgloss.NewStyle().Foreground(styles.InactiveBorderColor).
		Render(strings.Repeat("─", m.width))
}

func (m HistoryViewModel) footerRule() string {
	return lipgloss.NewStyle().Foreground(styles.InactiveBorderColor).
		Render(strings.Repeat("─", m.width))
}

func (m HistoryViewModel) scrollFooter() string {
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
