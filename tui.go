package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"golang.org/x/term"
)

const (
	ansiReset       = "\x1b[0m"
	ansiBold        = "\x1b[1m"
	ansiAltEnable   = "\x1b[?1049h"
	ansiAltDisable  = "\x1b[?1049l"
	ansiCursorHide  = "\x1b[?25l"
	ansiCursorShow  = "\x1b[?25h"
	ansiClearScreen = "\x1b[2J"
	ansiCursorHome  = "\x1b[H"
)

func ansiColor(layer, r, g, b int) string {
	return fmt.Sprintf("\x1b[%d;2;%d;%d;%dm", layer, r, g, b)
}

func ansiForeground(r, g, b int) string { return ansiColor(38, r, g, b) }
func ansiBackground(r, g, b int) string { return ansiColor(48, r, g, b) }

// CyanogenMod 11-inspired palette.
var (
	themeWhite      = ansiForeground(255, 255, 255)
	themeWhiteBold  = ansiBold + themeWhite
	themeGray       = ansiForeground(158, 158, 158)
	themeGreen      = ansiForeground(153, 204, 0)
	themeCyan       = ansiForeground(51, 181, 229)
	themeSurface    = ansiBackground(18, 18, 18)
	themeSurfaceAlt = ansiBackground(26, 26, 26)
	themeCyanDark   = ansiBackground(10, 46, 61)
	themeGreenDark  = ansiBackground(24, 36, 12)
)

type TUI struct {
	Width     int
	Height    int
	Model     Model
	ModelName string
	Events    chan<- Message
	ReqChan   chan<- string
	rawState  *term.State
	restore   sync.Once
}

type Model struct {
	BootText        string
	InputLine       []rune
	Chat            []Message
	InputY          int
	PendingRequests int
	SpinnerFrame    int
}

var spinnerFrames = [...]string{"▰▱▱", "▰▰▱", "▰▰▰", "▱▰▰", "▱▱▰", "▱▱▱"}

func NewTUI(events chan<- Message, reqChan chan<- string) *TUI {
	return &TUI{
		Events:  events,
		ReqChan: reqChan,
	}
}

func ansiMoveCursor(x, y int) string {
	return fmt.Sprintf("\x1b[%d;%dH", y, x)
}

func getTermSize() (int, int, error) {
	fd := int(os.Stdout.Fd())
	if !term.IsTerminal(fd) {
		return 0, 0, errors.New("output is not a terminal")
	}

	width, height, err := term.GetSize(fd)
	if err != nil {
		return 0, 0, errors.New("error getting terminal size")
	}

	return width, height, nil
}

// HandleInput runs in its own goroutine and turns raw keystrokes into
// Message values pushed on t.Events.
func (t *TUI) HandleInput() {
	for {
		buf, n := readInput()
		if n <= 0 {
			continue
		}

		switch buf[0] {
		case 3, 4: // Ctrl+C, Ctrl+D
			t.Events <- Message{MsgQuit, nil}
			return
		case 10, 13: // Enter (LF or CR, depending on terminal mode)
			t.Events <- Message{MsgEnter, nil}
		case 127:
			t.Events <- Message{MsgBackspace, nil}
		default:
			if r, _ := utf8.DecodeRune(buf[:n]); r != utf8.RuneError && unicode.IsPrint(r) {
				t.Events <- Message{MsgKeyPress, r}
			}
		}
	}
}

func (t *TUI) bootText() string {
	const inner = 44
	const boxW = inner + 2

	margin := (t.Width - boxW) / 2
	if margin < 0 {
		margin = 0
	}
	pad := strings.Repeat(" ", margin)

	type row struct {
		color string
		text  string
		boxed bool
	}
	rows := []row{
		{themeCyan, "┏" + strings.Repeat("━", inner) + "┓", false},
		{themeCyan, "┃" + strings.Repeat(" ", inner) + "┃", false},
		{themeWhiteBold, "         ██  ██   ██   ██      ███████      ", true},
		{themeWhiteBold, "         ██ ██    ██   ██      ██   ██      ", true},
		{themeWhiteBold, "         ████     ██   ██      ██   ██      ", true},
		{themeWhiteBold, "         ██ ██    ██   ██      ██   ██      ", true},
		{themeWhiteBold, "         ██  ██   ██   █████   ███████      ", true},
		{themeCyan, "┃" + strings.Repeat(" ", inner) + "┃", false},
		{themeWhite, "          ● kilobyte agent · v0.1a          ", true},
		{themeWhite, "   model qwen3:14b   press Ctrl+C to exit   ", true},
		{themeCyan, "┃" + strings.Repeat(" ", inner) + "┃", false},
		{themeCyan, "┗" + strings.Repeat("━", inner) + "┛", false},
	}

	var b strings.Builder
	for i, r := range rows {
		b.WriteString(pad)
		if r.boxed {
			b.WriteString(themeCyan + "┃" + ansiReset + r.color + r.text + ansiReset + themeCyan + "┃" + ansiReset)
		} else {
			b.WriteString(r.color + r.text + ansiReset)
		}
		if i < len(rows)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

func (t *TUI) Init() {
	var err error
	t.Width, t.Height, err = getTermSize()
	if err != nil {
		panic(err)
	}

	t.Model = Model{
		BootText:  t.bootText(),
		InputLine: []rune{},
		Chat:      []Message{},
		InputY:    t.Height - 3,
	}

	t.rawState, err = term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		panic(err)
	}

	fmt.Print(ansiAltEnable)
}

func (t *TUI) Restore() {
	t.restore.Do(func() {
		if t.rawState != nil {
			if err := term.Restore(int(os.Stdin.Fd()), t.rawState); err != nil {
				log.Printf("restore terminal: %v", err)
			}
		}
		fmt.Print(ansiReset + ansiCursorShow + ansiAltDisable + "Thanks for using Kilo Agent, hope it was fun!\n")
	})
}

// inputRows splits the input line into chunks of at most Width runes,
// always appending a final (possibly empty) row so the cursor row is exactly
// len(InputLine)/Width.
func (t *TUI) inputRows() []string {
	runes := t.Model.InputLine
	var rows []string
	for i := 0; i <= len(runes)/t.Width; i++ {
		start := i * t.Width
		end := start + t.Width
		if end > len(runes) {
			end = len(runes)
		}
		rows = append(rows, string(runes[start:end]))
	}
	return rows
}

// statusBar builds the fixed bottom row of the view: the generation spinner
// and model name on the left, and the current working directory on the right.
func (t *TUI) statusBar() string {
	left := t.ModelName
	if left == "" {
		left = "model"
	}
	if t.Model.PendingRequests > 0 {
		frame := spinnerFrames[t.Model.SpinnerFrame%len(spinnerFrames)]
		left = themeCyan + frame + " " + themeWhite + left
	}

	dir, err := os.Getwd()
	if err != nil {
		dir = "?"
	}
	right := dir

	if t.Width <= 0 {
		return ""
	}

	modelStr := left
	dirStr := right

	// Reserve one column for the separator space, then pad the rest.
	fillLen := t.Width - visibleRuneCount(modelStr) - utf8.RuneCountInString(dirStr) - 1
	var fill string
	if fillLen > 0 {
		fill = strings.Repeat(" ", fillLen)
	}

	return themeSurfaceAlt + themeWhite + modelStr + fill + " " + dirStr + ansiReset
}

func visibleRuneCount(s string) int {
	inEscape := false
	count := 0
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if r == 'm' {
				inEscape = false
			}
			continue
		}
		count++
	}
	return count
}

// msgColor returns the ANSI text color and background for a chat message type.
func msgColor(mt MsgType) (fg, bg string) {
	switch mt {
	case MsgUser:
		return themeWhite, themeCyanDark
	case MsgThinking:
		return themeGray, ""
	case MsgResponse:
		return themeWhite, themeSurface
	case MsgTool:
		return themeGreen, themeGreenDark
	default:
		return themeWhite, ""
	}
}

func styledLine(text, fg, bg string, width int) string {
	padding := width - utf8.RuneCountInString(text)
	if padding < 0 {
		padding = 0
	}
	return bg + fg + text + strings.Repeat(" ", padding) + ansiReset
}

// chatLines renders one blank separator between complete message entries.
// Streamed chunks are combined by appendAI before they reach this function.
func (t *TUI) chatLines() []string {
	var lines []string
	for i, message := range t.Model.Chat {
		if i > 0 {
			lines = append(lines, "")
		}

		fg, bg := msgColor(message.MsgType)
		for _, line := range strings.Split(fmt.Sprint(message.Data), "\n") {
			lines = append(lines, styledLine(line, fg, bg, t.Width))
		}
	}
	return lines
}

func placeBottom(destination, content []string, end int) {
	if end <= 0 || len(content) == 0 {
		return
	}
	if end > len(destination) {
		end = len(destination)
	}
	if len(content) > end {
		content = content[len(content)-end:]
	}
	copy(destination[end-len(content):end], content)
}

// view builds the full terminal frame (boot text, colored chat, input row)
// as a single ANSI string. It performs no I/O.
func (t *TUI) view() string {
	lines := make([]string, t.Height)

	inputRows := t.inputRows()
	for i, row := range inputRows {
		inputRows[i] = styledLine(row, themeWhite, themeSurfaceAlt, t.Width)
	}
	inputTop := t.Model.InputY - (len(inputRows) - 1)
	if inputTop < 0 {
		inputTop = 0
	}

	if len(t.Model.Chat) == 0 {
		boot := strings.Split(t.Model.BootText, "\n")
		start := (inputTop - len(boot)) / 2
		if start < 0 {
			start = 0
		}
		for i, line := range boot {
			if start+i >= inputTop {
				break
			}
			lines[start+i] = line
		}
	} else {
		// Keep one blank row between chat history and the input.
		placeBottom(lines, t.chatLines(), inputTop-1)
	}

	copy(lines[inputTop:], inputRows)

	// Reserve a fixed blank spacer row between the input and the status bar.
	if t.Height > 0 {
		lines[t.Height-1] = t.statusBar()
	}
	if t.Height > 1 {
		lines[t.Height-2] = ""
	}

	cursorX := len(t.Model.InputLine) % t.Width
	return ansiCursorHide + ansiClearScreen + ansiCursorHome + strings.Join(lines, "\r\n") +
		ansiMoveCursor(cursorX+1, t.Model.InputY+1) + ansiCursorShow
}

func (t *TUI) Render() {
	fmt.Print(t.view())
}

func (t *TUI) Update(msg Message) {
	switch msg.MsgType {
	case MsgKeyPress:
		r, ok := msg.Data.(rune)
		if ok {
			t.Model.InputLine = append(t.Model.InputLine, r)
		}
	case MsgEnter:
		prompt := string(t.Model.InputLine)
		if len(prompt) == 0 {
			return
		}
		t.Model.Chat = append(t.Model.Chat, Message{MsgUser, prompt})
		t.Model.InputLine = t.Model.InputLine[:0]
		t.Model.PendingRequests++
		t.Model.SpinnerFrame = 0
		t.ReqChan <- prompt
	case MsgBackspace:
		if len(t.Model.InputLine) == 0 {
			return
		}
		t.Model.InputLine = t.Model.InputLine[:len(t.Model.InputLine)-1]
	case MsgThinking, MsgResponse, MsgTool:
		text, ok := msg.Data.(string)
		if !ok {
			return
		}
		t.appendAI(text, msg.MsgType)
	case MsgGenerationDone:
		if t.Model.PendingRequests > 0 {
			t.Model.PendingRequests--
		}
	case MsgSpinnerTick:
		t.Model.SpinnerFrame = (t.Model.SpinnerFrame + 1) % len(spinnerFrames)
	default:
		log.Println("Invalid Update Message")
		return
	}
}

// appendAI combines streamed text chunks. Tool calls remain separate entries.
func (t *TUI) appendAI(text string, msgType MsgType) {
	if msgType != MsgTool && len(t.Model.Chat) > 0 {
		last := &t.Model.Chat[len(t.Model.Chat)-1]
		if last.MsgType == msgType {
			if prev, ok := last.Data.(string); ok {
				last.Data = prev + text
				return
			}
		}
	}
	t.Model.Chat = append(t.Model.Chat, Message{msgType, text})
}
