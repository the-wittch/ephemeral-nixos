package main

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// apps for another desktop than the one picked aren't shown, and without a desktop only the
// terminal ones are
func (m model) offered(a app) bool {
	if m.a.Desktop == "none" {
		return a.Category == "Terminal"
	}
	return len(a.Desktops) == 0 || slices.Contains(a.Desktops, m.a.Desktop)
}

// ticked by default on every desktop, or on the one picked
func (m model) isDefault(a app) bool { return a.Default || slices.Contains(a.DefaultOn, m.a.Desktop) }

// The ones ticked by default on some desktops follow the desktop until they're ticked or
// unticked by hand.
func (m *model) followDesktop() {
	for _, a := range m.catalog {
		if len(a.DefaultOn) > 0 && !m.touched[a.ID] {
			m.picked[a.ID] = m.isDefault(a)
		}
	}
}

// what goes to apps.json: the ticked apps, and -<name> for each default that was unticked
func (m model) appChoices() []string {
	var out []string
	for _, a := range m.catalog {
		switch {
		case !m.offered(a):
		case m.picked[a.ID]:
			out = append(out, a.ID)
		case m.isDefault(a):
			out = append(out, "-"+a.ID)
		}
	}
	return out
}

func (m model) appCategories() []string {
	seen := map[string]bool{}
	for _, a := range m.catalog {
		if m.offered(a) {
			seen[a.Category] = true
		}
	}
	var out []string
	for _, c := range rules.categories {
		if seen[c] {
			out = append(out, c)
			delete(seen, c)
		}
	}
	// the catalog only takes the ones in the order, but a test catalog can have others
	return append(out, slices.Sorted(maps.Keys(seen))...)
}

func (m model) appsIn(cat string) []app {
	var out []app
	for _, a := range m.catalog {
		if a.Category == cat && m.offered(a) {
			out = append(out, a)
		}
	}
	return out
}

// Each word of the search has to start a word of the app's name or description, as in
// ephemeral-apps, so "scan" finds a network scanner and not every name with s, c, a and n in it.
func appMatches(a app, query string) bool {
	split := func(s string) []string {
		return strings.FieldsFunc(strings.ToLower(s), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
	}
	words := split(a.Label + " " + a.Description)
	for _, q := range split(query) {
		if !slices.ContainsFunc(words, func(w string) bool { return strings.HasPrefix(w, q) }) {
			return false
		}
	}
	return true
}

// the apps offered here that match the search, from every category
func (m model) searchResults() []app {
	var out []app
	for _, a := range m.catalog {
		if m.offered(a) && appMatches(a, m.appQuery.Value()) {
			out = append(out, a)
		}
	}
	return out
}

func (m *model) closeSearch() {
	m.appSearching = false
	m.appQuery.SetValue("")
	m.appQuery.Blur()
	m.appIdx, m.appPane = 0, 0
}

func (m model) keySearch(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	res := m.searchResults()
	switch k.String() {
	case "enter", "esc":
		m.closeSearch()
		return m, nil
	case "up":
		m.appIdx = max(m.appIdx-1, 0)
		return m, nil
	case "down":
		m.appIdx = max(min(m.appIdx+1, len(res)-1), 0)
		return m, nil
	case "tab":
		if m.appIdx < len(res) {
			id := res[m.appIdx].ID
			m.picked[id] = !m.picked[id]
			m.touched[id] = true
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.appQuery, cmd = m.appQuery.Update(k)
	m.appIdx = 0
	return m, cmd
}

func (m model) keyApps(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.appSearching {
		return m.keySearch(k)
	}
	if k.String() == "enter" {
		m.a.Apps = m.appChoices()
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
	case "/":
		m.appSearching = true
		m.appIdx, m.appPane = 0, 1
		return m, m.appQuery.Focus()
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
	case "x", " ":
		if m.appPane == 1 && m.appIdx < len(apps) {
			id := apps[m.appIdx].ID
			m.picked[id] = !m.picked[id]
			m.touched[id] = true
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
	cats := m.appCategories()
	if len(cats) == 0 {
		out = append(out, sText.Render("Without a desktop, only terminal apps are offered, and there are none in the list."))
		return out, hint("enter", "next step", "esc", "back")
	}
	count := 0
	for _, a := range m.catalog {
		if m.offered(a) && m.picked[a.ID] {
			count++
		}
	}
	// on a short screen the categories and the panel come first; each category shows its own count
	if len(out)+2+len(cats)+appPanel <= m.bodyRows() {
		intro := sSub.Render("Some are ticked to start with. Change any of them later with ephemeral-apps.")
		if m.a.Desktop == "none" {
			intro = sSub.Render("Without a desktop, only terminal apps. Change them later with ephemeral-apps.")
		}
		picked := sMauve.Render(fmt.Sprintf("%d picked", count))
		out = append(out, intro+spaces(max(w-lipgloss.Width(intro)-lipgloss.Width(picked), 2))+picked, "")
	}

	apps := m.appsIn(cats[m.appCat])
	var left []string
	if m.appSearching {
		apps = m.searchResults()
		left = append(left, sMauve.Render(" / ")+m.appQuery.View(), "")
	}
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
		text := padRight(c, 17) + padLeft(count, 2)
		switch {
		case m.appSearching:
			left = append(left, "   "+sDim.Render(text)+" ")
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
		// the category only where it leaves room for the description; the panel below has it anyway
		if cat := " · " + a.Category; m.appSearching && rw-4-labelW-len(flags)-lipgloss.Width(cat) >= 24 {
			flags += cat
		}
		text := padRight(truncate(a.Label, labelW-2), labelW) + truncate(a.Description, rw-4-labelW-lipgloss.Width(flags))
		if i == m.appIdx && m.appPane == 1 {
			right = append(right, box+" "+sCursor.Render(padRight(text, rw-4-lipgloss.Width(flags)))+sDim.Render(flags))
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
		head := sBold.Render(a.Label) + sDim.Render(appFlags(a))
		if m.appSearching {
			head += sDim.Render(" · " + a.Category)
		}
		out = append(out, "", head, sSub.Render(truncate(a.Description, w)))
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
	if m.appSearching {
		if len(apps) == 0 {
			out = append(out, "", sSub.Render("Nothing matches; each word has to start a word in an app's name or description."))
		}
		return out, hint("type", "to search", "↑↓", "move", "tab", "tick", "enter", "done")
	}
	return out, hint("x", "tick", "/", "search", "tab", "switch side", "↑↓", "move", "enter", "next step", "esc", "back")
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
