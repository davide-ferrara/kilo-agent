package main

import (
	"os"
	"unicode/utf8"
)

// readInput reads a full keypress (1-4 bytes, UTF-8) in raw mode
func readInput() ([]byte, int) {
	buf := make([]byte, 4)
	n, err := os.Stdin.Read(buf[:1])
	if err != nil {
		return buf, -1
	}
	if n == 0 {
		return buf, 0
	}

	size := utf8.RuneLen(rune(buf[0]))
	if size <= 0 {
		size = 1
	}

	for n < size {
		m, err := os.Stdin.Read(buf[n:size])
		if err != nil {
			break
		}
		n += m
	}

	return buf, n
}
