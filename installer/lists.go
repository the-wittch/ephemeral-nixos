package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type item struct {
	Code, Desc string
}

func firstReadable(paths ...string) (string, bool) {
	for _, p := range paths {
		if f, err := os.Open(p); err == nil {
			f.Close()
			return p, true
		}
	}
	return "", false
}

func xkbRules(lists string) (string, bool) {
	return firstReadable(
		filepath.Join(lists, "base.lst"),
		"/etc/X11/xkb/rules/base.lst",
		"/run/current-system/sw/share/X11/xkb/rules/base.lst",
	)
}

// base.lst sections look like "! layout" followed by "  de   German"; variants are "  nodeadkeys   de: German (no dead keys)"
func parseXKB(path, section, layout string) []item {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	var items []item
	on := false
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "!") {
			on = strings.TrimSpace(strings.TrimPrefix(line, "!")) == section
			continue
		}
		fields := strings.Fields(line)
		if !on || len(fields) < 2 {
			continue
		}
		desc := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), fields[0]))
		if section == "variant" {
			if fields[1] != layout+":" {
				continue
			}
			desc = strings.TrimSpace(strings.TrimPrefix(desc, fields[1]))
		}
		items = append(items, item{Code: fields[0], Desc: desc})
	}
	return items
}

func locales(lists string) []item {
	p, ok := firstReadable(filepath.Join(lists, "locales"), "/run/current-system/sw/share/i18n/SUPPORTED")
	if !ok {
		return nil
	}
	f, err := os.Open(p)
	if err != nil {
		return nil
	}
	defer f.Close()
	var items []item
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		code, _, _ := strings.Cut(strings.TrimSpace(sc.Text()), "/")
		if strings.HasSuffix(code, ".UTF-8") {
			items = append(items, item{Code: code})
		}
	}
	return items
}

func timezones() []item {
	out, err := exec.Command("timedatectl", "list-timezones").Output()
	if err != nil {
		return []item{{Code: "UTC"}}
	}
	var items []item
	for _, z := range strings.Fields(string(out)) {
		items = append(items, item{Code: z})
	}
	return items
}

// put the default first, so it's what a bare enter picks
func defaultFirst(items []item, code string) []item {
	out := make([]item, 0, len(items))
	for _, it := range items {
		if it.Code == code {
			out = append(out, it)
		}
	}
	for _, it := range items {
		if it.Code != code {
			out = append(out, it)
		}
	}
	return out
}

type app struct {
	ID, Label, Category, Description, Warn string
	Flatpak, Unfree                        bool
}

var categoryOrder = []string{
	"Browsers", "Communication", "Office", "Media", "Graphics", "Gaming",
	"Development", "Security", "Terminal", "Utilities", "System", "Virtualization",
}

func parseCatalog(data []byte) ([]app, error) {
	var raw map[string]struct {
		Label       string  `json:"label"`
		Category    string  `json:"category"`
		Description string  `json:"description"`
		Flatpak     *string `json:"flatpak"`
		Warn        *string `json:"warn"`
		Unfree      bool    `json:"unfree"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	apps := make([]app, 0, len(raw))
	for id, a := range raw {
		apps = append(apps, app{
			ID: id, Label: a.Label, Category: a.Category, Description: a.Description,
			Flatpak: a.Flatpak != nil, Unfree: a.Unfree, Warn: str(a.Warn),
		})
	}
	sort.Slice(apps, func(i, j int) bool {
		return strings.ToLower(apps[i].Label) < strings.ToLower(apps[j].Label)
	})
	return apps, nil
}

// the config's catalog, built from the release this installer pins; the first build fetches nixpkgs
func loadCatalog(flakeRoot string) ([]app, error) {
	if p := os.Getenv("EPHEMERAL_CATALOG"); p != "" {
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		return parseCatalog(b)
	}
	ref := "path:" + flakeRoot
	out, err := exec.Command("nix", "build", "--no-link", "--print-out-paths", ref+"#app-catalog").Output()
	if err != nil {
		return nil, fmt.Errorf("nix build app-catalog: %w", err)
	}
	b, err := os.ReadFile(strings.TrimSpace(string(out)))
	if err != nil {
		return nil, err
	}
	return parseCatalog(b)
}
