package main

import (
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type reviewLine struct {
	label, value string
	step         stepID
	danger       bool
}

func (m model) reviewLines() []reviewLine {
	var d disk
	for _, x := range m.disks {
		if x.Path == m.a.Disk {
			d = x
		}
	}
	diskVal := strings.TrimSpace(m.a.Disk + "  " + d.Model + "  " + humanSize(d.Size))
	lines := []reviewLine{{"DISK", diskVal + "  erased", stDisk, true}}
	lines = append(lines, reviewLine{"DESKTOP", m.labelFor(stDesktop), stDesktop, false})
	if m.hasGPU {
		lines = append(lines, reviewLine{"GRAPHICS", "NVIDIA, " + m.a.Nvidia + " driver", stGraphics, false})
	}
	region := m.a.Layout
	if m.a.Variant != "" {
		region += " (" + m.a.Variant + ")"
	}
	lines = append(lines, reviewLine{"REGION", region + " · " + m.a.Locale + " · " + m.a.Timezone, stRegion, false})
	if m.a.Profile == "full" {
		var opts []string
		if m.a.Extras {
			opts = append(opts, "extras")
		}
		if m.a.OpenSnitch {
			opts = append(opts, "OpenSnitch")
		}
		if m.a.PassAgent {
			opts = append(opts, "Pass SSH agent ("+m.a.Vault+")")
		}
		switch i := m.a.dnsChoice(); {
		case m.a.DNS == "custom":
			opts = append(opts, "DNS over TLS to "+strings.Join(m.a.CustomDNS, ", "))
		case i > 0:
			opts = append(opts, dnsChoices[i].label+" DNS")
		}
		if m.a.Fingerprint {
			opts = append(opts, "fingerprint login")
		}
		v := "Full"
		if len(opts) > 0 {
			v += ", with " + strings.Join(opts, ", ")
		}
		lines = append(lines, reviewLine{"INSTALL", v, stOptions, false})
		apps := "none"
		if len(m.a.Apps) > 0 {
			var names []string
			for _, a := range m.catalog {
				if m.picked[a.ID] {
					names = append(names, a.Label)
				}
			}
			apps = strings.Join(names, ", ")
		}
		lines = append(lines, reviewLine{"APPS", apps, stApps, false})
	} else {
		lines = append(lines, reviewLine{"INSTALL", "Small; the rest later with ephemeral-setup", stType, false})
	}
	lines = append(lines,
		reviewLine{"MACHINE", m.a.Hostname + " · " + m.a.Username + " · " + m.a.Swap + " swap", stMachine, false},
		reviewLine{"SECRETS", "disk passphrase and login password set", stSecrets, false},
	)
	return lines
}

func (m model) keyReview(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	lines := m.reviewLines()
	switch k.String() {
	case "tab", "shift+tab":
		m.reviewFocus = 1 - m.reviewFocus
		if m.reviewFocus == 1 {
			return m, m.confirmIn.Focus()
		}
		m.confirmIn.Blur()
		return m, nil
	case "up":
		m.reviewFocus = 0
		m.confirmIn.Blur()
		if m.reviewSel > 0 {
			m.reviewSel--
		}
		return m, nil
	case "down":
		if m.reviewFocus == 0 && m.reviewSel < len(lines)-1 {
			m.reviewSel++
			return m, nil
		}
		m.reviewFocus = 1
		return m, m.confirmIn.Focus()
	case "enter":
		if m.reviewFocus == 0 {
			m.editing = true
			return m, m.goTo(lines[m.reviewSel].step)
		}
		if m.confirmIn.Value() != filepath.Base(m.a.Disk) {
			m.note = "Type " + filepath.Base(m.a.Disk) + " exactly to erase it."
			return m, nil
		}
		if !m.cfg.dryRun {
			id, err := diskIdentity(m.a.Disk)
			if err != nil {
				m.note = "Couldn't read " + m.a.Disk + ": " + err.Error()
				return m, nil
			}
			m.a.DiskID = id
		}
		return m.startInstall()
	}
	if m.reviewFocus == 1 {
		var cmd tea.Cmd
		m.confirmIn, cmd = m.confirmIn.Update(k)
		return m, cmd
	}
	return m, nil
}

func (m model) viewReview(w int) ([]string, string) {
	out := m.header(w)
	out = append(out, sBold.Render("Check everything.")+" "+sSub.Render("Nothing has been written yet. Pick a line to change it."), "")
	lines := m.reviewLines()
	// the header, the lines, the confirmation box and the dry-run line
	spaced := m.roomy() && 5+2*len(lines)+5 <= m.bodyRows()
	for i, l := range lines {
		val := truncate(l.value, w-16)
		if l.danger {
			val = sRed.Render(val)
		}
		line := sDim.Render(padRight(l.label, 10)) + val
		if m.reviewFocus == 0 && i == m.reviewSel {
			out = append(out, sSel.Render(" "+g.cursor+" ")+line+"   "+sDim.Render("enter to change"))
		} else {
			out = append(out, "   "+line)
		}
		if spaced {
			out = append(out, "")
		}
	}
	name := filepath.Base(m.a.Disk)
	prompt := "Type " + sRed.Bold(true).Render(name) + " to erase it and install:  "
	box := lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(cDim).Padding(0, 1).Width(w - 2)
	if m.reviewFocus == 1 {
		box = box.BorderForeground(cMauve)
	}
	if !spaced {
		out = append(out, "")
	}
	out = append(out, sSub.Render(truncate(m.a.Disk+" becomes a 1G boot partition and the rest encrypted, with a "+m.a.Swap+" swap file inside.", w)))
	out = append(out, strings.Split(box.Render(prompt+m.confirmIn.View()), "\n")...)
	if m.cfg.dryRun {
		out = append(out, sYellow.Render("Dry run: install.sh won't run."))
	}
	return out, hint("↑↓", "pick a line", "enter", "change it / install", "tab", "switch", "esc", "back")
}
