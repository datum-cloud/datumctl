package components

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"go.datum.net/datumctl/internal/console/data"
	"go.datum.net/datumctl/internal/console/styles"
)

// pluralizeKind returns "project" for n==1, "projects" for n>1.
func pluralizeKind(kind string, n int) string {
	s := strings.ToLower(kind)
	if n == 1 {
		return s
	}
	return s + "s"
}

// RenderQuotaTree renders a two-bucket (parent+child) tree for the S3 describe
// inline quota block. Falls back to flat rendering when tree is not applicable.
// The returned string does NOT include a trailing newline.
func RenderQuotaTree(tree data.TreeBuckets, width int, siblingsRestricted bool) string {
	if !tree.HasTree {
		return ""
	}

	muted := lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Muted)

	// Bar width: overhead = 3 (connector) + 20 (consumer label) + 2 (brackets) + 18 (counts) + 8 (suffix) = 51
	barWidth := max(10, width-51)

	var lines []string

	// Parent row: ├─ ConsumerKind ConsumerName   [bar] NNN/NNN
	hasTreeSiblingRow := len(tree.Siblings) > 0 && !siblingsRestricted
	parentConnector := "├─ "

	parentLabel := consumerLabel(tree.Parent.ConsumerKind, tree.Parent.ConsumerName)
	parentSibNote := ""
	if siblingsRestricted {
		parentSibNote = muted.Render(" (other projects' usage hidden)")
	}
	parentLine := muted.Render(parentConnector) + muted.Render(parentLabel+parentSibNote) +
		renderTreeBarCounts(*tree.Parent, barWidth)
	lines = append(lines, parentLine)

	// Sibling-consume row: │    sibling-consume   [bar] X of Limit by N projects   (you are ...)
	if hasTreeSiblingRow {
		lines = append(lines, renderSiblingConsumeLine(tree, barWidth, muted))
	} else if !siblingsRestricted && len(tree.Siblings) == 0 {
		// No siblings at all: no trunk line between parent and child
		// (handled by using └─ on parent if no sibling row)
		// Spec: two-row tree when no siblings — just parent + child, no │ row
	}

	// If no sibling-consume row, change parent connector to ├─ (it stays since child follows)
	// The parent always uses ├─ when a child follows.
	_ = parentConnector // already set above

	// Child row: └─ ConsumerKind ConsumerName   [bar] NNN/NNN
	childLabel := consumerLabel(tree.ActiveChild.ConsumerKind, tree.ActiveChild.ConsumerName)
	childLine := muted.Render("└─ ") + muted.Render(childLabel) +
		renderTreeBarCounts(*tree.ActiveChild, barWidth)
	lines = append(lines, childLine)

	return strings.Join(lines, "\n")
}

// consumerLabel formats a fixed-width (20-char) consumer label for tree rows.
// Format: "Kind Name" padded with spaces to 20 chars.
func consumerLabel(kind, name string) string {
	label := kind + " " + name
	runes := []rune(label)
	if len(runes) >= 20 {
		return string(runes[:20])
	}
	return label + strings.Repeat(" ", 20-len(runes))
}

// renderTreeBarCounts renders the bar + counts portion of a tree row.
func renderTreeBarCounts(b data.AllowanceBucket, barWidth int) string {
	if b.Limit == 0 {
		muted := lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Muted)
		bar := muted.Render(strings.Repeat("░", barWidth))
		counts := muted.Render(fmt.Sprintf(" %3d / %-3s (%-3s)", b.Allocated, "∞", "—"))
		return "[" + bar + "]" + counts
	}
	pct := float64(b.Allocated) / float64(b.Limit)
	pctInt := int(pct * 100)
	filled := min(barWidth, max(0, int(float64(barWidth)*pct)))
	empty := barWidth - filled
	barStr := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	countsStr := fmt.Sprintf(" %3d / %3d (%3d%%)", b.Allocated, b.Limit, pctInt)
	barStyle, suffixStyle, suffix := QuotaBarStyling(pctInt)
	result := "[" + barStyle.Render(barStr) + "]" + barStyle.Render(countsStr)
	if suffix != "" {
		result += suffixStyle.Render(suffix)
	} else {
		result += strings.Repeat(" ", 8)
	}
	return result
}


// renderSiblingConsumeLine builds the │    sibling-consume row for the S3 tree.
func renderSiblingConsumeLine(tree data.TreeBuckets, barWidth int, muted lipgloss.Style) string {
	var sibAllocated int64
	for _, s := range tree.Siblings {
		sibAllocated += s.Allocated
	}
	sibKind := ""
	if len(tree.Siblings) > 0 {
		sibKind = pluralizeKind(tree.Siblings[0].ConsumerKind, len(tree.Siblings))
	}
	sibLimit := tree.Parent.Limit
	pct := float64(0)
	if sibLimit > 0 {
		pct = float64(sibAllocated) / float64(sibLimit)
	}
	filled := min(barWidth, max(0, int(float64(barWidth)*pct)))
	empty := barWidth - filled
	barStr := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	countStr := fmt.Sprintf(" %3d of %-3d by %d %s", sibAllocated, sibLimit, len(tree.Siblings), sibKind)

	return muted.Render("│    sibling-consume  ") +
		muted.Render("["+barStr+"]") +
		muted.Render(countStr)
}

// RenderQuotaBlock renders a single AllowanceBucket as a 2-line block.
//
// Format:
//
//	  <label>                                 ← Accent bold (display name or short name)
//	  [<bar>] NNN / NNN (PPP%) <suffix>       ← bar tinted by threshold
//
// registrations is the optional slice of ResourceRegistration from the platform
// API; pass nil to fall back to the short-name (last "/" segment of ResourceType).
func RenderQuotaBlock(b data.AllowanceBucket, width int, registrations []data.ResourceRegistration) string {
	accentBold := lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Accent).Bold(true)

	label := resolveBlockLabel(b, registrations)
	header := accentBold.Render("  " + label)
	barLine := renderBarLine(b, width)

	return header + "\n" + barLine
}

// resolveBlockLabel returns the display name for a bucket's resource type.
// Uses data.ResolveDescription against registrations; falls back to the short
// name (last "/" segment of ResourceType) when no description is found.
func resolveBlockLabel(b data.AllowanceBucket, registrations []data.ResourceRegistration) string {
	group, name := data.SplitResourceType(b.ResourceType)
	if dn := data.ResolveDescription(registrations, group, name); dn != "" {
		return dn
	}
	return name
}

// renderBarLine produces: "  [<bar>] NNN / NNN (PPP%) <suffix>"
// Total visible width = width chars (before lipgloss renders escape codes).
func renderBarLine(b data.AllowanceBucket, width int) string {
	// overhead: 2 indent + 1 "[" + 1 "]" + 14 counts field + 8 suffix gutter = 26
	barWidth := max(10, width-26)

	if b.Limit == 0 {
		bar := lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Muted).Render(strings.Repeat("░", barWidth))
		counts := lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Muted).
			Render(fmt.Sprintf(" %3d / %-3s (%-3s)", b.Allocated, "∞", "—"))
		return "  [" + bar + "]" + counts
	}

	pct := float64(b.Allocated) / float64(b.Limit)
	pctInt := int(pct * 100)

	filled := min(barWidth, max(0, int(float64(barWidth)*pct)))
	empty := barWidth - filled
	barStr := strings.Repeat("█", filled) + strings.Repeat("░", empty)

	countsStr := fmt.Sprintf(" %3d / %3d (%3d%%)", b.Allocated, b.Limit, pctInt)

	barColor, suffixColor, suffix := QuotaBarStyling(pctInt)

	rendered := "  [" + barColor.Render(barStr) + "]" + barColor.Render(countsStr)
	if suffix != "" {
		rendered += suffixColor.Render(suffix)
	}
	return rendered
}

// QuotaBarStyling returns the bar style, suffix style, and textual suffix for a
// given integer percentage. Single source of truth for the threshold color grammar
// shared by RenderQuotaBlock (FB-010) and the quota banner (FB-012).
func QuotaBarStyling(pctInt int) (barStyle, suffixStyle lipgloss.Style, suffix string) {
	switch {
	case pctInt >= 100:
		s := lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Error).Bold(true)
		return s, s, " ⛔ full"
	case pctInt >= 90:
		s := lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Error)
		return s, s, " ⚠ near"
	case pctInt >= 70:
		s := lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Warning)
		return s, s, ""
	default:
		return lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Success), lipgloss.NewStyle(), ""
	}
}
