package main

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) keyRegion(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	p := m.region[m.regionFocus]
	switch k.String() {
	case "tab", "enter", "shift+tab":
		if err := p.commit(); err != nil {
			m.regionErr = err.Error()
			return m, nil
		}
		m.regionErr = ""
		if m.regionFocus == 0 {
			m.syncVariants()
		}
		p.blur()
		if k.String() == "shift+tab" {
			if m.regionFocus > 0 {
				m.regionFocus--
			}
		} else if m.regionFocus < len(m.region)-1 {
			m.regionFocus++
		} else if k.String() == "enter" {
			m.a.Layout, m.a.Variant = m.region[0].value, m.region[1].value
			m.a.Locale, m.a.Timezone = m.region[2].value, m.region[3].value
			m.liveLayout = switchConsoleLayout(m.a.Layout, m.a.Variant, m.cfg.dryRun)
			return m, m.next()
		}
		return m, m.region[m.regionFocus].focus()
	}
	return m, p.update(k)
}

// variants belong to one layout; "us,de" has none to pick from
func (m *model) syncVariants() {
	layout := m.region[0].value
	items := []item{{Code: "", Desc: "default"}}
	if rules, ok := xkbRules(m.cfg.listsDir); ok && !strings.Contains(layout, ",") {
		items = append(items, parseXKB(rules, "variant", layout)...)
	}
	value := m.region[1].value
	found := false
	for _, it := range items {
		if it.Code == value {
			found = true
		}
	}
	if !found {
		value = ""
	}
	m.region[1].setItems(items, value)
}

func (m model) viewRegion(w int) ([]string, string) {
	out := m.header(w)
	for i, p := range m.region {
		label := padRight(p.label, 18)
		if i == m.regionFocus {
			out = append(out, sMauve.Render(" "+g.cursor+" "+label)+p.input.View())
			for _, l := range p.listView(w - 21) {
				out = append(out, spaces(21)+l)
			}
			if m.regionErr != "" {
				out = append(out, spaces(21)+sRed.Render(g.failed+" "+m.regionErr))
			}
		} else {
			out = append(out, "   "+sText.Render(label)+sText.Render(p.displayValue()))
		}
		out = append(out, m.gap()...)
	}
	if !m.roomy() {
		out = append(out, "")
	}
	if m.region[0].value != "us" {
		out = append(out, sSub.Render("At boot you type the disk passphrase with "+m.region[0].value+". The passphrase screen switches to it"),
			sSub.Render("too, where it can, so you type it the same way there."))
	}
	return out, hint("type", "to search", "tab", "next field", "enter", "next step", "esc", "back")
}
