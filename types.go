package main

// MsgType is the contract between the UI and the agent: both sides only
// know about Message values flowing through channels, never about each other.
type MsgType int

const (
	MsgResponse MsgType = iota
	MsgThinking
	MsgTool
	MsgKeyPress
	MsgUser
	MsgEnter
	MsgBackspace
	MsgQuit
	MsgGenerationDone
	MsgSpinnerTick
)

type Message struct {
	MsgType MsgType
	Data    any
}
