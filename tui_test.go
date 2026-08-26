package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestChatLinesSeparatesMessages(t *testing.T) {
	tui := TUI{
		Width: 5,
		Model: Model{Chat: []Message{
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

func TestPlaceBottomClipsOldestLines(t *testing.T) {
	destination := make([]string, 5)
	placeBottom(destination, []string{"old", "middle", "new"}, 2)

	want := []string{"middle", "new", "", "", ""}
	if !reflect.DeepEqual(destination, want) {
		t.Fatalf("placeBottom() = %#v, want %#v", destination, want)
	}
}

func TestAppendAIKeepsToolCallsSeparate(t *testing.T) {
	tui := TUI{}
	tui.appendAI("read_file", MsgTool)
	tui.appendAI("ls", MsgTool)

	if len(tui.Model.Chat) != 2 {
		t.Fatalf("appendAI() stored %d tool messages, want 2", len(tui.Model.Chat))
	}
}

func TestAppendAICoalescesStreamedResponse(t *testing.T) {
	tui := TUI{}
	tui.appendAI("hel", MsgResponse)
	tui.appendAI("lo", MsgResponse)

	if len(tui.Model.Chat) != 1 || tui.Model.Chat[0].Data != "hello" {
		t.Fatalf("appendAI() chat = %#v, want one combined response", tui.Model.Chat)
	}
}

func TestStatusBarSpinnerTracksGeneration(t *testing.T) {
	tui := TUI{Width: 80, ModelName: "test-model"}

	if strings.Contains(tui.statusBar(), spinnerFrames[0]) {
		t.Fatal("statusBar() shows spinner while idle")
	}

	tui.Model.PendingRequests = 1
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

func TestViewUsesRawTerminalLineEndings(t *testing.T) {
	tui := TUI{
		Width:  20,
		Height: 4,
		Model:  Model{InputY: 1},
	}

	view := tui.view()
	if strings.Count(view, "\r\n") != tui.Height-1 {
		t.Fatalf("view() has %d CRLF line endings, want %d", strings.Count(view, "\r\n"), tui.Height-1)
	}
}
