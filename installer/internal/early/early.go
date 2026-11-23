// Package early tells Lip Gloss the background is dark before Bubble Tea asks the terminal.
//
// Bubble Tea v1 asks for the background colour (OSC 11) in its own init, and waits five seconds
// when nothing answers, as the Linux console doesn't, so the welcome screen came up five seconds
// late. The installer's colours are all fixed, so the answer changes nothing. This init runs
// first: since Go 1.21 packages initialise in import path order once their imports have,
// "ephemeral-install/..." sorts before "github.com/...", and Bubble Tea imports Lip Gloss too.
// The module path is "ephemeral-install" for that reason.
package early

import "github.com/charmbracelet/lipgloss"

func init() { lipgloss.SetHasDarkBackground(true) }
