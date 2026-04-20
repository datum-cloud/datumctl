package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"go.datum.net/datumctl/internal/tui/data"
	"go.datum.net/datumctl/internal/tui/styles"
)

// QuotaBannerModel renders quota usage above the ResourceTable in TablePane /
// NavPane. In single-bucket mode it renders one line per bucket. In tree mode
// it renders 2–3 lines (parent, optional sibling-consume, child).
type QuotaBannerModel struct {
	width              int
	buckets            []data.AllowanceBucket
	activeConsumerKind string
	activeConsumerName string
	siblingRestricted  bool
	registrations      []data.ResourceRegistration // nil until FB-014 fetch completes
}

func NewQuotaBannerModel(width int) QuotaBannerModel {
	return QuotaBannerModel{width: width}
}

func (m *QuotaBannerModel) SetSize(w int) { m.width = w }

func (m *QuotaBannerModel) SetBuckets(buckets []data.AllowanceBucket) {
	m.buckets = buckets
}

func (m *QuotaBannerModel) SetActiveConsumer(kind, name string) {
	m.activeConsumerKind = kind
	m.activeConsumerName = name
}

func (m *QuotaBannerModel) SetSiblingRestricted(v bool) {
	m.siblingRestricted = v
}

// SetRegistrations sets the ResourceRegistration slice used to resolve display
// names in banner labels. Pass nil to fall back to short-name labels.
func (m *QuotaBannerModel) SetRegistrations(regs []data.ResourceRegistration) {
	m.registrations = regs
}

// Height returns the number of rendered lines (0 when no buckets match).
func (m QuotaBannerModel) Height() int {
	if len(m.buckets) == 0 {
		return 0
	}
	tree := data.ClassifyTreeBuckets(m.buckets, m.activeConsumerKind, m.activeConsumerName)
	if !tree.HasTree {
		return len(m.buckets)
	}
	h := 2 // parent + child
	if len(tree.Siblings) > 0 && !m.siblingRestricted {
		h++ // sibling-consume row
	}
	return h
}

// HasBuckets reports whether any matching buckets are set.
func (m QuotaBannerModel) HasBuckets() bool { return len(m.buckets) > 0 }

// View returns the rendered banner, or "" when height is zero.
func (m QuotaBannerModel) View() string {
	if len(m.buckets) == 0 {
		return ""
	}
	tree := data.ClassifyTreeBuckets(m.buckets, m.activeConsumerKind, m.activeConsumerName)
	if !tree.HasTree {
		lines := make([]string, 0, len(m.buckets))
		for _, b := range m.buckets {
			lines = append(lines, renderBannerLineWithNames(b, m.width, m.registrations))
		}
		return strings.Join(lines, "\n")
	}
	return m.renderBannerTree(tree)
}

func renderBannerLineWithNames(b data.AllowanceBucket, width int, regs []data.ResourceRegistration) string {
	resourceLabel := bucketResourceLabel(b, regs)

	// Truncate display names that would overflow — reserve at most 40% of width
	// for the label in full form, 60% in compact. Short-name fallbacks are already
	// short enough that truncation is a no-op for them.
	fullBudget := max(8, width*40/100)
	resourceLabel = truncateLabelToWidth(resourceLabel, fullBudget)

	// Compute how many columns remain for the progress bar in full form.
	// overhead = indent(2) + resourceLabel + " • Quota "(9) + "[]"(2) +
	//            " NNN / NNN (NNN%)"(18) + suffix+gutter(8)
	overhead := 2 + lipgloss.Width(resourceLabel) + 9 + 2 + 18 + 8
	barWidth := width - overhead

	if barWidth < 10 {
		compactBudget := max(8, width*60/100)
		compactLabel := truncateLabelToWidth(bucketResourceLabel(b, regs), compactBudget)
		return renderBannerCompact(b, compactLabel, width)
	}
	return renderBannerFull(b, resourceLabel, barWidth)
}

// truncateLabelToWidth trims s to at most budget terminal cells, appending "…"
// when truncation occurs. Measures with lipgloss.Width to handle grapheme clusters.
func truncateLabelToWidth(s string, budget int) string {
	if budget < 1 {
		budget = 1
	}
	if lipgloss.Width(s) <= budget {
		return s
	}
	r := []rune(s)
	for len(r) > 0 && lipgloss.Width(string(r)+"…") > budget {
		r = r[:len(r)-1]
	}
	return string(r) + "…"
}

func renderBannerFull(b data.AllowanceBucket, resourceLabel string, barWidth int) string {
	muted := lipgloss.NewStyle().Foreground(styles.Muted)
	secondary := lipgloss.NewStyle().Foreground(styles.Secondary).Bold(true)

	prefix := "  " + secondary.Render(resourceLabel) + muted.Render("  • Quota ")

	if b.Limit == 0 {
		bar := muted.Render(strings.Repeat("░", barWidth))
		counts := muted.Render(fmt.Sprintf(" %3d / %-3s (%-3s)", b.Allocated, "∞", "—"))
		return prefix + "[" + bar + "]" + counts + strings.Repeat(" ", 8)
	}

	pct := float64(b.Allocated) / float64(b.Limit)
	pctInt := int(pct * 100)
	filled := min(barWidth, max(0, int(float64(barWidth)*pct)))
	empty := barWidth - filled
	barStr := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	countsStr := fmt.Sprintf(" %3d / %3d (%3d%%)", b.Allocated, b.Limit, pctInt)

	barStyle, suffixStyle, suffix := QuotaBarStyling(pctInt)
	rendered := prefix + "[" + barStyle.Render(barStr) + "]" + barStyle.Render(countsStr)

	if suffix != "" {
		suffixPadded := suffix + strings.Repeat(" ", max(0, 8-len([]rune(suffix))))
		rendered += suffixStyle.Render(suffixPadded)
	} else {
		rendered += strings.Repeat(" ", 8)
	}

	return rendered
}

func renderBannerCompact(b data.AllowanceBucket, resourceLabel string, _ int) string {
	muted := lipgloss.NewStyle().Foreground(styles.Muted)
	secondary := lipgloss.NewStyle().Foreground(styles.Secondary).Bold(true)

	prefix := "  " + secondary.Render(resourceLabel) + muted.Render(" • ")

	if b.Limit == 0 {
		counts := muted.Render(fmt.Sprintf("%d / ∞ (—)", b.Allocated))
		return prefix + counts
	}

	pct := float64(b.Allocated) / float64(b.Limit)
	pctInt := int(pct * 100)
	countsStr := fmt.Sprintf("%d / %d (%d%%)", b.Allocated, b.Limit, pctInt)

	_, suffixStyle, suffix := QuotaBarStyling(pctInt)
	barStyle, _, _ := QuotaBarStyling(pctInt)
	result := prefix + barStyle.Render(countsStr)
	if suffix != "" {
		result += suffixStyle.Render(suffix)
	}
	return result
}


// bucketResourceLabel returns the display name for a bucket's resource type.
// Uses data.ResolveDescription against regs; falls back to the short name
// (last "/" segment of ResourceType) when no description is found.
func bucketResourceLabel(b data.AllowanceBucket, regs []data.ResourceRegistration) string {
	group, name := data.SplitResourceType(b.ResourceType)
	if dn := data.ResolveDescription(regs, group, name); dn != "" {
		return dn
	}
	return name
}

// renderBannerTree renders a 2- or 3-line tree banner for a parent+child pair.
func (m QuotaBannerModel) renderBannerTree(tree data.TreeBuckets) string {
	// overhead: connector(3) + consumerLabel(20) + "[]"(2) + counts(18) + suffix(8)
	overhead := 3 + 20 + 2 + 18 + 8
	barWidth := m.width - overhead

	compact := barWidth < 10

	hasSibRow := len(tree.Siblings) > 0 && !m.siblingRestricted

	muted := lipgloss.NewStyle().Foreground(styles.Muted)

	var lines []string

	if compact {
		// Compact: connector + short kind label + counts only
		lines = append(lines, m.renderBannerTreeRowCompact("├─ ", *tree.Parent, m.siblingRestricted, muted))
		if hasSibRow {
			lines = append(lines, m.renderBannerSibCompact(tree, muted))
		}
		lines = append(lines, m.renderBannerTreeRowCompact("└─ ", *tree.ActiveChild, false, muted))
		return strings.Join(lines, "\n")
	}

	// Full form
	lines = append(lines, m.renderBannerTreeRowFull("├─ ", *tree.Parent, m.siblingRestricted, barWidth, muted))
	if hasSibRow {
		lines = append(lines, m.renderBannerSibFull(tree, barWidth, muted))
	}
	lines = append(lines, m.renderBannerTreeRowFull("└─ ", *tree.ActiveChild, false, barWidth, muted))
	return strings.Join(lines, "\n")
}

// renderBannerTreeRowFull renders one full-form tree row for the banner.
func (m QuotaBannerModel) renderBannerTreeRowFull(
	connector string, b data.AllowanceBucket,
	sibRestricted bool,
	barWidth int,
	muted lipgloss.Style,
) string {
	connectorRendered := muted.Render(connector)
	labelStr := consumerLabel(b.ConsumerKind, b.ConsumerName)
	labelRendered := muted.Render(labelStr)

	bar := renderBannerBarCounts(b, barWidth, muted)
	sibNote := ""
	if sibRestricted {
		sibNote = "  " + muted.Render("(other projects' usage hidden)")
	}
	return connectorRendered + labelRendered + bar + sibNote
}

// renderBannerSibFull renders the full-form sibling-consume row.
func (m QuotaBannerModel) renderBannerSibFull(tree data.TreeBuckets, barWidth int, muted lipgloss.Style) string {
	var sibAllocated int64
	for _, s := range tree.Siblings {
		sibAllocated += s.Allocated
	}
	sibLimit := tree.Parent.Limit
	pct := float64(0)
	if sibLimit > 0 {
		pct = float64(sibAllocated) / float64(sibLimit)
	}
	filled := min(barWidth, max(0, int(float64(barWidth)*pct)))
	empty := barWidth - filled
	barStr := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	sibKind := ""
	if len(tree.Siblings) > 0 {
		sibKind = pluralizeKind(tree.Siblings[0].ConsumerKind, len(tree.Siblings))
	}
	countStr := fmt.Sprintf(" %3d of %-3d by %d %s", sibAllocated, sibLimit, len(tree.Siblings), sibKind)
	return muted.Render("│    sibling-consume  ") +
		muted.Render("["+barStr+"]") +
		muted.Render(countStr)
}

// renderBannerTreeRowCompact renders one compact-form tree row for the banner.
func (m QuotaBannerModel) renderBannerTreeRowCompact(
	connector string, b data.AllowanceBucket,
	_ bool,
	muted lipgloss.Style,
) string {
	connectorRendered := muted.Render(connector)
	shortKind := strings.ToLower(b.ConsumerKind)
	kindRendered := muted.Render(shortKind)
	var counts string
	if b.Limit == 0 {
		counts = muted.Render(fmt.Sprintf(" %d / ∞ (—)", b.Allocated))
	} else {
		pct := float64(b.Allocated) / float64(b.Limit)
		pctInt := int(pct * 100)
		barStyle, suffixStyle, suffix := QuotaBarStyling(pctInt)
		counts = barStyle.Render(fmt.Sprintf(" %d / %d (%d%%)", b.Allocated, b.Limit, pctInt))
		if suffix != "" {
			counts += suffixStyle.Render(suffix)
		}
	}
	return connectorRendered + kindRendered + counts
}

// renderBannerSibCompact renders the compact sibling-consume row.
func (m QuotaBannerModel) renderBannerSibCompact(tree data.TreeBuckets, muted lipgloss.Style) string {
	var sibAllocated int64
	for _, s := range tree.Siblings {
		sibAllocated += s.Allocated
	}
	sibLimit := tree.Parent.Limit
	var counts string
	if sibLimit == 0 {
		counts = fmt.Sprintf(" %d / ∞ (—)", sibAllocated)
	} else {
		pct := float64(sibAllocated) / float64(sibLimit)
		pctInt := int(pct * 100)
		counts = fmt.Sprintf(" %d / %d (%d%%)", sibAllocated, sibLimit, pctInt)
	}
	return muted.Render("│    siblings") + muted.Render(counts) +
		muted.Render(fmt.Sprintf("  (%d %s)", len(tree.Siblings), pluralizeKind(tree.Siblings[0].ConsumerKind, len(tree.Siblings))))
}

// renderBannerBarCounts renders the bar+counts portion for full-form banner rows.
func renderBannerBarCounts(b data.AllowanceBucket, barWidth int, muted lipgloss.Style) string {
	if barWidth < 1 {
		barWidth = 1
	}
	if b.Limit == 0 {
		bar := muted.Render(strings.Repeat("░", barWidth))
		counts := muted.Render(fmt.Sprintf(" %3d / %-3s (%-3s)", b.Allocated, "∞", "—"))
		return "[" + bar + "]" + counts + strings.Repeat(" ", 8)
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
		suffixPadded := suffix + strings.Repeat(" ", max(0, 8-len([]rune(suffix))))
		result += suffixStyle.Render(suffixPadded)
	} else {
		result += strings.Repeat(" ", 8)
	}
	return result
}
