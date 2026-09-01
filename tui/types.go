package tui

// MsgType is the event contract shared by the terminal, update loop, and agent.
type MsgType int

const (
	MsgResponse MsgType = iota
	MsgThinking
	MsgTool
	MsgUser
	MsgQuit
	MsgGenerationDone
	MsgSpinnerTick
	MsgKey
	MsgPaste
	MsgUsage
	MsgError
	MsgNotice
)

type Message struct {
	MsgType MsgType
	Data    any
}

type Key int

const (
	KeyRune Key = iota
	KeyEnter
	KeyBackspace
	KeyDelete
	KeyLeft
	KeyRight
	KeyUp
	KeyDown
	KeyHome
	KeyEnd
	KeyPageUp
	KeyPageDown
	KeyTab
	KeyBackTab
	KeyEscape
	KeyWheelUp
	KeyWheelDown
)

type KeyEvent struct {
	Key   Key
	Rune  rune
	Ctrl  bool
	Alt   bool
	Shift bool
}

type Usage struct {
	PromptTokens int
	OutputTokens int
	ContextSize  int
	TokensPerSec float64
}

type EffectType int

const (
	EffectSendPrompt EffectType = iota
	EffectQuit
)

type Effect struct {
	Type EffectType
	Data string
}
