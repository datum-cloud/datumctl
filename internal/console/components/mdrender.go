package components

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"charm.land/lipgloss/v2"
	"go.datum.net/datumctl/internal/console/styles"
)

var (
	reBold        = regexp.MustCompile(`\*\*(.+?)\*\*`)
	reItalic      = regexp.MustCompile(`\*(.+?)\*`)
	reInlineCode  = regexp.MustCompile("`([^`]+)`")
	reStrikeThru  = regexp.MustCompile(`~~(.+?)~~`)
	reHeaderLine  = regexp.MustCompile(`^#{1,6}\s+(.+)$`)
	reTableSep    = regexp.MustCompile(`^\|[-| :]+\|$`)
	reTableRow    = regexp.MustCompile(`^\|(.+)\|$`)
	reNumList     = regexp.MustCompile(`^(\d+)\.\s+(.+)$`)
)

// renderMarkdown converts a subset of markdown to lipgloss-styled terminal
// output capped to maxWidth columns.  It handles bold, italic, inline code,
// fenced code blocks, headers, pipe tables, blockquotes, bullet lists, and
// hard word-wrap on long lines.
func renderMarkdown(text string, maxWidth int) string {
	if maxWidth < 8 {
		maxWidth = 8
	}

	defStyle := lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Primary)
	boldStyle := lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Primary).Bold(true)
	codeStyle := lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Accent)
	mutedStyle := lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Muted)
	headerStyle := lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Accent).Bold(true)
	warnStyle := lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Warning)

	lines := strings.Split(text, "\n")
	var out strings.Builder

	inFence := false
	var tableRows [][]string // buffered table rows
	var paraLines []string  // consecutive non-special lines forming one paragraph

	flushPara := func() {
		if len(paraLines) == 0 {
			return
		}
		// Join soft-break lines (single \n) into one paragraph, then word-wrap.
		joined := strings.Join(paraLines, " ")
		paraLines = nil
		plain := stripInlineMarkdown(joined)
		if utf8.RuneCountInString(plain) > maxWidth {
			wrappedLines := wordWrapPreserveStyle(joined, maxWidth, defStyle, boldStyle, codeStyle, warnStyle)
			for _, wl := range wrappedLines {
				out.WriteString(wl + "\n")
			}
		} else {
			out.WriteString(renderInlineFull(joined, defStyle, boldStyle, codeStyle, warnStyle) + "\n")
		}
	}

	flushTable := func() {
		if len(tableRows) == 0 {
			return
		}
		// compute column widths
		cols := 0
		for _, row := range tableRows {
			if len(row) > cols {
				cols = len(row)
			}
		}
		colW := make([]int, cols)
		for _, row := range tableRows {
			for i, cell := range row {
				plain := stripInlineMarkdown(cell)
				if utf8.RuneCountInString(plain) > colW[i] {
					colW[i] = utf8.RuneCountInString(plain)
				}
			}
		}
		// cap total width
		totalW := cols + 1 // separators
		for _, w := range colW {
			totalW += w + 2
		}
		if totalW > maxWidth {
			excess := totalW - maxWidth
			const minColWidth = 4
			for excess > 0 {
				widest, wi := 0, 0
				for i, w := range colW {
					if w > minColWidth && w > widest {
						widest, wi = w, i
					}
				}
				if widest == 0 {
					break
				}
				colW[wi]--
				excess--
			}
		}

		thStyle := lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Accent).Bold(true)
		for rowIdx, row := range tableRows {
			sepStyle := mutedStyle
			if rowIdx == 0 {
				sepStyle = thStyle
			}
			var sb strings.Builder
			sb.WriteString(sepStyle.Render("│"))
			for i := 0; i < cols; i++ {
				cell := ""
				if i < len(row) {
					cell = strings.TrimSpace(row[i])
				}
				plain := stripInlineMarkdown(cell)
				pad := colW[i] - utf8.RuneCountInString(plain)
				if pad < 0 {
					runes := []rune(plain)
					if colW[i] > 1 {
						plain = string(runes[:colW[i]-1]) + "…"
					} else {
						plain = "…"
					}
					pad = 0
				}
				var styled string
				if rowIdx == 0 {
					styled = thStyle.Render(plain)
				} else {
					styled = renderInline(cell, colW[i], defStyle, boldStyle, codeStyle)
				}
				sb.WriteString(" " + styled + strings.Repeat(" ", pad) + " ")
				sb.WriteString(sepStyle.Render("│"))
			}
			out.WriteString(sb.String() + "\n")
		}
		tableRows = nil
	}

	for _, line := range lines {
		// fenced code block toggle
		if strings.HasPrefix(line, "```") {
			flushPara()
			flushTable()
			inFence = !inFence
			if inFence {
				lang := strings.TrimPrefix(line, "```")
				if lang != "" {
					out.WriteString(mutedStyle.Render("  "+lang) + "\n")
				}
			}
			continue
		}
		if inFence {
			out.WriteString(codeStyle.Render("  "+line) + "\n")
			continue
		}

		// table separator row — skip rendering but don't flush yet
		if reTableSep.MatchString(strings.TrimSpace(line)) {
			flushPara()
			continue
		}

		// table data row — buffer
		if reTableRow.MatchString(strings.TrimSpace(line)) {
			flushPara()
			inner := strings.TrimSpace(line)
			inner = strings.TrimPrefix(inner, "|")
			inner = strings.TrimSuffix(inner, "|")
			cells := strings.Split(inner, "|")
			// Drop empty trailing cells produced by the final pipe in `| a | b |`.
			for len(cells) > 0 && strings.TrimSpace(cells[len(cells)-1]) == "" {
				cells = cells[:len(cells)-1]
			}
			tableRows = append(tableRows, cells)
			continue
		}

		// non-table line — flush any buffered table
		flushTable()

		trimmed := strings.TrimSpace(line)

		// empty line — paragraph break
		if trimmed == "" {
			flushPara()
			out.WriteString("\n")
			continue
		}

		// header
		if m := reHeaderLine.FindStringSubmatch(trimmed); m != nil {
			flushPara()
			out.WriteString(headerStyle.Render(m[1]) + "\n")
			continue
		}

		// blockquote
		if strings.HasPrefix(trimmed, ">") {
			flushPara()
			content := strings.TrimSpace(strings.TrimPrefix(trimmed, ">"))
			content = renderInlineFull(content, defStyle, boldStyle, codeStyle, warnStyle)
			out.WriteString(mutedStyle.Render("▎") + " " + content + "\n")
			continue
		}

		// bullet list items (-, *, +)
		if len(trimmed) > 0 && (trimmed[0] == '-' || trimmed[0] == '*' || trimmed[0] == '+') {
			flushPara()
			rest := strings.TrimSpace(trimmed[1:])
			prefix := mutedStyle.Render("•") + " "
			plain := stripInlineMarkdown(rest)
			if utf8.RuneCountInString(plain) > maxWidth-4 {
				wrappedLines := wordWrapPreserveStyle(rest, maxWidth-4, defStyle, boldStyle, codeStyle, warnStyle)
				for i, wl := range wrappedLines {
					if i == 0 {
						out.WriteString(prefix + wl + "\n")
					} else {
						out.WriteString("  " + wl + "\n")
					}
				}
			} else {
				styled := renderInlineFull(rest, defStyle, boldStyle, codeStyle, warnStyle)
				out.WriteString(prefix + styled + "\n")
			}
			continue
		}

		// numbered list
		if m := reNumList.FindStringSubmatch(trimmed); m != nil {
			flushPara()
			num := m[1]
			rest := m[2]
			styled := renderInlineFull(rest, defStyle, boldStyle, codeStyle, warnStyle)
			prefix := mutedStyle.Render(num+".") + " "
			out.WriteString(prefix + styled + "\n")
			continue
		}

		// horizontal rule
		if trimmed == "---" || trimmed == "***" || trimmed == "___" {
			flushPara()
			out.WriteString(mutedStyle.Render(strings.Repeat("─", maxWidth)) + "\n")
			continue
		}

		// normal paragraph line — buffer for soft-break joining
		paraLines = append(paraLines, trimmed)
	}

	flushPara()
	flushTable()
	return strings.TrimRight(out.String(), "\n")
}

// renderInlineFull renders inline markdown (bold, italic, inline code) with lipgloss styles.
func renderInlineFull(s string, def, bold, code, warn lipgloss.Style) string {
	// process bold first (** ... **)
	s = reBold.ReplaceAllStringFunc(s, func(m string) string {
		inner := reBold.FindStringSubmatch(m)[1]
		return bold.Render(inner)
	})
	// inline code
	s = reInlineCode.ReplaceAllStringFunc(s, func(m string) string {
		inner := reInlineCode.FindStringSubmatch(m)[1]
		return code.Render(inner)
	})
	// strikethrough → muted
	s = reStrikeThru.ReplaceAllStringFunc(s, func(m string) string {
		inner := reStrikeThru.FindStringSubmatch(m)[1]
		return warn.Render(inner)
	})
	// italic (single *) — apply after bold so ** is already consumed
	s = reItalic.ReplaceAllStringFunc(s, func(m string) string {
		inner := reItalic.FindStringSubmatch(m)[1]
		return def.Italic(true).Render(inner)
	})
	return def.Render(s)
}

// renderInline renders inline markdown for use within table cells (no outer wrap).
func renderInline(s string, _ int, def, bold, code lipgloss.Style) string {
	s = reBold.ReplaceAllStringFunc(s, func(m string) string {
		return bold.Render(reBold.FindStringSubmatch(m)[1])
	})
	s = reInlineCode.ReplaceAllStringFunc(s, func(m string) string {
		return code.Render(reInlineCode.FindStringSubmatch(m)[1])
	})
	return def.Render(s)
}

// stripInlineMarkdown removes markdown syntax to get the plain-text length.
func stripInlineMarkdown(s string) string {
	s = reBold.ReplaceAllString(s, "$1")
	s = reItalic.ReplaceAllString(s, "$1")
	s = reInlineCode.ReplaceAllString(s, "$1")
	s = reStrikeThru.ReplaceAllString(s, "$1")
	return s
}

// wordWrap wraps plain text to maxWidth runes per line.
func wordWrap(s string, maxWidth int) []string {
	if maxWidth <= 0 {
		return []string{s}
	}
	words := strings.Fields(s)
	var lines []string
	var cur strings.Builder
	curLen := 0
	for _, w := range words {
		wlen := utf8.RuneCountInString(w)
		if curLen+wlen+1 > maxWidth && curLen > 0 {
			lines = append(lines, cur.String())
			cur.Reset()
			curLen = 0
		}
		if curLen > 0 {
			cur.WriteByte(' ')
			curLen++
		}
		cur.WriteString(w)
		curLen += wlen
	}
	if cur.Len() > 0 {
		lines = append(lines, cur.String())
	}
	return lines
}

// wordWrapPreserveStyle wraps a paragraph respecting maxWidth, re-rendering
// each wrapped line through renderInlineFull.
func wordWrapPreserveStyle(s string, maxWidth int, def, bold, code, warn lipgloss.Style) []string {
	plain := stripInlineMarkdown(s)
	wrapped := wordWrap(plain, maxWidth)
	// simple approach: just use the full rendered string on the first line,
	// split visually by rune count on subsequent lines
	result := make([]string, len(wrapped))
	for i, line := range wrapped {
		result[i] = renderInlineFull(line, def, bold, code, warn)
	}
	return result
}
