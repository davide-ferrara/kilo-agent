package tui

import "fmt"

const (
	ansiReset        = "\x1b[0m"
	ansiBold         = "\x1b[1m"
	ansiAltEnable    = "\x1b[?1049h"
	ansiAltDisable   = "\x1b[?1049l"
	ansiCursorHide   = "\x1b[?25l"
	ansiCursorShow   = "\x1b[?25h"
	ansiClearScreen  = "\x1b[2J"
	ansiCursorHome   = "\x1b[H"
	ansiClearLine    = "\x1b[2K"
	ansiClearToEnd   = "\x1b[K"
	ansiPasteEnable  = "\x1b[?2004h"
	ansiPasteDisable = "\x1b[?2004l"
)

func ansiColor(layer, r, g, b int) string {
	return fmt.Sprintf("\x1b[%d;2;%d;%d;%dm", layer, r, g, b)
}

func ansiForeground(r, g, b int) string { return ansiColor(38, r, g, b) }
func ansiBackground(r, g, b int) string { return ansiColor(48, r, g, b) }

// CyanogenMod 11 blue on Omarchy's restrained, near-black surfaces.
var (
	themeWhite       = ansiForeground(224, 231, 235)
	themeWhiteBold   = ansiBold + ansiForeground(244, 248, 250)
	themeGray        = ansiForeground(120, 135, 143)
	themeGrayDark    = ansiForeground(76, 91, 99)
	themeGreen       = ansiForeground(153, 204, 0)
	themeRed         = ansiForeground(255, 92, 92)
	themeCyan        = ansiForeground(51, 181, 229)
	themeCyanBright  = ansiForeground(92, 207, 250)
	themeSurface     = ansiBackground(13, 17, 20)
	themeSurfaceAlt  = ansiBackground(22, 28, 32)
	themeSurfaceLift = ansiBackground(29, 38, 43)
	themeCyanDark    = ansiBackground(8, 42, 57)
	themeGreenDark   = ansiBackground(22, 38, 19)
	themeRedDark     = ansiBackground(48, 22, 25)
)

var spinnerFrames = [...]string{"◐", "◓", "◑", "◒"}
