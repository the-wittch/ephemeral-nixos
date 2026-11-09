package main

import (
	"os/exec"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var welcomeItems = []string{"Install", "Connect to Wi-Fi", "Open a shell"}

func (m model) keyWelcome(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	cur := m.listCursor[stWelcome]
	switch k.String() {
	case "left", "up", "shift+tab":
		if cur > 0 {
			cur--
		}
	case "right", "down", "tab":
		if cur < len(welcomeItems)-1 {
			cur++
		}
	case "q":
		m.quitTo = "shell"
		return m, tea.Quit
	case "enter":
		switch cur {
		case 0:
			if m.isOnline == nil || !*m.isOnline {
				m.note = "Connect to the internet first. The install downloads the system."
				return m, nil
			}
			if !m.uefi {
				m.note = "This computer started in legacy BIOS mode. Restart it in UEFI mode to install."
				return m, nil
			}
			// the list builds the config, so only once an install is under way
			return m, tea.Batch(m.goTo(stDisk), loadCatalogCmd(m.cfg))
		case 1:
			return m, tea.ExecProcess(exec.Command("nmtui"), func(error) tea.Msg { return execDoneMsg{} })
		case 2:
			m.quitTo = "shell"
			return m, tea.Quit
		}
	}
	m.listCursor[stWelcome] = cur
	return m, nil
}

func (m model) viewWelcome(w int) ([]string, string) {
	var out []string
	out = append(out, strings.Split(renderWord(), "\n")...)
	out = append(out, sSub.Render("NixOS on an encrypted disk, with a root that's wiped on every boot"), "")

	var status string
	switch {
	case m.isOnline == nil:
		status = m.spin() + sText.Render(" Checking the internet connection")
	case *m.isOnline:
		status = sGreen.Render(g.dot) + sText.Render(" Online")
	default:
		status = sYellow.Render(g.dot) + sText.Render(" Not connected. Connect to Wi-Fi or plug in a cable.")
	}
	var vers []string
	if m.cfg.version != "" {
		vers = append(vers, "installer "+m.cfg.version)
	}
	if m.configVer != "" {
		vers = append(vers, "config "+m.configVer)
	}
	ver := strings.Join(vers, " · ")
	out = append(out, status+spaces(max(w-lipgloss.Width(status)-lipgloss.Width(ver), 2))+sDim.Render(ver))
	facts := []string{}
	if m.uefi {
		facts = append(facts, g.dot+" UEFI")
	} else {
		facts = append(facts, g.warn+" not UEFI")
	}
	if m.memGiB > 0 {
		facts = append(facts, g.dot+" "+strconv.Itoa(m.memGiB)+" GiB memory")
	}
	out = append(out, sDim.Render(strings.Join(facts, "   ")), "")

	var items []string
	offline := m.isOnline != nil && !*m.isOnline
	for i, it := range welcomeItems {
		switch {
		case i == m.listCursor[stWelcome]:
			items = append(items, sCursor.Render(" "+g.cursor+" "+it+" "))
		case i == 0 && offline:
			items = append(items, " "+sDim.Render(it+" (needs internet)")+" ")
		default:
			items = append(items, " "+sText.Render(it)+" ")
		}
	}
	out = append(out, strings.Join(items, "   "), "")
	if m.cfg.dryRun {
		out = append(out, sYellow.Render("Dry run: nothing is written to any disk."))
	} else {
		out = append(out, sSub.Render("Nothing is written to disk until you confirm on the review screen."))
	}
	if m.logo != nil {
		logo := strings.Split(renderLogo(m.logo), "\n")
		if len(logo)+1+len(out) <= m.bodyRows() {
			out = append(append(logo, ""), out...)
		}
	}
	return out, hint("enter", "select", "←→", "move", "q", "quit")
}
