package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// wide enough to use a laptop console (160 columns at 1080p), narrow enough to read on a big one
const frameWidth = 140

func (m model) View() string {
	if m.width == 0 {
		return ""
	}
	if m.width < 80 || m.height < 24 {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
			sText.Render(fmt.Sprintf("Make the window at least 80×24 (it's %d×%d).", m.width, m.height)))
	}
	w := min(m.width-4, frameWidth)
	m.framed = m.roomy() && m.width >= 90
	var body []string
	var hints string
	switch m.step {
	case stWelcome:
		body, hints = m.viewWelcome(w)
	case stDisk:
		body, hints = m.viewDisk(w)
	case stDesktop, stGraphics, stType:
		body, hints = m.viewChoice(w)
	case stRegion:
		body, hints = m.viewRegion(w)
	case stOptions:
		body, hints = m.viewOptions(w)
	case stApps:
		body, hints = m.viewApps(w)
	case stMachine:
		body, hints = m.viewMachine(w)
	case stSecrets:
		body, hints = m.viewSecrets(w)
	case stReview:
		body, hints = m.viewReview(w)
	case stInstalling:
		body, hints = m.viewInstalling(w)
	case stDone, stFailed:
		body, hints = m.viewEnd(w)
	}
	problem := ""
	if p := m.problem(); p != "" {
		problem = sRed.Render(g.failed+" ") + sText.Render(truncate(p, w-2))
	}
	foot := []string{problem, hints}
	if m.roomy() {
		foot = []string{"", problem, "", hints}
	}
	h := m.bodyRows()
	if len(body) > h {
		body = body[:h]
	}
	for len(body) < h {
		body = append(body, "")
	}
	body = append(body, foot...)
	content := lipgloss.NewStyle().Width(w).Render(strings.Join(body, "\n"))
	if m.framed {
		content = m.frame(w, content)
	}
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

func (m model) problem() string {
	switch {
	case m.note != "":
		return m.note
	case m.step == stApps && m.catalogErr != nil:
		return "Couldn't load the app list: " + m.catalogErr.Error()
	}
	return ""
}

// whether there's height for a blank line between rows; the 24 px console font on a 768 px
// screen gives 32
func (m model) roomy() bool { return m.height >= 30 }

// the lines a screen has, above the error line and the keys
func (m model) bodyRows() int {
	if m.framed {
		return m.height - 2 - 4 - 2
	}
	if m.roomy() {
		return m.height - 2 - 4
	}
	return m.height - 2 - 2
}

func (m model) gap() []string {
	if m.roomy() {
		return []string{""}
	}
	return nil
}

func hint(pairs ...string) string {
	var parts []string
	for i := 0; i+1 < len(pairs); i += 2 {
		parts = append(parts, sDim.Render(pairs[i])+" "+sSub.Render(pairs[i+1]))
	}
	return strings.Join(parts, "   ")
}

func (m model) stepNumber() (n, of int) {
	steps := m.visible()
	for i, s := range steps {
		if s == m.step {
			n = i + 1
		}
	}
	return n, len(steps)
}

// what goes on the title row, or in the top of the frame
func (m model) crumbs() (left, right string) {
	switch m.step {
	case stWelcome:
		return "", ""
	case stInstalling:
		return "Ephemeral Nix", "installing · " + formatDuration(m.elapsed())
	case stDone, stFailed:
		state := "installed"
		switch {
		case m.showLog:
			state = "log"
		case m.step == stFailed:
			state = "stopped"
		}
		return "Ephemeral Nix", state + " · " + formatDuration(m.elapsed())
	}
	n, of := m.stepNumber()
	return "Ephemeral Nix", fmt.Sprintf("%d of %d · %s", n, of, stepName[m.step])
}

func (m model) header(w int) []string {
	n, of := m.stepNumber()
	done := w * n / max(of, 1)
	bar := sMauve.Render(strings.Repeat(g.barOn, done)) + sDim.Render(strings.Repeat(g.barOff, w-done))
	if m.framed {
		return []string{bar, ""}
	}
	left, right := m.crumbs()
	left, right = sTitle.Render(left), sDim.Render(right)
	gap := max(w-lipgloss.Width(left)-lipgloss.Width(right), 1)
	return []string{left + spaces(gap) + right, bar, ""}
}

func (m model) spin() string { return sMauve.Render(g.spinner[m.spinFrame%len(g.spinner)]) }

func row(selected bool, w int, text string) string {
	if selected {
		return sCursor.Render(padRight(" "+g.cursor+" "+text, w))
	}
	return "   " + text
}

func spaces(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat(" ", n)
}

func padRight(s string, n int) string { return s + spaces(n-lipgloss.Width(s)) }

func padLeft(s string, n int) string { return spaces(n-lipgloss.Width(s)) + s }

func truncate(s string, n int) string {
	if n <= 1 || lipgloss.Width(s) <= n {
		return s
	}
	r := []rune(s)
	for len(r) > 0 && lipgloss.Width(string(r))+1 > n {
		r = r[:len(r)-1]
	}
	return string(r) + "…"
}

func itoa(n int) string { return strconv.Itoa(n) }

const fieldLabel = 14

// A text field: an arrow when it has the cursor, then the label and what's typed. Under it, what
// to type while it has the cursor, or what's wrong with it. The line is there either way so
// moving between fields doesn't shift the screen.
func (m model) field(w int, focused bool, label, input, help string, err error) []string {
	line := "   " + sText.Render(padRight(label, fieldLabel)) + input
	if focused {
		line = sMauve.Render(" "+g.cursor+" "+padRight(label, fieldLabel)) + input
	}
	under := ""
	switch {
	case err != nil:
		under = sRed.Render(g.failed + " " + truncate(err.Error(), w-fieldLabel-5))
	case focused:
		under = sSub.Render(truncate(help, w-fieldLabel-3))
	}
	return []string{line, spaces(3+fieldLabel) + under}
}

func fieldHints(last bool) string {
	if last {
		return hint("type", "to fill it in", "enter", "next step", "↑↓", "move", "esc", "back")
	}
	return hint("type", "to fill it in", "enter", "next field", "↑↓", "move", "esc", "back")
}

// A line border round the screen, with the title and the step set into its top edge.
func (m model) frame(w int, content string) string {
	left, right := m.crumbs()
	top := sDim.Render("┌" + strings.Repeat("─", w+2) + "┐")
	if left != "" {
		fill := max(w-lipgloss.Width(left)-lipgloss.Width(right)-4, 1)
		top = sDim.Render("┌─ ") + sTitle.Render(left) + sDim.Render(" "+strings.Repeat("─", fill)+" ") +
			sDim.Render(right) + sDim.Render(" ─┐")
	}
	lines := []string{top}
	for _, l := range strings.Split(content, "\n") {
		lines = append(lines, sDim.Render("│ ")+padRight(l, w)+sDim.Render(" │"))
	}
	lines = append(lines, sDim.Render("└"+strings.Repeat("─", w+2)+"┘"))
	return strings.Join(lines, "\n")
}
