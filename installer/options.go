package main

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type optRow int

const (
	optExtras optRow = iota
	optOpenSnitch
	optPassAgent
	optVault
	optDNS
	optControlDID
	optCustomDNS
	optFingerprint
)

func (m model) optRows() []optRow {
	rows := []optRow{optExtras, optOpenSnitch, optPassAgent}
	if m.a.PassAgent {
		rows = append(rows, optVault)
	}
	rows = append(rows, optDNS)
	if m.a.ControlD {
		rows = append(rows, optControlDID)
	}
	if m.a.DNS == "custom" {
		rows = append(rows, optCustomDNS)
	}
	if m.hasReader {
		rows = append(rows, optFingerprint)
	}
	return rows
}

// which of dnsChoices the answers are on
func (a answers) dnsChoice() int {
	for i, c := range dnsChoices {
		if c.key == "controld" && a.ControlD || c.key == a.DNS && !a.ControlD {
			return i
		}
	}
	return 0
}

func (a *answers) setDNS(i int) {
	n := len(dnsChoices)
	key := dnsChoices[((i%n)+n)%n].key
	a.ControlD = key == "controld"
	a.DNS = key
	if a.ControlD {
		a.DNS = "none"
	}
}

func (m model) optRow() optRow {
	rows := m.optRows()
	return rows[min(m.optFocus, len(rows)-1)]
}

func (m *model) focusOpt(r optRow) tea.Cmd {
	for i, x := range m.optRows() {
		if x == r {
			m.optFocus = i
		}
	}
	m.vaultIn.Blur()
	m.controldIn.Blur()
	m.customDNSIn.Blur()
	switch r {
	case optVault:
		return m.vaultIn.Focus()
	case optControlDID:
		return m.controldIn.Focus()
	case optCustomDNS:
		return m.customDNSIn.Focus()
	}
	return nil
}

func (m model) optErrs() (vault, controld, custom error) {
	if m.a.PassAgent {
		vault = checkVault(strings.TrimSpace(m.vaultIn.Value()))
	}
	if m.a.ControlD {
		controld = checkControlD(strings.TrimSpace(m.controldIn.Value()))
	}
	if m.a.DNS == "custom" {
		custom = checkCustomDNS(m.customDNSIn.Value())
	}
	return vault, controld, custom
}

func (m model) keyOptions(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	rows := m.optRows()
	m.optFocus = min(m.optFocus, len(rows)-1)
	row := rows[m.optFocus]
	switch k.String() {
	case "up", "shift+tab":
		if m.optFocus > 0 {
			return m, m.focusOpt(rows[m.optFocus-1])
		}
		return m, nil
	case "down", "tab":
		if m.optFocus < len(rows)-1 {
			return m, m.focusOpt(rows[m.optFocus+1])
		}
		return m, nil
	case "enter":
		ve, ce, de := m.optErrs()
		switch {
		case ve != nil:
			m.note = "Vault name: " + ve.Error()
			return m, m.focusOpt(optVault)
		case ce != nil:
			m.note = "Resolver id: " + ce.Error()
			return m, m.focusOpt(optControlDID)
		case de != nil:
			m.note = "DNS servers: " + de.Error()
			return m, m.focusOpt(optCustomDNS)
		}
		if m.a.PassAgent {
			m.a.Vault = strings.TrimSpace(m.vaultIn.Value())
		}
		if m.a.ControlD {
			m.a.ControlDID = strings.TrimSpace(m.controldIn.Value())
		}
		if m.a.DNS == "custom" {
			m.a.CustomDNS = strings.Fields(m.customDNSIn.Value())
		}
		return m, m.next()
	}
	var cmd tea.Cmd
	switch row {
	case optVault:
		m.vaultIn, cmd = m.vaultIn.Update(k)
		return m, cmd
	case optControlDID:
		m.controldIn, cmd = m.controldIn.Update(k)
		return m, cmd
	case optCustomDNS:
		m.customDNSIn, cmd = m.customDNSIn.Update(k)
		return m, cmd
	}
	if row == optDNS {
		switch k.String() {
		case "x", " ", "right", "l":
			m.a.setDNS(m.a.dnsChoice() + 1)
		case "left", "h":
			m.a.setDNS(m.a.dnsChoice() - 1)
		}
		return m, nil
	}
	if s := k.String(); s != "x" && s != " " {
		return m, nil
	}
	switch row {
	case optExtras:
		m.a.Extras = !m.a.Extras
	case optOpenSnitch:
		m.a.OpenSnitch = !m.a.OpenSnitch
	case optPassAgent:
		m.a.PassAgent = !m.a.PassAgent
		if m.a.PassAgent {
			return m, m.focusOpt(optVault)
		}
	case optFingerprint:
		m.a.Fingerprint = !m.a.Fingerprint
	}
	return m, nil
}

func (m model) viewOptions(w int) ([]string, string) {
	out := m.header(w)
	out = append(out, sBold.Render("What else should the full install have?")+" "+sSub.Render("All of these can be changed later."), "")
	check := func(on bool) string {
		if on {
			return sGreen.Render(g.checked) + " "
		}
		return sDim.Render(g.unchecked) + " "
	}
	ve, ce, de := m.optErrs()
	rows := m.optRows()
	for i, r := range rows {
		focused := i == m.optFocus
		var head, desc string
		var err error
		indent := 4
		switch r {
		case optExtras:
			head, desc = check(m.a.Extras)+sText.Render("Extras"), "Steam, VS Code, Brave, VLC, atuin, comma and more"
		case optOpenSnitch:
			head, desc = check(m.a.OpenSnitch)+sText.Render("OpenSnitch"), "asks before a program connects to the internet"
		case optPassAgent:
			head, desc = check(m.a.PassAgent)+sText.Render("Pass SSH agent"), "Proton Pass hands out your SSH keys after you log in"
		case optVault:
			indent = 8
			head, desc = sText.Render(padRight("Vault name", 13))+m.vaultIn.View(), "the Proton Pass vault that holds your SSH keys"
			if m.vaultIn.Value() != "" {
				err = ve
			}
		case optDNS:
			c := dnsChoices[m.a.dnsChoice()]
			head, desc = sText.Render(padRight("DNS", 13))+sMauve.Render("< ")+sText.Render(c.label)+sMauve.Render(" >"), c.desc
		case optControlDID:
			indent = 8
			head, desc = sText.Render(padRight("Resolver id", 13))+m.controldIn.View(), "the part before .dns.controld.com in your endpoint's DNS-over-TLS address"
			if m.controldIn.Value() != "" {
				err = ce
			}
		case optCustomDNS:
			indent = 8
			head, desc = sText.Render(padRight("Servers", 13))+m.customDNSIn.View(), "address#name, separated by spaces; the name is the one on the server's certificate"
			if m.customDNSIn.Value() != "" {
				err = de
			}
		case optFingerprint:
			head, desc = check(m.a.Fingerprint)+sText.Render("Fingerprint"), "log in and use sudo with "+m.reader
		}
		if focused && (r == optVault || r == optControlDID || r == optCustomDNS) {
			desc = "type " + desc
		}
		cursor := spaces(indent)
		if focused {
			cursor = spaces(indent-3) + sMauve.Render(g.cursor+"  ")
		}
		out = append(out, cursor+head, spaces(indent+4)+sSub.Render(truncate(desc, w-indent-4)))
		if err != nil {
			out = append(out, spaces(indent+4)+sRed.Render(g.failed+" "+err.Error()))
		}
		// a field stays next to the box it belongs to
		if i+1 < len(rows) && (rows[i+1] == optVault || rows[i+1] == optControlDID || rows[i+1] == optCustomDNS) {
			continue
		}
		out = append(out, m.gap()...)
	}
	if m.laptop {
		out = append(out, sSub.Render("Found on this machine"), "    "+sGreen.Render(g.dot)+" "+sText.Render("a battery: laptop extras, thermald and battery reporting"))
	}
	switch m.optRow() {
	case optVault, optControlDID, optCustomDNS:
		return out, hint("type", "to fill it in", "↑↓", "move", "enter", "next step", "esc", "back")
	case optDNS:
		return out, hint("←→", "change", "↑↓", "move", "enter", "next step", "esc", "back")
	}
	return out, hint("x", "tick", "↑↓", "move", "enter", "next step", "esc", "back")
}
