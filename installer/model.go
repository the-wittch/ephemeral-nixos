package main

import (
	"errors"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type stepID int

const (
	stWelcome stepID = iota
	stDisk
	stDesktop
	stGraphics
	stRegion
	stType
	stOptions
	stApps
	stMachine
	stSecrets
	stReview
	stInstalling
	stDone
	stFailed
)

var questions = []stepID{stDisk, stDesktop, stGraphics, stRegion, stType, stOptions, stApps, stMachine, stSecrets, stReview}

var stepName = map[stepID]string{
	stDisk: "Disk", stDesktop: "Desktop", stGraphics: "Graphics", stRegion: "Region",
	stType: "Install type", stOptions: "Options", stApps: "Apps", stMachine: "Machine",
	stSecrets: "Secrets", stReview: "Review",
}

var errNoMatch = errors.New("nothing matches")

type config struct {
	scriptsDir, flakeRoot, listsDir, logoPath, version string
	dryRun, forceDisk, effects                         bool
}

type model struct {
	cfg           config
	width, height int
	step          stepID
	history       []stepID
	editing       bool
	a             answers
	note          string

	isOnline    *bool
	onlineRound int
	uefi        bool
	memGiB      int
	disks       []disk
	diskErr     error
	gpu         string
	hasGPU      bool
	laptop      bool
	reader      string
	hasReader   bool
	configVer   string
	logo        [][]cell
	spinFrame   int
	// spinFrame when the current screen appeared, for effects that play once
	stepShown int
	// set by View when there's room for the frame
	framed     bool
	listCursor map[stepID]int

	region      []*picker
	regionFocus int
	regionErr   string

	optFocus    int
	vaultIn     textinput.Model
	controldIn  textinput.Model
	customDNSIn textinput.Model

	catalog        []app
	catalogErr     error
	catalogLoading bool
	appCat         int
	appIdx         int
	appPane        int
	picked         map[string]bool

	machineIn    []textinput.Model
	machineFocus int

	secretIn    []textinput.Model
	secretFocus int

	reviewSel   int
	reviewFocus int
	confirmIn   textinput.Model

	run        runDir
	follow     *follower
	reattached bool
	curStep    int
	lines      []string
	started    time.Time
	finished   time.Time
	exitCode   int
	showLog    bool
	logView    viewport.Model
	endSel     int

	liveLayout string

	quitTo string
}

// round counts the checks started after nmtui, so an older round stops instead of running alongside
type onlineMsg struct {
	ok    bool
	round int
}
type catalogMsg struct {
	apps []app
	err  error
}
type tickMsg time.Time
type execDoneMsg struct{}

func newInput(value, placeholder string) textinput.Model {
	in := textinput.New()
	in.Prompt = ""
	in.SetValue(value)
	in.Placeholder = placeholder
	in.CharLimit = 128
	in.Cursor.Style = sCursor
	in.PlaceholderStyle = sDim
	return in
}

func newSecret() textinput.Model {
	in := newInput("", "")
	in.EchoMode = textinput.EchoPassword
	in.EchoCharacter = '•'
	return in
}

func newModel(cfg config) model {
	m := model{
		cfg:            cfg,
		uefi:           isUEFI(),
		memGiB:         memoryGiB(),
		laptop:         hasBattery(),
		configVer:      configVersion(cfg.flakeRoot),
		logo:           loadLogo(cfg.logoPath),
		listCursor:     map[stepID]int{},
		picked:         map[string]bool{},
		catalogLoading: true,
		a:              defaultAnswers(),
		run:            newRunDir(cfg.dryRun),
	}
	if m.run.running() {
		m.reattach()
	} else if m.run.died() {
		m.note = "The last install stopped without finishing. Its output is in /tmp/ephemeral-install.log."
	}
	m.gpu, m.hasGPU = nvidiaGPU()
	if m.hasGPU {
		m.a.Nvidia = "open"
	}
	m.reader, m.hasReader = fingerprintReader()
	m.a.Fingerprint = m.hasReader

	layouts, variants := []item{{Code: "us", Desc: "English (US)"}}, []item{{Code: "", Desc: "default"}}
	if rules, ok := xkbRules(cfg.listsDir); ok {
		if l := parseXKB(rules, "layout", ""); len(l) > 0 {
			layouts = defaultFirst(l, "us")
		}
		variants = append(variants, parseXKB(rules, "variant", "us")...)
	}
	locs := locales(cfg.listsDir)
	if len(locs) == 0 {
		locs = []item{{Code: "en_US.UTF-8"}}
	}
	m.region = []*picker{
		newPicker("Keyboard layout", layouts, "us", true, checkLayout),
		newPicker("Variant", variants, "", true, checkVariant),
		newPicker("Language", defaultFirst(locs, "en_US.UTF-8"), "en_US.UTF-8", false, checkLocale),
		newPicker("Time zone", defaultFirst(timezones(), "UTC"), "UTC", true, checkTimezone),
	}

	m.vaultIn = newInput(m.a.Vault, "Personal")
	m.controldIn = newInput("", "abcd1234")
	m.customDNSIn = newInput("", "194.242.2.2#dns.mullvad.net")
	// room for four servers, IPv4 and IPv6 for two providers
	m.customDNSIn.CharLimit = 256
	m.machineIn = []textinput.Model{
		newInput(m.a.Hostname, "nixos"),
		newInput(m.a.Username, "nixuser"),
		newInput(m.a.Swap, "4G"),
	}
	m.secretIn = []textinput.Model{newSecret(), newSecret(), newSecret(), newSecret()}
	m.confirmIn = newInput("", "")
	m.logView = viewport.New(80, 20)
	return m
}

func (m model) Init() tea.Cmd {
	return tea.Batch(checkOnline(0, 0), tick())
}

func checkOnline(delay time.Duration, round int) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(delay)
		return onlineMsg{online(), round}
	}
}

func tick() tea.Cmd {
	return tea.Tick(120*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func loadCatalogCmd(cfg config) tea.Cmd {
	return func() tea.Msg {
		apps, err := loadCatalog(cfg.flakeRoot)
		return catalogMsg{apps, err}
	}
}

func (m model) skipped(s stepID) bool {
	switch s {
	case stGraphics:
		return !m.hasGPU
	case stOptions, stApps:
		return m.a.Profile != "full"
	}
	return false
}

// the question steps that apply, for "3 of 10"
func (m model) visible() []stepID {
	var out []stepID
	for _, s := range questions {
		if !m.skipped(s) {
			out = append(out, s)
		}
	}
	return out
}

func (m *model) goTo(s stepID) tea.Cmd {
	m.history = append(m.history, m.step)
	m.step = s
	m.stepShown = m.spinFrame
	m.note = ""
	return m.enter()
}

func (m *model) next() tea.Cmd {
	if m.editing {
		// a full install needs its options answered even when the change came from the review
		if m.step == stType && m.a.Profile == "full" {
			return m.goTo(stOptions)
		}
		m.editing = false
		return m.goTo(stReview)
	}
	for i, s := range questions {
		if s != m.step {
			continue
		}
		for _, n := range questions[i+1:] {
			if !m.skipped(n) {
				return m.goTo(n)
			}
		}
	}
	return nil
}

func (m *model) back() tea.Cmd {
	if len(m.history) == 0 {
		return nil
	}
	m.step = m.history[len(m.history)-1]
	m.stepShown = m.spinFrame
	m.history = m.history[:len(m.history)-1]
	m.note = ""
	if m.step == stReview {
		m.editing = false
	}
	return m.enter()
}

// runs when a step is shown: focus its first field
func (m *model) enter() tea.Cmd {
	switch m.step {
	case stDisk:
		m.disks, m.diskErr = listDisks()
		m.placeDiskCursor()
	case stRegion:
		m.regionFocus = 0
		m.regionErr = ""
		return m.region[0].focus()
	case stOptions:
		m.optFocus = 0
		m.vaultIn.Blur()
		m.controldIn.Blur()
		m.customDNSIn.Blur()
	case stMachine:
		m.machineFocus = 0
		return m.focusMachine()
	case stSecrets:
		m.secretFocus = 0
		return m.focusSecret()
	case stReview:
		m.reviewFocus = 1
		m.confirmIn.SetValue("")
		return m.confirmIn.Focus()
	}
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.logView.Width = min(msg.Width-6, frameWidth-2)
		m.logView.Height = max(msg.Height-10, 5)
		return m, nil
	case onlineMsg:
		if msg.round != m.onlineRound {
			return m, nil
		}
		v := msg.ok
		was := m.isOnline
		m.isOnline = &v
		if m.step != stWelcome {
			return m, nil
		}
		// offline, the next thing to do is connect; once connected, it's the install
		changed := was == nil || *was != v
		switch cur := m.listCursor[stWelcome]; {
		case changed && !v && cur == 0:
			m.listCursor[stWelcome] = 1
		case changed && v && cur == 1:
			m.listCursor[stWelcome] = 0
		}
		if !v {
			return m, checkOnline(5*time.Second, m.onlineRound)
		}
		return m, nil
	case catalogMsg:
		m.catalog, m.catalogErr, m.catalogLoading = msg.apps, msg.err, false
		return m, nil
	case tickMsg:
		m.spinFrame++
		if m.step == stInstalling {
			m.poll()
		}
		return m, tick()
	case execDoneMsg:
		m.onlineRound++
		return m, checkOnline(0, m.onlineRound)
	case tea.KeyMsg:
		return m.key(msg)
	}
	return m, nil
}

func (m model) key(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	if k.String() == "ctrl+c" || (k.String() == "q" && m.step == stInstalling) {
		// the install carries on without the screens
		m.quitTo = "shell"
		return m, tea.Quit
	}
	// an error stays up until the next key, which either fixes it or says it again
	m.note = ""
	if k.String() == "esc" && m.step > stWelcome && m.step <= stReview {
		return m, m.back()
	}
	switch m.step {
	case stWelcome:
		return m.keyWelcome(k)
	case stDisk:
		return m.keyDisk(k)
	case stDesktop, stGraphics, stType:
		return m.keyChoice(k)
	case stRegion:
		return m.keyRegion(k)
	case stOptions:
		return m.keyOptions(k)
	case stApps:
		return m.keyApps(k)
	case stMachine:
		return m.keyMachine(k)
	case stSecrets:
		return m.keySecrets(k)
	case stReview:
		return m.keyReview(k)
	case stInstalling:
		if k.String() == "l" {
			m.showLog = !m.showLog
			return m, nil
		}
		if m.showLog {
			var cmd tea.Cmd
			m.logView, cmd = m.logView.Update(k)
			return m, cmd
		}
	case stDone, stFailed:
		return m.keyEnd(k)
	}
	return m, nil
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func (m model) age() int { return m.spinFrame - m.stepShown }
