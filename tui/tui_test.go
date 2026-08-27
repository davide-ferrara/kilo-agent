package tui

import (
	"bytes"
	"strings"
	"testing"
)

func TestRendererOnlyRepaintsChangedLines(t *testing.T) {
	var output bytes.Buffer
	renderer := NewRenderer(&output)
	renderer.Draw(Frame{Lines: []string{"one", "two"}, CursorX: 0, CursorY: 0})
	output.Reset()
	renderer.Draw(Frame{Lines: []string{"one", "changed"}, CursorX: 0, CursorY: 1})

	draw := output.String()
	if strings.Contains(draw, ansiClearScreen) {
		t.Fatal("incremental draw cleared the whole screen")
	}
	if strings.Contains(draw, ansiMoveCursor(1, 1)+ansiClearLine) {
		t.Fatal("incremental draw repainted an unchanged line")
	}
	if !strings.Contains(draw, ansiMoveCursor(1, 2)+ansiClearLine+"changed") {
		t.Fatal("incremental draw did not repaint the changed line")
	}
}

func TestChatLinesSeparatesMessages(t *testing.T) {
	tui := TUI{
		width: 5,
		model: Model{Chat: []Message{
			{MsgType: MsgUser, Data: "hello"},
			{MsgType: MsgThinking, Data: "hmm"},
			{MsgType: MsgResponse, Data: "one\ntwo"},
		}},
	}

	lines := tui.chatLines()
	if len(lines) != 6 {
		t.Fatalf("chatLines() returned %d lines, want 6", len(lines))
	}
	if lines[1] != "" || lines[3] != "" {
		t.Fatalf("chatLines() separators at lines 1 and 3 = %q and %q", lines[1], lines[3])
	}
	for _, index := range []int{0, 2, 4, 5} {
		if lines[index] == "" {
			t.Errorf("chatLines() message line %d is blank", index)
		}
	}
}

func TestStatusBarSpinnerTracksGeneration(t *testing.T) {
	tui := TUI{width: 80, modelName: "test-model"}

	if strings.Contains(tui.statusBar(), spinnerFrames[0]) {
		t.Fatal("statusBar() shows spinner while idle")
	}

	tui.model.PendingRequests = 1
	if !strings.Contains(tui.statusBar(), themeCyan+spinnerFrames[0]+" "+themeWhite+"test-model") {
		t.Fatal("statusBar() does not put the themed spinner before the model name")
	}

	tui.Update(Message{MsgType: MsgSpinnerTick})
	if !strings.Contains(tui.statusBar(), spinnerFrames[1]) {
		t.Fatal("statusBar() did not advance the spinner")
	}

	tui.Update(Message{MsgType: MsgGenerationDone})
	if strings.Contains(tui.statusBar(), spinnerFrames[1]) {
		t.Fatal("statusBar() shows spinner after generation completes")
	}
}

func TestStatusBarPaintsBackgroundThroughEndOfLine(t *testing.T) {
	tui := TUI{width: 80, modelName: "test-model", model: initialModel(nil)}
	bar := tui.statusBar()
	if visibleRuneCount(bar) != tui.width {
		t.Fatalf("status width = %d, want %d", visibleRuneCount(bar), tui.width)
	}
	if !strings.HasPrefix(bar, themeSurfaceLift) {
		t.Fatal("status bar does not start with its background")
	}
	if !strings.HasSuffix(bar, ansiClearToEnd+ansiReset) {
		t.Fatal("status bar does not erase the remaining line with its background")
	}
}

func TestInputBarPaintsBackgroundThroughEndOfLine(t *testing.T) {
	tui := TUI{width: 20, model: initialModel(nil)}
	rows := tui.inputRowsAtWidth(14)
	if len(rows) != 1 {
		t.Fatalf("input rows = %d, want 1", len(rows))
	}
	if visibleRuneCount(rows[0]) != tui.width {
		t.Fatalf("input width = %d, want %d", visibleRuneCount(rows[0]), tui.width)
	}
	if !strings.HasPrefix(rows[0], themeSurfaceAlt) {
		t.Fatal("input row does not start with its background")
	}
	if strings.Contains(rows[0], ansiReset+themeWhite) {
		t.Fatal("input row resets the background before its content")
	}
	if !strings.HasSuffix(rows[0], ansiClearToEnd+ansiReset) {
		t.Fatal("input row does not paint through the terminal edge")
	}
}

func TestInputAllocatesRowAtExactWrapBoundary(t *testing.T) {
	tui := TUI{width: 12, height: 8, model: initialModel(nil)}
	tui.model.Editor.Insert("123456") // Input width is terminal width minus six.
	rows := tui.inputRowsAtWidth(6)
	if len(rows) != 2 || visibleRuneCount(rows[1]) != tui.width {
		t.Fatalf("exact-boundary rows = %#v, want a filled row and an empty cursor row", rows)
	}
	frame := tui.frame()
	if frame.CursorX != 4 {
		t.Fatalf("wrapped cursor X = %d, want 4", frame.CursorX)
	}
	if frame.CursorY != tui.height-2 {
		t.Fatalf("wrapped cursor Y = %d, want %d", frame.CursorY, tui.height-2)
	}
}

func TestFrameClipsTallInputAroundCursor(t *testing.T) {
	tui := TUI{width: 12, height: 4, model: initialModel(nil)}
	tui.model.Editor.Insert("one\ntwo\nthree\nfour")

	frame := tui.frame()
	if len(frame.Lines) != tui.height {
		t.Fatalf("frame has %d rows, want %d", len(frame.Lines), tui.height)
	}
	if frame.CursorY < 0 || frame.CursorY >= tui.height-1 {
		t.Fatalf("cursor Y = %d, want it in the visible input viewport", frame.CursorY)
	}
	if !strings.Contains(frame.Lines[frame.CursorY], "four") {
		t.Fatalf("cursor row does not show the active input line: %q", frame.Lines[frame.CursorY])
	}
}

func TestCompletionWindowKeepsSelectionVisible(t *testing.T) {
	tui := TUI{width: 40, model: initialModel(nil)}
	for index := 0; index < 8; index++ {
		tui.model.Completion.Items = append(tui.model.Completion.Items, CompletionItem{Value: string(rune('a' + index))})
	}
	tui.model.Completion.Selected = 7
	lines := tui.completionLines(3)
	if len(lines) != 3 || !strings.Contains(lines[2], "h") || !strings.Contains(lines[2], "›") {
		t.Fatalf("completion window does not contain selected item: %#v", lines)
	}
}

func TestStatusBarNeverExceedsNarrowTerminal(t *testing.T) {
	for width := 1; width <= 30; width++ {
		tui := TUI{width: width, modelName: "a-very-long-model-name", model: initialModel(nil)}
		if got := visibleRuneCount(tui.statusBar()); got != width {
			t.Fatalf("status width at terminal width %d = %d", width, got)
		}
	}
}
