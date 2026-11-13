package main

import tea "github.com/charmbracelet/bubbletea"

func (m *model) placeDiskCursor() {
	cur := m.listCursor[stDisk]
	for i, d := range m.disks {
		if d.Path == m.a.Disk {
			cur = i
		}
	}
	if cur >= len(m.disks) || (cur < len(m.disks) && !m.diskUsable(m.disks[cur])) {
		cur = -1
		for i, d := range m.disks {
			if m.diskUsable(d) {
				cur = i
				break
			}
		}
	}
	m.listCursor[stDisk] = cur
}

func (m model) diskUsable(d disk) bool {
	return !(d.InUse || d.Installer || d.Member != "") || m.cfg.forceDisk
}

func (m model) keyDisk(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	cur := m.listCursor[stDisk]
	switch k.String() {
	case "up":
		for i := cur - 1; i >= 0; i-- {
			if m.diskUsable(m.disks[i]) {
				cur = i
				break
			}
		}
	case "down":
		for i := cur + 1; i < len(m.disks); i++ {
			if m.diskUsable(m.disks[i]) {
				cur = i
				break
			}
		}
	case "enter":
		if cur < 0 || cur >= len(m.disks) {
			return m, nil
		}
		m.a.Disk = m.disks[cur].Path
		return m, m.next()
	}
	m.listCursor[stDisk] = cur
	return m, nil
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
		case !m.diskUsable(d) && d.Installer:
			out = append(out, sDim.Render("   "+line+"  installer image"))
		case !m.diskUsable(d) && d.InUse:
			out = append(out, sDim.Render("   "+line+"  in use"))
		case !m.diskUsable(d):
			out = append(out, sDim.Render("   "+line+"  "+d.Member+" member"))
		case d.Size < minDiskSize:
			out = append(out, row(i == cur, w-14, line)+sYellow.Render("  "+g.warn+" small"))
		default:
			out = append(out, row(i == cur, w, line))
		}
	}
	if cur < 0 {
		out = append(out, "", sYellow.Render(g.warn+" No disk can be used. Unmount the one to install on, or start with FORCE_DISK=1."))
		return out, hint("esc", "back")
	}
	d := m.disks[cur]
	if d.Size < minDiskSize {
		out = append(out, "", sYellow.Render(g.warn+" ")+sText.Render(humanSize(d.Size)+" is tight: a full install with a few apps needs about 32G, and old versions"),
			sText.Render("  of the system stay until they're cleaned up. It works for a small install."))
	}
	if d.Tran == "usb" {
		out = append(out, "", sYellow.Render(g.warn+" ")+sText.Render("This is a USB drive. Make sure it isn't one you meant to keep, such as a backup."))
	}
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
