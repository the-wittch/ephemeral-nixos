package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

var installSteps = []string{
	"Get ready",
	"Partition and encrypt the disk",
	"Detect the hardware",
	"Build and install the system",
	"Install the boot loader",
}

// which step a line of install.sh output starts, or -1
func stepOf(line string) int {
	switch {
	case strings.HasPrefix(line, "==> partitioning"):
		return 1
	case strings.HasPrefix(line, "==> detecting hardware"),
		strings.HasPrefix(line, "==> using "),
		strings.HasPrefix(line, "==> warning: no nixos-generate-config"):
		return 2
	case strings.HasPrefix(line, "==> installing from"):
		return 3
	case strings.HasPrefix(line, "installing the boot loader"):
		return 4
	}
	return -1
}

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;?]*[A-Za-z]|\r`)

func cleanLine(s string) string { return printable(ansiRe.ReplaceAllString(s, "")) }

type lineMsg string
type installDoneMsg struct{ err error }

func (a answers) env(hash string) []string {
	apps := strings.Join(a.Apps, " ")
	vault := a.Vault
	if !a.PassAgent {
		vault = "Personal"
	}
	controld := ""
	if a.ControlD {
		controld = a.ControlDID
	}
	return append(os.Environ(),
		"DISK="+a.Disk,
		"CONFIRM=YES",
		"INSTALL_HOSTNAME="+a.Hostname,
		"INSTALL_USERNAME="+a.Username,
		"INSTALL_DESKTOP="+a.Desktop,
		"NVIDIA_DRIVER="+a.Nvidia,
		"KEYBOARD_LAYOUT="+a.Layout,
		"KEYBOARD_VARIANT="+a.Variant,
		"LOCALE="+a.Locale,
		"TIMEZONE="+a.Timezone,
		"PROFILE="+a.Profile,
		"EXTRAS="+boolStr(a.Extras && a.Profile == "full"),
		"OPENSNITCH="+boolStr(a.OpenSnitch),
		"PASS_SSH_AGENT="+boolStr(a.PassAgent && a.Profile == "full"),
		"PASS_SSH_VAULT="+vault,
		"CONTROLD_RESOLVER="+controld,
		"APPS="+apps,
		"FINGERPRINT="+boolStr(a.Fingerprint),
		"SWAP_SIZE="+a.Swap,
		"PASSWORD_HASH="+hash,
		"FLAKE_ATTR=nixos",
	)
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func hashPassword(pw string) (string, error) {
	cmd := exec.Command("mkpasswd", "-m", "yescrypt", "--stdin")
	cmd.Stdin = strings.NewReader(pw)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("mkpasswd: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// Runs install.sh with the answers. disko asks for the passphrase twice on stdin when it formats,
// so it goes in that way, never in the environment or on a command line.
func startInstall(scriptsDir string, a answers, dryRun bool) <-chan tea.Msg {
	ch := make(chan tea.Msg, 256)
	go func() {
		defer close(ch)
		if dryRun {
			ch <- installDoneMsg{err: simulate(ch)}
			return
		}
		hash, err := hashPassword(a.Password)
		if err != nil {
			ch <- installDoneMsg{err: err}
			return
		}
		cmd := exec.Command(filepath.Join(scriptsDir, "install.sh"))
		cmd.Dir = "/"
		cmd.Env = a.env(hash)
		cmd.Stdin = strings.NewReader(a.Passphrase + "\n" + a.Passphrase + "\n")
		pr, pw := io.Pipe()
		cmd.Stdout, cmd.Stderr = pw, pw
		if err := cmd.Start(); err != nil {
			ch <- installDoneMsg{err: err}
			return
		}
		go func() {
			pw.CloseWithError(cmd.Wait())
		}()
		sc := bufio.NewScanner(pr)
		sc.Buffer(make([]byte, 64*1024), 1024*1024)
		for sc.Scan() {
			ch <- lineMsg(cleanLine(sc.Text()))
		}
		err = sc.Err()
		ch <- installDoneMsg{err: err}
	}()
	return ch
}

func waitFor(ch <-chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return nil
		}
		return msg
	}
}

// --dry-run: the same output shape as install.sh, without touching anything
func simulate(ch chan<- tea.Msg) error {
	lines := []string{
		"==> a copy of this output goes to /tmp/ephemeral-install.log",
		"==> staging flake at /tmp/nixos-flake-stage.XXXXXX",
		"==> partitioning (LUKS passphrase next)",
		"==> the first build downloads nixpkgs and can sit at 0/N for several minutes",
		"+ cryptsetup -q luksFormat --type luks2 --pbkdf argon2id /dev/disk/by-partlabel/disk-main-luks",
		"+ mkfs.btrfs /dev/mapper/cryptroot -L nixos -f",
		"==> detecting hardware",
		"==> installing from /mnt/persist/etc/nixos",
		"copying path '/nix/store/…-linux-6.18.58' from 'https://cache.nixos.org'...",
		"copying path '/nix/store/…-plasma-workspace-6.6.2' from 'https://cache.nixos.org'...",
		"building '/nix/store/…-nixos-system-nixos.drv'...",
		"installing the boot loader...",
		"done. detach ISO, reboot, unlock LUKS, log in.",
	}
	for _, l := range lines {
		time.Sleep(700 * time.Millisecond)
		ch <- lineMsg(l)
	}
	if os.Getenv("EPHEMERAL_DRY_RUN_FAIL") == "1" {
		ch <- lineMsg("error: builder for '/nix/store/…-nvidia-x11-580.95.05.drv' failed with exit code 2")
		return fmt.Errorf("install.sh exited with status 1")
	}
	return nil
}
