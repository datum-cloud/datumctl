package components

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"go.datum.net/datumctl/internal/datumconfig"
	tuictx "go.datum.net/datumctl/internal/console/context"
	"go.datum.net/datumctl/internal/console/data"
	"go.datum.net/datumctl/internal/console/styles"
	"go.datum.net/datumctl/internal/discovery"
)

type ContextSwitchedMsg struct {
	Ctx tuictx.TUIContext
}

type treeEntry struct {
	isHeader bool
	label    string
	ctx      *datumconfig.DiscoveredContext
}

type CtxSwitcherModel struct {
	entries      []treeEntry
	cursor       int
	cfg          *datumconfig.ConfigV1Beta1
	dir          *discovery.Directory
	width        int
	height       int
	welcomeName  string // when set, renders a personalized post-login header
}

const ctxModalWidth = 60

// NewCtxSwitcherModel creates a context switcher over a live directory
// listing. A nil dir renders as a loading state.
func NewCtxSwitcherModel(cfg *datumconfig.ConfigV1Beta1, dir *discovery.Directory, width, height int) CtxSwitcherModel {
	m := CtxSwitcherModel{cfg: cfg, width: width, height: height}
	m.SetDirectory(dir)
	return m
}

// SetDirectory replaces the listed contexts with a fresh live listing.
func (m *CtxSwitcherModel) SetDirectory(dir *discovery.Directory) {
	m.dir = dir
	m.entries = nil
	if dir != nil {
		m.entries = buildTree(dir)
	}
	m.cursor = firstSelectableIdx(m.entries)
}

// NewPostLoginCtxSwitcherModel creates a context switcher with a personalized
// welcome header shown immediately after the user authenticates.
func NewPostLoginCtxSwitcherModel(cfg *datumconfig.ConfigV1Beta1, dir *discovery.Directory, width, height int, userName string) CtxSwitcherModel {
	m := NewCtxSwitcherModel(cfg, dir, width, height)
	// Use first name only for warmth
	firstName := userName
	if i := strings.Index(userName, " "); i > 0 {
		firstName = userName[:i]
	}
	m.welcomeName = firstName
	return m
}

func buildTree(dir *discovery.Directory) []treeEntry {
	contexts := dir.Contexts()
	var entries []treeEntry
	for _, o := range dir.Orgs {
		entries = append(entries, treeEntry{isHeader: true, label: dir.OrgDisplayName(o.Name)})
		for i := range contexts {
			ctx := &contexts[i]
			if ctx.OrganizationID != o.Name {
				continue
			}
			var label string
			if ctx.ProjectID != "" {
				label = dir.ProjectDisplayName(ctx.ProjectID)
			} else {
				label = "(org-wide)"
			}
			entries = append(entries, treeEntry{label: label, ctx: ctx})
		}
	}
	return entries
}

func firstSelectableIdx(entries []treeEntry) int {
	for i, e := range entries {
		if !e.isHeader {
			return i
		}
	}
	return 0
}

func (m CtxSwitcherModel) Init() tea.Cmd { return nil }

func (m CtxSwitcherModel) Update(msg tea.Msg) (CtxSwitcherModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			m.cursor = nextSelectable(m.entries, m.cursor, 1)
		case "k", "up":
			m.cursor = nextSelectable(m.entries, m.cursor, -1)
		case "enter":
			if m.cursor < 0 || m.cursor >= len(m.entries) {
				return m, nil
			}
			e := m.entries[m.cursor]
			if e.isHeader || e.ctx == nil {
				return m, nil
			}
			// Under --session or DATUM_SESSION the switch lasts only for this
			// console; the stored current context stays unchanged. The entry is
			// held in memory only and is dropped if the config is saved.
			if datumconfig.HasSessionOverride() {
				if m.cfg.ContextByName(e.ctx.Name) == nil {
					m.cfg.UpsertContext(*e.ctx)
				}
				datumconfig.SetOverrideContext(e.ctx.Name)
				newCtx := tuictx.FromConfig(m.cfg)
				newCtx.ApplyNames(m.dir)
				return m, func() tea.Msg { return ContextSwitchedMsg{Ctx: newCtx} }
			}
			m.cfg.SelectContext(*e.ctx)
			if err := datumconfig.SaveV1Beta1(m.cfg); err != nil {
				return m, func() tea.Msg {
					return data.LoadErrorMsg{Err: err, Severity: data.SeverityOfClassified(err)}
				}
			}
			newCtx := tuictx.FromConfig(m.cfg)
			newCtx.ApplyNames(m.dir)
			return m, func() tea.Msg { return ContextSwitchedMsg{Ctx: newCtx} }
		}
	}
	return m, nil
}

func nextSelectable(entries []treeEntry, cur, dir int) int {
	n := len(entries)
	if n == 0 {
		return cur
	}
	next := cur + dir
	for next >= 0 && next < n {
		if !entries[next].isHeader {
			return next
		}
		next += dir
	}
	return cur
}

func (m CtxSwitcherModel) View() string {
	headerStyle := lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Primary).Bold(true)
	selectedStyle := lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Accent).Bold(true)
	normalStyle := lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Secondary)
	currentStyle := lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Success)
	muted := lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Muted).Italic(true)
	accent := lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Accent).Bold(true)
	secondary := lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Secondary)

	var lines []string

	if m.welcomeName != "" {
		lines = append(lines, accent.Render("Welcome, "+m.welcomeName+"!"))
		lines = append(lines, secondary.Render("Choose where you'd like to work."))
		lines = append(lines, "")
	}

	if m.cfg == nil || len(m.entries) == 0 {
		msg := "No contexts available"
		if m.cfg != nil && m.dir == nil {
			msg = "Loading contexts…"
		}
		empty := lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Muted).
			Width(ctxModalWidth - 4).Align(lipgloss.Center).
			Render(msg)
		lines = append(lines, empty)
	} else {
		for i, e := range m.entries {
			if e.isHeader {
				lines = append(lines, headerStyle.Render("▾ "+e.label))
				continue
			}
			isCurrent := m.cfg != nil && e.ctx != nil && e.ctx.Name == m.cfg.CurrentContextName()
			indent := "  "
			var line string
			switch {
			case i == m.cursor && isCurrent:
				line = selectedStyle.Render(indent + "▸ ✓ " + e.label)
			case i == m.cursor:
				line = selectedStyle.Render(indent + "▸ " + strings.TrimSpace(e.label))
			case isCurrent:
				line = currentStyle.Render(indent + "✓ " + e.label)
			default:
				line = normalStyle.Render(indent + "  " + e.label)
			}
			lines = append(lines, line)
		}
	}

	footer := muted.Render("[Enter] switch  [Esc] close")
	lines = append(lines, "", footer)

	body := lipgloss.JoinVertical(lipgloss.Left, lines...)
	body = styles.SurfaceFill(body, ctxModalWidth, lipgloss.Height(body))
	modal := styles.OverlayStyle.Width(ctxModalWidth).Render(body)

	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center, modal,
		lipgloss.WithWhitespaceStyle(lipgloss.NewStyle().Background(styles.OverlayBackdrop)),
	)
}
