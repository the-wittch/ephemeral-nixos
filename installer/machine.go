package main

import tea "github.com/charmbracelet/bubbletea"

func (m model) machineErrs() []error {
	return []error{
		checkHostname(m.machineIn[0].Value()),
		checkUsername(m.machineIn[1].Value()),
		checkSwap(m.machineIn[2].Value()),
	}
}

func (m *model) focusMachine() tea.Cmd {
	var cmd tea.Cmd
	for i := range m.machineIn {
		if i == m.machineFocus {
			cmd = m.machineIn[i].Focus()
		} else {
			m.machineIn[i].Blur()
		}
	}
	return cmd
}

func (m model) keyMachine(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch k.String() {
	case "up", "shift+tab":
		if m.machineFocus > 0 {
			m.machineFocus--
		}
		return m, m.focusMachine()
	case "down", "tab":
		if m.machineFocus < len(m.machineIn)-1 {
			m.machineFocus++
		}
		return m, m.focusMachine()
	case "enter":
		for i, err := range m.machineErrs() {
			if err != nil {
				m.machineFocus = i
				return m, m.focusMachine()
			}
		}
		m.a.Hostname, m.a.Username, m.a.Swap = m.machineIn[0].Value(), m.machineIn[1].Value(), m.machineIn[2].Value()
		return m, m.next()
	}
	var cmd tea.Cmd
	m.machineIn[m.machineFocus], cmd = m.machineIn[m.machineFocus].Update(k)
	return m, cmd
}

func (m model) viewMachine(w int) ([]string, string) {
	out := m.header(w)
	labels := []string{"Hostname", "Username", "Swap"}
	notes := []string{"", "", "a swap file on the encrypted disk"}
	errs := m.machineErrs()
	for i, in := range m.machineIn {
		label := padRight(labels[i], 12)
		if i == m.machineFocus {
			label = sMauve.Render(label)
		} else {
			label = sText.Render(label)
		}
		line := "   " + label + in.View()
		if notes[i] != "" {
			line += "  " + sDim.Render(notes[i])
		}
		out = append(out, line)
		if errs[i] != nil && in.Value() != "" {
			out = append(out, spaces(15)+sRed.Render(g.failed+" "+errs[i].Error()))
		}
	}
	out = append(out, "", sSub.Render("These go into settings.json, which you can change after the install."))
	return out, hint("tab", "next field", "enter", "next step", "esc", "back")
}
