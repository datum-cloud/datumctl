package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"go.datum.net/datumctl/internal/tui/data"
	"go.datum.net/datumctl/internal/tui/styles"
)

type DetailViewModel struct {
	vp           viewport.Model
	resourceKind string
	resourceName string
	loading      bool
	spinner      spinner.Model
	width        int
	height       int
	focused      bool
	mode         string // "describe", "yaml", or "conditions"; empty treated as "describe" (FB-018 adds "conditions")
}

func NewDetailViewModel(width, height int) DetailViewModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	m := DetailViewModel{
		vp:      viewport.New(width, height),
		spinner: s,
	}
	m.SetSize(width, height)
	return m
}

func (m DetailViewModel) Init() tea.Cmd { return m.spinner.Tick }

func (m DetailViewModel) Update(msg tea.Msg) (DetailViewModel, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down", "k", "up", "pgup", "pgdown", "g", "G":
			m.vp, cmd = m.vp.Update(msg)
		}
	default:
		m.vp, cmd = m.vp.Update(msg)
	}
	return m, cmd
}

func (m DetailViewModel) View() string {
	var content string
	if m.height < 6 {
		content = m.vp.View()
	} else {
		content = lipgloss.JoinVertical(lipgloss.Left,
			m.titleBar(),
			m.titleRule(),
			m.vp.View(),
			m.footerRule(),
			m.scrollFooter(),
		)
	}
	return styles.PaneBorder(m.focused).Render(content)
}

func (m *DetailViewModel) SetContent(content string) {
	m.vp.SetContent(content)
	m.vp.GotoTop()
}

func (m *DetailViewModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	vpH := h
	if h >= 6 {
		vpH = max(h-4, 1)
	}
	m.vp.Width = w
	m.vp.Height = vpH
}

func (m *DetailViewModel) SetResourceContext(kind, name string) {
	m.resourceKind = kind
	m.resourceName = name
}

func (m *DetailViewModel) SetLoading(loading bool) {
	m.loading = loading
}

func (m *DetailViewModel) SetFocused(focused bool) {
	m.focused = focused
}

func (m *DetailViewModel) SetMode(mode string) {
	m.mode = mode
}

func (m *DetailViewModel) ScrollToTop() {
	m.vp.GotoTop()
}

func (m DetailViewModel) ResourceKind() string  { return m.resourceKind }
func (m DetailViewModel) ResourceName() string  { return m.resourceName }
func (m DetailViewModel) Loading() bool         { return m.loading }
func (m DetailViewModel) Width() int            { return m.width }
func (m DetailViewModel) Mode() string          { return m.mode }
func (m DetailViewModel) Spinner() spinner.Model { return m.spinner } // AC#24

func (m DetailViewModel) titleBar() string {
	w := m.width
	accentBold := lipgloss.NewStyle().Foreground(styles.Accent).Bold(true)
	muted := lipgloss.NewStyle().Foreground(styles.Muted)

	leftText := accentBold.Render(m.resourceKind) +
		muted.Render(" / ") +
		accentBold.Render(m.resourceName)

	if m.mode != "" {
		leftText += muted.Render("  ") + accentBold.Render(m.mode)
	}

	if m.loading {
		leftText += muted.Render("  ") + m.spinner.View() + muted.Render(" loading…")
	}

	var rightText string
	if !m.loading {
		yHint := "[y] yaml"
		if m.mode == "yaml" {
			yHint = "[y] describe"
		}
		cHint := "[C] toggle conditions" // AC#6
		if m.mode == "conditions" {
			cHint = "[C] toggle conditions"
		}
		eHint := "[E] events" // FB-024
		if m.mode == "events" {
			eHint = "[E] describe"
		}
		rightText = muted.Render("[j/k] scroll  " + yHint + "  " + cHint + "  " + eHint + "  [x] delete  [Esc] back")
	}

	gap := w - lipgloss.Width(leftText) - lipgloss.Width(rightText)

	// Truncate resource name if it doesn't fit
	if gap < 2 && !m.loading {
		rightWidth := lipgloss.Width(rightText)
		kindWidth := lipgloss.Width(accentBold.Render(m.resourceKind))
		separatorWidth := lipgloss.Width(muted.Render(" / "))
		available := w - rightWidth - kindWidth - separatorWidth - 4
		if available > 0 {
			name := m.resourceName
			if len([]rune(name)) > available {
				runes := []rune(name)
				name = string(runes[:available]) + "…"
			}
			leftText = accentBold.Render(m.resourceKind) +
				muted.Render(" / ") +
				accentBold.Render(name)
			gap = w - lipgloss.Width(leftText) - rightWidth
		} else {
			// Drop keybind hint entirely
			rightText = ""
			gap = w - lipgloss.Width(leftText)
		}
	}

	if gap < 0 {
		gap = 0
	}

	return leftText + strings.Repeat(" ", gap) + rightText
}

func (m DetailViewModel) titleRule() string {
	return lipgloss.NewStyle().Foreground(styles.InactiveBorderColor).
		Render(strings.Repeat("─", m.width))
}

func (m DetailViewModel) footerRule() string {
	return lipgloss.NewStyle().Foreground(styles.InactiveBorderColor).
		Render(strings.Repeat("─", m.width))
}

func (m DetailViewModel) scrollFooter() string {
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

// conditionRow holds extracted fields from a single .status.conditions[] entry.
type conditionRow struct {
	Type               string
	Status             string
	Reason             string
	Message            string
	LastTransitionTime string // formatted "2006-01-02 15:04:05"; "" if missing/unparseable
}

// parseConditionRow extracts a conditionRow from an interface{} condition entry.
// Missing or wrong-type fields default to "". Never panics on any input (AC#13).
func parseConditionRow(raw interface{}) conditionRow {
	m, ok := raw.(map[string]interface{})
	if !ok {
		return conditionRow{}
	}
	strField := func(key string) string {
		v, ok := m[key]
		if !ok {
			return ""
		}
		s, ok := v.(string)
		if !ok {
			return ""
		}
		return s
	}
	ltt := ""
	if raw := strField("lastTransitionTime"); raw != "" {
		if t, err := time.Parse(time.RFC3339, raw); err == nil {
			ltt = t.UTC().Format("2006-01-02 15:04:05")
		}
	}
	return conditionRow{
		Type:               strField("type"),
		Status:             strField("status"),
		Reason:             strField("reason"),
		Message:            strField("message"),
		LastTransitionTime: ltt,
	}
}

// RenderConditionsTable parses .status.conditions from raw and renders a width-banded table.
// Width bands: [0,40) unusable, [40,60) narrow (T/S/R), [60,80) standard (T/S/R/LTT),
// [80,∞) wide (T/S/R/M/LTT). Non-Ready rows (status != "True") rendered in styles.Warning.
// Returns a muted placeholder when conditions are absent, empty, or unparseable (AC#11/12/13).
func RenderConditionsTable(raw *unstructured.Unstructured, width int) string { // AC#1
	mutedStyle := lipgloss.NewStyle().Foreground(styles.Muted)
	if width < 40 { // AC#18
		return mutedStyle.Render("Terminal too narrow — widen to 40+ columns")
	}

	conditions, found, err := unstructuredNestedSlice(raw.Object, "status", "conditions")
	if err != nil { // AC#13 — malformed .status structure
		return mutedStyle.Render("Conditions unavailable for this resource type.")
	}
	if !found || len(conditions) == 0 { // AC#11 / AC#12
		return mutedStyle.Render("No conditions reported for this resource.")
	}

	rows := make([]conditionRow, 0, len(conditions))
	for _, c := range conditions {
		rows = append(rows, parseConditionRow(c))
	}

	return renderConditionsBody(rows, width)
}

// unstructuredNestedSlice is a thin wrapper around k8s unstructured helpers to
// allow clear separation of the error path (malformed structure) from not-found.
func unstructuredNestedSlice(obj map[string]interface{}, fields ...string) ([]interface{}, bool, error) {
	return unstructured.NestedSlice(obj, fields...)
}

// RenderEventsTable renders the events table for the DetailPane events sub-view.
// Width bands (D6): [0,40) unusable, [40,60) narrow (T/R/A), [60,80) standard (T/R/A/C), [80,∞) wide (T/R/A/M/C). // AC#1
func RenderEventsTable(events []data.EventRow, loading bool, fetchErr error, rc data.ResourceClient, width int, sp spinner.Model) string {
	mutedStyle := lipgloss.NewStyle().Foreground(styles.Muted)
	if loading { // AC#24
		return mutedStyle.Render(sp.View() + " Loading events…")
	}
	if width < 40 { // AC#21
		return mutedStyle.Render("Terminal too narrow")
	}
	if fetchErr != nil {
		return renderEventsError(fetchErr, rc, width)
	}
	if len(events) == 0 { // AC#15
		return mutedStyle.Render("No events recorded for this resource.")
	}
	return renderEventsBody(events, width)
}

// renderEventsError routes fetchErr through the FB-022 canonical error path via RenderErrorBlock.
func renderEventsError(fetchErr error, rc data.ResourceClient, width int) string {
	title, detail := titleAndDetailForError(fetchErr, "Could not fetch events")
	sev := ErrorSeverityOf(fetchErr, rc)
	return RenderErrorBlock(ErrorBlock{
		Title:    title,
		Detail:   detail,
		Actions:  actionsForSeverity(sev, "back"),
		Severity: sev,
		Width:    width,
	})
}

// eventDisplayRow holds pre-formatted cell strings for a single events table row. // AC#16
type eventDisplayRow struct {
	Type    string
	Reason  string
	Age     string
	Message string
	Count   string
	Warning bool
}

// parseEventRow converts an EventRow to display strings. Zero/missing fields become "" (rendered as —). // AC#16
func parseEventRow(r data.EventRow) eventDisplayRow {
	d := eventDisplayRow{
		Type:    r.Type,
		Reason:  r.Reason,
		Message: r.Message,
		Warning: strings.EqualFold(r.Type, "Warning"), // AC#17
	}

	// Age: use LastTimestamp, fall back to EventTime, render — for zero/future. // AC#16
	ts := r.LastTimestamp
	if ts.IsZero() {
		ts = r.EventTime
	}
	if !ts.IsZero() {
		since := time.Since(ts)
		if since > 0 {
			d.Age = formatEventAge(since)
		}
		// future/clock-skew → d.Age stays ""
	}

	// Count == 0 renders as — per spec §2h. // AC#16
	if r.Count > 0 {
		d.Count = fmt.Sprintf("%d", r.Count)
	}

	return d
}

func formatEventAge(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}

// eventCell pads or truncates s to exactly w chars (appending … if truncated).
func eventCell(s string, w int) string {
	if s == "" {
		return strings.Repeat(" ", w-1) + "—"
	}
	if len(s) > w {
		if w > 1 {
			return s[:w-1] + "…"
		}
		return s[:w]
	}
	return s + strings.Repeat(" ", w-len(s))
}

type eventsBand int

const (
	eventsUnusable  eventsBand = iota // <40
	eventsNarrow                      // [40,60) — Type Reason Age
	eventsStandard                    // [60,80) — Type Reason Age Count
	eventsWide                        // [80,∞)  — Type Reason Age Message Count
)

func eventsWidthBand(width int) eventsBand {
	switch {
	case width >= 80:
		return eventsWide
	case width >= 60:
		return eventsStandard
	case width >= 40:
		return eventsNarrow
	default:
		return eventsUnusable
	}
}

// renderEventsBody renders the header + separator + data rows at the appropriate width band. // AC#18/19/20
func renderEventsBody(rows []data.EventRow, width int) string {
	band := eventsWidthBand(width)
	warningStyle := lipgloss.NewStyle().Foreground(styles.Warning)

	const typeW = 8
	const ageW = 12
	const countW = 5

	// Compute Reason width: cap at 24, min 10.
	reasonW := 10
	for _, r := range rows {
		if len(r.Reason) > reasonW {
			reasonW = len(r.Reason)
		}
	}
	if reasonW > 24 {
		reasonW = 24
	}

	// Message fills remaining width in wide band.
	msgW := width - typeW - ageW - countW - reasonW - 10 // 10 = column padding allowance
	if msgW < 10 {
		msgW = 10
	}

	var sb strings.Builder

	// Header row. // AC#2
	var header, sep string
	switch band {
	case eventsWide:
		header = "  " + eventCell("Type", typeW) + "  " + eventCell("Reason", reasonW) + "  " + eventCell("Age", ageW) + "  " + eventCell("Message", msgW) + "  " + eventCell("Count", countW)
		sep = "  " + strings.Repeat("─", typeW) + "  " + strings.Repeat("─", reasonW) + "  " + strings.Repeat("─", ageW) + "  " + strings.Repeat("─", msgW) + "  " + strings.Repeat("─", countW)
	case eventsStandard:
		header = "  " + eventCell("Type", typeW) + "  " + eventCell("Reason", reasonW) + "  " + eventCell("Age", ageW) + "  " + eventCell("Count", countW)
		sep = "  " + strings.Repeat("─", typeW) + "  " + strings.Repeat("─", reasonW) + "  " + strings.Repeat("─", ageW) + "  " + strings.Repeat("─", countW)
	case eventsNarrow:
		header = "  " + eventCell("Type", typeW) + "  " + eventCell("Reason", reasonW) + "  " + eventCell("Age", ageW)
		sep = "  " + strings.Repeat("─", typeW) + "  " + strings.Repeat("─", reasonW) + "  " + strings.Repeat("─", ageW)
	}
	sb.WriteString(header + "\n")
	sb.WriteString(sep + "\n")

	for _, r := range rows {
		d := parseEventRow(r)
		var line string
		switch band {
		case eventsWide:
			line = "  " + eventCell(d.Type, typeW) + "  " + eventCell(d.Reason, reasonW) + "  " + eventCell(d.Age, ageW) + "  " + eventCell(d.Message, msgW) + "  " + eventCell(d.Count, countW)
		case eventsStandard:
			line = "  " + eventCell(d.Type, typeW) + "  " + eventCell(d.Reason, reasonW) + "  " + eventCell(d.Age, ageW) + "  " + eventCell(d.Count, countW)
		case eventsNarrow:
			line = "  " + eventCell(d.Type, typeW) + "  " + eventCell(d.Reason, reasonW) + "  " + eventCell(d.Age, ageW)
		}
		if d.Warning { // AC#17
			sb.WriteString(warningStyle.Render(line) + "\n")
		} else {
			sb.WriteString(line + "\n")
		}
	}

	return strings.TrimRight(sb.String(), "\n")
}

// renderConditionsBody builds the header + separator + data rows for the given band.
func renderConditionsBody(rows []conditionRow, width int) string {
	warningStyle := lipgloss.NewStyle().Foreground(styles.Warning)

	// Compute column widths from data.
	typeW := 12
	for _, r := range rows {
		if l := len([]rune(r.Type)); l > typeW {
			typeW = l
		}
	}
	if typeW > 24 {
		typeW = 24
	}
	reasonW := 18
	for _, r := range rows {
		if l := len([]rune(r.Reason)); l > reasonW {
			reasonW = l
		}
	}
	if reasonW > 24 {
		reasonW = 24
	}
	const statusW = 7
	const lttW = 19

	dash := "—"

	cellStr := func(s string, w int) string {
		if s == "" {
			s = dash
		}
		runes := []rune(s)
		if len(runes) > w {
			return string(runes[:w-1]) + "…"
		}
		return s + strings.Repeat(" ", w-len(runes))
	}

	var sb strings.Builder

	// Band-specific rendering. AC#15/16/17/18/19
	switch {
	case width >= 80: // wide — all 5 columns // AC#15
		msgW := width - typeW - statusW - reasonW - lttW - 10 // 10 = 5 × 2-char padding
		if msgW < 1 {
			msgW = 1
		}
		header := "  " + cellStr("Type", typeW) + "  " + cellStr("Status", statusW) + "  " +
			cellStr("Reason", reasonW) + "  " + cellStr("Message", msgW) + "  " +
			cellStr("LastTransitionTime", lttW)
		sep := "  " + strings.Repeat("─", typeW) + "  " + strings.Repeat("─", statusW) + "  " +
			strings.Repeat("─", reasonW) + "  " + strings.Repeat("─", msgW) + "  " +
			strings.Repeat("─", lttW)
		sb.WriteString(header + "\n")
		sb.WriteString(sep + "\n")
		for _, r := range rows {
			line := "  " + cellStr(r.Type, typeW) + "  " + cellStr(r.Status, statusW) + "  " +
				cellStr(r.Reason, reasonW) + "  " + cellStr(r.Message, msgW) + "  " +
				cellStr(r.LastTransitionTime, lttW)
			if r.Status != "True" { // AC#14
				sb.WriteString(warningStyle.Render(line) + "\n")
			} else {
				sb.WriteString(line + "\n")
			}
		}
	case width >= 60: // standard — drop Message // AC#16
		header := "  " + cellStr("Type", typeW) + "  " + cellStr("Status", statusW) + "  " +
			cellStr("Reason", reasonW) + "  " + cellStr("LastTransitionTime", lttW)
		sep := "  " + strings.Repeat("─", typeW) + "  " + strings.Repeat("─", statusW) + "  " +
			strings.Repeat("─", reasonW) + "  " + strings.Repeat("─", lttW)
		sb.WriteString(header + "\n")
		sb.WriteString(sep + "\n")
		for _, r := range rows {
			line := "  " + cellStr(r.Type, typeW) + "  " + cellStr(r.Status, statusW) + "  " +
				cellStr(r.Reason, reasonW) + "  " + cellStr(r.LastTransitionTime, lttW)
			if r.Status != "True" { // AC#14
				sb.WriteString(warningStyle.Render(line) + "\n")
			} else {
				sb.WriteString(line + "\n")
			}
		}
	default: // narrow [40,60) — drop Message + LastTransitionTime // AC#17
		header := "  " + cellStr("Type", typeW) + "  " + cellStr("Status", statusW) + "  " +
			cellStr("Reason", reasonW)
		sep := "  " + strings.Repeat("─", typeW) + "  " + strings.Repeat("─", statusW) + "  " +
			strings.Repeat("─", reasonW)
		sb.WriteString(header + "\n")
		sb.WriteString(sep + "\n")
		for _, r := range rows {
			line := "  " + cellStr(r.Type, typeW) + "  " + cellStr(r.Status, statusW) + "  " +
				cellStr(r.Reason, reasonW)
			if r.Status != "True" { // AC#14
				sb.WriteString(warningStyle.Render(line) + "\n")
			} else {
				sb.WriteString(line + "\n")
			}
		}
	}

	return strings.TrimRight(sb.String(), "\n")
}

