package main

import (
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// logo.txt is the wallpaper's snowflake as truecolor characters (see logo.sh).
// Each cell is "\e[38;2;R;G;Bm" and a character.
var cellRe = regexp.MustCompile(`\x1b\[38;2;(\d+);(\d+);(\d+)m(.)|\x1b\[0m|(.)`)

type cell struct {
	ch      string
	r, g, b int
	colored bool
}

func parseLogo(text string) [][]cell {
	var rows [][]cell
	for _, line := range strings.Split(strings.TrimRight(text, "\n"), "\n") {
		var row []cell
		for _, m := range cellRe.FindAllStringSubmatch(line, -1) {
			switch {
			case m[4] != "":
				r, _ := strconv.Atoi(m[1])
				gr, _ := strconv.Atoi(m[2])
				b, _ := strconv.Atoi(m[3])
				row = append(row, cell{ch: m[4], r: r, g: gr, b: b, colored: true})
			case m[5] != "":
				row = append(row, cell{ch: m[5]})
			}
		}
		rows = append(rows, row)
	}
	return rows
}

func loadLogo(path string) [][]cell {
	if path == "" {
		return nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return parseLogo(string(b))
}

// For its first frames the snowflake draws in alternating mauve and blue, then in its own colours.
func renderLogo(rows [][]cell, frame, age int, effects bool) string {
	truecolor := lipgloss.ColorProfile() == termenv.TrueColor
	flash := effects && age < logoFlash
	var sb strings.Builder
	for i, row := range rows {
		if i > 0 {
			sb.WriteByte('\n')
		}
		for j, c := range row {
			if !c.colored || c.ch == " " {
				sb.WriteString(c.ch)
				continue
			}
			var col lipgloss.TerminalColor
			switch {
			case flash && (frame+i+j)%2 == 0:
				col = cMauve
			case flash:
				col = cBlue
			case truecolor:
				col = lipgloss.Color("#" + hex2(c.r) + hex2(c.g) + hex2(c.b))
			case c.r > c.g:
				col = cMauve
			default:
				col = cBlue
			}
			sb.WriteString(lipgloss.NewStyle().Foreground(col).Render(c.ch))
		}
	}
	return sb.String()
}

func hex2(v int) string {
	s := strconv.FormatInt(int64(v), 16)
	if len(s) == 1 {
		return "0" + s
	}
	return s
}

var wordRows = []string{
	"█▀▀ █▀█ █ █ █▀▀ █▀▄▀█ █▀▀ █▀█ ▄▀█ █     █▄ █ █ ▀▄▀",
	"██▄ █▀▀ █▀█ ██▄ █ ▀ █ ██▄ █▀▄ █▀█ █▄▄   █ ▀█ █ █ █",
}

// the wallpaper's purple to blue across the word; on the console, spaced capitals in mauve
// The title settles out of scrambled characters, left to right, over its first wordSpan ticks.
func renderWord(frame, age int, effects bool) string {
	frac := 1.0
	if effects {
		frac = revealFrac(age, wordSpan)
	}
	if !g.blockWord {
		return sTitle.Render(scramble("E P H E M E R A L   N I X", frame, frac))
	}
	from, to := [3]float64{0xa8, 0x55, 0xf7}, [3]float64{0x38, 0xbd, 0xf8}
	var lines []string
	for y, row := range wordRows {
		runes := []rune(row)
		shown := int(float64(len(runes)) * frac)
		var sb strings.Builder
		for i, r := range runes {
			if i >= shown && r != ' ' {
				r = glitchChar(frame + i*7 + y*13)
			}
			t := float64(i) / float64(len(runes)-1)
			c := "#"
			for k := 0; k < 3; k++ {
				c += hex2(int(from[k] + (to[k]-from[k])*t))
			}
			sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(c)).Bold(true).Render(string(r)))
		}
		lines = append(lines, sb.String())
	}
	return strings.Join(lines, "\n")
}
