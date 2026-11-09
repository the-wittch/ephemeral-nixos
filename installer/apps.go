package main

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m model) appCategories() []string {
	seen := map[string]bool{}
	for _, a := range m.catalog {
		seen[a.Category] = true
	}
	var out []string
	for _, c := range categoryOrder {
		if seen[c] {
			out = append(out, c)
			delete(seen, c)
		}
	}
	for c := range seen {
		out = append(out, c)
	}
	return out
}

func (m model) appsIn(cat string) []app {
	var out []app
	for _, a := range m.catalog {
		if a.Category == cat {
			out = append(out, a)
		}
	}
	return out
}

func (m model) keyApps(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	if k.String() == "enter" {
		m.a.Apps = nil
		for _, a := range m.catalog {
			if m.picked[a.ID] {
				m.a.Apps = append(m.a.Apps, a.ID)
			}
		}
		return m, m.next()
	}
	if k.String() == "r" && !m.catalogLoading && (m.catalogErr != nil || len(m.catalog) == 0) {
		m.catalogLoading, m.catalogErr = true, nil
		return m, loadCatalogCmd(m.cfg)
	}
	cats := m.appCategories()
	if len(cats) == 0 {
		return m, nil
	}
	apps := m.appsIn(cats[m.appCat])
	switch k.String() {
	case "tab", "right", "left", "shift+tab":
		m.appPane = 1 - m.appPane
	case "up":
		if m.appPane == 0 && m.appCat > 0 {
			m.appCat--
			m.appIdx = 0
		} else if m.appPane == 1 && m.appIdx > 0 {
			m.appIdx--
		}
	case "down":
		if m.appPane == 0 && m.appCat < len(cats)-1 {
			m.appCat++
			m.appIdx = 0
		} else if m.appPane == 1 && m.appIdx < len(apps)-1 {
			m.appIdx++
		}
	case " ":
		if m.appPane == 1 && m.appIdx < len(apps) {
			id := apps[m.appIdx].ID
			m.picked[id] = !m.picked[id]
		}
	}
	return m, nil
}

func (m model) viewApps(w int) ([]string, string) {
	out := m.header(w)
	if m.catalogLoading {
		out = append(out, m.spin()+sText.Render(" Loading the app list. The first time takes a few minutes."), "",
			sSub.Render("You can skip this and pick apps after the install with ephemeral-apps."))
		return out, hint("enter", "skip", "esc", "back")
	}
	if m.catalogErr != nil || len(m.catalog) == 0 {
		out = append(out, sText.Render("The app list didn't load, so there's nothing to pick here."), "",
			sSub.Render("Try again, or go on and pick apps after the install with ephemeral-apps."),
			sSub.Render("The full error is in "+uiLogPath()+"."))
		return out, hint("r", "try again", "enter", "next step", "esc", "back")
	}
	count := 0
	for _, on := range m.picked {
		if on {
			count++
		}
	}
	cats := m.appCategories()
	// on a short screen the categories and the panel come first; each category shows its own count
	if len(out)+2+len(cats)+appPanel <= m.bodyRows() {
		intro := sSub.Render("Optional. Everything here can be added later with ephemeral-apps.")
		picked := sMauve.Render(fmt.Sprintf("%d picked", count))
		out = append(out, intro+spaces(max(w-lipgloss.Width(intro)-lipgloss.Width(picked), 2))+picked, "")
	}

	apps := m.appsIn(cats[m.appCat])
	var left []string
	for i, c := range cats {
		n := 0
		for _, a := range m.appsIn(c) {
			if m.picked[a.ID] {
				n++
			}
		}
		count := ""
		if n > 0 {
			count = strconv.Itoa(n)
		}
		text := padRight(c, 16) + padLeft(count, 2)
		switch {
		case i == m.appCat && m.appPane == 0:
			left = append(left, sCursor.Render(" "+g.cursor+" "+text+" "))
		case i == m.appCat:
			left = append(left, sMauve.Render(" "+g.cursor+" "+text+" "))
		default:
			left = append(left, "   "+sText.Render(text)+" ")
		}
	}
	rw := w - 26
	labelW := 0
	for _, a := range apps {
		labelW = max(labelW, lipgloss.Width(a.Label))
	}
	labelW = min(labelW, 20) + 2

	// the list gets the rows the header and the panel don't use
	rows := max(m.bodyRows()-len(out)-appPanel, 3)
	start, end := 0, len(apps)
	if len(apps) > rows {
		rows -= 2
		start = min(max(m.appIdx-rows/2, 0), len(apps)-rows)
		end = start + rows
	}
	var right []string
	if start > 0 {
		right = append(right, sDim.Render(fmt.Sprintf("  ↑ %d more", start)))
	}
	for i := start; i < end; i++ {
		a := apps[i]
		box := sDim.Render(g.unchecked)
		if m.picked[a.ID] {
			box = sGreen.Render(g.checked)
		}
		flags := appFlags(a)
		text := padRight(truncate(a.Label, labelW-2), labelW) + truncate(a.Description, rw-4-labelW-len(flags))
		if i == m.appIdx && m.appPane == 1 {
			right = append(right, box+" "+sCursor.Render(padRight(text, rw-4-len(flags)))+sDim.Render(flags))
		} else {
			right = append(right, box+" "+sText.Render(text)+sDim.Render(flags))
		}
	}
	if end < len(apps) {
		right = append(right, sDim.Render(fmt.Sprintf("  ↓ %d more", len(apps)-end)))
	}
	for i := 0; i < max(len(left), len(right)); i++ {
		l, r := "", ""
		if i < len(left) {
			l = left[i]
		}
		if i < len(right) {
			r = right[i]
		}
		out = append(out, padRight(l, 26)+r)
	}

	// the selected app in full, since its row may be cut short
	if m.appPane == 1 && m.appIdx < len(apps) {
		a := apps[m.appIdx]
		out = append(out, "", sBold.Render(a.Label)+sDim.Render(appFlags(a)), sSub.Render(truncate(a.Description, w)))
		if a.Warn != "" {
			warn := strings.Split(lipgloss.NewStyle().Width(w-2).Render(a.Warn), "\n")
			for i, l := range warn[:min(len(warn), 2)] {
				lead := "  "
				if i == 0 {
					lead = g.warn + " "
				}
				out = append(out, sYellow.Render(lead+strings.TrimRight(l, " ")))
			}
		}
	}
	return out, hint("space", "tick", "tab", "switch side", "↑↓", "move", "enter", "next step", "esc", "back")
}

func appFlags(a app) string {
	flags := ""
	if a.Unfree {
		flags += " unfree"
	}
	if a.Flatpak {
		flags += " flathub"
	}
	return flags
}

// lines under the list for the selected app: a blank, its name, its description, two for a warning
const appPanel = 5
