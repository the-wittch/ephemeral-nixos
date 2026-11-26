package main

import (
	"errors"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var secretLabels = []string{"Passphrase", "Again", "Password", "Again"}

// what's wrong with one field on its own, so enter can stop there
func (m model) secretErr(i int) error {
	v := func(i int) string { return m.secretIn[i].Value() }
	switch i {
	case 0:
		if err := checkPassphrase(v(0), ""); err != nil {
			return err
		}
		return m.handheldErr(v(0))
	case 1:
		if v(1) == "" {
			return errors.New("type the passphrase again")
		}
		if err := checkPassphrase(v(0), v(1)); err != nil {
			return err
		}
		return m.handheldErr(v(0))
	case 2:
		return checkPassword(v(2), "")
	default:
		if v(3) == "" {
			return errors.New("type the password again")
		}
		return checkPassword(v(2), v(3))
	}
}

func (m *model) focusSecret() tea.Cmd {
	var cmd tea.Cmd
	for i := range m.secretIn {
		if i == m.secretFocus {
			cmd = m.secretIn[i].Focus()
		} else {
			m.secretIn[i].Blur()
		}
	}
	return cmd
}

func (m model) keySecrets(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	last := len(m.secretIn) - 1
	switch k.String() {
	case "up", "shift+tab":
		if m.secretFocus > 0 {
			m.secretFocus--
		}
		return m, m.focusSecret()
	case "down", "tab":
		if m.secretFocus < last {
			m.secretFocus++
		}
		return m, m.focusSecret()
	case "enter":
		if err := m.secretErr(m.secretFocus); err != nil {
			m.note = secretLabels[m.secretFocus] + ": " + err.Error()
			return m, nil
		}
		if m.secretFocus < last {
			m.secretFocus++
			return m, m.focusSecret()
		}
		for i := range m.secretIn {
			if err := m.secretErr(i); err != nil {
				m.secretFocus = i
				m.note = secretLabels[i] + ": " + err.Error()
				return m, m.focusSecret()
			}
		}
		m.a.Passphrase, m.a.Password = m.secretIn[0].Value(), m.secretIn[2].Value()
		return m, m.next()
	}
	var cmd tea.Cmd
	m.secretIn[m.secretFocus], cmd = m.secretIn[m.secretFocus].Update(k)
	return m, cmd
}

func (m model) viewSecrets(w int) ([]string, string) {
	out := m.header(w)
	help := []string{
		"type the passphrase; it stays hidden",
		"type it again to check it",
		"type the password you'll log in with",
		"type it again to check it",
	}
	field := func(i int) []string {
		var err error
		// a field is only marked wrong once something's typed in it, and the second copy not while
		// it's still on its way to matching
		v := m.secretIn[i].Value()
		if v != "" {
			err = m.secretErr(i)
		}
		if (i == 1 || i == 3) && i == m.secretFocus && strings.HasPrefix(m.secretIn[i-1].Value(), v) {
			err = nil
		}
		lines := m.field(w, i == m.secretFocus, secretLabels[i], m.secretIn[i].View(), help[i], err)
		if (i == 1 || i == 3) && v != "" && m.secretErr(i) == nil {
			lines[0] += "  " + sGreen.Render(g.done+" matches")
		}
		return lines
	}
	out = append(out, sBold.Render("Disk passphrase")+"  "+sSub.Render("unlocks the disk at every boot"))
	out = append(out, m.gap()...)
	out = append(out, field(0)...)
	out = append(out, field(1)...)
	out = append(out, m.gap()...)
	switch {
	case m.liveLayout == m.a.Layout && m.a.Layout != "us":
		out = append(out, "   "+sGreen.Render(g.done)+" "+sSub.Render("This screen types with "+m.a.Layout+", like the prompt at boot."))
	case m.a.Layout != "us":
		out = append(out, "   "+sYellow.Render(g.warn+" ")+sSub.Render("This screen can't switch to "+m.a.Layout+", but you'll type it with "+m.a.Layout+" at boot."),
			"     "+sSub.Render("Use characters that are in the same place on both keyboards."))
	}
	if m.a.Desktop == "handheld" {
		switch {
		case m.padTyping:
			out = append(out, "   "+sCursor.Render(" Typing with the controller ")+"  "+
				sText.Render("press your buttons, ")+sBold.Render("View")+sText.Render(" deletes, ")+
				sBold.Render("Menu")+sText.Render(" finishes"))
		case m.padUsed && m.onPassphrase():
			out = append(out, "   "+sText.Render("Press ")+sBold.Render("Y")+
				sText.Render(" to type it with the controller, as you will at boot."))
		}
		out = append(out, "   "+sText.Render("Twelve or more presses of these, the only ones the controller has at boot:"))
		out = append(out, buttonTable(usableButtons(rules.handheld, m.vendor), w-5)...)
	} else {
		out = append(out, "   "+sSub.Render("A few random words are easy to type and hard to guess."))
	}
	out = append(out, "   "+sYellow.Render(g.warn+" ")+sSub.Render("It can't be recovered. Keep it somewhere other than this computer."), "")
	out = append(out, m.gap()...)
	out = append(out, sBold.Render("Login password")+"  "+sSub.Render("for "+m.a.Username))
	out = append(out, m.gap()...)
	out = append(out, field(2)...)
	out = append(out, field(3)...)
	return out, fieldHints(m.secretFocus == len(m.secretIn)-1)
}

func (m model) handheldErr(s string) error {
	if m.a.Desktop != "handheld" {
		return nil
	}
	return checkHandheldPassphrase(s, usableButtons(rules.handheld, m.vendor))
}

// The button table in even columns, each button with what it types, as many columns as fit.
func buttonTable(buttons []handheldButton, w int) []string {
	name := func(b handheldButton) string {
		if b.Only != "" {
			return b.Button + " (" + b.Only + ")"
		}
		return b.Button
	}
	widest := 0
	for _, b := range buttons {
		widest = max(widest, lipgloss.Width(name(b)))
	}
	cell := widest + 6
	cols := max(min((w-5)/cell, 4), 1)
	rows := (len(buttons) + cols - 1) / cols
	var out []string
	for r := range rows {
		line := "     "
		// down each column first, as the docs' table reads
		for c := range cols {
			i := c*rows + r
			if i >= len(buttons) {
				break
			}
			b := buttons[i]
			line += sText.Render(padRight(name(b), widest)) + "  " + sMauve.Bold(true).Render(b.Character) + "   "
		}
		out = append(out, strings.TrimRight(line, " "))
	}
	return out
}
