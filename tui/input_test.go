package tui

import (
	"strings"
	"testing"
)

func TestInputDecoderNavigationAndModifiers(t *testing.T) {
	decoder := NewInputDecoder(strings.NewReader("\x1b[D\x1b[1;5C\x1b[1;2A\x1bb"))
	tests := []KeyEvent{
		{Key: KeyLeft},
		{Key: KeyRight, Ctrl: true},
		{Key: KeyUp, Shift: true},
		{Key: KeyRune, Rune: 'b', Alt: true},
	}
	for index, want := range tests {
		message, err := decoder.Read()
		if err != nil {
			t.Fatal(err)
		}
		if got := message.Data.(KeyEvent); got != want {
			t.Fatalf("key %d = %#v, want %#v", index, got, want)
		}
	}
}

func TestInputDecoderMultilineEnter(t *testing.T) {
	decoder := NewInputDecoder(strings.NewReader("\x1b[13;2u\x1b\r"))
	for index, want := range []KeyEvent{
		{Key: KeyEnter, Shift: true},
		{Key: KeyEnter, Alt: true},
	} {
		message, err := decoder.Read()
		if err != nil {
			t.Fatal(err)
		}
		if got := message.Data.(KeyEvent); got != want {
			t.Fatalf("multiline enter %d = %#v, want %#v", index, got, want)
		}
	}
}

func TestInputDecoderBracketedPaste(t *testing.T) {
	decoder := NewInputDecoder(strings.NewReader("\x1b[200~hello\nworld\x1b[201~"))
	message, err := decoder.Read()
	if err != nil {
		t.Fatal(err)
	}
	if message.MsgType != MsgPaste || message.Data != "hello\nworld" {
		t.Fatalf("paste = %#v", message)
	}
}

func TestInputDecoderMouseWheel(t *testing.T) {
	decoder := NewInputDecoder(strings.NewReader("\x1b[<64;20;8M\x1b[<65;20;8M"))
	for index, want := range []Key{KeyWheelUp, KeyWheelDown} {
		message, err := decoder.Read()
		if err != nil {
			t.Fatal(err)
		}
		if got := message.Data.(KeyEvent).Key; got != want {
			t.Fatalf("wheel event %d = %v, want %v", index, got, want)
		}
	}
}
