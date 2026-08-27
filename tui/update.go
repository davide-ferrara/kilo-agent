package tui

import (
	"fmt"
	"strings"
)

type Model struct {
	Editor          Editor
	Chat            []Message
	PendingRequests int
	SpinnerFrame    int
	ScrollOffset    int
	InputWidth      int
	Files           []string
	Completion      Completion
	Usage           Usage
	LastError       string
}

func initialModel(files []string) Model {
	return Model{
		Editor:     Editor{PreferredColumn: -1, HistoryIndex: 0},
		InputWidth: 72,
		Files:      files,
		Usage:      Usage{ContextSize: 32768},
	}
}

// UpdateModel is the Elm update function: no terminal I/O and no agent calls.
func UpdateModel(model Model, msg Message) (Model, []Effect) {
	var effects []Effect
	switch msg.MsgType {
	case MsgKey:
		key, ok := msg.Data.(KeyEvent)
		if ok {
			effects = updateKey(&model, key)
		}
	case MsgPaste:
		if text, ok := msg.Data.(string); ok {
			model.Editor.Insert(normalizePaste(text))
			updateCompletion(&model)
		}
	case MsgThinking, MsgResponse, MsgTool, MsgError, MsgNotice:
		if text, ok := msg.Data.(string); ok {
			appendStream(&model, text, msg.MsgType)
			if msg.MsgType == MsgError {
				model.LastError = text
			}
		}
	case MsgGenerationDone:
		if model.PendingRequests > 0 {
			model.PendingRequests--
		}
	case MsgSpinnerTick:
		model.SpinnerFrame = (model.SpinnerFrame + 1) % len(spinnerFrames)
	case MsgUsage:
		if usage, ok := msg.Data.(Usage); ok {
			if usage.ContextSize == 0 {
				usage.ContextSize = model.Usage.ContextSize
			}
			model.Usage = usage
		}
	case MsgQuit:
		effects = append(effects, Effect{Type: EffectQuit})
	}
	return model, effects
}

func updateKey(model *Model, key KeyEvent) []Effect {
	if key.Ctrl {
		if effects, handled := updateControlKey(model, key); handled {
			return effects
		}
	}
	if key.Alt && (key.Rune == 'b' || key.Key == KeyLeft) {
		model.Editor.MoveWord(-1)
	} else if key.Alt && (key.Rune == 'f' || key.Key == KeyRight) {
		model.Editor.MoveWord(1)
	} else {
		switch key.Key {
		case KeyRune:
			model.Editor.Insert(string(key.Rune))
		case KeyEnter:
			if key.Shift || key.Alt {
				model.Editor.Insert("\n")
				model.Completion = Completion{}
				break
			}
			if model.Completion.Kind == CompletionCommand && applyCompletion(model) {
				return submit(model)
			}
			if applyCompletion(model) {
				return nil
			}
			return submit(model)
		case KeyBackspace:
			model.Editor.Backspace()
		case KeyDelete:
			model.Editor.Delete()
		case KeyLeft:
			model.Editor.Move(-1)
		case KeyRight:
			model.Editor.Move(1)
		case KeyHome:
			model.Editor.Cursor = lineStart(model.Editor.Text, model.Editor.Cursor)
			model.Editor.PreferredColumn = -1
		case KeyEnd:
			model.Editor.Cursor = lineEnd(model.Editor.Text, model.Editor.Cursor)
			model.Editor.PreferredColumn = -1
		case KeyUp:
			if len(model.Completion.Items) > 0 {
				model.Completion.Selected = wrapIndex(model.Completion.Selected-1, len(model.Completion.Items))
				return nil
			}
			model.Editor.MoveVertical(-1, model.InputWidth)
		case KeyDown:
			if len(model.Completion.Items) > 0 {
				model.Completion.Selected = wrapIndex(model.Completion.Selected+1, len(model.Completion.Items))
				return nil
			}
			model.Editor.MoveVertical(1, model.InputWidth)
		case KeyPageUp:
			model.ScrollOffset += 8
			return nil
		case KeyPageDown:
			model.ScrollOffset = clamp(model.ScrollOffset-8, 0, model.ScrollOffset)
			return nil
		case KeyWheelUp:
			model.ScrollOffset += 3
			return nil
		case KeyWheelDown:
			model.ScrollOffset = clamp(model.ScrollOffset-3, 0, model.ScrollOffset)
			return nil
		case KeyTab:
			if !applyCompletion(model) {
				model.Editor.Insert("    ")
			}
		case KeyBackTab, KeyEscape:
			model.Completion = Completion{}
			return nil
		}
	}
	model.ScrollOffset = 0
	updateCompletion(model)
	return nil
}

func updateControlKey(model *Model, key KeyEvent) ([]Effect, bool) {
	switch key.Rune {
	case 'A':
		model.Editor.Cursor = lineStart(model.Editor.Text, model.Editor.Cursor)
		model.Editor.PreferredColumn = -1
	case 'E':
		model.Editor.Cursor = lineEnd(model.Editor.Text, model.Editor.Cursor)
		model.Editor.PreferredColumn = -1
	case 'B':
		model.Editor.Move(-1)
	case 'F':
		model.Editor.Move(1)
	case 'W':
		model.Editor.DeleteWord()
	case 'U':
		model.Editor.KillToStart()
	case 'K':
		model.Editor.KillToEnd()
	case 'Y':
		model.Editor.Insert(string(model.Editor.KillRing))
	case 'P':
		model.Editor.Recall(-1)
	case 'N':
		model.Editor.Recall(1)
	case 'L':
		model.Chat = nil
		model.ScrollOffset = 0
	case 'C':
		if len(model.Editor.Text) > 0 {
			model.Editor.Clear()
			model.Completion = Completion{}
			return nil, true
		}
		return []Effect{{Type: EffectQuit}}, true
	case 'D':
		if len(model.Editor.Text) == 0 {
			return []Effect{{Type: EffectQuit}}, true
		}
		model.Editor.Delete()
	case 'J', 'M':
		return submit(model), true
	default:
		if key.Key == KeyLeft {
			model.Editor.MoveWord(-1)
			return nil, true
		}
		if key.Key == KeyRight {
			model.Editor.MoveWord(1)
			return nil, true
		}
		return nil, false
	}
	updateCompletion(model)
	return nil, true
}

func submit(model *Model) []Effect {
	if applyCompletion(model) {
		return nil
	}
	prompt := strings.TrimSpace(model.Editor.String())
	if prompt == "" {
		return nil
	}
	model.Editor.Commit()
	model.Completion = Completion{}
	model.ScrollOffset = 0
	if strings.HasPrefix(prompt, "/") {
		return runCommand(model, prompt)
	}
	model.Chat = append(model.Chat, Message{MsgType: MsgUser, Data: prompt})
	model.PendingRequests++
	model.SpinnerFrame = 0
	return []Effect{{Type: EffectSendPrompt, Data: prompt}}
}

func runCommand(model *Model, command string) []Effect {
	switch strings.Fields(command)[0] {
	case "/help":
		model.Chat = append(model.Chat, Message{MsgType: MsgNotice, Data: helpText()})
	case "/clear":
		model.Chat = nil
	case "/tokens":
		u := model.Usage
		percent := contextPercent(u)
		model.Chat = append(model.Chat, Message{MsgType: MsgNotice, Data: fmt.Sprintf("Context: %d / %d tokens (%.1f%%)\nLast response: %d tokens at %.1f tk/s", u.PromptTokens, u.ContextSize, percent, u.OutputTokens, u.TokensPerSec)})
	case "/quit", "/exit":
		return []Effect{{Type: EffectQuit}}
	default:
		model.Chat = append(model.Chat, Message{MsgType: MsgError, Data: "Unknown command. Type /help to see what is available."})
	}
	return nil
}

func helpText() string {
	return strings.Join([]string{
		"Commands",
		"  /help    commands and keyboard shortcuts",
		"  /clear   clear the conversation",
		"  /tokens  detailed model usage",
		"  /quit    leave Kilo Agent",
		"",
		"Editing",
		"  @path selects a project file · / opens commands",
		"  arrows move/select · Shift+Enter adds a line · Tab accepts",
		"  PgUp/PgDn scrolls the conversation · drag selects text",
		"  Ctrl+A/E start/end · Ctrl+W delete word · Ctrl+U/K kill line",
		"  Ctrl+Y yank · Ctrl+P/N history · Alt+B/F move by word",
		"  Ctrl+C clears input (or quits when empty) · Ctrl+D quits when empty",
		"  Ctrl+Shift+C copy · Ctrl+Shift+V / Shift+Insert paste",
	}, "\n")
}

func appendStream(model *Model, text string, msgType MsgType) {
	if text == "" {
		return
	}
	if msgType != MsgTool && msgType != MsgError && msgType != MsgNotice && len(model.Chat) > 0 {
		last := &model.Chat[len(model.Chat)-1]
		if last.MsgType == msgType {
			if previous, ok := last.Data.(string); ok {
				last.Data = previous + text
				return
			}
		}
	}
	model.Chat = append(model.Chat, Message{MsgType: msgType, Data: text})
}

func normalizePaste(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	return strings.ReplaceAll(text, "\r", "\n")
}

func lineStart(text []rune, cursor int) int {
	for cursor > 0 && text[cursor-1] != '\n' {
		cursor--
	}
	return cursor
}

func lineEnd(text []rune, cursor int) int {
	for cursor < len(text) && text[cursor] != '\n' {
		cursor++
	}
	return cursor
}

func wrapIndex(index, length int) int {
	if length == 0 {
		return 0
	}
	return (index%length + length) % length
}

func contextPercent(usage Usage) float64 {
	if usage.ContextSize <= 0 {
		return 0
	}
	return float64(usage.PromptTokens) * 100 / float64(usage.ContextSize)
}
