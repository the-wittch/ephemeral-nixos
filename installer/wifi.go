package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// The Wi-Fi screen: networks from NetworkManager, and a password typed here, so a handheld
// can get online with the controller. nmtui, from a shell, is still there for a hidden network
// or a work login.
type network struct {
	ssid     string
	signal   int
	security string
}

type wifiListMsg struct {
	nets []network
	err  error
}

type wifiDoneMsg struct{ err error }

func scanWifi() tea.Cmd {
	return func() tea.Msg {
		out, err := exec.Command("nmcli", "-t", "-e", "yes", "-f", "SSID,SIGNAL,SECURITY",
			"device", "wifi", "list", "--rescan", "yes").Output()
		if err != nil {
			return wifiListMsg{err: fmt.Errorf("couldn't list Wi-Fi networks: %w", err)}
		}
		return wifiListMsg{nets: parseWifi(string(out))}
	}
}

// nmcli -t -e yes: fields split on ':', with ':' and '\' inside a field escaped by '\'. The
// strongest of each name is kept, and hidden ones, which have none, are left out.
func parseWifi(out string) []network {
	var nets []network
	seen := map[string]int{}
	for _, line := range strings.Split(out, "\n") {
		f := splitEscaped(line)
		if len(f) != 3 || f[0] == "" {
			continue
		}
		sig, _ := strconv.Atoi(f[1])
		n := network{f[0], sig, f[2]}
		if i, ok := seen[n.ssid]; ok {
			if sig > nets[i].signal {
				nets[i] = n
			}
			continue
		}
		seen[n.ssid] = len(nets)
		nets = append(nets, n)
	}
	return nets
}

func splitEscaped(line string) []string {
	var out []string
	var cur strings.Builder
	esc := false
	for _, r := range line {
		switch {
		case esc:
			cur.WriteRune(r)
			esc = false
		case r == '\\':
			esc = true
		case r == ':':
			out = append(out, cur.String())
			cur.Reset()
		default:
			cur.WriteRune(r)
		}
	}
	return append(out, cur.String())
}

func (n network) open() bool { return n.security == "" || n.security == "--" }

// where the profile goes; the install copies the folder to the new system
const nmConnections = "/etc/NetworkManager/system-connections"

const wifiProfile = "ephemeral-wifi"

// The profile is written as NetworkManager's own keyfile, readable by root only, so the password
// is kept for the new system and never on a command line. The name goes in as bytes, so nothing
// in it can be read as keyfile syntax.
func wifiKeyfile(n network, password string) string {
	var ssid strings.Builder
	for _, c := range []byte(n.ssid) {
		ssid.WriteString(strconv.Itoa(int(c)) + ";")
	}
	var b strings.Builder
	fmt.Fprintf(&b, "[connection]\nid=%s\ntype=wifi\n\n[wifi]\nmode=infrastructure\nssid=%s\n\n", wifiProfile, ssid.String())
	if !n.open() {
		mgmt := "wpa-psk"
		if strings.Contains(n.security, "WPA3") && !strings.Contains(n.security, "WPA2") {
			mgmt = "sae"
		}
		fmt.Fprintf(&b, "[wifi-security]\nkey-mgmt=%s\npsk=%s\n\n", mgmt, keyfileString(password))
	}
	b.WriteString("[ipv4]\nmethod=auto\n\n[ipv6]\nmethod=auto\n")
	return b.String()
}

// GLib's escapes for a key file value: a backslash, and a space that would otherwise be dropped
// from the start.
func keyfileString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	if strings.HasPrefix(s, " ") {
		s = `\s` + s[1:]
	}
	return s
}

func connectWifi(n network, password string) tea.Cmd {
	return func() tea.Msg {
		path := nmConnections + "/" + wifiProfile + ".nmconnection"
		if err := os.WriteFile(path+".new", []byte(wifiKeyfile(n, password)), 0o600); err != nil {
			return wifiDoneMsg{err}
		}
		if err := os.Rename(path+".new", path); err != nil {
			return wifiDoneMsg{err}
		}
		if out, err := exec.Command("nmcli", "connection", "reload").CombinedOutput(); err != nil {
			return wifiDoneMsg{nmcliErr(out, err)}
		}
		out, err := exec.Command("nmcli", "connection", "up", "id", wifiProfile).CombinedOutput()
		if err != nil {
			// a wrong password isn't kept for the new system
			_ = os.Remove(path)
			_ = exec.Command("nmcli", "connection", "reload").Run()
		}
		return wifiDoneMsg{nmcliErr(out, err)}
	}
}

func nmcliErr(out []byte, err error) error {
	if err == nil {
		return nil
	}
	if msg := strings.TrimSpace(string(out)); msg != "" {
		return fmt.Errorf("%s", strings.TrimPrefix(msg, "Error: "))
	}
	return err
}

func (m model) keyWifi(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.wifiBusy {
		return m, nil
	}
	if m.wifiPass.Focused() {
		switch k.String() {
		case "esc":
			m.wifiPass.Blur()
			m.wifiPass.SetValue("")
			return m, nil
		case "enter":
			if m.wifiPass.Value() == "" {
				m.note = "Type the network's password."
				return m, nil
			}
			m.wifiBusy = true
			pw := m.wifiPass.Value()
			m.wifiPass.SetValue("")
			m.wifiPass.Blur()
			return m, connectWifi(m.wifiNets[m.wifiSel], pw)
		}
		var cmd tea.Cmd
		m.wifiPass, cmd = m.wifiPass.Update(k)
		return m, cmd
	}
	switch k.String() {
	case "esc":
		return m, m.back()
	case "up":
		if m.wifiSel > 0 {
			m.wifiSel--
		}
	case "down":
		if m.wifiSel < len(m.wifiNets)-1 {
			m.wifiSel++
		}
	case "r", "f2":
		m.wifiScanning = true
		return m, scanWifi()
	case "enter":
		if m.wifiSel >= len(m.wifiNets) {
			return m, nil
		}
		n := m.wifiNets[m.wifiSel]
		if n.open() {
			m.wifiBusy = true
			return m, connectWifi(n, "")
		}
		m.osk.open = m.padUsed
		return m, m.wifiPass.Focus()
	}
	return m, nil
}

func (m model) viewWifi(w int) ([]string, string) {
	out := []string{sBold.Render("Wi-Fi") + "  " + sSub.Render("pick a network"), ""}
	switch {
	case m.wifiScanning:
		out = append(out, m.spin()+sText.Render(" Looking for networks"))
	case m.wifiErr != nil:
		out = append(out, sRed.Render(g.failed+" ")+sText.Render(m.wifiErr.Error()))
	case len(m.wifiNets) == 0:
		out = append(out, sText.Render("No networks found. Press r or Y to look again."))
	}
	for i, n := range m.wifiNets {
		lock := ""
		if !n.open() {
			lock = sDim.Render("  " + n.security)
		}
		line := fmt.Sprintf("%-32s %3d%%", truncate(n.ssid, 32), n.signal)
		if i == m.wifiSel {
			out = append(out, sCursor.Render(" "+g.cursor+" "+line+" ")+lock)
		} else {
			out = append(out, "   "+sText.Render(line)+lock)
		}
	}
	if m.wifiPass.Focused() || m.wifiBusy {
		out = append(out, "", sText.Render("Password for "+m.wifiNets[m.wifiSel].ssid), "  "+m.wifiPass.View())
	}
	if m.wifiBusy {
		out = append(out, "", m.spin()+sText.Render(" Connecting"))
	}
	out = append(out, "", sSub.Render("For a hidden network or a work login, open a shell from the first screen and run nmtui."))
	if m.wifiPass.Focused() {
		return out, hint("enter", "connect", "f2 / Y", "keyboard", "esc", "cancel")
	}
	return out, hint("enter", "connect", "↑↓", "move", "r / Y", "look again", "esc", "back")
}
