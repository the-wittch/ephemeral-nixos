package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"testing"
)

func TestMain(m *testing.M) {
	var err error
	if rules, err = loadContract(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

// EPHEMERAL_VERDICTS is testdata/validation.json's values with the config's own verdict on each,
// worked out in Nix from the same contract (modules/installer.nix). Go's regexp and the config's
// have to agree on every one.
func TestValidationMatchesConfig(t *testing.T) {
	b, err := os.ReadFile(os.Getenv("EPHEMERAL_VERDICTS"))
	if err != nil {
		t.Fatal(err)
	}
	var verdicts map[string][]struct {
		Value string
		Ok    bool
	}
	if err := json.Unmarshal(b, &verdicts); err != nil {
		t.Fatal(err)
	}
	checks := map[string]func(string) error{
		"hostname":         checkHostname,
		"username":         checkUsername,
		"swapSize":         checkSwap,
		"timeZone":         checkTimezone,
		"keyboardLayout":   checkLayout,
		"keyboardVariant":  checkVariant,
		"locale":           checkLocale,
		"passSshVault":     checkVault,
		"controldResolver": checkControlD,
		"customDns":        checkCustomDNS,
		"desktop":          checkDesktop,
		"nvidia":           checkNvidia,
		"profile":          checkProfile,
		"encryptedDns":     checkDNS,
		"sshKeys":          checkSSHKey,
	}
	for key, cases := range verdicts {
		check, ok := checks[key]
		if !ok {
			t.Errorf("no Go check for %q", key)
			continue
		}
		for _, c := range cases {
			if got := check(c.Value) == nil; got != c.Ok {
				t.Errorf("%s %q: the config says %v, the screens %v", key, c.Value, c.Ok, got)
			}
		}
	}
	for key := range checks {
		if len(verdicts[key]) == 0 {
			t.Errorf("testdata/validation.json has no values for %q", key)
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
