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
	stServer
	stSecrets
	stReview
	stInstalling
	stDone
	stFailed
	// from the first screen, not one of the questions
	stWifi
)

var questions = []stepID{stDisk, stDesktop, stGraphics, stRegion, stType, stOptions, stApps, stMachine, stServer, stSecrets, stReview}

var stepName = map[stepID]string{
	stDisk: "Disk", stDesktop: "Desktop", stGraphics: "Graphics", stRegion: "Region",
	stType: "Install type", stOptions: "Options", stApps: "Apps", stMachine: "Machine", stServer: "Server",
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
	// the DMI sys_vendor, for the buttons only some handhelds have
	vendor    string
	reader    string
	hasReader bool
	setupMode bool
	configVer string
	logo      [][]cell
	spinFrame int
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
	appSearching   bool
	appQuery       textinput.Model
	picked         map[string]bool
	// the apps ticked or unticked by hand
	touched map[string]bool

	machineIn    []textinput.Model
	machineFocus int

	serverFocus int
	keyIn       textinput.Model
	// the GitHub user whose keys are on their way
	fetching string

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

	osk osk
	// handheld-input is running, so a controller may be all there is
	padUsed bool
	// whether the controller types the passphrase now, as at boot
	padPassphrase bool

	wifiNets     []network
	wifiSel      int
	wifiErr      error
	wifiScanning bool
	wifiBusy     bool
	wifiPass     textinput.Model

	liveLayout string

	quitTo string
}

// round counts the checks started after a Wi-Fi connection, so an older round stops instead of
// running alongside
type onlineMsg struct {
	ok    bool
	round int
}
type catalogMsg struct {
	apps []app
	err  error
}
type tickMsg time.Time

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
		vendor:         dmiVendor(),
		padUsed:        padRunning(),
		configVer:      configVersion(cfg.flakeRoot),
		logo:           loadLogo(cfg.logoPath),
		listCursor:     map[stepID]int{},
		picked:         map[string]bool{},
		touched:        map[string]bool{},
		catalogLoading: true,
		a:              defaultAnswers(),
		run:            newRunDir(cfg.dryRun),
	}
	if m.run.running() {
		m.reattach()
	} else if m.run.died() {
		m.note = "The last install stopped without finishing. Its output is in /tmp/ephemeral-install.log."
	}
	m.gpu, m.hasGPU = nvidiaGPU(cfg.scriptsDir)
	if m.hasGPU {
		m.a.Nvidia = "open"
	}
	if isHandheld() && rules.allows("desktop", "handheld") {
		m.a.Desktop = "handheld"
	}
	m.reader, m.hasReader = fingerprintReader(cfg.scriptsDir)
	m.a.Fingerprint = m.hasReader
	m.setupMode = setupMode()

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
	m.appQuery = newInput("", "name or what it does")
	m.appQuery.Width = 18
	m.keyIn = newInput("", "ssh-ed25519 AAAA… or github:yourname")
	// an RSA key is 750 characters or so
	m.keyIn.CharLimit = 4096
	m.secretIn = []textinput.Model{newSecret(), newSecret(), newSecret(), newSecret()}
	m.confirmIn = newInput("", "")
	m.wifiPass = newSecret()
	m.wifiPass.CharLimit = 63
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
		// a change to the desktop can make it a server, which needs its keys
		if m.step != stServer && m.a.serverUnanswered() {
			return m.goTo(stServer)
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
	case stApps:
		m.followDesktop()
		// a change of desktop can leave fewer categories than before
		if m.appCat >= len(m.appCategories()) {
			m.appCat, m.appIdx, m.appPane = 0, 0, 0
		}
	case stMachine:
		m.machineFocus = 0
		return m.focusMachine()
	case stServer:
		if m.a.Desktop == "none" {
			m.a.Server = true
		}
		m.serverFocus = 0
		if m.a.serverUnanswered() {
			m.serverFocus = 1
		}
		return m.focusServer()
	case stSecrets:
		m.secretFocus = 0
		return m.focusSecret()
	case stWifi:
		m.wifiSel, m.wifiErr, m.wifiBusy, m.wifiScanning = 0, nil, false, true
		return scanWifi()
	case stReview:
		m.followDesktop()
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
		// so a long key scrolls inside its field instead of off the screen
		m.keyIn.Width = max(min(msg.Width-4, frameWidth)-30, 20)
		return m, nil
	case keysMsg:
		return m.gotKeys(msg), nil
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
		for _, a := range m.catalog {
			if _, seen := m.picked[a.ID]; a.Default && !seen {
				m.picked[a.ID] = true
			}
		}
		m.followDesktop()
		return m, nil
	case tickMsg:
		m.spinFrame++
		if m.step == stInstalling {
			m.poll()
		}
		return m, tick()
	case wifiListMsg:
		m.wifiScanning, m.wifiErr = false, msg.err
		m.wifiNets = msg.nets
		m.wifiSel = min(m.wifiSel, max(len(m.wifiNets)-1, 0))
		return m, nil
	case wifiDoneMsg:
		m.wifiBusy = false
		if msg.err != nil {
			m.note = "Couldn't connect: " + msg.err.Error()
			return m, nil
		}
		m.onlineRound++
		return m, tea.Batch(m.back(), checkOnline(0, m.onlineRound))
	case tea.KeyMsg:
		next, cmd := m.key(msg)
		nm := next.(model)
		return nm, tea.Batch(cmd, nm.syncPad())
	}
	return m, nil
}

// On a handheld's passphrase fields the controller types the passphrase as it will at boot, and
// everywhere else it moves around the screens.
func (m *model) syncPad() tea.Cmd {
	want := m.step == stSecrets && m.a.Desktop == "handheld" && m.secretFocus <= 1
	if !m.padUsed || want == m.padPassphrase {
		return nil
	}
	m.padPassphrase = want
	return func() tea.Msg {
		setPadMode(want)
		return nil
	}
}

func (m model) key(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	if k.String() == "ctrl+c" || (k.String() == "q" && m.step == stInstalling) {
		// the install carries on without the screens
		m.quitTo = "shell"
		return m, tea.Quit
	}
	// an error stays up until the next key, which either fixes it or says it again
	m.note = ""
	if m.osk.open {
		if next, cmd, used := m.keyOSK(k); used {
			return next, cmd
		}
	} else if k.String() == "f2" {
		// on the app list it opens the search to type in
		if m.step == stApps && !m.appSearching {
			next, cmd := m.screenKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
			nm := next.(model)
			nm.osk = osk{open: true}
			return nm, cmd
		}
		if m.focusedInput() != nil {
			m.osk = osk{open: true}
			return m, nil
		}
	}
	return m.screenKey(k)
}

// screenKey is the key as the screen takes it, typed or from the on-screen keyboard.
func (m model) screenKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	// in the app search it closes the search instead
	if k.String() == "esc" && m.step > stWelcome && m.step <= stReview && !(m.step == stApps && m.appSearching) {
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
	case stServer:
		return m.keyServer(k)
	case stSecrets:
		return m.keySecrets(k)
	case stReview:
		return m.keyReview(k)
	case stWifi:
		return m.keyWifi(k)
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
