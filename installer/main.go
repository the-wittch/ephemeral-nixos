package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"

	// before Bubble Tea's init, so it doesn't wait on the terminal
	_ "ephemeral-install/internal/early"
)

var version = ""

// what went wrong in the screens themselves; install.sh keeps its own log
var uiLog = log.New(io.Discard, "", log.LstdFlags)

// The passphrase and password are Go strings, which can't be wiped, so a core dump would hold
// them. Not dumpable also keeps other processes from attaching to read them.
func noCoreDumps() error {
	const prSetDumpable = 4
	if _, _, errno := syscall.RawSyscall(syscall.SYS_PRCTL, prSetDumpable, 0, 0); errno != 0 {
		return errno
	}
	return nil
}

func main() {
	if err := noCoreDumps(); err != nil {
		fmt.Fprintln(os.Stderr, "warning: couldn't turn off core dumps:", err)
	}
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
	noEffects := flag.Bool("no-effects", false, "no animations; EPHEMERAL_NO_EFFECTS=1 does the same")
	flag.Parse()
	cfg.flakeRoot = filepath.Clean(filepath.Join(cfg.scriptsDir, ".."))
	cfg.forceDisk = os.Getenv("FORCE_DISK") == "1"
	cfg.effects = !*noEffects && os.Getenv("EPHEMERAL_NO_EFFECTS") != "1"
	cfg.version = version

	if !cfg.dryRun && os.Geteuid() != 0 {
		fmt.Fprintln(os.Stderr, "run it as root: sudo ephemeral-install, or add --dry-run to look around")
		os.Exit(1)
	}
	if _, err := os.Stat(filepath.Join(cfg.scriptsDir, "install.sh")); err != nil && !cfg.dryRun {
		fmt.Fprintf(os.Stderr, "no install.sh in %s (set --scripts)\n", cfg.scriptsDir)
		os.Exit(1)
	}
	var err error
	if rules, err = loadContract(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	// the same settings install.sh gets from common.sh
	if b, err := os.ReadFile(filepath.Join(cfg.scriptsDir, "nix.conf")); err == nil {
		os.Setenv("NIX_CONFIG", os.Getenv("NIX_CONFIG")+"\n"+string(b))
	}

	logPath := uiLogPath()
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
		if w := answersFileWarning(answersFile); w != "" {
			fmt.Fprintln(os.Stderr, "warning:", w)
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

func uiLogPath() string { return filepath.Join(os.TempDir(), "ephemeral-install-ui.log") }
