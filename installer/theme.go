package main

import (
	"os"

	"github.com/charmbracelet/lipgloss"
)

// Catppuccin Mocha. The ANSI numbers are the slots of the ISO's console palette (modules/iso.nix),
// so the plain console shows the same colors, just fewer of them.
var (
	cText    = lipgloss.CompleteColor{TrueColor: "#cdd6f4", ANSI256: "189", ANSI: "7"}
	cSub     = lipgloss.CompleteColor{TrueColor: "#a6adc8", ANSI256: "146", ANSI: "15"}
	cDim     = lipgloss.CompleteColor{TrueColor: "#6c7086", ANSI256: "60", ANSI: "8"}
	cSurface = lipgloss.CompleteColor{TrueColor: "#313244", ANSI256: "237", ANSI: "8"}
	cBase    = lipgloss.CompleteColor{TrueColor: "#1e1e2e", ANSI256: "235", ANSI: "0"}
	cMauve   = lipgloss.CompleteColor{TrueColor: "#cba6f7", ANSI256: "183", ANSI: "5"}
	cBlue    = lipgloss.CompleteColor{TrueColor: "#89b4fa", ANSI256: "111", ANSI: "4"}
	cGreen   = lipgloss.CompleteColor{TrueColor: "#a6e3a1", ANSI256: "151", ANSI: "2"}
	cYellow  = lipgloss.CompleteColor{TrueColor: "#f9e2af", ANSI256: "223", ANSI: "3"}
	cRed     = lipgloss.CompleteColor{TrueColor: "#f38ba8", ANSI256: "211", ANSI: "1"}
)

var (
	sText   = lipgloss.NewStyle().Foreground(cText)
	sBold   = lipgloss.NewStyle().Foreground(cText).Bold(true)
	sSub    = lipgloss.NewStyle().Foreground(cSub)
	sDim    = lipgloss.NewStyle().Foreground(cDim)
	sMauve  = lipgloss.NewStyle().Foreground(cMauve)
	sTitle  = lipgloss.NewStyle().Foreground(cMauve).Bold(true)
	sBlue   = lipgloss.NewStyle().Foreground(cBlue)
	sGreen  = lipgloss.NewStyle().Foreground(cGreen)
	sYellow = lipgloss.NewStyle().Foreground(cYellow)
	sRed    = lipgloss.NewStyle().Foreground(cRed)
	sCursor = lipgloss.NewStyle().Foreground(cBase).Background(cMauve)
	sSel    = lipgloss.NewStyle().Background(cSurface)
	sTag    = lipgloss.NewStyle().Foreground(cText).Background(cSurface)
)

// The Linux console's 512-glyph fonts lack ✓, ▸, ○, braille and half blocks, so it gets a set
// made of what Lat2-Terminus16 and ter-v24n both have.
type glyphSet struct {
	cursor, done, failed, pending, dot, warn, barOn, barOff, rule, checked, unchecked string
	spinner                                                                           []string
	blockWord                                                                         bool
}

var richGlyphs = glyphSet{
	cursor: "▸", done: "✓", failed: "✗", pending: "○", dot: "●", warn: "▲",
	barOn: "━", barOff: "━", rule: "━", checked: "[x]", unchecked: "[ ]",
	spinner:   []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
	blockWord: true,
}

var consoleGlyphs = glyphSet{
	cursor: "▶", done: "■", failed: "x", pending: "·", dot: "•", warn: "▲",
	barOn: "█", barOff: "░", rule: "─", checked: "[x]", unchecked: "[ ]",
	spinner: []string{"|", "/", "-", "\\"},
}

func pickGlyphs() glyphSet {
	if os.Getenv("TERM") == "linux" || os.Getenv("EPHEMERAL_CONSOLE_GLYPHS") == "1" {
		return consoleGlyphs
	}
	return richGlyphs
}

var g = pickGlyphs()
