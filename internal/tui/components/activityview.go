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

// NeedNextActivityPageMsg is returned by ActivityViewModel.Update when the
// viewport has scrolled into the pagination sentinel and the next page should
// be fetched.
type NeedNextActivityPageMsg struct{}

// ActivityViewModel renders the activity timeline for a single resource.
type ActivityViewModel struct {
	width, height int
	focused       bool
	vp            viewport.Model
	spinner       spinner.Model
	loading       bool
	loadingMore   bool
	err           error
	unauthorized  bool
	rows          []data.ActivityRow
	nextContinue  string
	resourceKind  string
	resourceName  string
}

// NewActivityViewModel constructs an ActivityViewModel.
func NewActivityViewModel(width, height int) ActivityViewModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	m := ActivityViewModel{spinner: s}
	m.SetSize(width, height)
	return m
}

func (m ActivityViewModel) Init() tea.Cmd { return m.spinner.Tick }

func (m ActivityViewModel) Update(msg tea.Msg) (ActivityViewModel, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			prevOffset := m.vp.YOffset
			m.vp, cmd = m.vp.Update(msg)
			if m.nextContinue != "" && !m.loadingMore && m.vp.YOffset > prevOffset {
				if m.sentinelVisible() {
					return m, tea.Batch(cmd, func() tea.Msg { return NeedNextActivityPageMsg{} })
				}
			}
			return m, cmd
		case "k", "up", "pgup", "pgdown", "g", "G":
			m.vp, cmd = m.vp.Update(msg)
			return m, cmd
		}
	default:
		m.vp, cmd = m.vp.Update(msg)
	}
	return m, cmd
}

func (m ActivityViewModel) View() string {
	var content string
	if m.height < 6 {
		content = m.vp.View()
	} else {
		content = lipgloss.JoinVertical(lipgloss.Left,
			m.titleBar(),
			m.titleRule(),
			m.columnHeader(),
			m.vp.View(),
			m.footerRule(),
			m.scrollFooter(),
		)
	}
	return styles.PaneBorder(m.focused).Render(content)
}

func (m *ActivityViewModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	vpH := h
	if h >= 6 {
		// chrome: titleBar + titleRule + columnHeader + footerRule + scrollFooter = 5 lines
		vpH = max(h-5, 1)
	}
	m.vp.Width = w
	m.vp.Height = vpH
}

func (m *ActivityViewModel) SetFocused(focused bool) { m.focused = focused }

func (m *ActivityViewModel) SetResourceContext(kind, name string) {
	m.resourceKind = kind
	m.resourceName = name
}

func (m *ActivityViewModel) SetLoading(loading bool) {
	m.loading = loading
	m.refreshContent()
}

func (m *ActivityViewModel) SetLoadingMore(loadingMore bool) {
	m.loadingMore = loadingMore
	m.refreshContent()
}

func (m *ActivityViewModel) SetRows(rows []data.ActivityRow, nextContinue string) {
	m.rows = rows
	m.nextContinue = nextContinue
	m.err = nil
	m.unauthorized = false
	m.refreshContent()
}

func (m *ActivityViewModel) AppendRows(rows []data.ActivityRow, nextContinue string) {
	m.rows = append(m.rows, rows...)
	m.nextContinue = nextContinue
	m.loadingMore = false
	m.refreshContent()
}

func (m *ActivityViewModel) SetError(err error, unauthorized bool) {
	m.err = err
	m.unauthorized = unauthorized
	m.loading = false
	m.loadingMore = false
	m.refreshContent()
}

// Reset clears all row data and resets to the initial loading state for a new
// resource.
func (m *ActivityViewModel) Reset() {
	m.rows = nil
	m.nextContinue = ""
	m.err = nil
	m.unauthorized = false
	m.loading = false
	m.loadingMore = false
	m.vp.GotoTop()
	m.refreshContent()
}

func (m ActivityViewModel) HasRows() bool { return len(m.rows) > 0 }

func (m ActivityViewModel) NextContinue() string { return m.nextContinue }

// sentinelVisible returns true when the viewport's bottom row overlaps with
// the sentinel line (last line of content).
func (m ActivityViewModel) sentinelVisible() bool {
	totalLines := strings.Count(m.vp.View(), "\n") + 1
	return m.vp.YOffset+m.vp.Height >= totalLines
}

func (m *ActivityViewModel) refreshContent() {
	m.vp.SetContent(m.buildContent())
}

func (m ActivityViewModel) buildContent() string {
	switch {
	case m.loading:
		return "\n  " + m.spinner.View() + "  loading activity…\n"
	case m.unauthorized:
		muted := lipgloss.NewStyle().Foreground(styles.Muted)
		return "\n" +
			muted.Render("  Activity is not enabled for this project.") + "\n" +
			muted.Render("  Enable the activity service in your project settings") + "\n" +
			muted.Render("  to collect audit events and resource changes.") + "\n\n" +
			muted.Render("  [Esc] back to describe") + "\n"
	case m.err != nil:
		title, detail := titleAndDetailForError(m.err, "Could not load activity")
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
			muted.Render("  No activity recorded for this resource.") + "\n" +
			muted.Render("  (Activity is collected for the last 30 days.)") + "\n"
	}

	return m.renderRows()
}

func (m ActivityViewModel) renderRows() string {
	timeW, originW, actorW, srcW, summaryW := m.columnWidths()

	var sb strings.Builder
	var prevDate string

	for _, row := range m.rows {
		// Day-boundary separator.
		dateStr := row.Timestamp.Local().Format("2006-01-02")
		if prevDate != "" && dateStr != prevDate {
			muted := lipgloss.NewStyle().Foreground(styles.Muted)
			sb.WriteString(muted.Render("  ── "+dateStr+" ──") + "\n")
		}
		prevDate = dateStr

		sb.WriteString(m.renderRow(row, timeW, originW, actorW, srcW, summaryW))
		sb.WriteByte('\n')
	}

	// Pagination sentinel / end marker.
	muted := lipgloss.NewStyle().Foreground(styles.Muted)
	sb.WriteByte('\n')
	switch {
	case m.loadingMore:
		sb.WriteString(muted.Render("  " + m.spinner.View() + "  loading more…"))
	case m.nextContinue != "":
		centered := strings.Repeat(" ", max(0, (m.width-22)/2))
		sb.WriteString(muted.Render(centered + "(more — scroll ↓)"))
	default:
		centered := strings.Repeat(" ", max(0, (m.width-20)/2))
		sb.WriteString(muted.Render(centered + "— end of activity —"))
	}
	sb.WriteByte('\n')

	return sb.String()
}

// columnWidths returns (timeW, originW, actorW, srcW, summaryW) based on m.width.
// At <50 cols ACTOR is dropped; at <30 SRC too; at <20 only TIME+SUMMARY.
func (m ActivityViewModel) columnWidths() (timeW, originW, actorW, srcW, summaryW int) {
	w := m.width
	timeW = 12 // "HH:MM:SS    " (padded to 12)
	originW = 6 // "audit " / "event "
	srcW = 7    // "human  " / "system "
	const separators = 4

	switch {
	case w < 20:
		// Only TIME + SUMMARY
		summaryW = max(1, w-timeW-1)
		return timeW, 0, 0, 0, summaryW
	case w < 30:
		// TIME + ORIGIN + SUMMARY
		summaryW = max(1, w-timeW-originW-2)
		return timeW, originW, 0, 0, summaryW
	case w < 50:
		// Drop ACTOR
		summaryW = max(1, w-timeW-originW-srcW-separators+1)
		return timeW, originW, 0, srcW, summaryW
	default:
		// Full layout: TIME + ORIGIN + ACTOR + SRC + SUMMARY
		// Actor grows with width; summary gets remainder.
		actorW = min(24, max(17, w-timeW-originW-srcW-separators-16))
		summaryW = max(1, w-timeW-originW-actorW-srcW-separators)
		return timeW, originW, actorW, srcW, summaryW
	}
}

func (m ActivityViewModel) renderRow(
	row data.ActivityRow,
	timeW, originW, actorW, srcW, summaryW int,
) string {
	isEvent := row.Origin == "event"

	// Styles
	secondary := lipgloss.NewStyle().Foreground(styles.Secondary)
	muted := lipgloss.NewStyle().Foreground(styles.Muted)
	accent := lipgloss.NewStyle().Foreground(styles.Accent)
	def := lipgloss.NewStyle()

	timeCell := padRight(formatActivityTime(row.Timestamp), timeW)

	var originCell string
	if isEvent {
		originCell = muted.Render(padRight(row.Origin, originW))
	} else {
		originCell = secondary.Render(padRight(row.Origin, originW))
	}

	var actorCell string
	if actorW > 0 {
		actor := truncate(row.ActorDisplay, actorW)
		actorCell = def.Render(padRight(actor, actorW))
	}

	var srcCell string
	if srcW > 0 {
		src := truncate(row.ChangeSource, srcW)
		switch row.ChangeSource {
		case "human":
			srcCell = accent.Render(padRight(src, srcW))
		case "system":
			srcCell = muted.Render(padRight(src, srcW))
		default:
			srcCell = secondary.Render(padRight(src, srcW))
		}
	}

	summary := truncate(row.Summary, summaryW)
	var summaryCell string
	if isEvent {
		summaryCell = muted.Render(summary)
	} else {
		summaryCell = def.Render(summary)
	}

	parts := []string{timeCell}
	if originW > 0 {
		parts = append(parts, originCell)
	}
	if actorW > 0 {
		parts = append(parts, actorCell)
	}
	if srcW > 0 {
		parts = append(parts, srcCell)
	}
	parts = append(parts, summaryCell)

	return strings.Join(parts, " ")
}

func (m ActivityViewModel) columnHeader() string {
	timeW, originW, actorW, srcW, summaryW := m.columnWidths()
	style := lipgloss.NewStyle().Foreground(styles.Secondary).Bold(true)

	parts := []string{padRight("TIME", timeW)}
	if originW > 0 {
		parts = append(parts, padRight("ORIGIN", originW))
	}
	if actorW > 0 {
		parts = append(parts, padRight("ACTOR", actorW))
	}
	if srcW > 0 {
		parts = append(parts, padRight("SRC", srcW))
	}
	parts = append(parts, padRight("SUMMARY", summaryW))

	return style.Render(strings.Join(parts, " "))
}

func (m ActivityViewModel) titleBar() string {
	w := m.width
	accentBold := lipgloss.NewStyle().Foreground(styles.Accent).Bold(true)
	muted := lipgloss.NewStyle().Foreground(styles.Muted)

	leftText := accentBold.Render(m.resourceKind) +
		muted.Render(" / ") +
		accentBold.Render(m.resourceName)

	rightText := muted.Render("[a] describe  [Esc] back")

	gap := max(1, w-lipgloss.Width(leftText)-lipgloss.Width(rightText))
	return leftText + strings.Repeat(" ", gap) + rightText
}

func (m ActivityViewModel) titleRule() string {
	return lipgloss.NewStyle().Foreground(styles.InactiveBorderColor).
		Render(strings.Repeat("─", m.width))
}

func (m ActivityViewModel) footerRule() string {
	return lipgloss.NewStyle().Foreground(styles.InactiveBorderColor).
		Render(strings.Repeat("─", m.width))
}

func (m ActivityViewModel) scrollFooter() string {
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

// formatActivityTime formats a timestamp according to the spec (§4).
func formatActivityTime(t time.Time) string {
	local := t.Local()
	now := time.Now().Local()
	if local.Year() == now.Year() && local.Month() == now.Month() && local.Day() == now.Day() {
		return local.Format("15:04:05")
	}
	if local.Year() == now.Year() {
		return local.Format("Jan 02 15:04")
	}
	return local.Format("2006-01-02")
}

func padRight(s string, n int) string {
	r := []rune(s)
	if len(r) >= n {
		return string(r[:n])
	}
	return s + strings.Repeat(" ", n-len(r))
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= 1 {
		return "…"
	}
	return string(r[:n-1]) + "…"
}
