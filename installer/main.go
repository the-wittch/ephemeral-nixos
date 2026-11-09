package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
)

var version = ""

func main() {
	exe, _ := os.Executable()
	defaultScripts := "/etc/ephemeral-nixos/scripts"
	if _, err := os.Stat(defaultScripts); err != nil {
		defaultScripts = filepath.Join(filepath.Dir(exe), "..", "share", "ephemeral-nixos", "scripts")
	}

	var cfg config
	flag.StringVar(&cfg.scriptsDir, "scripts", defaultScripts, "the directory with install.sh")
	flag.StringVar(&cfg.listsDir, "lists", envOr("EPHEMERAL_LISTS", "/etc/ephemeral-nixos-lists"), "keyboard and language lists")
	flag.StringVar(&cfg.logoPath, "logo", os.Getenv("EPHEMERAL_LOGO"), "the snowflake as truecolor characters")
	flag.BoolVar(&cfg.dryRun, "dry-run", false, "go through the screens without installing anything")
	flag.Parse()
	cfg.flakeRoot = filepath.Clean(filepath.Join(cfg.scriptsDir, ".."))
	cfg.forceDisk = os.Getenv("FORCE_DISK") == "1"
	cfg.version = version

	if !cfg.dryRun && os.Geteuid() != 0 {
		fmt.Fprintln(os.Stderr, "run it as root: sudo ephemeral-install, or add --dry-run to look around")
		os.Exit(1)
	}
	if _, err := os.Stat(filepath.Join(cfg.scriptsDir, "install.sh")); err != nil && !cfg.dryRun {
		fmt.Fprintf(os.Stderr, "no install.sh in %s (set --scripts)\n", cfg.scriptsDir)
		os.Exit(1)
	}
	os.Setenv("NIX_CONFIG", os.Getenv("NIX_CONFIG")+"\nexperimental-features = nix-command flakes\nhttp2 = false\ndownload-attempts = 5\nconnect-timeout = 10\n")

	final, err := tea.NewProgram(newModel(cfg), tea.WithAltScreen()).Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	if m, ok := final.(model); ok && m.quitTo == "reboot" && !cfg.dryRun {
		if err := exec.Command("systemctl", "reboot").Run(); err != nil {
			fmt.Fprintln(os.Stderr, "couldn't reboot:", err)
			os.Exit(1)
		}
	}
}
