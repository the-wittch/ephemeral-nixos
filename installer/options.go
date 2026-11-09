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
	optControlD
	optControlDID
	optFingerprint
)

func (m model) optRows() []optRow {
	rows := []optRow{optExtras, optOpenSnitch, optPassAgent}
	if m.a.PassAgent {
		rows = append(rows, optVault)
	}
	rows = append(rows, optControlD)
	if m.a.ControlD {
		rows = append(rows, optControlDID)
	}
	if m.hasReader {
		rows = append(rows, optFingerprint)
	}
	return rows
}

func (m model) keyOptions(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	rows := m.optRows()
	if m.optFocus >= len(rows) {
		m.optFocus = len(rows) - 1
	}
	row := rows[m.optFocus]
	moveTo := func(i int) tea.Cmd {
		m.optFocus = i
		m.vaultIn.Blur()
		m.controldIn.Blur()
		switch m.optRows()[i] {
		case optVault:
			return m.vaultIn.Focus()
		case optControlDID:
			return m.controldIn.Focus()
		}
		return nil
	}
	switch k.String() {
	case "up", "shift+tab":
		if m.optFocus > 0 {
			return m, moveTo(m.optFocus - 1)
		}
		return m, nil
	case "down", "tab":
		if m.optFocus < len(rows)-1 {
			return m, moveTo(m.optFocus + 1)
		}
		return m, nil
	case "enter":
		if m.a.PassAgent {
			if err := checkVault(strings.TrimSpace(m.vaultIn.Value())); err != nil {
				m.note = "Vault: " + err.Error()
				return m, nil
			}
			m.a.Vault = strings.TrimSpace(m.vaultIn.Value())
		}
		if m.a.ControlD {
			if err := checkControlD(strings.TrimSpace(m.controldIn.Value())); err != nil {
				m.note = "Control D: " + err.Error()
				return m, nil
			}
			m.a.ControlDID = strings.TrimSpace(m.controldIn.Value())
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
	}
	if k.String() == " " {
		switch row {
		case optExtras:
			m.a.Extras = !m.a.Extras
		case optOpenSnitch:
			m.a.OpenSnitch = !m.a.OpenSnitch
		case optPassAgent:
			m.a.PassAgent = !m.a.PassAgent
		case optControlD:
			m.a.ControlD = !m.a.ControlD
		case optFingerprint:
			m.a.Fingerprint = !m.a.Fingerprint
		}
	}
	return m, nil
}

func (m model) viewOptions(w int) ([]string, string) {
	out := m.header(w)
	check := func(on bool) string {
		if on {
			return sGreen.Render(g.checked)
		}
		return sDim.Render(g.unchecked)
	}
	rows := m.optRows()
	for i, r := range rows {
		focused := i == m.optFocus
		var line string
		switch r {
		case optExtras:
			line = check(m.a.Extras) + " " + padRight("Extras", 16) + sSub.Render("Steam, VS Code, Brave, VLC, atuin, comma and more")
		case optOpenSnitch:
			line = check(m.a.OpenSnitch) + " " + padRight("OpenSnitch", 16) + sSub.Render("ask before programs connect out")
		case optPassAgent:
			line = check(m.a.PassAgent) + " " + padRight("Pass SSH agent", 16) + sSub.Render("Proton Pass serves your SSH keys at login")
		case optVault:
			line = "      " + padRight("Vault", 12) + m.vaultIn.View()
		case optControlD:
			line = check(m.a.ControlD) + " " + padRight("Control D", 16) + sSub.Render("DNS over TLS; needs a resolver id")
		case optControlDID:
			line = "      " + padRight("Resolver id", 12) + m.controldIn.View()
		case optFingerprint:
			line = check(m.a.Fingerprint) + " " + padRight("Fingerprint", 16) + sSub.Render("login and sudo with "+m.reader)
		}
		if focused {
			out = append(out, sMauve.Render(" "+g.cursor+" ")+line)
		} else {
			out = append(out, "   "+line)
		}
	}
	if m.laptop {
		out = append(out, "", sSub.Render("Found on this machine"), "   "+sGreen.Render(g.dot)+" "+sText.Render("a battery: laptop extras, thermald and battery reporting"))
	}
	return out, hint("space", "tick", "↑↓", "move", "enter", "next step", "esc", "back")
}
