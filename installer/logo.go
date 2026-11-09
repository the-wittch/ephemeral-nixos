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

func renderLogo(rows [][]cell) string {
	truecolor := lipgloss.ColorProfile() == termenv.TrueColor
	var sb strings.Builder
	for i, row := range rows {
		if i > 0 {
			sb.WriteByte('\n')
		}
		for _, c := range row {
			if !c.colored || c.ch == " " {
				sb.WriteString(c.ch)
				continue
			}
			var col lipgloss.TerminalColor
			switch {
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
func renderWord() string {
	if !g.blockWord {
		return sTitle.Render("E P H E M E R A L   N I X")
	}
	from, to := [3]float64{0xa8, 0x55, 0xf7}, [3]float64{0x38, 0xbd, 0xf8}
	var lines []string
	for _, row := range wordRows {
		runes := []rune(row)
		var sb strings.Builder
		for i, r := range runes {
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
