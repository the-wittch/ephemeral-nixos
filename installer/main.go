package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
)

var version = ""

// what went wrong in the screens themselves; install.sh keeps its own log
var uiLog = log.New(io.Discard, "", log.LstdFlags)

func main() {
	exe, _ := os.Executable()
	share := filepath.Join(filepath.Dir(exe), "..", "share")
	defaultScripts := "/etc/ephemeral-nixos/scripts"
	if _, err := os.Stat(defaultScripts); err != nil {
		defaultScripts = filepath.Join(share, "ephemeral-nixos", "scripts")
	}
	defaultLists := "/etc/ephemeral-nixos-lists"
	if _, err := os.Stat(defaultLists); err != nil {
		defaultLists = filepath.Join(share, "ephemeral-nixos-lists")
	}

	var cfg config
	var answersFile string
	flag.StringVar(&cfg.scriptsDir, "scripts", defaultScripts, "the directory with install.sh")
	flag.StringVar(&cfg.listsDir, "lists", envOr("EPHEMERAL_LISTS", defaultLists), "keyboard and language lists")
	flag.StringVar(&cfg.logoPath, "logo", os.Getenv("EPHEMERAL_LOGO"), "the snowflake as truecolor characters")
	flag.BoolVar(&cfg.dryRun, "dry-run", false, "go through the screens without installing anything")
	flag.StringVar(&answersFile, "answers", "", "install from a JSON file of answers, without the screens")
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

	logPath := filepath.Join(os.TempDir(), "ephemeral-install-ui.log")
	if f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600); err == nil {
		defer f.Close()
		uiLog.SetOutput(f)
	}

	if answersFile != "" {
		a, err := readAnswers(answersFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(2)
		}
		os.Exit(runHeadless(cfg, a))
	}

	final, err := tea.NewProgram(newModel(cfg), tea.WithAltScreen()).Run()
	if err != nil {
		uiLog.Printf("the screens stopped: %v", err)
		fmt.Fprintf(os.Stderr, "error: %v\nDetails are in %s.\n", err, logPath)
	}
	m, ok := final.(model)
	if !ok {
		os.Exit(1)
	}
	if m.step == stInstalling {
		fmt.Println("The install keeps running. Run ephemeral-install again to follow it.")
	}
	if m.quitTo == "reboot" && !cfg.dryRun {
		if err := exec.Command("systemctl", "reboot").Run(); err != nil {
			fmt.Fprintln(os.Stderr, "couldn't reboot:", err)
			os.Exit(1)
		}
	}
}
