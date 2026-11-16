package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"testing"
)

// testdata/validation.json is shared with modules/tests/_scripts.sh, so the screens and install.sh
// can't drift apart on what they accept
func TestValidationMatchesInstallSh(t *testing.T) {
	b, err := os.ReadFile("testdata/validation.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases map[string]struct{ Ok, Bad []string }
	if err := json.Unmarshal(b, &cases); err != nil {
		t.Fatal(err)
	}
	checks := map[string]func(string) error{
		"hostname":  checkHostname,
		"username":  checkUsername,
		"swap":      checkSwap,
		"timezone":  checkTimezone,
		"vault":     checkVault,
		"controld":  checkControlD,
		"dns":       checkDNS,
		"customDns": checkCustomDNS,
	}
	for name, c := range cases {
		check, ok := checks[name]
		if !ok {
			t.Errorf("no Go check for %q", name)
			continue
		}
		for _, v := range c.Ok {
			if err := check(v); err != nil {
				t.Errorf("%s %q refused: %v", name, v, err)
			}
		}
		for _, v := range c.Bad {
			if check(v) == nil {
				t.Errorf("%s %q accepted", name, v)
			}
		}
	}
	for name := range checks {
		if _, ok := cases[name]; !ok {
			t.Errorf("testdata/validation.json has no cases for %q", name)
		}
	}
}

func TestSecretsRefuseControlCharacters(t *testing.T) {
	for _, s := range []string{"long enough\nsecond", "long enough\ttab", "long enough\x1b[A"} {
		if checkPassphrase(s, "") == nil {
			t.Errorf("passphrase %q accepted", s)
		}
		if checkPassword(s, "") == nil {
			t.Errorf("password %q accepted", s)
		}
	}
	if err := checkPassphrase("correct horse battery staple", ""); err != nil {
		t.Errorf("a plain passphrase was refused: %v", err)
	}
}

func TestAnswersFileReadableByOthers(t *testing.T) {
	dir := t.TempDir()
	open := filepath.Join(dir, "open.json")
	closed := filepath.Join(dir, "closed.json")
	for path, mode := range map[string]os.FileMode{open: 0o644, closed: 0o600} {
		if err := os.WriteFile(path, []byte("{}"), mode); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(path, mode); err != nil {
			t.Fatal(err)
		}
	}
	if answersFileWarning(open) == "" {
		t.Error("no warning for an answers file other users can read")
	}
	if w := answersFileWarning(closed); w != "" {
		t.Errorf("warned about a 600 answers file: %s", w)
	}
}

func TestInheritedOverridesArentPassedOn(t *testing.T) {
	t.Setenv("FORCE_DISK", "1")
	t.Setenv("ALLOW_SAME_PARTLABELS", "1")
	t.Setenv("HARDWARE_CONFIG", "/tmp/evil.nix")
	t.Setenv("EPHEMERAL_KEEP_ME", "yes")
	a := defaultAnswers()
	a.Disk = "/dev/vda"
	env := a.env("hash")
	for _, kv := range env {
		for _, key := range []string{"FORCE_DISK=", "ALLOW_SAME_PARTLABELS=", "HARDWARE_CONFIG="} {
			if strings.HasPrefix(kv, key) {
				t.Errorf("%s reached install.sh from the installer's own environment", kv)
			}
		}
	}
	if !slices.Contains(env, "EPHEMERAL_KEEP_ME=yes") {
		t.Error("an ordinary variable from the environment was dropped")
	}
	a.Hardware = "/etc/hardware.nix"
	if !slices.Contains(a.env("hash"), "HARDWARE_CONFIG=/etc/hardware.nix") {
		t.Error("hardwareConfig from the answers didn't reach install.sh")
	}
}

// the same line modules/tests/_scripts.sh expects from disk_identity in common.sh
func TestIdentityMatchesInstallSh(t *testing.T) {
	got := formatIdentity("SERIAL=\"\" WWN=\"\" SIZE=\"68719476736\" MODEL=\"Same Disk\"\n", "pci-0000:00:14.0-usb-0:2:1.0-scsi-0:0:0:0\n")
	want := `SERIAL="" WWN="" SIZE="68719476736" MODEL="Same Disk" ID_PATH="pci-0000:00:14.0-usb-0:2:1.0-scsi-0:0:0:0"`
	if got != want {
		t.Errorf("identity line\n got %s\nwant %s", got, want)
	}
}

func TestNoCoreDumps(t *testing.T) {
	if err := noCoreDumps(); err != nil {
		t.Fatal(err)
	}
	const prGetDumpable = 3
	v, _, errno := syscall.RawSyscall(syscall.SYS_PRCTL, prGetDumpable, 0, 0)
	if errno != 0 || v != 0 {
		t.Errorf("the process is still dumpable (%d, %v)", v, errno)
	}
}
