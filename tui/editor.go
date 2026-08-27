package tui

import "unicode"

type Editor struct {
	Text            []rune
	Cursor          int
	PreferredColumn int
	KillRing        []rune
	History         []string
	HistoryIndex    int
	HistoryDraft    []rune
}

func (e *Editor) String() string { return string(e.Text) }

func (e *Editor) Insert(text string) {
	e.Cursor = clamp(e.Cursor, 0, len(e.Text))
	runes := []rune(text)
	e.Text = append(e.Text, make([]rune, len(runes))...)
	copy(e.Text[e.Cursor+len(runes):], e.Text[e.Cursor:len(e.Text)-len(runes)])
	copy(e.Text[e.Cursor:], runes)
	e.Cursor += len(runes)
	e.PreferredColumn = -1
}

func (e *Editor) Backspace() {
	if e.Cursor == 0 {
		return
	}
	copy(e.Text[e.Cursor-1:], e.Text[e.Cursor:])
	e.Text = e.Text[:len(e.Text)-1]
	e.Cursor--
	e.PreferredColumn = -1
}

func (e *Editor) Delete() {
	if e.Cursor >= len(e.Text) {
		return
	}
	copy(e.Text[e.Cursor:], e.Text[e.Cursor+1:])
	e.Text = e.Text[:len(e.Text)-1]
	e.PreferredColumn = -1
}

func (e *Editor) Move(delta int) {
	e.Cursor = clamp(e.Cursor+delta, 0, len(e.Text))
	e.PreferredColumn = -1
}

func (e *Editor) MoveWord(direction int) {
	if direction < 0 {
		for e.Cursor > 0 && unicode.IsSpace(e.Text[e.Cursor-1]) {
			e.Cursor--
		}
		for e.Cursor > 0 && !unicode.IsSpace(e.Text[e.Cursor-1]) {
			e.Cursor--
		}
	} else {
		for e.Cursor < len(e.Text) && !unicode.IsSpace(e.Text[e.Cursor]) {
			e.Cursor++
		}
		for e.Cursor < len(e.Text) && unicode.IsSpace(e.Text[e.Cursor]) {
			e.Cursor++
		}
	}
	e.PreferredColumn = -1
}

func (e *Editor) MoveVertical(direction, width int) {
	if width < 1 {
		return
	}
	positions := cursorPositions(e.Text, width)
	position := positions[clamp(e.Cursor, 0, len(e.Text))]
	row, column := position.row, position.column
	if e.PreferredColumn >= 0 {
		column = e.PreferredColumn
	} else {
		e.PreferredColumn = column
	}
	targetRow := row + direction
	if targetRow < 0 || targetRow > positions[len(positions)-1].row {
		return
	}
	e.Cursor = cursorIndexAtColumn(positions, targetRow, column)
}

func (e *Editor) DeleteWord() {
	end := e.Cursor
	start := end
	for start > 0 && unicode.IsSpace(e.Text[start-1]) {
		start--
	}
	for start > 0 && !unicode.IsSpace(e.Text[start-1]) {
		start--
	}
	e.KillRing = append(e.KillRing[:0], e.Text[start:end]...)
	e.Text = append(e.Text[:start], e.Text[end:]...)
	e.Cursor = start
	e.PreferredColumn = -1
}

func (e *Editor) KillToStart() {
	start := lineStart(e.Text, e.Cursor)
	e.KillRing = append(e.KillRing[:0], e.Text[start:e.Cursor]...)
	e.Text = append(e.Text[:start], e.Text[e.Cursor:]...)
	e.Cursor = start
	e.PreferredColumn = -1
}

func (e *Editor) KillToEnd() {
	end := lineEnd(e.Text, e.Cursor)
	e.KillRing = append(e.KillRing[:0], e.Text[e.Cursor:end]...)
	e.Text = append(e.Text[:e.Cursor], e.Text[end:]...)
	e.PreferredColumn = -1
}

func (e *Editor) Clear() {
	e.Text = e.Text[:0]
	e.Cursor = 0
	e.HistoryIndex = len(e.History)
	e.HistoryDraft = nil
	e.PreferredColumn = -1
}

func (e *Editor) Commit() string {
	text := e.String()
	if text != "" && (len(e.History) == 0 || e.History[len(e.History)-1] != text) {
		e.History = append(e.History, text)
	}
	e.Clear()
	return text
}

func (e *Editor) Recall(direction int) {
	if len(e.History) == 0 {
		return
	}
	if e.HistoryIndex < 0 || e.HistoryIndex > len(e.History) {
		e.HistoryIndex = len(e.History)
	}
	if direction < 0 && e.HistoryIndex == len(e.History) {
		e.HistoryDraft = append(e.HistoryDraft[:0], e.Text...)
	}
	e.HistoryIndex = clamp(e.HistoryIndex+direction, 0, len(e.History))
	if e.HistoryIndex == len(e.History) {
		e.Text = append(e.Text[:0], e.HistoryDraft...)
	} else {
		e.Text = []rune(e.History[e.HistoryIndex])
	}
	e.Cursor = len(e.Text)
	e.PreferredColumn = -1
}

func cursorPosition(text []rune, cursor, width int) (row, column int) {
	positions := cursorPositions(text, width)
	position := positions[clamp(cursor, 0, len(text))]
	return position.row, position.column
}

type cursorPoint struct {
	row    int
	column int
}

func cursorPositions(text []rune, width int) []cursorPoint {
	width = max(width, 1)
	positions := make([]cursorPoint, len(text)+1)
	row, column := 0, 0
	wrapped := false
	for i, r := range text {
		if r == '\n' {
			if !wrapped {
				row++
			}
			column = 0
			wrapped = false
			positions[i+1] = cursorPoint{row: row, column: column}
			continue
		}
		cells := runeWidth(r)
		if column > 0 && column+cells > width {
			row++
			column = 0
		}
		column += cells
		wrapped = column >= width
		if wrapped {
			row++
			column = 0
		}
		positions[i+1] = cursorPoint{row: row, column: column}
	}
	return positions
}

func cursorIndexAtColumn(positions []cursorPoint, row, target int) int {
	bestIndex := -1
	bestColumn := -1
	for index, position := range positions {
		if position.row != row || position.column > target {
			continue
		}
		if position.column >= bestColumn {
			bestIndex = index
			bestColumn = position.column
		}
	}
	if bestIndex >= 0 {
		return bestIndex
	}
	for index, position := range positions {
		if position.row == row {
			return index
		}
	}
	return 0
}

func clamp(value, low, high int) int {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}
