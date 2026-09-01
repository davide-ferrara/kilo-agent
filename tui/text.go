package tui

import (
	"strings"
	"unicode"
)

func runeWidth(r rune) int {
	if r == 0 || r == '\n' || r == '\r' || unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Me, r) {
		return 0
	}
	if r < 32 || (r >= 0x7f && r < 0xa0) {
		return 0
	}
	if r >= 0x1100 && (r <= 0x115f || r == 0x2329 || r == 0x232a ||
		(r >= 0x2e80 && r <= 0xa4cf) || (r >= 0xac00 && r <= 0xd7a3) ||
		(r >= 0xf900 && r <= 0xfaff) || (r >= 0xfe10 && r <= 0xfe6f) ||
		(r >= 0xff00 && r <= 0xff60) || (r >= 0xffe0 && r <= 0xffe6) ||
		(r >= 0x1f300 && r <= 0x1faff) || (r >= 0x20000 && r <= 0x3fffd)) {
		return 2
	}
	return 1
}

// runeCellWidths returns terminal-cell widths with the context needed for
// emoji variation sequences. Some symbols (for example U+2708 AIRPLANE) are
// one cell as text, but terminals render "\u2708\ufe0f" as a two-cell emoji. A
// rune-at-a-time width calculation misses that extra cell and pads a styled
// line past the terminal edge.
func runeCellWidths(runes []rune) []int {
	widths := make([]int, len(runes))
	for index, r := range runes {
		widths[index] = runeWidth(r)
		if r == '\ufe0f' && index > 0 && widths[index-1] == 1 {
			widths[index-1] = 2
		}
	}
	return widths
}

func cellWidth(text string) int {
	width := 0
	for _, cells := range runeCellWidths([]rune(text)) {
		width += cells
	}
	return width
}

func visibleRuneCount(text string) int {
	var visible strings.Builder
	inEscape := false
	for _, r := range text {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if (r >= '@' && r <= '~') && r != '[' {
				inEscape = false
			}
			continue
		}
		visible.WriteRune(r)
	}
	return cellWidth(visible.String())
}

func hardWrap(text string, width int) []string {
	if width < 1 {
		return []string{""}
	}
	var lines []string
	var line strings.Builder
	column := 0
	runes := []rune(text)
	widths := runeCellWidths(runes)
	for index, r := range runes {
		if r == '\n' {
			lines = append(lines, line.String())
			line.Reset()
			column = 0
			continue
		}
		cells := widths[index]
		if column > 0 && column+cells > width {
			lines = append(lines, line.String())
			line.Reset()
			column = 0
		}
		line.WriteRune(r)
		column += cells
	}
	lines = append(lines, line.String())
	return lines
}

func wordWrap(text string, width int) []string {
	if width < 1 {
		return []string{""}
	}
	paragraphs := strings.Split(text, "\n")
	lines := make([]string, 0, len(paragraphs))
	for _, paragraph := range paragraphs {
		if paragraph == "" {
			lines = append(lines, "")
			continue
		}
		var line strings.Builder
		for _, word := range strings.Fields(paragraph) {
			wordWidth := cellWidth(word)
			if line.Len() > 0 && cellWidth(line.String())+1+wordWidth <= width {
				line.WriteByte(' ')
				line.WriteString(word)
				continue
			}
			if line.Len() > 0 {
				lines = append(lines, line.String())
				line.Reset()
			}
			wrapped := hardWrap(word, width)
			lines = append(lines, wrapped[:len(wrapped)-1]...)
			line.WriteString(wrapped[len(wrapped)-1])
		}
		lines = append(lines, line.String())
	}
	return lines
}

func truncateCells(text string, width int) string {
	if width <= 0 {
		return ""
	}
	if cellWidth(text) <= width {
		return text
	}
	if width == 1 {
		return "…"
	}
	var b strings.Builder
	used := 0
	runes := []rune(text)
	widths := runeCellWidths(runes)
	for index, r := range runes {
		cells := widths[index]
		if used+cells > width-1 {
			break
		}
		b.WriteRune(r)
		used += cells
	}
	return b.String() + "…"
}

func styledLine(text, foreground, background string, width int) string {
	padding := width - cellWidth(text)
	if padding < 0 {
		padding = 0
	}
	return background + foreground + text + strings.Repeat(" ", padding) + ansiReset
}
