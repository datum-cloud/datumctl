package components

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"go.datum.net/datumctl/internal/datumconfig"
	tuictx "go.datum.net/datumctl/internal/console/context"
	"go.datum.net/datumctl/internal/console/data"
	"go.datum.net/datumctl/internal/console/styles"
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
	entries []treeEntry
	cursor  int
	cfg     *datumconfig.ConfigV1Beta1
	width   int
	height  int
}

const ctxModalWidth = 60

func NewCtxSwitcherModel(cfg *datumconfig.ConfigV1Beta1, width, height int) CtxSwitcherModel {
	m := CtxSwitcherModel{cfg: cfg, width: width, height: height}
	if cfg != nil {
		m.entries = buildTree(cfg)
		m.cursor = firstSelectableIdx(m.entries)
	}
	return m
}

func buildTree(cfg *datumconfig.ConfigV1Beta1) []treeEntry {
	activeSession := ""
	if s := cfg.ActiveSessionEntry(); s != nil {
		activeSession = s.Name
	}

	seen := map[string]bool{}
	var orgOrder []string
	for _, ctx := range cfg.Contexts {
		if activeSession != "" && ctx.Session != activeSession {
			continue
		}
		if !seen[ctx.OrganizationID] {
			seen[ctx.OrganizationID] = true
			orgOrder = append(orgOrder, ctx.OrganizationID)
		}
	}

	var entries []treeEntry
	for _, orgID := range orgOrder {
		orgName := cfg.OrgDisplayName(orgID)
		entries = append(entries, treeEntry{isHeader: true, label: orgName})
		for i := range cfg.Contexts {
			ctx := &cfg.Contexts[i]
			if activeSession != "" && ctx.Session != activeSession {
				continue
			}
			if ctx.OrganizationID != orgID {
				continue
			}
			var label string
			if ctx.ProjectID != "" {
				label = cfg.ProjectDisplayName(ctx.ProjectID)
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
			m.cfg.CurrentContext = e.ctx.Name
			if err := datumconfig.SaveV1Beta1(m.cfg); err != nil {
				return m, func() tea.Msg {
					return data.LoadErrorMsg{Err: err, Severity: data.SeverityOfClassified(err)}
				}
			}
			newCtx := tuictx.FromConfig(m.cfg)
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

	var lines []string

	if m.cfg == nil || len(m.cfg.Contexts) == 0 {
		empty := lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Muted).
			Width(ctxModalWidth - 4).Align(lipgloss.Center).
			Render("No contexts available")
		lines = append(lines, empty)
	} else {
		for i, e := range m.entries {
			if e.isHeader {
				lines = append(lines, headerStyle.Render("▾ "+e.label))
				continue
			}
			isCurrent := m.cfg != nil && e.ctx != nil && e.ctx.Name == m.cfg.CurrentContext
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
