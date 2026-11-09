package main

import tea "github.com/charmbracelet/bubbletea"

func (m model) secretErrs() (error, error) {
	return checkPassphrase(m.secretIn[0].Value(), m.secretIn[1].Value()),
		checkPassword(m.secretIn[2].Value(), m.secretIn[3].Value())
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
	switch k.String() {
	case "up", "shift+tab":
		if m.secretFocus > 0 {
			m.secretFocus--
		}
		return m, m.focusSecret()
	case "down", "tab":
		if m.secretFocus < len(m.secretIn)-1 {
			m.secretFocus++
		}
		return m, m.focusSecret()
	case "enter":
		if m.secretFocus < len(m.secretIn)-1 {
			m.secretFocus++
			return m, m.focusSecret()
		}
		pe, we := m.secretErrs()
		switch {
		case pe != nil || m.secretIn[1].Value() == "":
			m.secretFocus = 0
			if pe == nil {
				m.secretFocus = 1
			}
			return m, m.focusSecret()
		case we != nil || m.secretIn[3].Value() == "":
			m.secretFocus = 2
			if we == nil {
				m.secretFocus = 3
			}
			return m, m.focusSecret()
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
	pe, we := m.secretErrs()
	field := func(i int, label string) string {
		l := padRight(label, 14)
		if i == m.secretFocus {
			l = sMauve.Render(l)
		} else {
			l = sText.Render(l)
		}
		return "   " + l + m.secretIn[i].View()
	}
	status := func(err error, again string) string {
		switch {
		case again == "":
			return ""
		case err != nil:
			return "  " + sRed.Render(g.failed+" "+err.Error())
		default:
			return "  " + sGreen.Render(g.done+" matches")
		}
	}
	out = append(out, sBold.Render("Disk passphrase")+"  "+sSub.Render("unlocks the disk at every boot"))
	out = append(out, field(0, "Passphrase"))
	if v := m.secretIn[0].Value(); v != "" && len(v) < 8 {
		out = append(out, spaces(17)+sRed.Render(g.failed+" at least 8 characters"))
	}
	out = append(out, field(1, "Again")+status(pe, m.secretIn[1].Value()))
	switch {
	case m.liveLayout == m.a.Layout && m.a.Layout != "us":
		out = append(out, "   "+sGreen.Render(g.done)+" "+sSub.Render("This screen types with "+m.a.Layout+", like the prompt at boot."))
	case m.a.Layout != "us":
		out = append(out, "   "+sYellow.Render(g.warn+" ")+sSub.Render("This screen can't switch to "+m.a.Layout+", but you'll type it with "+m.a.Layout+" at boot."),
			"     "+sSub.Render("Use characters that are in the same place on both keyboards."))
	}
	out = append(out, "   "+sSub.Render("A few random words make a passphrase that's easy to type and hard to guess."))
	out = append(out, "   "+sYellow.Render(g.warn+" ")+sSub.Render("It can't be recovered. Keep it somewhere other than this computer."), "")
	out = append(out, sBold.Render("Login password")+"  "+sSub.Render("for "+m.a.Username))
	out = append(out, field(2, "Password"))
	out = append(out, field(3, "Again")+status(we, m.secretIn[3].Value()))
	return out, hint("tab", "next field", "enter", "next step", "esc", "back")
}
