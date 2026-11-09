package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// A model as the screens start it, on made-up hardware: one usable disk, no NVIDIA card.
func newTestModel(t *testing.T) model {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("TMPDIR", dir)
	disks := filepath.Join(dir, "lsblk.json")
	if err := os.WriteFile(disks, []byte(lsblkSample), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("EPHEMERAL_LSBLK_JSON", disks)
	dryRunPause = time.Millisecond

	m := newModel(config{dryRun: true, listsDir: dir, scriptsDir: dir, flakeRoot: dir})
	m.uefi, m.hasGPU, m.hasReader = true, false, false
	m.a.Nvidia, m.a.Fingerprint = "none", false
	online := true
	m.isOnline = &online
	m = update(t, m, tea.WindowSizeMsg{Width: 110, Height: 36})
	return update(t, m, catalogMsg{apps: []app{
		{ID: "zed", Label: "Zed", Category: "Development"},
		{ID: "docker", Label: "Docker", Category: "Development", Warn: "root-equivalent"},
	}})
}

func update(t *testing.T, m model, msg tea.Msg) model {
	t.Helper()
	next, _ := m.Update(msg)
	return next.(model)
}

var (
	enter = tea.KeyMsg{Type: tea.KeyEnter}
	esc   = tea.KeyMsg{Type: tea.KeyEsc}
	tab   = tea.KeyMsg{Type: tea.KeyTab}
	up    = tea.KeyMsg{Type: tea.KeyUp}
	down  = tea.KeyMsg{Type: tea.KeyDown}
	space = tea.KeyMsg{Type: tea.KeySpace}
	ctrlU = tea.KeyMsg{Type: tea.KeyCtrlU}
)

func typed(s string) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)} }

func press(t *testing.T, m model, keys ...tea.KeyMsg) model {
	t.Helper()
	for _, k := range keys {
		m = update(t, m, k)
		// every screen has to draw whatever state a key leaves it in
		_ = m.View()
	}
	return m
}

func wantStep(t *testing.T, m model, want stepID) {
	t.Helper()
	if m.step != want {
		t.Fatalf("on %q, want %q (note: %q)", stepName[m.step], stepName[want], m.note)
	}
}

// welcome, disk, desktop, region (four fields), install type
func toInstallType(t *testing.T, m model) model {
	t.Helper()
	m = press(t, m, enter)
	wantStep(t, m, stDisk)
	m = press(t, m, enter)
	wantStep(t, m, stDesktop)
	m = press(t, m, enter)
	wantStep(t, m, stRegion)
	m = press(t, m, enter, enter, enter, enter)
	wantStep(t, m, stType)
	return m
}

func toReviewFromMachine(t *testing.T, m model) model {
	t.Helper()
	wantStep(t, m, stMachine)
	m = press(t, m, enter)
	wantStep(t, m, stSecrets)
	m = press(t, m, typed("correct horse"), tab, typed("correct horse"), tab, typed("pw"), tab, typed("pw"), enter)
	wantStep(t, m, stReview)
	return m
}

// up leaves the confirmation for the list, which starts at its first line
func pickReviewLine(t *testing.T, m model, s stepID) model {
	t.Helper()
	m = press(t, m, up)
	for range m.reviewLines() {
		if m.reviewLines()[m.reviewSel].step == s {
			return m
		}
		m = press(t, m, down)
	}
	t.Fatalf("no review line for %q", stepName[s])
	return m
}

func TestFullInstallFlow(t *testing.T) {
	m := newTestModel(t)
	if n := len(m.visible()); n != 9 {
		t.Errorf("without a GPU there should be 9 steps, got %d", n)
	}
	m = toInstallType(t, m)
	m = press(t, m, enter)
	wantStep(t, m, stOptions)
	m = press(t, m, space)
	if !m.a.Extras {
		t.Error("space should tick extras")
	}
	m = press(t, m, enter)
	wantStep(t, m, stApps)
	m = press(t, m, tab, space, enter)
	m = toReviewFromMachine(t, m)

	if len(m.a.Apps) != 1 || m.a.Apps[0] != "zed" {
		t.Errorf("apps: %v", m.a.Apps)
	}
	if m.a.Disk != "/dev/nvme0n1" || m.a.Desktop != "plasma" || m.a.Profile != "full" {
		t.Errorf("answers: %+v", m.a)
	}
	if m.a.Passphrase != "correct horse" || m.a.Password != "pw" {
		t.Error("secrets weren't kept")
	}
}

func TestBackKeepsAnswers(t *testing.T) {
	m := newTestModel(t)
	m = press(t, m, enter, enter, down, enter)
	wantStep(t, m, stRegion)
	if m.a.Desktop != "gnome" {
		t.Fatalf("desktop: %q", m.a.Desktop)
	}
	m = press(t, m, esc)
	wantStep(t, m, stDesktop)
	if m.choiceCursor(stDesktop) != 1 {
		t.Error("going back should land on the answer already given")
	}
	m = press(t, m, esc, esc)
	wantStep(t, m, stWelcome)
}

func TestSmallSkipsOptionsAndApps(t *testing.T) {
	m := toInstallType(t, newTestModel(t))
	m = press(t, m, down, enter)
	if m.a.Profile != "minimal" || len(m.visible()) != 7 {
		t.Errorf("profile %q, %d steps", m.a.Profile, len(m.visible()))
	}
	m = toReviewFromMachine(t, m)
	m = press(t, m, esc)
	wantStep(t, m, stSecrets)
}

func TestGraphicsOnlyWithNVIDIA(t *testing.T) {
	m := newTestModel(t)
	m.hasGPU, m.gpu, m.a.Nvidia = true, "GeForce RTX 4070 SUPER", "open"
	m = press(t, m, enter, enter, enter)
	wantStep(t, m, stGraphics)
	m = press(t, m, down, enter)
	wantStep(t, m, stRegion)
	if m.a.Nvidia != "legacy" {
		t.Errorf("nvidia: %q", m.a.Nvidia)
	}
}

func TestChangeFromReviewComesBack(t *testing.T) {
	m := toInstallType(t, newTestModel(t))
	m = press(t, m, enter, enter, enter)
	m = toReviewFromMachine(t, m)

	m = pickReviewLine(t, m, stDesktop)
	m = press(t, m, enter)
	wantStep(t, m, stDesktop)
	m = press(t, m, down, down, enter)
	wantStep(t, m, stReview)
	if m.a.Desktop != "niri" {
		t.Errorf("desktop: %q", m.a.Desktop)
	}
}

func TestSwitchingToFullFromReviewAsksTheOptions(t *testing.T) {
	m := toInstallType(t, newTestModel(t))
	m = press(t, m, down, enter)
	m = toReviewFromMachine(t, m)

	m = pickReviewLine(t, m, stType)
	m = press(t, m, enter, up, enter)
	wantStep(t, m, stOptions)
	m = press(t, m, enter)
	wantStep(t, m, stReview)
}

func TestInvalidInputStaysPut(t *testing.T) {
	m := toInstallType(t, newTestModel(t))
	m = press(t, m, enter, enter, enter)
	wantStep(t, m, stMachine)
	m = press(t, m, ctrlU, typed("My Laptop"), enter)
	wantStep(t, m, stMachine)
	m = press(t, m, ctrlU, typed("my-laptop"), enter)
	wantStep(t, m, stSecrets)

	m = press(t, m, typed("short"), tab, typed("short"), tab, typed("pw"), tab, typed("pw"), enter)
	wantStep(t, m, stSecrets)
	if m.secretFocus != 0 {
		t.Errorf("a short passphrase should send the cursor back to it, got field %d", m.secretFocus)
	}
}

func TestInstallNeedsTheDiskName(t *testing.T) {
	m := toInstallType(t, newTestModel(t))
	m = press(t, m, down, enter)
	m = toReviewFromMachine(t, m)

	m = press(t, m, typed("sda"), enter)
	wantStep(t, m, stReview)
	m = press(t, m, ctrlU, typed("nvme0n1"), enter)
	wantStep(t, m, stInstalling)
	m = waitForEnd(t, m)
	wantStep(t, m, stDone)
	if m.curStep != len(installSteps)-1 {
		t.Errorf("the last step reached was %d", m.curStep)
	}
}

func TestLeavingKeepsTheInstallRunning(t *testing.T) {
	m := toInstallType(t, newTestModel(t))
	dryRunPause = 100 * time.Millisecond
	m = press(t, m, down, enter)
	m = toReviewFromMachine(t, m)
	m = press(t, m, typed("nvme0n1"), enter)
	wantStep(t, m, stInstalling)

	next, cmd := m.Update(typed("q"))
	if cmd == nil || cmd() != tea.Quit() || next.(model).quitTo != "shell" {
		t.Fatal("q should quit the screens and leave a shell")
	}
	if !m.run.running() {
		t.Fatal("the install stopped with the screens")
	}

	again := newModel(m.cfg)
	wantStep(t, again, stInstalling)
	if !again.reattached {
		t.Error("a second start should follow the running install")
	}
	again = waitForEnd(t, update(t, again, tea.WindowSizeMsg{Width: 110, Height: 36}))
	wantStep(t, again, stDone)
	if len(again.lines) == 0 || again.lines[0] != "==> a copy of this output goes to /tmp/ephemeral-install.log" {
		t.Errorf("reattaching should read the output from the start, got %q", again.lines)
	}
}

func waitForEnd(t *testing.T, m model) model {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for m.step == stInstalling && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
		m = update(t, m, tickMsg(time.Now()))
		_ = m.View()
	}
	return m
}
