package components

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"go.datum.net/datumctl/internal/tui/data"
	"go.datum.net/datumctl/internal/tui/styles"
)

type resourceTypeItem struct {
	rt data.ResourceType
}

func (i resourceTypeItem) Title() string       { return i.rt.Name }
func (i resourceTypeItem) Description() string { return i.rt.Group }
func (i resourceTypeItem) FilterValue() string { return i.rt.Name }

type compactDelegate struct {
	focused bool
	counts  map[string]int
}

func (d compactDelegate) Height() int  { return 1 }
func (d compactDelegate) Spacing() int { return 0 }

func (d compactDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

func (d compactDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	rti, ok := item.(resourceTypeItem)
	if !ok {
		return
	}
	name := rti.rt.Name
	if d.counts != nil {
		if n, ok := d.counts[rti.rt.Name]; ok && n > 0 {
			name = fmt.Sprintf("%s (%d)", name, n)
		}
	}
	if len(name) > 20 {
		name = name[:19] + "…"
	}

	if index == m.Index() && d.focused {
		fmt.Fprint(w, lipgloss.NewStyle().Foreground(styles.Primary).Bold(true).Render("▸ "+name))
	} else if index == m.Index() {
		fmt.Fprint(w, lipgloss.NewStyle().Foreground(styles.Secondary).Render("▸ "+name))
	} else {
		fmt.Fprint(w, lipgloss.NewStyle().Foreground(styles.Muted).Render("  "+name))
	}
}

type NavSidebarModel struct {
	list    list.Model
	counts  map[string]int
	focused bool
	width   int
	height  int
}

func NewNavSidebarModel(width, height int) NavSidebarModel {
	delegate := compactDelegate{counts: make(map[string]int)}
	l := list.New(nil, delegate, width, height)
	l.SetShowTitle(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(false)
	l.SetShowStatusBar(false)
	l.Styles.NoItems = lipgloss.NewStyle().Foreground(styles.Muted).Padding(1, 2)
	return NavSidebarModel{list: l, counts: make(map[string]int), width: width, height: height}
}

func (m NavSidebarModel) Init() tea.Cmd { return nil }

func (m NavSidebarModel) Update(msg tea.Msg) (NavSidebarModel, tea.Cmd) {
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m NavSidebarModel) View() string {
	return styles.PaneBorder(m.focused).Width(m.width).Height(m.height).Render(m.list.View())
}

func (m *NavSidebarModel) SetItems(types []data.ResourceType) {
	items := make([]list.Item, len(types))
	for i, rt := range types {
		items[i] = resourceTypeItem{rt: rt}
	}
	m.list.SetItems(items)
}

func (m *NavSidebarModel) SetCount(typeName string, n int) {
	if m.counts == nil {
		m.counts = make(map[string]int)
	}
	m.counts[typeName] = n
	d := compactDelegate{focused: m.focused, counts: m.counts}
	m.list.SetDelegate(d)
}

func (m *NavSidebarModel) SetFocused(focused bool) {
	m.focused = focused
	d := compactDelegate{focused: focused, counts: m.counts}
	m.list.SetDelegate(d)
}

func (m NavSidebarModel) SelectedType() (data.ResourceType, bool) {
	item := m.list.SelectedItem()
	if item == nil {
		return data.ResourceType{}, false
	}
	rti, ok := item.(resourceTypeItem)
	if !ok {
		return data.ResourceType{}, false
	}
	return rti.rt, true
}
