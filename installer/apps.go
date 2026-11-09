package main

import (
	"fmt"
	"strconv"

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
	intro := sSub.Render("Optional. Everything here can be added later with ephemeral-apps.")
	picked := sMauve.Render(fmt.Sprintf("%d picked", count))
	out = append(out, intro+spaces(max(w-lipgloss.Width(intro)-lipgloss.Width(picked), 2))+picked, "")

	cats := m.appCategories()
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
	var right []string
	for i, a := range apps {
		box := sDim.Render(g.unchecked)
		if m.picked[a.ID] {
			box = sGreen.Render(g.checked)
		}
		flags := ""
		if a.Unfree {
			flags += " unfree"
		}
		if a.Flatpak {
			flags += " flathub"
		}
		text := padRight(a.Label, 18) + truncate(a.Description, rw-30)
		if i == m.appIdx && m.appPane == 1 {
			right = append(right, box+" "+sCursor.Render(padRight(text, rw-8))+sDim.Render(flags))
			if a.Warn != "" {
				right = append(right, "    "+sYellow.Render(g.warn+" "+truncate(a.Warn, rw-8)))
			}
		} else {
			right = append(right, box+" "+sText.Render(text)+sDim.Render(flags))
		}
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
	return out, hint("space", "tick", "tab", "switch side", "↑↓", "move", "enter", "next step", "esc", "back")
}
