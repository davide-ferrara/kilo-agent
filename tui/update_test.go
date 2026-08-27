package tui

import (
	"strings"
	"testing"
)

func TestFileCompletionAndPromptEffect(t *testing.T) {
	model := initialModel([]string{"README.md", "docs/user guide.md"})
	for _, r := range "@read" {
		model, _ = UpdateModel(model, Message{MsgType: MsgKey, Data: KeyEvent{Key: KeyRune, Rune: r}})
	}
	if len(model.Completion.Items) != 1 || model.Completion.Items[0].Value != "@README.md" {
		t.Fatalf("completion = %#v", model.Completion.Items)
	}
	model, effects := UpdateModel(model, Message{MsgType: MsgKey, Data: KeyEvent{Key: KeyEnter}})
	if model.Editor.String() != "@README.md " || len(effects) != 0 {
		t.Fatalf("accepted file = %q, effects %#v", model.Editor.String(), effects)
	}
	model.Editor.Insert("summarize")
	model, effects = UpdateModel(model, Message{MsgType: MsgKey, Data: KeyEvent{Key: KeyEnter}})
	if len(effects) != 1 || effects[0].Type != EffectSendPrompt || effects[0].Data != "@README.md summarize" {
		t.Fatalf("submit effects = %#v", effects)
	}
}

func TestHelpCommandRendersLocally(t *testing.T) {
	model := initialModel(nil)
	model.Editor.Insert("/help")
	model, effects := UpdateModel(model, Message{MsgType: MsgKey, Data: KeyEvent{Key: KeyEnter}})
	if len(effects) != 0 || len(model.Chat) != 1 {
		t.Fatalf("help effects/chat = %#v / %#v", effects, model.Chat)
	}
	if !strings.Contains(model.Chat[0].Data.(string), "Ctrl+A/E") {
		t.Fatal("help does not document readline bindings")
	}
}

func TestResponseAndInputWordWrapping(t *testing.T) {
	input := hardWrap("123456789", 4)
	if len(input) != 3 {
		t.Fatalf("input wraps to %d rows", len(input))
	}
	message := renderMessage(Message{MsgType: MsgResponse, Data: "one two three"}, 7)
	if len(message) != 2 {
		t.Fatalf("output wraps to %d rows", len(message))
	}
}

func TestDiffLinesReceiveSemanticColors(t *testing.T) {
	lines := renderMessage(Message{MsgType: MsgResponse, Data: "```diff\n-old\n+new\n```"}, 40)
	if !strings.Contains(lines[1], themeRedDark) || !strings.Contains(lines[2], themeGreenDark) {
		t.Fatalf("diff colors missing: %#v", lines)
	}
}

func TestAssistantAndThinkingLinesKeepTerminalBackground(t *testing.T) {
	for _, message := range []Message{
		{MsgType: MsgResponse, Data: "answer"},
		{MsgType: MsgThinking, Data: "thinking"},
	} {
		lines := renderMessage(message, 20)
		if strings.Contains(lines[0], themeSurface) ||
			strings.Contains(lines[0], themeSurfaceAlt) ||
			strings.Contains(lines[0], themeSurfaceLift) {
			t.Fatalf("message %v painted a surface background: %q", message.MsgType, lines[0])
		}
	}
	if !strings.Contains(renderMessage(Message{MsgType: MsgThinking, Data: "thinking"}, 20)[0], themeGray) {
		t.Fatal("thinking line does not use grey foreground")
	}
}

func TestMouseWheelScrollsConversation(t *testing.T) {
	model := initialModel(nil)
	model, _ = UpdateModel(model, keyMessage(KeyEvent{Key: KeyWheelUp}))
	if model.ScrollOffset != 3 {
		t.Fatalf("wheel up offset = %d", model.ScrollOffset)
	}
	model, _ = UpdateModel(model, keyMessage(KeyEvent{Key: KeyWheelDown}))
	if model.ScrollOffset != 0 {
		t.Fatalf("wheel down offset = %d", model.ScrollOffset)
	}
}

func TestShiftEnterInsertsNewlineWithoutSubmitting(t *testing.T) {
	model := initialModel(nil)
	model.Editor.Insert("first")
	model, effects := UpdateModel(model, keyMessage(KeyEvent{Key: KeyEnter, Shift: true}))
	if got := model.Editor.String(); got != "first\n" || len(effects) != 0 {
		t.Fatalf("shift-enter input/effects = %q / %#v", got, effects)
	}
}

func TestChatLinesUseExactlyOneBoundaryBlank(t *testing.T) {
	tui := TUI{width: 40, model: Model{Chat: []Message{
		{MsgType: MsgResponse, Data: "first\n\n"},
		{MsgType: MsgResponse, Data: "\nsecond"},
	}}}
	lines := tui.chatLines()
	if len(lines) != 3 || lines[1] != "" {
		t.Fatalf("chat boundary spacing = %#v, want two messages separated by one blank row", lines)
	}
}

func TestCodeFencePreservesIndentation(t *testing.T) {
	lines := renderMessage(Message{MsgType: MsgResponse, Data: "```go\n  indented()\n```"}, 40)
	if !strings.Contains(lines[1], "  indented()") {
		t.Fatalf("code indentation was collapsed: %q", lines[1])
	}
}

func TestStreamingPreservesScrolledContent(t *testing.T) {
	tui := TUI{width: 8, model: initialModel(nil)}
	tui.model.Chat = []Message{{MsgType: MsgResponse, Data: "one two three four"}}
	tui.model.ScrollOffset = 2
	before := len(tui.chatLines()) - tui.model.ScrollOffset
	tui.Update(Message{MsgType: MsgResponse, Data: " five six"})
	after := len(tui.chatLines()) - tui.model.ScrollOffset
	if after != before {
		t.Fatalf("stream moved scrolled viewport anchor from %d to %d", before, after)
	}
}
