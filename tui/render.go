package tui

import (
	"fmt"
	"io"
	"strings"
)

type Frame struct {
	Lines   []string
	CursorX int
	CursorY int
}

type Renderer struct {
	out      io.Writer
	previous []string
}

func NewRenderer(out io.Writer) *Renderer { return &Renderer{out: out} }

func (r *Renderer) Reset() { r.previous = nil }

// Draw repaints only changed terminal rows. This avoids full-screen flicker
// while streamed tokens arrive, but still clears correctly on the first frame.
func (r *Renderer) Draw(frame Frame) {
	var b strings.Builder
	b.WriteString(ansiCursorHide)
	if len(r.previous) == 0 || len(r.previous) != len(frame.Lines) {
		b.WriteString(ansiClearScreen + ansiCursorHome)
		for index, line := range frame.Lines {
			if index > 0 {
				b.WriteString("\r\n")
			}
			b.WriteString(line)
		}
	} else {
		for index, line := range frame.Lines {
			if line == r.previous[index] {
				continue
			}
			b.WriteString(ansiMoveCursor(1, index+1) + ansiClearLine + line)
		}
	}
	b.WriteString(ansiMoveCursor(frame.CursorX+1, frame.CursorY+1) + ansiCursorShow)
	_, _ = io.WriteString(r.out, b.String())
	r.previous = append(r.previous[:0], frame.Lines...)
}

func (t *TUI) frame() Frame {
	width := max(t.width, 1)
	height := max(t.height, 1)
	lines := make([]string, height)
	inputWidth := t.editorWidth()
	tallInput := t.inputRowsAtWidth(inputWidth)
	cursorRow, cursorColumn := cursorPosition(t.model.Editor.Text, t.model.Editor.Cursor, inputWidth)
	inputCapacity := max(0, height-1)
	inputStart := 0
	if len(tallInput) > inputCapacity {
		inputStart = clamp(cursorRow-inputCapacity+1, 0, len(tallInput)-inputCapacity)
	}
	input := tallInput[inputStart:min(inputStart+inputCapacity, len(tallInput))]
	inputTop := height - 1 - len(input)
	completionCapacity := max(0, min(5, inputTop-1))
	completion := t.completionLines(completionCapacity)
	completionTop := inputTop - len(completion)
	chatEnd := max(0, completionTop-1)

	if len(t.model.Chat) == 0 {
		placeCentered(lines[:chatEnd], strings.Split(t.bootText(), "\n"), width)
	} else {
		placeScrolled(lines[:chatEnd], t.chatLines(), t.model.ScrollOffset)
	}
	copy(lines[completionTop:inputTop], completion)
	copy(lines[inputTop:height-1], input)
	lines[height-1] = t.statusBar()

	cursorY := inputTop + cursorRow - inputStart
	return Frame{Lines: lines, CursorX: min(t.inputPrefixWidth()+cursorColumn, width-1), CursorY: clamp(cursorY, 0, height-1)}
}

func (t *TUI) inputRowsAtWidth(width int) []string {
	rows := hardWrap(t.model.Editor.String(), width)
	// A cursor immediately after a completely filled row belongs at column zero
	// of the following row. Reserve that row before another key is inserted so
	// the terminal never performs its own implicit wrap into unrelated content.
	cursorRow, _ := cursorPosition(t.model.Editor.Text, t.model.Editor.Cursor, width)
	for len(rows) <= cursorRow {
		rows = append(rows, "")
	}
	result := make([]string, len(rows))
	for index, row := range rows {
		prompt := themeSurfaceAlt + strings.Repeat(" ", t.inputPrefixWidth())
		if index == 0 && t.width >= 3 {
			prompt = themeSurfaceAlt + themeCyan + "❯ "
			if t.width >= 7 {
				prompt += themeGrayDark + "│ "
			}
		}
		padding := max(0, width-cellWidth(row))
		suffix := strings.Repeat(" ", max(0, t.width-t.inputPrefixWidth()-width))
		result[index] = prompt + themeWhite + row + strings.Repeat(" ", padding) + suffix +
			ansiClearToEnd + ansiReset
	}
	return result
}

func (t *TUI) inputPrefixWidth() int {
	if t.width >= 7 {
		return 4
	}
	if t.width >= 3 {
		return 2
	}
	return 0
}

func (t *TUI) completionLines(limit int) []string {
	items := t.model.Completion.Items
	if len(items) == 0 || limit == 0 {
		return nil
	}
	selected := clamp(t.model.Completion.Selected, 0, len(items)-1)
	start := clamp(selected-limit+1, 0, max(0, len(items)-limit))
	if len(items)-start > limit {
		items = items[start : start+limit]
	} else {
		items = items[start:]
	}
	lines := make([]string, len(items))
	for index, item := range items {
		marker, background, foreground := "  ", themeSurface, themeGray
		if start+index == selected {
			marker, background, foreground = "› ", themeCyanDark, themeWhiteBold
		}
		available := max(t.width-cellWidth(marker), 0)
		description := truncateCells("  "+item.Description, max(0, available/2))
		valueWidth := max(0, available-cellWidth(description))
		text := marker + truncateCells(item.Value, valueWidth) + description
		lines[index] = styledLine(text, foreground, background, t.width)
	}
	return lines
}

func (t *TUI) chatLines() []string {
	var lines []string
	for _, message := range t.model.Chat {
		rendered := renderMessage(message, max(t.width, 1))
		if len(rendered) == 0 {
			continue
		}
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, rendered...)
	}
	return lines
}

func renderMessage(message Message, width int) []string {
	foreground, background := messageColors(message.MsgType)
	text := fmt.Sprint(message.Data)
	sources := strings.Split(text, "\n")
	for len(sources) > 0 && strings.TrimSpace(sources[0]) == "" {
		sources = sources[1:]
	}
	for len(sources) > 0 && strings.TrimSpace(sources[len(sources)-1]) == "" {
		sources = sources[:len(sources)-1]
	}
	var rendered []string
	inFence := false
	inDiff := false
	for _, source := range sources {
		trimmed := strings.TrimSpace(source)
		openingFence := strings.HasPrefix(trimmed, "```") && !inFence
		closingFence := strings.HasPrefix(trimmed, "```") && inFence
		if openingFence {
			inFence = true
			inDiff = strings.HasPrefix(trimmed, "```diff")
		}
		wrapped := wordWrap(source, width)
		if inFence {
			wrapped = hardWrap(source, width)
		}
		for _, line := range wrapped {
			fg, bg := foreground, background
			if inDiff || strings.HasPrefix(trimmed, "```diff") {
				switch {
				case strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++"):
					fg, bg = themeGreen, themeGreenDark
				case strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---"):
					fg, bg = themeRed, themeRedDark
				case strings.HasPrefix(line, "@@"):
					fg, bg = themeCyanBright, themeCyanDark
				default:
					fg, bg = themeGray, themeSurface
				}
			}
			rendered = append(rendered, styledLine(line, fg, bg, width))
		}
		if closingFence {
			inFence = false
			inDiff = false
		}
	}
	return rendered
}

func messageColors(kind MsgType) (string, string) {
	switch kind {
	case MsgUser:
		return themeWhiteBold, themeCyanDark
	case MsgThinking:
		return themeGray, ""
	case MsgTool:
		return themeGreen, themeGreenDark
	case MsgError:
		return themeRed, themeRedDark
	case MsgNotice:
		return themeCyanBright, themeSurfaceAlt
	default:
		return themeWhite, ""
	}
}

func (t *TUI) statusBar() string {
	if t.width <= 0 {
		return ""
	}
	model := t.modelName
	if model == "" {
		model = "model"
	}
	state := "●"
	stateColor := themeGreen
	if t.model.PendingRequests > 0 {
		state = spinnerFrames[t.model.SpinnerFrame%len(spinnerFrames)]
		stateColor = themeCyan
	}
	u := t.model.Usage
	metrics := fmt.Sprintf("ctx %.0f%% · %d tok · %.1f tk/s", contextPercent(u), u.PromptTokens+u.OutputTokens, u.TokensPerSec)
	metricsWidth := cellWidth(metrics) + 1
	showMetrics := t.width >= metricsWidth+6
	modelRoom := max(0, t.width-3)
	if showMetrics {
		modelRoom = max(1, t.width-metricsWidth-4)
	}
	model = truncateCells(model, modelRoom)
	leftPlainWidth := 3 + cellWidth(model)
	if t.width < 3 {
		prefix := truncateCells(" "+state, t.width)
		return themeSurfaceLift + stateColor + prefix + strings.Repeat(" ", max(0, t.width-cellWidth(prefix))) + ansiClearToEnd + ansiReset
	}
	rightPlain := ""
	if showMetrics {
		rightPlain = metrics + " "
	}
	available := max(0, t.width-leftPlainWidth-cellWidth(rightPlain))
	// Reapply the background after the reset embedded in the left segment. The
	// final EL (Erase in Line) uses the active background for every remaining
	// terminal cell, including terminals with unusual width accounting.
	return themeSurfaceLift + " " + stateColor + state + " " + themeWhite + model +
		themeSurfaceLift + strings.Repeat(" ", available) + themeGray + rightPlain + ansiClearToEnd + ansiReset
}

func (t *TUI) bootText() string {
	width := max(t.width, 1)
	if width < 24 {
		return themeCyanBright + ansiBold + "KILO" + ansiReset + "\n" +
			themeGray + "@ files · / help" + ansiReset
	}
	inner := min(52, width-2)
	model := t.modelName
	if model == "" {
		model = "local model"
	}
	rows := []string{themeCyan + "┏" + strings.Repeat("━", inner) + "┓" + ansiReset}
	if inner >= 44 {
		for _, logo := range []string{
			"██  ██   ██   ██      ███████",
			"██ ██    ██   ██      ██   ██",
			"████     ██   ██      ██   ██",
			"██ ██    ██   ██      ██   ██",
			"██  ██   ██   █████   ███████",
		} {
			rows = append(rows, bootBoxRow(inner, logo, themeWhiteBold))
		}
	} else {
		rows = append(rows, bootBoxRow(inner, "K I L O", themeWhiteBold))
	}
	rows = append(rows,
		bootBoxRow(inner, "", themeWhite),
		bootBoxRow(inner, "● kilobyte agent · terminal workspace", themeWhite),
		bootBoxRow(inner, "model: "+model, themeGray),
		bootBoxRow(inner, "", themeWhite),
		bootBoxRow(inner, "@ attach a file   / browse commands", themeCyanBright),
		bootBoxRow(inner, "arrows edit · PgUp scroll · drag selects", themeGray),
		bootBoxRow(inner, "Enter sends · Ctrl+C clears or exits", themeGray),
		themeCyan+"┗"+strings.Repeat("━", inner)+"┛"+ansiReset,
	)
	return strings.Join(rows, "\n")
}

func bootBoxRow(inner int, text, color string) string {
	text = truncateCells(text, inner)
	left := max(0, (inner-cellWidth(text))/2)
	right := max(0, inner-cellWidth(text)-left)
	return themeCyan + "┃" + color + strings.Repeat(" ", left) + text +
		strings.Repeat(" ", right) + themeCyan + "┃" + ansiReset
}

func placeScrolled(destination, content []string, offset int) {
	maxOffset := max(0, len(content)-len(destination))
	end := len(content) - clamp(offset, 0, maxOffset)
	start := max(0, end-len(destination))
	visible := content[start:end]
	copy(destination[len(destination)-len(visible):], visible)
}

func placeCentered(destination, content []string, width int) {
	start := max(0, (len(destination)-len(content))/2)
	for index, line := range content {
		if start+index >= len(destination) {
			break
		}
		left := max(0, (width-visibleRuneCount(line))/2)
		destination[start+index] = strings.Repeat(" ", left) + line
	}
}

func ansiMoveCursor(x, y int) string { return fmt.Sprintf("\x1b[%d;%dH", y, x) }
