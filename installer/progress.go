package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m model) startInstall() (tea.Model, tea.Cmd) {
	if err := launch(m.cfg, m.run, m.a); err != nil {
		uiLog.Printf("starting install.sh: %v", err)
		m.note = "Couldn't start the install: " + err.Error()
		return m, nil
	}
	m.history = nil
	m.step = stInstalling
	m.stepShown = m.spinFrame
	m.reattached = false
	m.resetProgress(time.Now())
	return m, nil
}

func (m *model) resetProgress(started time.Time) {
	m.follow = &follower{path: m.run.out}
	m.curStep = 0
	m.lines = nil
	m.exitCode = 0
	m.started = started
	m.note = ""
	m.logView.SetContent("")
}

// The disk check refuses a disk with something mounted or open on it, which a failed install
// leaves behind; install.sh undoes only what it recorded opening.
func teardown(cfg config) error {
	if cfg.dryRun {
		return nil
	}
	out, err := exec.Command(filepath.Join(cfg.scriptsDir, "install.sh"), "--teardown").CombinedOutput()
	if err != nil {
		uiLog.Printf("install.sh --teardown: %v\n%s", err, out)
		msg := strings.TrimSpace(string(out))
		if i := strings.LastIndex(msg, "\n"); i >= 0 {
			msg = msg[i+1:]
		}
		return errors.New(strings.TrimPrefix(msg, "error: "))
	}
	return nil
}

// picks up an install that a quit or crashed run of the screens left going
func (m *model) reattach() {
	m.step = stInstalling
	m.stepShown = m.spinFrame
	m.reattached = true
	started := time.Now()
	if fi, err := os.Stat(m.run.pid); err == nil {
		started = fi.ModTime()
	}
	m.resetProgress(started)
	m.poll()
}

func (m *model) poll() {
	for _, line := range m.follow.next() {
		if s := stepOf(line); s > m.curStep {
			m.curStep = s
		}
		if strings.TrimSpace(line) != "" && !isMarker(line) {
			m.lines = append(m.lines, line)
		}
	}
	m.logView.SetContent(strings.Join(m.lines, "\n"))
	if !m.showLog {
		m.logView.GotoBottom()
	}
	code, done := m.run.finished()
	if !done {
		return
	}
	m.finished = time.Now()
	m.exitCode = code
	m.endSel = 0
	m.showLog = false
	if code == 0 {
		m.step = stDone
		m.stepShown = m.spinFrame
	} else {
		m.step = stFailed
		m.stepShown = m.spinFrame
		uiLog.Printf("install.sh exited with status %d", code)
	}
}

// the answers are only in memory, so an install picked up again can't be repeated from here
func (m model) endItems() []string {
	switch {
	case m.step == stDone:
		return []string{"Reboot", "Open a shell"}
	case m.reattached:
		return []string{"Show the full log", "Start over", "Open a shell"}
	}
	return []string{"Try again", "Change an answer", "Show the full log", "Open a shell"}
}

func (m model) keyEnd(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.showLog {
		switch k.String() {
		case "l", "esc", "q":
			m.showLog = false
			return m, nil
		}
		var cmd tea.Cmd
		m.logView, cmd = m.logView.Update(k)
		return m, cmd
	}
	items := m.endItems()
	switch k.String() {
	case "up", "left":
		if m.endSel > 0 {
			m.endSel--
		}
	case "down", "right", "tab":
		if m.endSel < len(items)-1 {
			m.endSel++
		}
	case "enter":
		switch items[m.endSel] {
		case "Reboot":
			m.quitTo = "reboot"
			return m, tea.Quit
		case "Open a shell":
			m.quitTo = "shell"
			return m, tea.Quit
		case "Try again":
			return m.startInstall()
		case "Change an answer":
			if err := teardown(m.cfg); err != nil {
				m.note = "Couldn't close what the last try left open: " + err.Error()
				return m, nil
			}
			m.step = stReview
			m.stepShown = m.spinFrame
			return m, m.enter()
		case "Start over":
			if err := teardown(m.cfg); err != nil {
				m.note = "Couldn't close what the last try left open: " + err.Error()
				return m, nil
			}
			fresh := newModel(m.cfg)
			fresh.width, fresh.height = m.width, m.height
			fresh.logView.Width, fresh.logView.Height = m.logView.Width, m.logView.Height
			fresh.isOnline = m.isOnline
			fresh.step = stWelcome
			fresh.reattached = false
			return fresh, nil
		case "Show the full log":
			m.showLog = true
			m.logView.GotoBottom()
		}
	}
	return m, nil
}

func (m model) elapsed() time.Duration {
	end := m.finished
	if m.step == stInstalling || end.IsZero() {
		end = time.Now()
	}
	return end.Sub(m.started).Round(time.Second)
}

// the title row of the install screens; framed, it's in the border instead
func (m model) titleLines(w int) []string {
	if m.framed {
		return nil
	}
	left, right := m.crumbs()
	left, right = sTitle.Render(left), sDim.Render(right)
	return []string{left + spaces(max(w-lipgloss.Width(left)-lipgloss.Width(right), 1)) + right}
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
	out := append(m.titleLines(w), m.installBar(w), "")
	if m.showLog {
		out = append(out, strings.Split(m.logView.View(), "\n")...)
		return out, hint("↑↓", "scroll", "l", "back to the steps")
	}
	title := "Installing"
	if !m.reattached {
		title += " on " + m.a.Disk
	}
	out = append(out, sBold.Render(title), "")
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
	return out, hint("l", "full log", "q", "leave it running") + "   " + sSub.Render("saved to /tmp/ephemeral-install.log too")
}

func (m model) viewEnd(w int) ([]string, string) {
	var out []string
	if m.showLog {
		out = append(m.titleLines(w), "")
		out = append(out, strings.Split(m.logView.View(), "\n")...)
		return out, hint("↑↓", "scroll", "esc", "back")
	}
	if m.step == stDone {
		out = append(m.titleLines(w), m.installBar(w), "")
		installed := "Installed."
		if m.cfg.effects {
			installed = scramble(installed, m.spinFrame, revealFrac(m.age(), wordSpan))
		}
		out = append(out, sGreen.Bold(true).Render(g.done+" "+installed)+" "+sText.Render("Remove the USB stick and reboot."), "")
		user := m.a.Username
		if m.reattached {
			user = "the user you picked"
		}
		out = append(out, "   "+sText.Render("At boot, unlock the disk with your passphrase, then log in as "+user+"."), "")
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
		out = append(m.titleLines(w), "")
		out = append(out, sRed.Bold(true).Render(g.failed+" The install stopped at:")+" "+sText.Render(installSteps[m.curStep]), "")
		if l := lastError(m.lines); l != "" {
			out = append(out, sRed.Render(truncate(l, w)))
		}
		out = append(out, sDim.Render(fmt.Sprintf("install.sh stopped with status %d", m.exitCode)))
		if !m.reattached {
			out = append(out, "", sSub.Render("Your answers are kept. Trying again formats the disk again from the start."))
		}
		out = append(out, "")
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
