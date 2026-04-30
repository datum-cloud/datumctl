package layout

type Dimensions struct {
	Width, Height int
}

const HeaderHeight = 8 // 5 wordmark + 1 separator + 1 user/org + 1 ns/pane/refresh
const FooterHeight = 2 // 1 top-border line + 1 content line
const FilterBarHeight = 1

func MainArea(totalHeight int) int {
	return MainAreaWithFilter(totalHeight, false)
}

func MainAreaWithFilter(totalHeight int, filterVisible bool) int {
	h := totalHeight - HeaderHeight - FooterHeight
	if filterVisible {
		h -= FilterBarHeight
	}
	if h < 0 {
		return 0
	}
	return h
}

// SidebarWidth returns a sidebar width that scales with the terminal: ~20% of
// total width, clamped to [16, 32]. This keeps the sidebar proportional across
// narrow and wide terminals without being a fixed column count.
func SidebarWidth(totalWidth int) int {
	w := totalWidth / 5
	if w < 16 {
		return 16
	}
	if w > 32 {
		return 32
	}
	return w
}

func TableWidth(totalWidth, sidebarWidth int) int {
	w := totalWidth - sidebarWidth
	if w < 0 {
		return 0
	}
	return w
}

// TableColumnWidths distributes a table's width across Name, Namespace, Status, and Age columns.
// When namespaced is true, a 20-wide Namespace column is included; otherwise ns = 0.
func TableColumnWidths(tableWidth int, namespaced bool) (name, ns, status, age int) {
	status = 14
	age = 8
	if namespaced {
		ns = 20
	}
	name = tableWidth - ns - status - age
	if name < 0 {
		name = 0
	}
	return name, ns, status, age
}
