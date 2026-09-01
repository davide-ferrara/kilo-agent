package tui

import (
	"bufio"
	"bytes"
	"io"
	"strings"
	"unicode/utf8"
)

const bracketedPasteEnd = "\x1b[201~"

// InputDecoder translates terminal escape sequences into editor-level keys.
type InputDecoder struct{ r *bufio.Reader }

func NewInputDecoder(r io.Reader) *InputDecoder {
	return &InputDecoder{r: bufio.NewReader(r)}
}

func (d *InputDecoder) Read() (Message, error) {
	b, err := d.r.ReadByte()
	if err != nil {
		return Message{}, err
	}
	switch b {
	case '\r', '\n':
		return keyMessage(KeyEvent{Key: KeyEnter}), nil
	case '\t':
		return keyMessage(KeyEvent{Key: KeyTab}), nil
	case 0x7f, 0x08:
		return keyMessage(KeyEvent{Key: KeyBackspace}), nil
	case 0x1b:
		return d.readEscape()
	}
	if b > 0 && b < 0x20 {
		return keyMessage(KeyEvent{Key: KeyRune, Rune: rune(b + '@'), Ctrl: true}), nil
	}
	return d.readRune(b)
}

func (d *InputDecoder) readRune(first byte) (Message, error) {
	if first < utf8.RuneSelf {
		return keyMessage(KeyEvent{Key: KeyRune, Rune: rune(first)}), nil
	}
	buf := []byte{first}
	for !utf8.FullRune(buf) && len(buf) < utf8.UTFMax {
		b, err := d.r.ReadByte()
		if err != nil {
			return Message{}, err
		}
		buf = append(buf, b)
	}
	r, _ := utf8.DecodeRune(buf)
	return keyMessage(KeyEvent{Key: KeyRune, Rune: r}), nil
}

func (d *InputDecoder) readEscape() (Message, error) {
	b, err := d.r.ReadByte()
	if err != nil {
		if err == io.EOF {
			return keyMessage(KeyEvent{Key: KeyEscape}), nil
		}
		return Message{}, err
	}
	if b != '[' && b != 'O' {
		if b == '\r' || b == '\n' {
			return keyMessage(KeyEvent{Key: KeyEnter, Alt: true}), nil
		}
		msg, err := d.readRune(b)
		if err == nil {
			event := msg.Data.(KeyEvent)
			event.Alt = true
			msg.Data = event
		}
		return msg, err
	}
	sequence, err := d.readCSI(b)
	if err != nil {
		return Message{}, err
	}
	if sequence == "[200~" {
		return d.readPaste()
	}
	return keyMessage(decodeCSI(sequence)), nil
}

func (d *InputDecoder) readCSI(prefix byte) (string, error) {
	buf := []byte{prefix}
	for len(buf) < 16 {
		b, err := d.r.ReadByte()
		if err != nil {
			return "", err
		}
		buf = append(buf, b)
		if b >= 0x40 && b <= 0x7e {
			return string(buf), nil
		}
	}
	return string(buf), nil
}

func (d *InputDecoder) readPaste() (Message, error) {
	var content bytes.Buffer
	marker := []byte(bracketedPasteEnd)
	for {
		b, err := d.r.ReadByte()
		if err != nil {
			return Message{}, err
		}
		content.WriteByte(b)
		if content.Len() >= len(marker) && bytes.HasSuffix(content.Bytes(), marker) {
			text := content.Bytes()[:content.Len()-len(marker)]
			return Message{MsgType: MsgPaste, Data: string(text)}, nil
		}
	}
}

func decodeCSI(sequence string) KeyEvent {
	// SGR mouse buttons 64 and 65 are wheel up and wheel down. Coordinates
	// are ignored because the conversation is a single scroll viewport.
	if strings.HasPrefix(sequence, "[<64;") && strings.HasSuffix(sequence, "M") {
		return KeyEvent{Key: KeyWheelUp}
	}
	if strings.HasPrefix(sequence, "[<65;") && strings.HasSuffix(sequence, "M") {
		return KeyEvent{Key: KeyWheelDown}
	}
	keys := map[string]Key{
		"[A": KeyUp, "OA": KeyUp, "[B": KeyDown, "OB": KeyDown,
		"[C": KeyRight, "OC": KeyRight, "[D": KeyLeft, "OD": KeyLeft,
		"[H": KeyHome, "OH": KeyHome, "[F": KeyEnd, "OF": KeyEnd,
		"[1~": KeyHome, "[4~": KeyEnd, "[3~": KeyDelete,
		"[5~": KeyPageUp, "[6~": KeyPageDown, "[Z": KeyBackTab,
	}
	if key, ok := keys[sequence]; ok {
		return KeyEvent{Key: key, Shift: sequence == "[Z"}
	}
	modified := map[byte]Key{
		'A': KeyUp, 'B': KeyDown, 'C': KeyRight, 'D': KeyLeft,
		'H': KeyHome, 'F': KeyEnd,
	}
	if len(sequence) >= 5 && sequence[0] == '[' && sequence[1] == '1' && sequence[2] == ';' {
		if key, ok := modified[sequence[len(sequence)-1]]; ok {
			return modifiedKey(key, sequence[3])
		}
	}
	if strings.HasPrefix(sequence, "[13;") && strings.HasSuffix(sequence, "u") && len(sequence) >= 6 {
		return modifiedKey(KeyEnter, sequence[4])
	}
	return KeyEvent{Key: KeyEscape}
}

func modifiedKey(key Key, modifier byte) KeyEvent {
	event := KeyEvent{Key: key}
	switch modifier {
	case '2':
		event.Shift = true
	case '3':
		event.Alt = true
	case '4':
		event.Shift, event.Alt = true, true
	case '5':
		event.Ctrl = true
	case '6':
		event.Shift, event.Ctrl = true, true
	case '7':
		event.Alt, event.Ctrl = true, true
	case '8':
		event.Shift, event.Alt, event.Ctrl = true, true, true
	}
	return event
}

func keyMessage(key KeyEvent) Message { return Message{MsgType: MsgKey, Data: key} }
