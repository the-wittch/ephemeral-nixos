package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type partition struct {
	Name, FSType, Label string
	Size                uint64
}

type disk struct {
	Path, Name, Tran, Model string
	Size                    uint64
	InUse                   bool
	// an ISO written to it, like the stick this runs from; refused even when it isn't mounted
	Installer bool
	// "LVM", "ZFS", "RAID" or empty
	Member     string
	Partitions []partition
}

type lsblkDev struct {
	Name        string     `json:"name"`
	Path        string     `json:"path"`
	Type        string     `json:"type"`
	Size        uint64     `json:"size"`
	Tran        *string    `json:"tran"`
	Model       *string    `json:"model"`
	FSType      *string    `json:"fstype"`
	Label       *string    `json:"label"`
	PartLabel   *string    `json:"partlabel"`
	Mountpoints []*string  `json:"mountpoints"`
	Children    []lsblkDev `json:"children"`
}

func str(p *string) string {
	if p == nil {
		return ""
	}
	return strings.TrimSpace(*p)
}

// in use the way common.sh's disk_in_use sees it: mounted, swap, or under LUKS, LVM or RAID
func inUse(d lsblkDev) bool {
	for _, m := range d.Mountpoints {
		if str(m) != "" {
			return true
		}
	}
	if d.Type == "crypt" || d.Type == "lvm" || strings.HasPrefix(d.Type, "raid") {
		return true
	}
	for _, c := range d.Children {
		if inUse(c) {
			return true
		}
	}
	return false
}

// booted with copytoram, the installer stick isn't mounted, so in use alone misses it
func hasISO(d lsblkDev) bool {
	if str(d.FSType) == "iso9660" {
		return true
	}
	for _, c := range d.Children {
		if hasISO(c) {
			return true
		}
	}
	return false
}

func storageMember(d lsblkDev) string {
	switch f := str(d.FSType); {
	case f == "LVM2_member":
		return "LVM"
	case f == "zfs_member":
		return "ZFS"
	case strings.HasSuffix(f, "_raid_member"):
		return "RAID"
	}
	for _, c := range d.Children {
		if k := storageMember(c); k != "" {
			return k
		}
	}
	return ""
}

// the same command as disk_identity in common.sh, so install.sh can compare the two as they are
func diskIdentity(path string) (string, error) {
	out, err := exec.Command("lsblk", "-dbP", "-o", "SERIAL,WWN,SIZE,MODEL", "--", path).Output()
	if err != nil {
		return "", fmt.Errorf("lsblk: %w", err)
	}
	return strings.TrimRight(string(out), "\n"), nil
}

func parseDisks(data []byte) ([]disk, error) {
	var out struct {
		Devices []lsblkDev `json:"blockdevices"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	var disks []disk
	for _, d := range out.Devices {
		// zram is swap in memory, not somewhere to install
		if d.Type != "disk" || strings.HasPrefix(d.Name, "zram") {
			continue
		}
		dk := disk{Path: d.Path, Name: d.Name, Tran: str(d.Tran), Model: str(d.Model), Size: d.Size, InUse: inUse(d), Installer: hasISO(d), Member: storageMember(d)}
		for _, p := range d.Children {
			if p.Type != "part" {
				continue
			}
			label := str(p.PartLabel)
			if l := str(p.Label); l != "" {
				label = l
			}
			dk.Partitions = append(dk.Partitions, partition{Name: p.Name, FSType: str(p.FSType), Label: label, Size: p.Size})
		}
		disks = append(disks, dk)
	}
	return disks, nil
}

func listDisks() ([]disk, error) {
	if p := os.Getenv("EPHEMERAL_LSBLK_JSON"); p != "" {
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		return parseDisks(b)
	}
	out, err := exec.Command("lsblk", "-J", "-b", "-o", "NAME,PATH,TYPE,SIZE,TRAN,MODEL,MOUNTPOINTS,FSTYPE,LABEL,PARTLABEL").Output()
	if err != nil {
		return nil, fmt.Errorf("lsblk: %w", err)
	}
	return parseDisks(out)
}

func humanSize(b uint64) string {
	units := []string{"B", "K", "M", "G", "T"}
	v := float64(b)
	i := 0
	for v >= 1024 && i < len(units)-1 {
		v /= 1024
		i++
	}
	if v >= 100 || i == 0 {
		return fmt.Sprintf("%.0f%s", v, units[i])
	}
	return fmt.Sprintf("%.1f%s", v, units[i])
}

func readTrim(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// vendor 10de, pci class 03 (display)
func nvidiaGPU() (string, bool) {
	devs, _ := filepath.Glob("/sys/bus/pci/devices/*")
	for _, d := range devs {
		if readTrim(d+"/vendor") != "0x10de" || !strings.HasPrefix(readTrim(d+"/class"), "0x03") {
			continue
		}
		name := "an NVIDIA card"
		if out, err := exec.Command("lspci", "-s", filepath.Base(d)).Output(); err == nil {
			s := string(out)
			if i := strings.Index(s, ": "); i >= 0 {
				s = s[i+2:]
			}
			s = revRe.ReplaceAllString(strings.TrimSpace(s), "")
			if i := strings.LastIndex(s, "["); i >= 0 && strings.HasSuffix(s, "]") {
				s = s[i+1 : len(s)-1]
			}
			if s != "" {
				name = s
			}
		}
		return printable(name), true
	}
	return "", false
}

func hasBattery() bool {
	m, _ := filepath.Glob("/sys/class/power_supply/BAT*")
	return len(m) > 0
}

// usb vendors that make fingerprint readers, the same list as common.sh
func fingerprintReader() (string, bool) {
	vendors := map[string]bool{"27c6": true, "06cb": true, "138a": true, "04f3": true, "1c7a": true, "2808": true, "08ff": true, "147e": true, "10a5": true}
	devs, _ := filepath.Glob("/sys/bus/usb/devices/*")
	for _, d := range devs {
		v := readTrim(d + "/idVendor")
		if !vendors[v] {
			continue
		}
		name := strings.TrimSpace(readTrim(d+"/manufacturer") + " " + readTrim(d+"/product"))
		if name == "" {
			name = "a fingerprint reader"
		}
		return printable(fmt.Sprintf("%s (%s)", name, v)), true
	}
	return "", false
}

func memoryGiB() int {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) >= 2 && fields[0] == "MemTotal:" {
			kb, _ := strconv.Atoi(fields[1])
			return (kb + 512*1024) / (1024 * 1024)
		}
	}
	return 0
}

func isUEFI() bool {
	_, err := os.Stat("/sys/firmware/efi")
	return err == nil
}

func online() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://cache.nixos.org/nix-cache-info", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

var revRe = regexp.MustCompile(`\s*\(rev [0-9a-f]+\)$`)

var configTagRe = regexp.MustCompile(`ephemeral-nixos-config/(v[0-9]+\.[0-9]+\.[0-9]+)`)

func configVersion(flakeRoot string) string {
	b, err := os.ReadFile(filepath.Join(flakeRoot, "flake.nix"))
	if err != nil {
		return ""
	}
	if m := configTagRe.FindSubmatch(b); m != nil {
		return string(m[1])
	}
	return ""
}

// device names end up on screen; drop anything that isn't printable
func printable(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, s)
}

// The console keymaps from kbd are mostly named after the XKB layout. Switching the live
// console means the passphrase is typed here the way it will be at the boot prompt.
// Returns the layout now active, or "" when it couldn't be switched.
func switchConsoleLayout(layout, variant string, dryRun bool) string {
	if dryRun || os.Getenv("TERM") != "linux" || strings.Contains(layout, ",") {
		return ""
	}
	keymap := layout
	if layout == "gb" {
		keymap = "uk"
	}
	if variant != "" {
		keymap += "-" + variant
	}
	out, err := exec.Command("loadkeys", keymap).CombinedOutput()
	if err != nil {
		uiLog.Printf("loadkeys %s: %v: %s", keymap, err, strings.TrimSpace(string(out)))
		return ""
	}
	return layout
}

// below this a full install fills up fast, since old generations stay until they're cleaned up
const minDiskSize = 32 << 30
