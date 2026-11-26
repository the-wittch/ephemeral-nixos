package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

var f2 = tea.KeyMsg{Type: tea.KeyF2}

func TestOnScreenKeyboardTypesIntoTheField(t *testing.T) {
	m := newTestModel(t)
	m.width, m.height = 120, 40
	m.step = stMachine
	m.enter()
	// 1 from the top row, then q below it
	m = press(t, m, ctrlU, f2, enter, down, enter)
	if !m.osk.open {
		t.Fatal("F2 didn't open the keyboard")
	}
	// Shift, at the start of the bottom row, then a capital A
	m = press(t, m, down, down, down, enter, up, up, enter)
	if got := m.machineIn[0].Value(); got != "1qA" {
		t.Errorf("typed %q, want 1qA", got)
	}
	// Delete, then Done, which moves on to the user name with the keyboard still up
	m = press(t, m, down, down, right, right, right, enter, right, enter)
	if got := m.machineIn[0].Value(); got != "1q" {
		t.Errorf("after Delete the hostname is %q, want 1q", got)
	}
	if m.machineFocus != 1 || !m.osk.open {
		t.Errorf("after Done: field %d, keyboard open %v; want the user name, open", m.machineFocus, m.osk.open)
	}
	m = press(t, m, esc)
	if m.osk.open || m.step != stMachine {
		t.Errorf("esc should close the keyboard and stay on the screen; open %v, on %q", m.osk.open, stepName[m.step])
	}
}

func TestKeyboardReachesEveryPrintableCharacter(t *testing.T) {
	have := map[rune]bool{' ': true}
	for _, layer := range [][][]string{oskLetters, oskSymbols} {
		for _, row := range layer {
			for _, k := range row {
				for _, r := range k {
					have[r] = true
					if r >= 'a' && r <= 'z' {
						have[r-'a'+'A'] = true
					}
				}
			}
		}
	}
	for r := rune(0x20); r < 0x7f; r++ {
		if !have[r] {
			t.Errorf("%q isn't on the keyboard", r)
		}
	}
}

func TestF2OnTheAppListSearches(t *testing.T) {
	m := newTestModel(t)
	m.width, m.height = 120, 40
	m.step = stApps
	m = press(t, m, f2)
	if !m.appSearching || !m.osk.open {
		t.Errorf("F2 on the app list: searching %v, keyboard %v; want both", m.appSearching, m.osk.open)
	}
}

func TestParseWifi(t *testing.T) {
	out := "Home\\:Net:70:WPA2\nHome\\:Net:40:WPA2\n:90:WPA2\nCafe:55:--\nBack\\\\slash:20:WPA3\n"
	got := parseWifi(out)
	want := []network{{"Home:Net", 70, "WPA2"}, {"Cafe", 55, "--"}, {"Back\\slash", 20, "WPA3"}}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("network %d: got %v, want %v", i, got[i], want[i])
		}
	}
	if !got[1].open() || got[0].open() {
		t.Error("Cafe is open and Home isn't")
	}
}

func TestWifiKeyfile(t *testing.T) {
	got := wifiKeyfile(network{"a;b", 50, "WPA2"}, ` p\ss`)
	for _, want := range []string{"ssid=97;59;98;\n", "key-mgmt=wpa-psk\n", `psk=\sp\\ss` + "\n"} {
		if !strings.Contains(got, want) {
			t.Errorf("no %q in\n%s", want, got)
		}
	}
	if open := wifiKeyfile(network{"Cafe", 50, "--"}, ""); strings.Contains(open, "wifi-security") {
		t.Errorf("an open network got a password section:\n%s", open)
	}
	if sae := wifiKeyfile(network{"New", 50, "WPA3"}, "longenough"); !strings.Contains(sae, "key-mgmt=sae\n") {
		t.Errorf("WPA3 alone should be sae:\n%s", sae)
	}
}
