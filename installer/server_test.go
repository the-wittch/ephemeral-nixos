package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const (
	testKey  = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIMSJH062QE//0aztsSAqLUYZjcQNSxw66LXQo8vT+ecu alice@laptop"
	otherKey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGJq3S2DS9yNDqYV0EZJy1yVExmn2JqP6m0ZDp6mGmXn"
)

// welcome to Machine on a plasma or headless full install, the defaults otherwise
func toServer(t *testing.T, m model, headless bool) model {
	t.Helper()
	m = press(t, m, enter, enter)
	wantStep(t, m, stDesktop)
	if headless {
		for range m.choices(stDesktop) {
			m = press(t, m, down)
		}
		if m.a.Desktop != "none" {
			t.Fatalf("the last desktop should be none, got %q", m.a.Desktop)
		}
	}
	m = press(t, m, enter, enter, enter, enter, enter, enter, enter)
	wantStep(t, m, stApps)
	m = press(t, m, enter, enter, enter, enter)
	wantStep(t, m, stServer)
	return m
}

func TestServerNeedsAKey(t *testing.T) {
	m := toServer(t, newTestModel(t), false)
	m = press(t, m, enter)
	wantStep(t, m, stSecrets)
	if m.a.Server {
		t.Error("a desktop isn't a server unless it's ticked")
	}

	m = press(t, m, esc, typed("x"))
	if !m.a.Server || m.serverFocus != 1 {
		t.Fatalf("ticking it should go to the key field: server %v, focus %d", m.a.Server, m.serverFocus)
	}
	m = press(t, m, enter)
	wantStep(t, m, stServer)
	if !strings.Contains(m.problem(), "at least one") {
		t.Errorf("enter without a key should say why it stays: %q", m.problem())
	}
	// the key without its type, as when only the middle was copied
	m = press(t, m, typed(strings.Fields(testKey)[1]), enter)
	if len(m.a.SSHKeys) != 0 || !strings.Contains(m.problem(), "public key, such as") {
		t.Errorf("something that isn't a whole public key was taken: %v, %q", m.a.SSHKeys, m.problem())
	}
	m = press(t, m, ctrlU, typed(testKey), enter)
	if !slices.Equal(m.a.SSHKeys, []string{testKey}) || m.keyIn.Value() != "" {
		t.Errorf("keys %v, field %q", m.a.SSHKeys, m.keyIn.Value())
	}
	m = press(t, m, typed(strings.TrimSuffix(testKey, " alice@laptop")+" again"), enter)
	if len(m.a.SSHKeys) != 1 {
		t.Errorf("the same key with another comment was added twice: %v", m.a.SSHKeys)
	}
	m = press(t, m, typed(otherKey), enter, down, typed("x"))
	if !slices.Equal(m.a.SSHKeys, []string{otherKey}) {
		t.Errorf("x on the first key should remove it: %v", m.a.SSHKeys)
	}
	m = press(t, m, enter)
	wantStep(t, m, stSecrets)

	m.a.Disk = "/dev/vda"
	env := strings.Join(m.a.env(), "\n")
	if !strings.Contains(env, "SERVER=true\n") || !strings.Contains(env, "SSH_KEYS="+otherKey+"\n") {
		t.Errorf("install.sh doesn't get the server answers:\n%s", env)
	}
}

func TestNoDesktopIsAServer(t *testing.T) {
	m := newTestModel(t)
	m = update(t, m, catalogMsg{apps: []app{
		{ID: "firefox", Label: "Firefox", Category: "Browsers", Default: true},
		{ID: "htop", Label: "htop", Category: "Terminal", Default: true},
		{ID: "tmux", Label: "tmux", Category: "Terminal"},
	}})
	m = press(t, m, enter, enter)
	for range m.choices(stDesktop) {
		m = press(t, m, down)
	}
	m = press(t, m, enter, enter, enter, enter, enter, enter)
	wantStep(t, m, stOptions)
	for _, r := range m.optRows() {
		if r == optOpenSnitch {
			t.Error("OpenSnitch is offered without a desktop to ask its questions on")
		}
	}
	m = press(t, m, enter)
	wantStep(t, m, stApps)
	if cats := m.appCategories(); !slices.Equal(cats, []string{"Terminal"}) {
		t.Errorf("without a desktop only terminal apps should be offered, got %v", cats)
	}
	m = press(t, m, enter)
	if !slices.Equal(m.a.Apps, []string{"htop"}) {
		t.Errorf("apps: %v", m.a.Apps)
	}

	m = press(t, m, enter, enter, enter)
	wantStep(t, m, stServer)
	if !m.a.Server || m.serverFocus != 1 {
		t.Fatalf("without a desktop it starts as a server on the key field: server %v, focus %d", m.a.Server, m.serverFocus)
	}
	m = press(t, m, up, typed("x"))
	if !m.a.Server {
		t.Error("a machine without a desktop could be unticked as a server")
	}
	m = press(t, m, down, typed(testKey), enter, enter)
	m = toReviewFromSecrets(t, m)
	if l := m.reviewLines(); !slices.ContainsFunc(l, func(r reviewLine) bool { return r.label == "SERVER" && strings.Contains(r.value, "1 key,") }) {
		t.Errorf("the review doesn't show the server: %+v", l)
	}
}

func toReviewFromSecrets(t *testing.T, m model) model {
	t.Helper()
	wantStep(t, m, stSecrets)
	m = press(t, m, typed("correct horse"), tab, typed("correct horse"), tab, typed("pw"), tab, typed("pw"), enter)
	wantStep(t, m, stReview)
	return m
}

func TestNoDesktopFromReviewAsksForKeys(t *testing.T) {
	m := toServer(t, newTestModel(t), false)
	m = toReviewFromSecrets(t, press(t, m, enter))
	m = pickReviewLine(t, m, stDesktop)
	m = press(t, m, enter)
	for range m.choices(stDesktop) {
		m = press(t, m, down)
	}
	m = press(t, m, enter)
	wantStep(t, m, stServer)
	m = press(t, m, typed(testKey), enter, enter)
	wantStep(t, m, stReview)
	if !m.a.Server {
		t.Error("the change to no desktop didn't make it a server")
	}
}

func TestKeysFromGitHub(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/alice.keys":
			fmt.Fprintf(w, "%s\nssh-dss AAAAB3NzaC1kc3MAAACBAP\n%s\n", strings.TrimSuffix(testKey, " alice@laptop"), otherKey)
		case "/nokeys.keys":
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	was := githubKeysURL
	githubKeysURL = srv.URL + "/%s.keys"
	defer func() { githubKeysURL = was }()

	fetch := func(m model, name string) model {
		t.Helper()
		m = press(t, m, ctrlU, typed("github:"+name))
		next, cmd := m.Update(enter)
		m = next.(model)
		if cmd == nil {
			t.Fatalf("github:%s didn't start a fetch (note %q)", name, m.note)
		}
		return update(t, m, cmd())
	}

	m := toServer(t, newTestModel(t), false)
	m = press(t, m, typed("x"))
	m = fetch(m, "alice")
	want := []string{strings.TrimSuffix(testKey, " alice@laptop") + " github:alice", otherKey + " github:alice"}
	if !slices.Equal(m.a.SSHKeys, want) {
		t.Errorf("keys:\n%q\nwant\n%q", m.a.SSHKeys, want)
	}
	if !strings.Contains(m.problem(), "left out 1") {
		t.Errorf("the DSA key should be left out and said so: %q", m.problem())
	}
	if m = fetch(m, "nobody"); !strings.Contains(m.problem(), "no user called nobody") {
		t.Errorf("an unknown user: %q", m.problem())
	}
	if m = fetch(m, "nokeys"); !strings.Contains(m.problem(), "no SSH keys") {
		t.Errorf("a user without keys: %q", m.problem())
	}
	m = press(t, m, ctrlU, typed("github:../x"), enter)
	if m.fetching != "" || !strings.Contains(m.problem(), "GitHub user name") {
		t.Errorf("a name that isn't GitHub's shape went into the URL: %q", m.problem())
	}
	if len(m.a.SSHKeys) != 2 {
		t.Errorf("failed fetches changed the keys: %v", m.a.SSHKeys)
	}
}

// a fetch that's no longer wanted, like after a second one, changes nothing
func TestOnlyTheLatestFetchCounts(t *testing.T) {
	m := toServer(t, newTestModel(t), false)
	m.fetching = "bob"
	m = update(t, m, keysMsg{name: "alice", keys: []string{testKey}})
	if len(m.a.SSHKeys) != 0 || m.fetching != "bob" {
		t.Errorf("keys %v, fetching %q", m.a.SSHKeys, m.fetching)
	}
}

func TestServerAnswers(t *testing.T) {
	dir := t.TempDir()
	base := `"diskDevice":"/dev/vda","passphrase":"correct horse","password":"pw"`
	for name, c := range map[string]struct{ json, want string }{
		"no keys":       {`"server":true`, "sshKeys: a server needs at least one"},
		"no desktop":    {`"desktop":"none","sshKeys":["` + testKey + `"]`, `server: has to be true with "desktop": "none"`},
		"no type":       {`"server":true,"sshKeys":["AAAAC3NzaC1lZDI1NTE5AAAAIMSJH062QE"]`, "public key, such as"},
		"quote in name": {`"server":true,"sshKeys":["` + testKey + `\"x"]`, "sshKeys"},
	} {
		p := filepath.Join(dir, strings.ReplaceAll(name, " ", "-")+".json")
		os.WriteFile(p, []byte("{"+base+","+c.json+"}"), 0o600)
		if _, err := readAnswers(p); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: want %q, got %v", name, c.want, err)
		}
	}
	good := filepath.Join(dir, "good.json")
	os.WriteFile(good, []byte(`{`+base+`,"desktop":"none","server":true,"sshKeys":["`+testKey+`"]}`), 0o600)
	a, err := readAnswers(good)
	if err != nil {
		t.Fatal(err)
	}
	if !a.Server || !slices.Equal(a.SSHKeys, []string{testKey}) {
		t.Errorf("answers: %+v", a)
	}

	a.Server = false
	for _, kv := range a.env() {
		if kv == "SSH_KEYS="+testKey {
			t.Error("keys went to install.sh for a machine that isn't a server")
		}
	}
}

func TestUnlockFingerprint(t *testing.T) {
	lines := []string{"==> installing", "  256 SHA256:Zk3uP0c8aW9v+2QxYq1tR5sN7mLb4Hd6Jf0eKg2nXyI initrd (ED25519)", "done"}
	if got := unlockFingerprint(lines); got != "SHA256:Zk3uP0c8aW9v+2QxYq1tR5sN7mLb4Hd6Jf0eKg2nXyI" {
		t.Errorf("got %q", got)
	}
	if got := unlockFingerprint([]string{"256 SHA256:abc root@nixos (ED25519)"}); got != "" {
		t.Errorf("another key's fingerprint was taken: %q", got)
	}
}

func TestShortKey(t *testing.T) {
	if got := shortKey(testKey); got != "ssh-ed25519 …6LXQo8vT+ecu alice@laptop" {
		t.Errorf("got %q", got)
	}
}
