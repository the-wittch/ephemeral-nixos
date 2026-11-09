package main

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

const frameWidth = 100

func (m model) View() string {
	if m.width == 0 {
		return ""
	}
	if m.width < 80 || m.height < 24 {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
			sText.Render(fmt.Sprintf("Make the window at least 80×24 (it's %d×%d).", m.width, m.height)))
	}
	w := min(m.width-4, frameWidth)
	var body []string
	var hints string
	switch m.step {
	case stWelcome:
		body, hints = m.viewWelcome(w)
	case stDisk:
		body, hints = m.viewDisk(w)
	case stDesktop, stGraphics, stType:
		body, hints = m.viewChoice(w)
	case stRegion:
		body, hints = m.viewRegion(w)
	case stOptions:
		body, hints = m.viewOptions(w)
	case stApps:
		body, hints = m.viewApps(w)
	case stMachine:
		body, hints = m.viewMachine(w)
	case stSecrets:
		body, hints = m.viewSecrets(w)
	case stReview:
		body, hints = m.viewReview(w)
	case stInstalling:
		body, hints = m.viewInstalling(w)
	case stDone, stFailed:
		body, hints = m.viewEnd(w)
	}
	if m.note != "" {
		body = append(body, "", sYellow.Render(g.warn+" ")+sText.Render(m.note))
	}
	h := min(m.height-2, 34)
	if len(body) > h-2 {
		body = body[:h-2]
	}
	for len(body) < h-2 {
		body = append(body, "")
	}
	body = append(body, "", hints)
	content := lipgloss.NewStyle().Width(w).Render(strings.Join(body, "\n"))
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

func hint(pairs ...string) string {
	var parts []string
	for i := 0; i+1 < len(pairs); i += 2 {
		parts = append(parts, sDim.Render(pairs[i])+" "+sSub.Render(pairs[i+1]))
	}
	return strings.Join(parts, "   ")
}

func (m model) header(w int) []string {
	steps := m.visible()
	n := 0
	for i, s := range steps {
		if s == m.step {
			n = i + 1
		}
	}
	left := sTitle.Render("Ephemeral Nix")
	right := sDim.Render(fmt.Sprintf("%d of %d · %s", n, len(steps), stepName[m.step]))
	gap := max(w-lipgloss.Width(left)-lipgloss.Width(right), 1)
	done := w * n / max(len(steps), 1)
	bar := sMauve.Render(strings.Repeat(g.barOn, done)) + sDim.Render(strings.Repeat(g.barOff, w-done))
	return []string{left + spaces(gap) + right, bar, ""}
}

func (m model) spin() string { return sMauve.Render(g.spinner[m.spinFrame%len(g.spinner)]) }

func row(selected bool, w int, text string) string {
	if selected {
		return sCursor.Render(padRight(" "+g.cursor+" "+text, w))
	}
	return "   " + text
}

func (m model) viewWelcome(w int) ([]string, string) {
	var out []string
	if m.logo != nil && m.height >= 34 {
		out = append(out, strings.Split(renderLogo(m.logo), "\n")...)
		out = append(out, "")
	}
	out = append(out, strings.Split(renderWord(), "\n")...)
	out = append(out, sSub.Render("NixOS on an encrypted disk, with a root that's wiped on every boot"), "")

	var status string
	switch {
	case m.isOnline == nil:
		status = m.spin() + sText.Render(" Checking the internet connection")
	case *m.isOnline:
		status = sGreen.Render(g.dot) + sText.Render(" Online")
	default:
		status = sYellow.Render(g.dot) + sText.Render(" No internet connection yet. Set one up to install.")
	}
	ver := "config " + m.configVer
	if m.cfg.version != "" {
		ver = "installer " + m.cfg.version + " · " + ver
	}
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
	for i, it := range welcomeItems {
		if i == m.listCursor[stWelcome] {
			items = append(items, sCursor.Render(" "+g.cursor+" "+it+" "))
		} else {
			items = append(items, " "+sText.Render(it)+" ")
		}
	}
	out = append(out, strings.Join(items, "   "), "")
	if m.cfg.dryRun {
		out = append(out, sYellow.Render("Dry run: nothing is written to any disk."))
	} else {
		out = append(out, sSub.Render("Nothing is written to disk until you confirm on the review screen."))
	}
	return out, hint("enter", "select", "←→", "move", "q", "quit")
}

func (m model) viewDisk(w int) ([]string, string) {
	out := m.header(w)
	out = append(out, sBold.Render("Which disk should NixOS go on?")+" "+sSub.Render("Everything on it is erased."), "")
	if m.diskErr != nil {
		out = append(out, sRed.Render(g.failed+" Couldn't list the disks: "+m.diskErr.Error()))
		return out, hint("esc", "back")
	}
	out = append(out, sDim.Render("   "+padRight("DEVICE", 18)+padRight("SIZE", 9)+padRight("TYPE", 7)+"MODEL"))
	cur := m.listCursor[stDisk]
	for i, d := range m.disks {
		line := padRight(d.Path, 18) + padRight(humanSize(d.Size), 9) + padRight(d.Tran, 7) + truncate(d.Model, 30)
		switch {
		case !m.diskUsable(d):
			out = append(out, sDim.Render("   "+line+"  in use"))
		default:
			out = append(out, row(i == cur, w, line))
		}
	}
	if cur < 0 {
		out = append(out, "", sYellow.Render(g.warn+" Every disk is in use. Unmount the one to install on, or start with FORCE_DISK=1."))
		return out, hint("esc", "back")
	}
	d := m.disks[cur]
	out = append(out, "", sSub.Render("On "+d.Path+" now"))
	if len(d.Partitions) == 0 {
		out = append(out, sDim.Render("   no partitions"))
	}
	for _, p := range d.Partitions {
		line := "   " + padRight(p.Name, 14) + padRight(humanSize(p.Size), 9) + padRight(p.FSType, 8) + p.Label
		out = append(out, sText.Render(line)+"  "+sYellow.Render(g.warn+" erased"))
	}
	out = append(out, "", sSub.Render("Dual boot isn't supported. Back up anything you want to keep first."))
	return out, hint("enter", "choose", "↑↓", "move", "esc", "back")
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
		label := padRight(c.label, 9) + c.desc
		if i == cur {
			if c.tag != "" {
				label += "  (" + c.tag + ")"
			}
			out = append(out, row(true, w, truncate(label, w-4)))
		} else {
			line := padRight(c.label, 9) + sSub.Render(truncate(c.desc, w-16))
			if c.tag != "" {
				line += "  " + sTag.Render(" "+c.tag+" ")
			}
			out = append(out, "   "+line)
		}
	}
	out = append(out, "")
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

func (m model) viewRegion(w int) ([]string, string) {
	out := m.header(w)
	for i, p := range m.region {
		label := padRight(p.label, 18)
		if i == m.regionFocus {
			out = append(out, sMauve.Render(label)+p.input.View())
			for _, l := range p.listView(w - 18) {
				out = append(out, spaces(18)+l)
			}
			if m.regionErr != "" {
				out = append(out, spaces(18)+sRed.Render(g.failed+" "+m.regionErr))
			}
			continue
		}
		out = append(out, sText.Render(label)+sText.Render(p.displayValue()))
	}
	out = append(out, "")
	if m.region[0].value != "us" {
		out = append(out, sYellow.Render(g.warn+" ")+sSub.Render("At boot you type the disk passphrase with "+m.region[0].value+". If this screen's keyboard"),
			sSub.Render("  isn't "+m.region[0].value+" yet, pick a passphrase that types the same on both."))
	}
	return out, hint("type", "to search", "tab", "next field", "enter", "next step", "esc", "back")
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

func (m model) viewApps(w int) ([]string, string) {
	out := m.header(w)
	if m.catalogLoading {
		out = append(out, m.spin()+sText.Render(" Loading the app list. The first time takes a few minutes."), "",
			sSub.Render("You can skip this and pick apps after the install with ephemeral-apps."))
		return out, hint("enter", "skip", "esc", "back")
	}
	if m.catalogErr != nil || len(m.catalog) == 0 {
		out = append(out, sYellow.Render(g.warn+" ")+sText.Render("Couldn't load the app list. Pick apps after the install with ephemeral-apps."))
		return out, hint("enter", "next step", "esc", "back")
	}
	count := 0
	for _, on := range m.picked {
		if on {
			count++
		}
	}
	intro := sSub.Render("Optional. Everything here can be added later with ephemeral-apps.")
	picked := sMauve.Render(fmt.Sprintf("%d picked", count))
	out = append(out, intro+spaces(max(w-lipgloss.Width(intro)-lipgloss.Width(picked), 2))+picked, "")

	cats := m.appCategories()
	apps := m.appsIn(cats[m.appCat])
	var left []string
	for i, c := range cats {
		n := 0
		for _, a := range m.appsIn(c) {
			if m.picked[a.ID] {
				n++
			}
		}
		count := ""
		if n > 0 {
			count = strconv.Itoa(n)
		}
		text := padRight(c, 16) + padLeft(count, 2)
		switch {
		case i == m.appCat && m.appPane == 0:
			left = append(left, sCursor.Render(" "+g.cursor+" "+text+" "))
		case i == m.appCat:
			left = append(left, sMauve.Render(" "+g.cursor+" "+text+" "))
		default:
			left = append(left, "   "+sText.Render(text)+" ")
		}
	}
	rw := w - 26
	var right []string
	for i, a := range apps {
		box := sDim.Render(g.unchecked)
		if m.picked[a.ID] {
			box = sGreen.Render(g.checked)
		}
		flags := ""
		if a.Unfree {
			flags += " unfree"
		}
		if a.Flatpak {
			flags += " flathub"
		}
		text := padRight(a.Label, 18) + truncate(a.Description, rw-30)
		if i == m.appIdx && m.appPane == 1 {
			right = append(right, box+" "+sCursor.Render(padRight(text, rw-8))+sDim.Render(flags))
			if a.Warn != "" {
				right = append(right, "    "+sYellow.Render(g.warn+" "+truncate(a.Warn, rw-8)))
			}
		} else {
			right = append(right, box+" "+sText.Render(text)+sDim.Render(flags))
		}
	}
	for i := 0; i < max(len(left), len(right)); i++ {
		l, r := "", ""
		if i < len(left) {
			l = left[i]
		}
		if i < len(right) {
			r = right[i]
		}
		out = append(out, padRight(l, 26)+r)
	}
	return out, hint("space", "tick", "tab", "switch side", "↑↓", "move", "enter", "next step", "esc", "back")
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
	if m.a.Layout != "us" {
		out = append(out, "   "+sSub.Render("At boot you type it with the "+m.a.Layout+" layout."))
	}
	out = append(out, "   "+sYellow.Render(g.warn+" ")+sSub.Render("It can't be recovered. Keep it somewhere other than this computer."), "")
	out = append(out, sBold.Render("Login password")+"  "+sSub.Render("for "+m.a.Username))
	out = append(out, field(2, "Password"))
	out = append(out, field(3, "Again")+status(we, m.secretIn[3].Value()))
	return out, hint("tab", "next field", "enter", "next step", "esc", "back")
}

func (m model) viewReview(w int) ([]string, string) {
	out := m.header(w)
	out = append(out, sBold.Render("Check everything.")+" "+sSub.Render("Nothing has been written yet. Pick a line to change it."), "")
	for i, l := range m.reviewLines() {
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
	}
	name := filepath.Base(m.a.Disk)
	prompt := "Type " + sRed.Bold(true).Render(name) + " to erase it and install:  "
	box := lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(cDim).Padding(0, 1).Width(w - 4)
	if m.reviewFocus == 1 {
		box = box.BorderForeground(cMauve)
	}
	out = append(out, "")
	out = append(out, strings.Split(box.Render(prompt+m.confirmIn.View()), "\n")...)
	if m.cfg.dryRun {
		out = append(out, sYellow.Render("Dry run: install.sh won't run."))
	}
	return out, hint("↑↓", "pick a line", "enter", "change it / install", "tab", "switch", "esc", "back")
}

func (m model) installHeader(w int, state string) string {
	left := sTitle.Render("Ephemeral Nix")
	right := sDim.Render(state + " · " + formatDuration(m.elapsed()))
	return left + spaces(max(w-lipgloss.Width(left)-lipgloss.Width(right), 1)) + right
}

func (m model) stepList(failedAt int) []string {
	var out []string
	for i, s := range installSteps {
		switch {
		case failedAt >= 0 && i == failedAt:
			out = append(out, "   "+sRed.Render(g.failed)+" "+sBold.Render(s))
		case i < m.curStep || (m.step == stDone):
			out = append(out, "   "+sGreen.Render(g.done)+" "+sText.Render(s))
		case i == m.curStep && failedAt < 0:
			out = append(out, "   "+m.spin()+" "+sBold.Render(s))
		default:
			out = append(out, "   "+sDim.Render(g.pending+" "+s))
		}
	}
	return out
}

func (m model) viewInstalling(w int) ([]string, string) {
	out := []string{m.installHeader(w, "installing"), ""}
	if m.showLog {
		out = append(out, strings.Split(m.logView.View(), "\n")...)
		return out, hint("↑↓", "scroll", "l", "back to the steps")
	}
	out = append(out, sBold.Render("Installing on "+m.a.Disk), "")
	out = append(out, m.stepList(-1)...)
	out = append(out, "", sDim.Render("latest output "+strings.Repeat(g.rule, max(w-15, 0))))
	start := max(len(m.lines)-6, 0)
	for i, l := range m.lines[start:] {
		st := sDim
		if i == len(m.lines[start:])-1 {
			st = sSub
		}
		out = append(out, st.Render(truncate(l, w)))
	}
	return out, hint("l", "full log") + "   " + sSub.Render("The output is also saved to /tmp/ephemeral-install.log")
}

func (m model) viewEnd(w int) ([]string, string) {
	var out []string
	if m.showLog {
		out = []string{m.installHeader(w, "log"), ""}
		out = append(out, strings.Split(m.logView.View(), "\n")...)
		return out, hint("↑↓", "scroll", "esc", "back")
	}
	if m.step == stDone {
		out = []string{m.installHeader(w, "installed"), ""}
		out = append(out, sGreen.Bold(true).Render(g.done+" Installed.")+" "+sText.Render("Remove the USB stick and reboot."), "")
		out = append(out, "   "+sText.Render("At boot, unlock the disk with your passphrase, then log in as "+m.a.Username+"."), "")
		out = append(out, sSub.Render("After the first login"))
		for _, c := range [][2]string{
			{"ephemeral-apps", "pick more apps"},
			{"ephemeral-setup", "change the options from this install"},
			{"ephemeral-help", "every command this system adds"},
		} {
			out = append(out, "   "+sBlue.Render(padRight(c[0], 18))+sText.Render(c[1]))
		}
		out = append(out, "", sSub.Render("The config is in /persist/etc/nixos. A read-only copy of this install is kept"),
			sSub.Render("in /persist/etc/nixos-install to rebuild back to."), "")
	} else {
		out = []string{m.installHeader(w, "stopped"), ""}
		out = append(out, sRed.Bold(true).Render(g.failed+" The install stopped at:")+" "+sText.Render(installSteps[m.curStep]), "")
		if l := lastError(m.lines); l != "" {
			out = append(out, sRed.Render(truncate(l, w)))
		}
		if m.installErr != nil {
			out = append(out, sDim.Render(truncate(m.installErr.Error(), w)))
		}
		out = append(out, "", sSub.Render("Your answers are kept. Trying again formats the disk again from the start."), "")
		out = append(out, m.stepList(m.curStep)...)
		out = append(out, "")
	}
	for i, it := range m.endItems() {
		out = append(out, row(i == m.endSel, 28, it))
	}
	if m.step == stFailed {
		out = append(out, "", sSub.Render("To report it, open an issue at github.com/the-wittch/ephemeral-nixos and attach"),
			sSub.Render("/tmp/ephemeral-install.log. It has no passwords or passphrases in it."))
	}
	return out, hint("enter", "select", "↑↓", "move")
}

func lastError(lines []string) string {
	for i := len(lines) - 1; i >= 0; i-- {
		l := strings.ToLower(lines[i])
		if strings.HasPrefix(l, "error") || strings.Contains(l, "error:") {
			return lines[i]
		}
	}
	return ""
}

func formatDuration(d time.Duration) string {
	m := int(d.Minutes())
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%d:%02d", m, s)
}

func spaces(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat(" ", n)
}

func padRight(s string, n int) string { return s + spaces(n-lipgloss.Width(s)) }

func padLeft(s string, n int) string { return spaces(n-lipgloss.Width(s)) + s }

func truncate(s string, n int) string {
	if n <= 1 || lipgloss.Width(s) <= n {
		return s
	}
	r := []rune(s)
	for len(r) > 0 && lipgloss.Width(string(r))+1 > n {
		r = r[:len(r)-1]
	}
	return string(r) + "…"
}

func itoa(n int) string { return strconv.Itoa(n) }
