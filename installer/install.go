package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
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

// The answers, keyed like settings.json, plus what install.sh takes besides it.
// --answers reads them from a file; the screens fill them in.
type answers struct {
	Disk        string   `json:"diskDevice"`
	Desktop     string   `json:"desktop"`
	Nvidia      string   `json:"nvidia"`
	Layout      string   `json:"keyboardLayout"`
	Variant     string   `json:"keyboardVariant"`
	Locale      string   `json:"locale"`
	Timezone    string   `json:"timeZone"`
	Profile     string   `json:"profile"`
	Extras      bool     `json:"extras"`
	OpenSnitch  bool     `json:"useOpenSnitch"`
	PassAgent   bool     `json:"usePassSshAgent"`
	Vault       string   `json:"passSshVault"`
	ControlD    bool     `json:"-"`
	ControlDID  string   `json:"controldResolver"`
	Fingerprint bool     `json:"fingerprint"`
	Apps        []string `json:"apps"`
	Hostname    string   `json:"hostname"`
	Username    string   `json:"username"`
	Swap        string   `json:"swapSize"`
	Passphrase  string   `json:"passphrase"`
	Password    string   `json:"password"`
	Hardware    string   `json:"hardwareConfig"`
}

func defaultAnswers() answers {
	return answers{
		Desktop: "plasma", Nvidia: "none", Layout: "us", Locale: "en_US.UTF-8", Timezone: "UTC",
		Profile: "full", OpenSnitch: true, Vault: "Personal",
		Hostname: "nixos", Username: "nixuser", Swap: "4G",
	}
}

func readAnswers(path string) (answers, error) {
	a := defaultAnswers()
	b, err := os.ReadFile(path)
	if err != nil {
		return a, err
	}
	if err := json.Unmarshal(b, &a); err != nil {
		return a, fmt.Errorf("%s: %w", path, err)
	}
	a.ControlD = a.ControlDID != ""
	return a, a.check()
}

func (a answers) check() error {
	var errs []error
	add := func(field string, err error) {
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", field, err))
		}
	}
	if a.Disk == "" {
		add("diskDevice", errors.New("missing"))
	}
	add("hostname", checkHostname(a.Hostname))
	add("username", checkUsername(a.Username))
	add("swapSize", checkSwap(a.Swap))
	add("timeZone", checkTimezone(a.Timezone))
	add("keyboardLayout", checkLayout(a.Layout))
	add("keyboardVariant", checkVariant(a.Variant))
	add("locale", checkLocale(a.Locale))
	add("passphrase", checkPassphrase(a.Passphrase, ""))
	add("password", checkPassword(a.Password, ""))
	if a.PassAgent {
		add("passSshVault", checkVault(a.Vault))
	}
	if a.ControlD {
		add("controldResolver", checkControlD(a.ControlDID))
	}
	return errors.Join(errs...)
}

func (a answers) env(hash string) []string {
	vault := a.Vault
	if !a.PassAgent {
		vault = "Personal"
	}
	controld := ""
	if a.ControlD {
		controld = a.ControlDID
	}
	env := append(os.Environ(),
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
		"APPS="+strings.Join(a.Apps, " "),
		"FINGERPRINT="+boolStr(a.Fingerprint),
		"SWAP_SIZE="+a.Swap,
		"PASSWORD_HASH="+hash,
		"FLAKE_ATTR=nixos",
	)
	if a.Hardware != "" {
		env = append(env, "HARDWARE_CONFIG="+a.Hardware)
	}
	return env
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

// Where a running install keeps its output and, once it ends, its exit status. install.sh runs
// in a session of its own writing there, so the screens can quit, crash or reattach without
// taking the install down with them.
type runDir struct{ dir, out, status, pid string }

func newRunDir(dryRun bool) runDir {
	dir := "/run/ephemeral-install"
	if dryRun {
		dir = filepath.Join(os.TempDir(), "ephemeral-install-dry-run")
	}
	return runDir{dir, filepath.Join(dir, "output"), filepath.Join(dir, "status"), filepath.Join(dir, "pid")}
}

// whether an earlier install is still going
func (r runDir) running() bool {
	if _, err := os.Stat(r.status); err == nil {
		return false
	}
	pid, err := strconv.Atoi(readTrim(r.pid))
	if err != nil || pid <= 0 {
		return false
	}
	return syscall.Kill(pid, 0) == nil
}

func (r runDir) finished() (code int, done bool) {
	s := readTrim(r.status)
	if s == "" {
		return 0, false
	}
	code, err := strconv.Atoi(s)
	if err != nil {
		return 1, true
	}
	return code, true
}

// Starts install.sh. disko asks for the passphrase twice on stdin when it formats, so it goes
// in that way, through a pipe filled and closed before the start: never in the environment or
// on a command line.
func launch(cfg config, r runDir, a answers) error {
	if err := os.RemoveAll(r.dir); err != nil {
		return err
	}
	if err := os.MkdirAll(r.dir, 0o700); err != nil {
		return err
	}
	out, err := os.OpenFile(r.out, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer out.Close()

	script := filepath.Join(cfg.scriptsDir, "install.sh")
	hash := "dry-run"
	if cfg.dryRun {
		script = filepath.Join(r.dir, "pretend-install.sh")
		if err := os.WriteFile(script, []byte(pretendInstall), 0o700); err != nil {
			return err
		}
	} else if hash, err = hashPassword(a.Password); err != nil {
		return err
	}
	pr, pw, err := os.Pipe()
	if err != nil {
		return err
	}
	_, err = io.WriteString(pw, a.Passphrase+"\n"+a.Passphrase+"\n")
	pw.Close()
	if err != nil {
		pr.Close()
		return err
	}
	defer pr.Close()

	cmd := exec.Command("sh", "-c", `"$0"; echo "$?" > "$1"`, script, r.status)
	cmd.Dir = "/"
	cmd.Env = a.env(hash)
	if cfg.dryRun {
		cmd.Env = append(cmd.Env, fmt.Sprintf("DRY_RUN_PAUSE=%.3f", dryRunPause.Seconds()))
	}
	cmd.Stdin, cmd.Stdout, cmd.Stderr = pr, out, out
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	if err := os.WriteFile(r.pid, []byte(strconv.Itoa(cmd.Process.Pid)), 0o600); err != nil {
		return err
	}
	go cmd.Wait()
	return nil
}

// reads the output as it grows, a whole line at a time
type follower struct {
	path    string
	off     int64
	partial []byte
}

func (f *follower) next() []string {
	file, err := os.Open(f.path)
	if err != nil {
		return nil
	}
	defer file.Close()
	if _, err := file.Seek(f.off, io.SeekStart); err != nil {
		return nil
	}
	b, _ := io.ReadAll(file)
	f.off += int64(len(b))
	b = append(f.partial, b...)
	i := bytes.LastIndexByte(b, '\n')
	if i < 0 {
		f.partial = b
		return nil
	}
	f.partial = append([]byte(nil), b[i+1:]...)
	var lines []string
	for _, l := range strings.Split(string(b[:i]), "\n") {
		lines = append(lines, cleanLine(l))
	}
	return lines
}

// --answers: no screens, the output on stdout, install.sh's exit status as ours
func runHeadless(cfg config, a answers) int {
	r := newRunDir(cfg.dryRun)
	if err := launch(cfg, r, a); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	f := follower{path: r.out}
	for {
		for _, l := range f.next() {
			fmt.Println(l)
		}
		if code, done := r.finished(); done {
			for _, l := range f.next() {
				fmt.Println(l)
			}
			return code
		}
		time.Sleep(200 * time.Millisecond)
	}
}

var dryRunPause = 700 * time.Millisecond

// --dry-run runs this in place of install.sh, the same way: detached, with the passphrase on
// stdin. It prints what install.sh would and touches nothing.
const pretendInstall = `#!/bin/sh
read -r one
read -r two
if [ -z "$one" ] || [ "$one" != "$two" ]; then
  echo "error: the passphrase didn't arrive twice on stdin"
  exit 3
fi
for line in \
  "==> a copy of this output goes to /tmp/ephemeral-install.log" \
  "==> partitioning (LUKS passphrase next)" \
  "==> the first build downloads nixpkgs and can sit at 0/N for several minutes" \
  "+ cryptsetup -q luksFormat --type luks2 --pbkdf argon2id /dev/disk/by-partlabel/disk-main-luks" \
  "+ mkfs.btrfs /dev/mapper/cryptroot -L nixos -f" \
  "==> detecting hardware" \
  "==> installing from /mnt/persist/etc/nixos" \
  "copying path '/nix/store/...-linux-6.18.58' from 'https://cache.nixos.org'..." \
  "building '/nix/store/...-nixos-system-nixos.drv'..." \
  "installing the boot loader..."; do
  sleep "$DRY_RUN_PAUSE"
  echo "$line"
done
if [ "$EPHEMERAL_DRY_RUN_FAIL" = 1 ]; then
  echo "error: builder for '/nix/store/...-nvidia-x11-580.95.05.drv' failed with exit code 2"
  exit 1
fi
echo "done. detach ISO, reboot, unlock LUKS, log in."
`
