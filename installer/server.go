package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// a variable so the tests can point it at a server of their own
var githubKeysURL = "https://github.com/%s.keys"

type keysMsg struct {
	name    string
	keys    []string
	skipped int
	err     error
}

func fetchKeys(name string) tea.Cmd {
	return func() tea.Msg {
		keys, skipped, err := githubKeys(name)
		return keysMsg{name, keys, skipped, err}
	}
}

// GitHub lists a user's keys without comments, so each gets one saying where it came from
func githubKeys(name string) (keys []string, skipped int, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf(githubKeysURL, name), nil)
	if err != nil {
		return nil, 0, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode == http.StatusNotFound:
		return nil, 0, fmt.Errorf("GitHub has no user called %s", name)
	case resp.StatusCode != http.StatusOK:
		return nil, 0, fmt.Errorf("GitHub answered %s", resp.Status)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, 0, err
	}
	for _, line := range strings.Split(string(b), "\n") {
		if line = strings.TrimSpace(line); line == "" {
			continue
		}
		if k := line + " github:" + name; checkSSHKey(k) == nil {
			keys = append(keys, k)
		} else {
			skipped++
		}
	}
	if len(keys) == 0 {
		return nil, skipped, fmt.Errorf("%s has no SSH keys on GitHub that can be used", name)
	}
	return keys, skipped, nil
}

// the same key with another comment is still the same key
func addKeys(have []string, more ...string) []string {
	out := slices.Clone(have)
	for _, k := range more {
		if !slices.ContainsFunc(out, func(h string) bool { return keyBody(h) == keyBody(k) }) {
			out = append(out, k)
		}
	}
	return out
}

func keyBody(k string) string {
	f := strings.Fields(k)
	return strings.Join(f[:min(2, len(f))], " ")
}

// the type, the end of the key and the comment fit on a line where the whole key doesn't
func shortKey(k string) string {
	f := strings.Fields(k)
	if len(f) < 2 {
		return k
	}
	body := f[1]
	if len(body) > 12 {
		body = "…" + body[len(body)-12:]
	}
	return strings.Join(append([]string{f[0], body}, f[2:]...), " ")
}

// the tick box, then with it ticked the key field and a row for each key
func (m model) serverRows() int {
	if !m.a.Server {
		return 1
	}
	return 2 + len(m.a.SSHKeys)
}

func (m *model) focusServer() tea.Cmd {
	m.serverFocus = min(m.serverFocus, m.serverRows()-1)
	if m.serverFocus == 1 {
		return m.keyIn.Focus()
	}
	m.keyIn.Blur()
	return nil
}

// what the Server step still needs before the install can go ahead
func (a answers) serverUnanswered() bool {
	return a.Desktop == "none" && !a.Server || a.Server && len(a.SSHKeys) == 0
}

func (m model) keyServer(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch k.String() {
	case "up", "shift+tab":
		m.serverFocus = max(m.serverFocus-1, 0)
		return m, m.focusServer()
	case "down", "tab":
		m.serverFocus++
		return m, m.focusServer()
	case "enter":
		if m.serverFocus == 1 && strings.TrimSpace(m.keyIn.Value()) != "" {
			return m.addKey()
		}
		if m.fetching != "" {
			m.note = "Still fetching the keys from GitHub."
			return m, nil
		}
		if m.a.serverUnanswered() {
			m.note = "SSH keys: a server needs at least one, or nobody can log in to it"
			m.serverFocus = 1
			return m, m.focusServer()
		}
		return m, m.next()
	}
	switch {
	case m.serverFocus == 0:
		if s := k.String(); s != "x" && s != " " {
			return m, nil
		}
		if m.a.Desktop == "none" {
			m.note = "Without a desktop, SSH is the only way in, so it has to be a server."
			return m, nil
		}
		m.a.Server = !m.a.Server
		if m.a.Server && len(m.a.SSHKeys) == 0 {
			m.serverFocus = 1
			return m, m.focusServer()
		}
	case m.serverFocus == 1:
		var cmd tea.Cmd
		m.keyIn, cmd = m.keyIn.Update(k)
		return m, cmd
	default:
		switch k.String() {
		case "x", "delete", "backspace":
			i := m.serverFocus - 2
			m.a.SSHKeys = slices.Delete(slices.Clone(m.a.SSHKeys), i, i+1)
			return m, m.focusServer()
		}
	}
	return m, nil
}

func (m model) addKey() (tea.Model, tea.Cmd) {
	v := strings.TrimSpace(m.keyIn.Value())
	if name, ok := strings.CutPrefix(v, "github:"); ok {
		if err := checkGitHubName(name); err != nil {
			m.note = "SSH keys: " + err.Error()
			return m, nil
		}
		m.fetching = name
		return m, fetchKeys(name)
	}
	if err := checkSSHKey(v); err != nil {
		m.note = "SSH keys: " + err.Error()
		return m, nil
	}
	m.a.SSHKeys = addKeys(m.a.SSHKeys, v)
	m.keyIn.SetValue("")
	return m, nil
}

func (m model) gotKeys(msg keysMsg) model {
	if msg.name != m.fetching {
		return m
	}
	m.fetching = ""
	if msg.err != nil {
		m.note = "SSH keys: " + msg.err.Error()
		return m
	}
	m.a.SSHKeys = addKeys(m.a.SSHKeys, msg.keys...)
	m.keyIn.SetValue("")
	if msg.skipped > 0 {
		m.note = fmt.Sprintf("SSH keys: left out %d of %s's keys, of a type the config doesn't take", msg.skipped, msg.name)
	}
	return m
}

func (m model) viewServer(w int) ([]string, string) {
	out := m.header(w)
	out = append(out, sBold.Render("Is it a server?")+" "+sSub.Render("Reached from other computers, with or without a desktop."), "")
	check := sDim.Render(g.unchecked) + " "
	if m.a.Server {
		check = sGreen.Render(g.checked) + " "
	}
	desc := "stays on, and takes SSH logins with keys only"
	if m.a.Desktop == "none" {
		desc = "without a desktop it has to be: SSH is the only way in"
	}
	cursor := func(i, indent int) string {
		if i == m.serverFocus {
			return spaces(indent-3) + sMauve.Render(g.cursor+"  ")
		}
		return spaces(indent)
	}
	out = append(out, cursor(0, 4)+check+sText.Render("Server"), spaces(8)+sSub.Render(truncate(desc, w-8)))
	if !m.a.Server {
		return out, hint("x", "tick", "enter", "next step", "esc", "back")
	}
	out = append(out, m.gap()...)
	help := "paste a public key, or github:yourname to fetch yours from GitHub"
	if m.serverFocus == 1 {
		help = "type or " + help
	}
	out = append(out, cursor(1, 8)+sText.Render(padRight("SSH key", 13))+m.keyIn.View(), spaces(12)+sSub.Render(truncate(help, w-12)))
	if m.fetching != "" {
		out = append(out, spaces(12)+m.spin()+" "+sSub.Render("fetching github.com/"+m.fetching+".keys"))
	}
	out = append(out, m.gap()...)
	if len(m.a.SSHKeys) == 0 {
		out = append(out, spaces(8)+sSub.Render("No keys yet. The computers with these keys can log in."))
	}
	for i, k := range m.a.SSHKeys {
		line := cursor(2+i, 8) + sGreen.Render(g.dot) + " " + sText.Render(truncate(shortKey(k), w-14))
		if 2+i == m.serverFocus {
			line += "  " + sDim.Render("x to remove")
		}
		out = append(out, line)
	}
	out = append(out, m.gap()...)
	out = append(out,
		sSub.Render("Unlock the disk after a reboot with ssh -p 2222 root@<address>."),
		sSub.Render("That needs Ethernet; Wi-Fi only starts once the disk is unlocked."))
	switch {
	case m.serverFocus == 1:
		return out, hint("enter", "add", "↑↓", "move", "esc", "back")
	case m.serverFocus >= 2:
		return out, hint("x", "remove", "↑↓", "move", "enter", "next step", "esc", "back")
	}
	return out, hint("x", "tick", "↑↓", "move", "enter", "next step", "esc", "back")
}
