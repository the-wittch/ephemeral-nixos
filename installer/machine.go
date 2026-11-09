package main

import tea "github.com/charmbracelet/bubbletea"

var machineFields = []struct{ label, help string }{
	{"Hostname", "the computer's name on the network, such as my-laptop"},
	{"Username", "the account you log in with"},
	{"Swap", "the size of a swap file on the encrypted disk, such as 4G or 8192M"},
}

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
	last := len(m.machineIn) - 1
	switch k.String() {
	case "up", "shift+tab":
		if m.machineFocus > 0 {
			m.machineFocus--
		}
		return m, m.focusMachine()
	case "down", "tab":
		if m.machineFocus < last {
			m.machineFocus++
		}
		return m, m.focusMachine()
	case "enter":
		errs := m.machineErrs()
		if err := errs[m.machineFocus]; err != nil {
			m.note = machineFields[m.machineFocus].label + ": " + err.Error()
			return m, nil
		}
		if m.machineFocus < last {
			m.machineFocus++
			return m, m.focusMachine()
		}
		for i, err := range errs {
			if err != nil {
				m.machineFocus = i
				m.note = machineFields[i].label + ": " + err.Error()
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
	out = append(out, sBold.Render("About this computer")+"  "+sSub.Render("all of it can be changed in settings.json later"), "")
	errs := m.machineErrs()
	for i, f := range machineFields {
		var err error
		if m.machineIn[i].Value() != "" {
			err = errs[i]
		}
		out = append(out, m.field(w, i == m.machineFocus, f.label, m.machineIn[i].View(), f.help, err)...)
	}
	return out, fieldHints(m.machineFocus == len(m.machineIn)-1)
}
