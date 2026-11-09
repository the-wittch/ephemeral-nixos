package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestValidation(t *testing.T) {
	ok := map[string]func(string) error{
		"my-laptop": checkHostname, "alice": checkUsername, "_build-2": checkUsername,
		"16G": checkSwap, "8192M": checkSwap, "Europe/Berlin": checkTimezone,
		"us,de": checkLayout, "": checkVariant, "nodeadkeys": checkVariant,
		"de_DE.UTF-8": checkLocale, "Personal": checkVault, "abcd-1234": checkControlD,
	}
	for v, f := range ok {
		if err := f(v); err != nil {
			t.Errorf("%q should pass: %v", v, err)
		}
	}
	bad := map[string]func(string) error{
		"My Laptop": checkHostname, "-x": checkHostname, "Alice": checkUsername, "1abc": checkUsername,
		"16GB": checkSwap, "Europe/Berlin;rm": checkTimezone, "US": checkLayout, "de_DE": checkLocale,
		"a\"b": checkVault, "a$b": checkVault, "": checkVault, "a b": checkControlD,
	}
	for v, f := range bad {
		if f(v) == nil {
			t.Errorf("%q should fail", v)
		}
	}
	if checkPassphrase("short", "") == nil || checkPassphrase("long enough", "different") == nil {
		t.Error("passphrase checks")
	}
	if checkPassphrase("long enough", "long enough") != nil || checkPassword("x", "x") != nil {
		t.Error("matching secrets should pass")
	}
}

const lsblkSample = `{"blockdevices": [
 {"name":"nvme0n1","path":"/dev/nvme0n1","type":"disk","size":1000204886016,"tran":"nvme","model":"Samsung SSD 990 PRO 1TB","mountpoints":[null],"fstype":null,"label":null,"partlabel":null,
  "children":[
   {"name":"nvme0n1p1","path":"/dev/nvme0n1p1","type":"part","size":536870912,"tran":null,"model":null,"mountpoints":[null],"fstype":"vfat","label":null,"partlabel":"EFI"},
   {"name":"nvme0n1p2","path":"/dev/nvme0n1p2","type":"part","size":322122547200,"tran":null,"model":null,"mountpoints":[null],"fstype":"ntfs","label":"Windows","partlabel":null}]},
 {"name":"sda","path":"/dev/sda","type":"disk","size":30752636928,"tran":"usb","model":"SanDisk Ultra Fit","mountpoints":[null],"fstype":"iso9660","label":"EPHEMERAL_NIX","partlabel":null,
  "children":[{"name":"sda1","path":"/dev/sda1","type":"part","size":30752636928,"tran":null,"model":null,"mountpoints":["/iso"],"fstype":"iso9660","label":null,"partlabel":null}]},
 {"name":"sdb","path":"/dev/sdb","type":"disk","size":500107862016,"tran":"sata","model":"Old disk","mountpoints":[null],"fstype":null,"label":null,"partlabel":null,
  "children":[{"name":"sdb1","path":"/dev/sdb1","type":"part","size":500106813440,"tran":null,"model":null,"mountpoints":[null],"fstype":"crypto_LUKS","label":null,"partlabel":null,
    "children":[{"name":"cryptroot","path":"/dev/mapper/cryptroot","type":"crypt","size":500090036224,"tran":null,"model":null,"mountpoints":[null],"fstype":"btrfs","label":null,"partlabel":null}]}]},
 {"name":"zram0","path":"/dev/zram0","type":"disk","size":8589934592,"tran":null,"model":null,"mountpoints":["[SWAP]"],"fstype":null,"label":null,"partlabel":null}
]}`

func TestParseDisks(t *testing.T) {
	disks, err := parseDisks([]byte(lsblkSample))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"/dev/nvme0n1": false, "/dev/sda": true, "/dev/sdb": true}
	if len(disks) != len(want) {
		t.Fatalf("got %d disks", len(disks))
	}
	for _, d := range disks {
		if d.InUse != want[d.Path] {
			t.Errorf("%s in use: got %v", d.Path, d.InUse)
		}
	}
	if p := disks[0].Partitions; len(p) != 2 || p[0].Label != "EFI" || p[1].Label != "Windows" {
		t.Errorf("partitions: %+v", p)
	}
}

func TestStepOf(t *testing.T) {
	cases := map[string]int{
		"==> partitioning (LUKS passphrase next)":          1,
		"==> detecting hardware":                           2,
		"==> using /tmp/hw.nix":                            2,
		"==> installing from /mnt/persist/etc/nixos":       3,
		"installing the boot loader...":                    4,
		"copying path '/nix/store/x' from 'https://cache'": -1,
	}
	for line, want := range cases {
		if got := stepOf(line); got != want {
			t.Errorf("%q: got %d, want %d", line, got, want)
		}
	}
}

func TestEnvHasNoSecrets(t *testing.T) {
	a := answers{Disk: "/dev/vda", Passphrase: "disk secret", Password: "login secret", Profile: "full"}
	for _, kv := range a.env("$y$hash") {
		if strings.Contains(kv, "disk secret") || strings.Contains(kv, "login secret") {
			t.Errorf("a secret is in the environment: %s", kv)
		}
	}
}

func TestParseXKB(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "base.lst")
	os.WriteFile(p, []byte("! layout\n  us              English (US)\n  de              German\n\n! variant\n  nodeadkeys      de: German (no dead keys)\n  intl            us: English (US, intl.)\n"), 0o644)
	if l := parseXKB(p, "layout", ""); len(l) != 2 || l[1].Code != "de" || l[1].Desc != "German" {
		t.Errorf("layouts: %+v", l)
	}
	if v := parseXKB(p, "variant", "de"); len(v) != 1 || v[0].Code != "nodeadkeys" || v[0].Desc != "German (no dead keys)" {
		t.Errorf("variants: %+v", v)
	}
}

func TestParseLogo(t *testing.T) {
	rows := parseLogo(" \x1b[38;2;126;186;228m@\x1b[38;2;82;119;195m#\x1b[0m\n.")
	if len(rows) != 2 || len(rows[0]) != 3 || rows[0][1].ch != "@" || rows[0][1].r != 126 || rows[1][0].ch != "." {
		t.Errorf("logo: %+v", rows)
	}
}

func TestParseCatalog(t *testing.T) {
	apps, err := parseCatalog([]byte(`{"zed":{"label":"Zed","category":"Development","description":"fast code editor","flatpak":null,"warn":null,"unfree":false},
		"docker":{"label":"Docker","category":"Development","description":"the real Docker","flatpak":null,"warn":"root-equivalent","unfree":false}}`))
	if err != nil || len(apps) != 2 || apps[0].Label != "Docker" || apps[0].Warn != "root-equivalent" {
		t.Errorf("catalog: %+v %v", apps, err)
	}
}

func TestReadAnswers(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "good.json")
	os.WriteFile(good, []byte(`{"hostname":"installtest","username":"alice","diskDevice":"/dev/vda","swapSize":"1G",
		"timeZone":"UTC","desktop":"gnome","profile":"full","nvidia":"none","keyboardLayout":"us","keyboardVariant":"",
		"locale":"en_US.UTF-8","extras":false,"laptop":false,"fingerprint":false,"useOpenSnitch":true,
		"useControlD":false,"usePassSshAgent":false,"apps":["glow"],"passphrase":"correct horse","password":"pw"}`), 0o600)
	a, err := readAnswers(good)
	if err != nil {
		t.Fatal(err)
	}
	if a.Disk != "/dev/vda" || a.Hostname != "installtest" || len(a.Apps) != 1 || a.ControlD || !a.OpenSnitch {
		t.Errorf("answers: %+v", a)
	}

	bad := filepath.Join(dir, "bad.json")
	os.WriteFile(bad, []byte(`{"hostname":"My Laptop","diskDevice":"/dev/vda","passphrase":"short","password":"pw"}`), 0o600)
	_, err = readAnswers(bad)
	if err == nil || !strings.Contains(err.Error(), "hostname") || !strings.Contains(err.Error(), "passphrase") {
		t.Errorf("want hostname and passphrase errors, got %v", err)
	}
}

func TestFollower(t *testing.T) {
	p := filepath.Join(t.TempDir(), "output")
	f := follower{path: p}
	os.WriteFile(p, []byte("==> partitioning\npart"), 0o600)
	if got := f.next(); len(got) != 1 || got[0] != "==> partitioning" {
		t.Errorf("first read: %q", got)
	}
	fh, _ := os.OpenFile(p, os.O_APPEND|os.O_WRONLY, 0o600)
	fh.WriteString("ial line\n\x1b[1mbold\x1b[0m\n")
	fh.Close()
	if got := f.next(); len(got) != 2 || got[0] != "partial line" || got[1] != "bold" {
		t.Errorf("second read: %q", got)
	}
}

func TestHeadlessDryRun(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	dryRunPause = time.Millisecond
	a := defaultAnswers()
	a.Disk, a.Passphrase, a.Password = "/dev/vda", "correct horse", "pw"
	if code := runHeadless(config{dryRun: true}, a); code != 0 {
		t.Errorf("dry run exited %d", code)
	}
	t.Setenv("EPHEMERAL_DRY_RUN_FAIL", "1")
	if code := runHeadless(config{dryRun: true}, a); code != 1 {
		t.Errorf("failing dry run exited %d", code)
	}
}

func TestDryRunNeedsThePassphrase(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	dryRunPause = time.Millisecond
	a := defaultAnswers()
	a.Disk, a.Password = "/dev/vda", "pw"
	if code := runHeadless(config{dryRun: true}, a); code != 3 {
		t.Errorf("an empty passphrase should reach the script empty and fail it, got %d", code)
	}
}
