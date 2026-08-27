package tui

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	"golang.org/x/term"
)

type TUI struct {
	width     int
	height    int
	model     Model
	modelName string
	events    chan<- Message
	rawState  *term.State
	restore   sync.Once
	renderer  *Renderer
}

func New(events chan<- Message) *TUI {
	return &TUI{events: events, renderer: NewRenderer(os.Stdout)}
}

func getTermSize() (int, int, error) {
	fd := int(os.Stdout.Fd())
	if !term.IsTerminal(fd) {
		return 0, 0, errors.New("output is not a terminal")
	}
	width, height, err := term.GetSize(fd)
	if err != nil {
		return 0, 0, fmt.Errorf("get terminal size: %w", err)
	}
	return width, height, nil
}

func (t *TUI) Init(modelName string) error {
	width, height, err := getTermSize()
	if err != nil {
		return err
	}
	t.width, t.height = width, height
	t.modelName = modelName
	t.model = initialModel(discoverFiles(".", 2000))
	t.model.InputWidth = t.editorWidth()
	t.rawState, err = term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("enable raw terminal input: %w", err)
	}
	fmt.Print(ansiAltEnable + ansiPasteEnable)
	return nil
}

func (t *TUI) Restore() {
	t.restore.Do(func() {
		if t.rawState != nil {
			if err := term.Restore(int(os.Stdin.Fd()), t.rawState); err != nil {
				log.Printf("restore terminal: %v", err)
			}
		}
		fmt.Print(ansiPasteDisable + ansiReset + ansiCursorShow + ansiAltDisable + "Kilo Agent closed cleanly.\n")
	})
}

func (t *TUI) HandleInput() {
	decoder := NewInputDecoder(os.Stdin)
	for {
		message, err := decoder.Read()
		if err != nil {
			if err != io.EOF {
				t.events <- Message{MsgType: MsgError, Data: "terminal input: " + err.Error()}
			}
			t.events <- Message{MsgType: MsgQuit}
			return
		}
		t.events <- message
	}
}

func (t *TUI) Update(message Message) []Effect {
	t.model.InputWidth = t.editorWidth()
	previousLines := 0
	wasScrolled := t.model.ScrollOffset > 0 && isConversationMessage(message.MsgType)
	if wasScrolled {
		previousLines = len(t.chatLines())
	}
	model, effects := UpdateModel(t.model, message)
	t.model = model
	if wasScrolled {
		t.model.ScrollOffset += max(0, len(t.chatLines())-previousLines)
	}
	return effects
}

func (t *TUI) Render() { t.renderer.Draw(t.frame()) }

func (t *TUI) Pending() bool { return t.model.PendingRequests > 0 }

func (t *TUI) Resize() error {
	width, height, err := getTermSize()
	if err != nil {
		return err
	}
	t.width, t.height = width, height
	t.model.InputWidth = t.editorWidth()
	t.model.Editor.PreferredColumn = -1
	t.renderer.Reset()
	return nil
}

func (t *TUI) editorWidth() int {
	switch {
	case t.width >= 7:
		return t.width - 6
	case t.width >= 3:
		return t.width - 2
	default:
		return 1
	}
}

func isConversationMessage(kind MsgType) bool {
	return kind == MsgThinking || kind == MsgResponse || kind == MsgTool || kind == MsgError || kind == MsgNotice
}
