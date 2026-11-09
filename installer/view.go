package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const frameWidth = 100

func (m model) View() string {
	if m.width == 0 {
		return ""
	}
	if m.width < 80 || m.height < 24 {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
			sText.Render(fmt.Sprintf("Make the window at least 80×24 (it's %d×%d).", m.width, m.height)))
	}
	w := min(m.width-4, frameWidth)
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
	if m.note != "" {
		body = append(body, "", sYellow.Render(g.warn+" ")+sText.Render(m.note))
	}
	h := min(m.height-2, 34)
	if len(body) > h-2 {
		body = body[:h-2]
	}
	for len(body) < h-2 {
		body = append(body, "")
	}
	body = append(body, "", hints)
	content := lipgloss.NewStyle().Width(w).Render(strings.Join(body, "\n"))
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

func hint(pairs ...string) string {
	var parts []string
	for i := 0; i+1 < len(pairs); i += 2 {
		parts = append(parts, sDim.Render(pairs[i])+" "+sSub.Render(pairs[i+1]))
	}
	return strings.Join(parts, "   ")
}

func (m model) header(w int) []string {
	steps := m.visible()
	n := 0
	for i, s := range steps {
		if s == m.step {
			n = i + 1
		}
	}
	left := sTitle.Render("Ephemeral Nix")
	right := sDim.Render(fmt.Sprintf("%d of %d · %s", n, len(steps), stepName[m.step]))
	gap := max(w-lipgloss.Width(left)-lipgloss.Width(right), 1)
	done := w * n / max(len(steps), 1)
	bar := sMauve.Render(strings.Repeat(g.barOn, done)) + sDim.Render(strings.Repeat(g.barOff, w-done))
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
