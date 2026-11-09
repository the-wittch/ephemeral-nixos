package main

import tea "github.com/charmbracelet/bubbletea"

type choice struct{ value, label, desc, tag string }

func (m model) choices(s stepID) []choice {
	switch s {
	case stDesktop:
		return []choice{
			{"plasma", "Plasma", "KDE Plasma 6, tiling with Krohnkite, colors from the wallpaper", ""},
			{"gnome", "GNOME", "GNOME", ""},
			{"niri", "niri", "scrolling tiles, with the Noctalia shell", ""},
			{"umbriel", "Umbriel", "a new compositor, with the Noctalia shell", "may change"},
		}
	case stGraphics:
		return []choice{
			{"open", "open", "RTX 20 / GTX 16 series and newer", ""},
			{"legacy", "legacy", "GTX 900 / GTX 10 series", ""},
			{"none", "none", "keep the open-source nouveau driver", ""},
		}
	case stType:
		return []choice{
			{"full", "Full", "dotfiles, the theme, snapshots, OpenSnitch, Syncthing, Podman, Flatpak, and the options on the next screen", ""},
			{"minimal", "Small", "a plain desktop now; run ephemeral-setup for the rest later", ""},
		}
	}
	return nil
}

func (m *model) answerFor(s stepID) *string {
	switch s {
	case stDesktop:
		return &m.a.Desktop
	case stGraphics:
		return &m.a.Nvidia
	case stType:
		return &m.a.Profile
	}
	return nil
}

func (m model) choiceCursor(s stepID) int {
	cur := m.answerFor(s)
	for i, c := range m.choices(s) {
		if c.value == *cur {
			return i
		}
	}
	return 0
}

func (m model) keyChoice(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	cs := m.choices(m.step)
	cur := m.choiceCursor(m.step)
	switch k.String() {
	case "up":
		if cur > 0 {
			cur--
		}
	case "down":
		if cur < len(cs)-1 {
			cur++
		}
	case "enter":
		return m, m.next()
	}
	*m.answerFor(m.step) = cs[cur].value
	return m, nil
}

func (m model) labelFor(s stepID) string {
	cur := *m.answerFor(s)
	for _, c := range m.choices(s) {
		if c.value == cur {
			return c.label
		}
	}
	return cur
}

func (m model) viewChoice(w int) ([]string, string) {
	out := m.header(w)
	switch m.step {
	case stDesktop:
		out = append(out, sBold.Render("Which desktop?"), "")
	case stGraphics:
		out = append(out, sBold.Render("Found an NVIDIA card:")+" "+sText.Render(m.gpu), "")
	case stType:
		out = append(out, sBold.Render("How much to install?"), "")
	}
	cur := m.choiceCursor(m.step)
	for i, c := range m.choices(m.step) {
		tag := ""
		if c.tag != "" {
			tag = "  " + sTag.Render(" "+c.tag+" ")
		}
		switch {
		case m.roomy() && i == cur:
			out = append(out, sCursor.Render(" "+g.cursor+" "+c.label+" ")+tag, "     "+sSub.Render(truncate(c.desc, w-5)), "")
		case m.roomy():
			out = append(out, "   "+sText.Render(c.label)+tag, "     "+sSub.Render(truncate(c.desc, w-5)), "")
		case i == cur:
			paren := ""
			if c.tag != "" {
				paren = "  (" + c.tag + ")"
			}
			out = append(out, row(true, w, truncate(padRight(c.label, 9)+c.desc+paren, w-4)))
		default:
			out = append(out, "   "+padRight(c.label, 9)+sSub.Render(truncate(c.desc, w-16))+tag)
		}
	}
	if !m.roomy() {
		out = append(out, "")
	}
	switch m.step {
	case stDesktop:
		out = append(out, sSub.Render("You can switch later in settings.json; your files stay."))
	case stGraphics:
		out = append(out, sSub.Render("The driver is compiled during the install, which adds a few minutes."))
	case stType:
		out = append(out, sSub.Render("Small can be turned into Full at any time with ephemeral-setup."))
	}
	return out, hint("enter", "choose", "↑↓", "move", "esc", "back")
}
