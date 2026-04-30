package components

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"go.datum.net/datumctl/internal/console/data"
	"go.datum.net/datumctl/internal/console/styles"
)

type resourceTypeItem struct {
	rt data.ResourceType
}

func (i resourceTypeItem) Title() string       { return i.rt.Name }
func (i resourceTypeItem) Description() string { return i.rt.Group }
func (i resourceTypeItem) FilterValue() string { return i.rt.Name }

// headerItem is a non-selectable group header that appears above each group's
// resource-type rows in the sidebar list.
type headerItem struct{ label string }

func (h headerItem) Title() string       { return h.label }
func (h headerItem) Description() string { return "" }
func (h headerItem) FilterValue() string { return "" }

type compactDelegate struct {
	focused bool
	counts  map[string]int
	width   int
}

func (d compactDelegate) Height() int  { return 1 }
func (d compactDelegate) Spacing() int { return 0 }

func (d compactDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

func (d compactDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	// Group headers render as bold-muted section labels with 1-space indent.
	if h, ok := item.(headerItem); ok {
		available := d.width
		label := h.label
		if available > 0 && len(label) > available-2 {
			if available-2 > 0 {
				label = label[:available-2] + "…"
			} else {
				label = ""
			}
		}
		fmt.Fprint(w, lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Muted).Bold(true).Render(" "+label))
		return
	}

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
		fmt.Fprint(w, lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Primary).Bold(true).Render("▸ "+name))
	} else if index == m.Index() {
		fmt.Fprint(w, lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Secondary).Render("▸ "+name))
	} else {
		fmt.Fprint(w, lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Muted).Render("  "+name))
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
	delegate := compactDelegate{counts: make(map[string]int), width: width}
	l := list.New(nil, delegate, width, height)
	l.SetShowTitle(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(false)
	l.SetShowStatusBar(false)
	l.Styles.NoItems = lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Muted).Padding(1, 2)
	return NavSidebarModel{list: l, counts: make(map[string]int), width: width, height: height}
}

func (m NavSidebarModel) Init() tea.Cmd { return nil }

func (m NavSidebarModel) Update(msg tea.Msg) (NavSidebarModel, tea.Cmd) {
	// Detect navigation key direction before delegating to the list, so we can
	// post-process a cursor that lands on a headerItem.
	var navDir int // -1 = up, +1 = down, 0 = no nav key
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "j", "down":
			navDir = 1
		case "k", "up":
			navDir = -1
		}
	}

	// Record cursor index before the list update so we can restore it if the
	// skip logic cannot find a non-header item to land on (boundary case).
	preUpdateIdx := m.list.Index()

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)

	// Skip over any headerItem the cursor landed on after a navigation key.
	if navDir != 0 {
		for {
			sel := m.list.SelectedItem()
			if sel == nil {
				break
			}
			if _, isHeader := sel.(headerItem); !isHeader {
				break
			}
			prevIdx := m.list.Index()
			if navDir > 0 {
				m.list.CursorDown()
			} else {
				m.list.CursorUp()
			}
			if m.list.Index() == prevIdx {
				// Cursor did not move — restore to the position before the key
				// was processed so the overall key press is a no-op.
				m.list.Select(preUpdateIdx)
				break
			}
		}
	}

	return m, cmd
}

func (m NavSidebarModel) View() string {
	return styles.PaneBorder(m.focused).Render(styles.SurfaceFill(m.list.View(), m.width, m.height))
}

// SetItems groups, filters, and sorts resource types, inserts group headers,
// then loads the result into the bubbles list. The previously selected type
// name is restored when it remains in the new list.
func (m *NavSidebarModel) SetItems(types []data.ResourceType) {
	// Capture the currently selected type name for cursor restoration.
	prevName := ""
	if rt, ok := m.SelectedType(); ok {
		prevName = rt.Name
	}

	// Filter out hidden platform types.
	visible := make([]data.ResourceType, 0, len(types))
	for _, rt := range types {
		if !data.ShouldHideResourceType(rt) {
			visible = append(visible, rt)
		}
	}

	// Group by API group.
	grouped := make(map[string][]data.ResourceType)
	for _, rt := range visible {
		grouped[rt.Group] = append(grouped[rt.Group], rt)
	}

	// Sort types within each group case-insensitively by name.
	for g := range grouped {
		sort.Slice(grouped[g], func(i, j int) bool {
			return strings.ToLower(grouped[g][i].Name) < strings.ToLower(grouped[g][j].Name)
		})
	}

	// Collect all groups: non-core groups sorted alphabetically by display
	// name, with the core group ("") always placed last.
	nonCoreGroups := make([]string, 0, len(grouped))
	hasCoreGroup := false
	for g := range grouped {
		if g == "" {
			hasCoreGroup = true
		} else {
			nonCoreGroups = append(nonCoreGroups, g)
		}
	}
	sort.Slice(nonCoreGroups, func(i, j int) bool {
		return data.GroupDisplayName(nonCoreGroups[i]) < data.GroupDisplayName(nonCoreGroups[j])
	})

	orderedGroups := nonCoreGroups
	if hasCoreGroup {
		orderedGroups = append(orderedGroups, "")
	}

	// Build the flat item list with interleaved headers.
	items := make([]list.Item, 0, len(visible)+len(orderedGroups))
	for _, g := range orderedGroups {
		items = append(items, headerItem{label: data.GroupDisplayName(g)})
		for _, rt := range grouped[g] {
			items = append(items, resourceTypeItem{rt: rt})
		}
	}

	m.list.SetItems(items)

	// Always reset cursor to 0 first, then restore if the previously selected
	// type is still present.
	m.list.Select(0)

	if prevName != "" {
		for i, item := range items {
			if rti, ok := item.(resourceTypeItem); ok && rti.rt.Name == prevName {
				m.list.Select(i)
				break
			}
		}
	}

	// Ensure the cursor does not rest on a headerItem after SetItems.
	m.skipHeaderForward()
}

// skipHeaderForward advances the cursor forward past any headerItem it currently
// rests on. Called after SetItems to guarantee the initial cursor position is
// always a resourceTypeItem.
func (m *NavSidebarModel) skipHeaderForward() {
	items := m.list.Items()
	for {
		sel := m.list.SelectedItem()
		if sel == nil {
			break
		}
		if _, isHeader := sel.(headerItem); !isHeader {
			break
		}
		idx := m.list.Index()
		if idx+1 >= len(items) {
			break
		}
		m.list.CursorDown()
	}
}

func (m *NavSidebarModel) SetCount(typeName string, n int) {
	if m.counts == nil {
		m.counts = make(map[string]int)
	}
	m.counts[typeName] = n
	d := compactDelegate{focused: m.focused, counts: m.counts, width: m.width}
	m.list.SetDelegate(d)
}

func (m *NavSidebarModel) SetFocused(focused bool) {
	m.focused = focused
	d := compactDelegate{focused: focused, counts: m.counts, width: m.width}
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
