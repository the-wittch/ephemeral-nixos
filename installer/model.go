package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

type answers struct {
	Disk, Desktop, Nvidia, Layout, Variant, Locale, Timezone, Profile string
	Extras, OpenSnitch, PassAgent, ControlD, Fingerprint              bool
	Vault, ControlDID                                                 string
	Apps                                                              []string
	Hostname, Username, Swap                                          string
	Passphrase, Password                                              string
}

type config struct {
	scriptsDir, flakeRoot, listsDir, logoPath, version string
	dryRun, forceDisk                                  bool
}

type model struct {
	cfg           config
	width, height int
	step          stepID
	history       []stepID
	editing       bool
	a             answers
	note          string

	isOnline   *bool
	uefi       bool
	memGiB     int
	disks      []disk
	diskErr    error
	gpu        string
	hasGPU     bool
	laptop     bool
	reader     string
	hasReader  bool
	configVer  string
	logo       [][]cell
	spinFrame  int
	listCursor map[stepID]int

	region      []*picker
	regionFocus int
	regionErr   string

	optFocus   int
	vaultIn    textinput.Model
	controldIn textinput.Model

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

	installCh  <-chan tea.Msg
	curStep    int
	lines      []string
	started    time.Time
	finished   time.Time
	installErr error
	showLog    bool
	logView    viewport.Model
	endSel     int

	quitTo string
}

type onlineMsg bool
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
		a: answers{
			Desktop: "plasma", Nvidia: "none", Layout: "us", Locale: "en_US.UTF-8", Timezone: "UTC",
			Profile: "full", OpenSnitch: true, Vault: "Personal",
			Hostname: "nixos", Username: "nixuser", Swap: "4G",
		},
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
	return tea.Batch(checkOnline(0), tick())
}

func checkOnline(delay time.Duration) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(delay)
		return onlineMsg(online())
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

func (m *model) placeDiskCursor() {
	cur := m.listCursor[stDisk]
	for i, d := range m.disks {
		if d.Path == m.a.Disk {
			cur = i
		}
	}
	if cur >= len(m.disks) || (cur < len(m.disks) && !m.diskUsable(m.disks[cur])) {
		cur = -1
		for i, d := range m.disks {
			if m.diskUsable(d) {
				cur = i
				break
			}
		}
	}
	m.listCursor[stDisk] = cur
}

func (m model) diskUsable(d disk) bool { return !d.InUse || m.cfg.forceDisk }

func (m *model) focusMachine() tea.Cmd {
	var cmd tea.Cmd
	for i := range m.machineIn {
		if i == m.machineFocus {
			cmd = m.machineIn[i].Focus()
		} else {
			m.machineIn[i].Blur()
		}
	}
	return cmd
}

func (m *model) focusSecret() tea.Cmd {
	var cmd tea.Cmd
	for i := range m.secretIn {
		if i == m.secretFocus {
			cmd = m.secretIn[i].Focus()
		} else {
			m.secretIn[i].Blur()
		}
	}
	return cmd
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.logView.Width = min(msg.Width-6, frameWidth-2)
		m.logView.Height = max(msg.Height-10, 5)
		return m, nil
	case onlineMsg:
		v := bool(msg)
		m.isOnline = &v
		if !v && m.step == stWelcome {
			return m, checkOnline(5 * time.Second)
		}
		return m, nil
	case catalogMsg:
		m.catalog, m.catalogErr, m.catalogLoading = msg.apps, msg.err, false
		return m, nil
	case tickMsg:
		m.spinFrame++
		return m, tick()
	case execDoneMsg:
		return m, checkOnline(0)
	case lineMsg:
		line := string(msg)
		if s := stepOf(line); s > m.curStep {
			m.curStep = s
		}
		if strings.TrimSpace(line) != "" {
			m.lines = append(m.lines, line)
			m.logView.SetContent(strings.Join(m.lines, "\n"))
			m.logView.GotoBottom()
		}
		return m, waitFor(m.installCh)
	case installDoneMsg:
		m.finished = time.Now()
		m.installErr = msg.err
		m.endSel = 0
		m.showLog = false
		if msg.err != nil {
			m.step = stFailed
		} else {
			m.step = stDone
		}
		return m, nil
	case tea.KeyMsg:
		return m.key(msg)
	}
	return m, nil
}

func (m model) key(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	if k.String() == "ctrl+c" {
		if m.step == stInstalling {
			m.note = "The install is running. Stopping it halfway would leave the disk half set up."
			return m, nil
		}
		m.quitTo = "shell"
		return m, tea.Quit
	}
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

var welcomeItems = []string{"Install", "Set up the network", "Open a shell"}

func (m model) keyWelcome(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	cur := m.listCursor[stWelcome]
	switch k.String() {
	case "left", "up", "shift+tab":
		if cur > 0 {
			cur--
		}
	case "right", "down", "tab":
		if cur < len(welcomeItems)-1 {
			cur++
		}
	case "q":
		m.quitTo = "shell"
		return m, tea.Quit
	case "enter":
		switch cur {
		case 0:
			if m.isOnline == nil || !*m.isOnline {
				m.note = "Connect to the internet first. The install downloads the system."
				return m, nil
			}
			if !m.uefi {
				m.note = "This computer started in legacy BIOS mode. Restart it in UEFI mode to install."
				return m, nil
			}
			// the list builds the config, so only once an install is under way
			return m, tea.Batch(m.goTo(stDisk), loadCatalogCmd(m.cfg))
		case 1:
			return m, tea.ExecProcess(exec.Command("nmtui"), func(error) tea.Msg { return execDoneMsg{} })
		case 2:
			m.quitTo = "shell"
			return m, tea.Quit
		}
	}
	m.listCursor[stWelcome] = cur
	return m, nil
}

func (m model) keyDisk(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	cur := m.listCursor[stDisk]
	switch k.String() {
	case "up":
		for i := cur - 1; i >= 0; i-- {
			if m.diskUsable(m.disks[i]) {
				cur = i
				break
			}
		}
	case "down":
		for i := cur + 1; i < len(m.disks); i++ {
			if m.diskUsable(m.disks[i]) {
				cur = i
				break
			}
		}
	case "enter":
		if cur < 0 || cur >= len(m.disks) {
			return m, nil
		}
		m.a.Disk = m.disks[cur].Path
		return m, m.next()
	}
	m.listCursor[stDisk] = cur
	return m, nil
}

type choice struct{ value, label, desc, tag string }

func (m model) choices(s stepID) []choice {
	switch s {
	case stDesktop:
		return []choice{
			{"plasma", "Plasma", "KDE Plasma 6, tiling with Krohnkite, colors from the wallpaper", ""},
			{"gnome", "GNOME", "GNOME", ""},
			{"niri", "niri", "scrolling tiles, with the Noctalia shell", ""},
			{"umbriel", "Umbriel", "a new compositor, with the Noctalia shell", "may change"},
		}
	case stGraphics:
		return []choice{
			{"open", "open", "RTX 20 / GTX 16 series and newer", ""},
			{"legacy", "legacy", "GTX 900 / GTX 10 series", ""},
			{"none", "none", "keep the open-source nouveau driver", ""},
		}
	case stType:
		return []choice{
			{"full", "Full", "dotfiles, the theme, snapshots, OpenSnitch, Syncthing, Podman, Flatpak, and the options on the next screen", ""},
			{"minimal", "Small", "a plain desktop now; run ephemeral-setup for the rest later", ""},
		}
	}
	return nil
}

func (m *model) answerFor(s stepID) *string {
	switch s {
	case stDesktop:
		return &m.a.Desktop
	case stGraphics:
		return &m.a.Nvidia
	case stType:
		return &m.a.Profile
	}
	return nil
}

func (m model) choiceCursor(s stepID) int {
	cur := m.answerFor(s)
	for i, c := range m.choices(s) {
		if c.value == *cur {
			return i
		}
	}
	return 0
}

func (m model) keyChoice(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	cs := m.choices(m.step)
	cur := m.choiceCursor(m.step)
	switch k.String() {
	case "up":
		if cur > 0 {
			cur--
		}
	case "down":
		if cur < len(cs)-1 {
			cur++
		}
	case "enter":
		return m, m.next()
	}
	*m.answerFor(m.step) = cs[cur].value
	return m, nil
}

func (m model) keyRegion(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	p := m.region[m.regionFocus]
	switch k.String() {
	case "tab", "enter", "shift+tab":
		if err := p.commit(); err != nil {
			m.regionErr = err.Error()
			return m, nil
		}
		m.regionErr = ""
		if m.regionFocus == 0 {
			m.syncVariants()
		}
		p.blur()
		if k.String() == "shift+tab" {
			if m.regionFocus > 0 {
				m.regionFocus--
			}
		} else if m.regionFocus < len(m.region)-1 {
			m.regionFocus++
		} else if k.String() == "enter" {
			m.a.Layout, m.a.Variant = m.region[0].value, m.region[1].value
			m.a.Locale, m.a.Timezone = m.region[2].value, m.region[3].value
			return m, m.next()
		}
		return m, m.region[m.regionFocus].focus()
	}
	return m, p.update(k)
}

// variants belong to one layout; "us,de" has none to pick from
func (m *model) syncVariants() {
	layout := m.region[0].value
	items := []item{{Code: "", Desc: "default"}}
	if rules, ok := xkbRules(m.cfg.listsDir); ok && !strings.Contains(layout, ",") {
		items = append(items, parseXKB(rules, "variant", layout)...)
	}
	value := m.region[1].value
	found := false
	for _, it := range items {
		if it.Code == value {
			found = true
		}
	}
	if !found {
		value = ""
	}
	m.region[1].setItems(items, value)
}

type optRow int

const (
	optExtras optRow = iota
	optOpenSnitch
	optPassAgent
	optVault
	optControlD
	optControlDID
	optFingerprint
)

func (m model) optRows() []optRow {
	rows := []optRow{optExtras, optOpenSnitch, optPassAgent}
	if m.a.PassAgent {
		rows = append(rows, optVault)
	}
	rows = append(rows, optControlD)
	if m.a.ControlD {
		rows = append(rows, optControlDID)
	}
	if m.hasReader {
		rows = append(rows, optFingerprint)
	}
	return rows
}

func (m model) keyOptions(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	rows := m.optRows()
	if m.optFocus >= len(rows) {
		m.optFocus = len(rows) - 1
	}
	row := rows[m.optFocus]
	moveTo := func(i int) tea.Cmd {
		m.optFocus = i
		m.vaultIn.Blur()
		m.controldIn.Blur()
		switch m.optRows()[i] {
		case optVault:
			return m.vaultIn.Focus()
		case optControlDID:
			return m.controldIn.Focus()
		}
		return nil
	}
	switch k.String() {
	case "up", "shift+tab":
		if m.optFocus > 0 {
			return m, moveTo(m.optFocus - 1)
		}
		return m, nil
	case "down", "tab":
		if m.optFocus < len(rows)-1 {
			return m, moveTo(m.optFocus + 1)
		}
		return m, nil
	case "enter":
		if m.a.PassAgent {
			if err := checkVault(strings.TrimSpace(m.vaultIn.Value())); err != nil {
				m.note = "Vault: " + err.Error()
				return m, nil
			}
			m.a.Vault = strings.TrimSpace(m.vaultIn.Value())
		}
		if m.a.ControlD {
			if err := checkControlD(strings.TrimSpace(m.controldIn.Value())); err != nil {
				m.note = "Control D: " + err.Error()
				return m, nil
			}
			m.a.ControlDID = strings.TrimSpace(m.controldIn.Value())
		}
		return m, m.next()
	}
	var cmd tea.Cmd
	switch row {
	case optVault:
		m.vaultIn, cmd = m.vaultIn.Update(k)
		return m, cmd
	case optControlDID:
		m.controldIn, cmd = m.controldIn.Update(k)
		return m, cmd
	}
	if k.String() == " " {
		switch row {
		case optExtras:
			m.a.Extras = !m.a.Extras
		case optOpenSnitch:
			m.a.OpenSnitch = !m.a.OpenSnitch
		case optPassAgent:
			m.a.PassAgent = !m.a.PassAgent
		case optControlD:
			m.a.ControlD = !m.a.ControlD
		case optFingerprint:
			m.a.Fingerprint = !m.a.Fingerprint
		}
	}
	return m, nil
}

func (m model) appCategories() []string {
	seen := map[string]bool{}
	for _, a := range m.catalog {
		seen[a.Category] = true
	}
	var out []string
	for _, c := range categoryOrder {
		if seen[c] {
			out = append(out, c)
			delete(seen, c)
		}
	}
	for c := range seen {
		out = append(out, c)
	}
	return out
}

func (m model) appsIn(cat string) []app {
	var out []app
	for _, a := range m.catalog {
		if a.Category == cat {
			out = append(out, a)
		}
	}
	return out
}

func (m model) keyApps(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	if k.String() == "enter" {
		m.a.Apps = nil
		for _, a := range m.catalog {
			if m.picked[a.ID] {
				m.a.Apps = append(m.a.Apps, a.ID)
			}
		}
		return m, m.next()
	}
	cats := m.appCategories()
	if len(cats) == 0 {
		return m, nil
	}
	apps := m.appsIn(cats[m.appCat])
	switch k.String() {
	case "tab", "right", "left", "shift+tab":
		m.appPane = 1 - m.appPane
	case "up":
		if m.appPane == 0 && m.appCat > 0 {
			m.appCat--
			m.appIdx = 0
		} else if m.appPane == 1 && m.appIdx > 0 {
			m.appIdx--
		}
	case "down":
		if m.appPane == 0 && m.appCat < len(cats)-1 {
			m.appCat++
			m.appIdx = 0
		} else if m.appPane == 1 && m.appIdx < len(apps)-1 {
			m.appIdx++
		}
	case " ":
		if m.appPane == 1 && m.appIdx < len(apps) {
			id := apps[m.appIdx].ID
			m.picked[id] = !m.picked[id]
		}
	}
	return m, nil
}

func (m model) machineErrs() []error {
	return []error{
		checkHostname(m.machineIn[0].Value()),
		checkUsername(m.machineIn[1].Value()),
		checkSwap(m.machineIn[2].Value()),
	}
}

func (m model) keyMachine(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch k.String() {
	case "up", "shift+tab":
		if m.machineFocus > 0 {
			m.machineFocus--
		}
		return m, m.focusMachine()
	case "down", "tab":
		if m.machineFocus < len(m.machineIn)-1 {
			m.machineFocus++
		}
		return m, m.focusMachine()
	case "enter":
		for i, err := range m.machineErrs() {
			if err != nil {
				m.machineFocus = i
				return m, m.focusMachine()
			}
		}
		m.a.Hostname, m.a.Username, m.a.Swap = m.machineIn[0].Value(), m.machineIn[1].Value(), m.machineIn[2].Value()
		return m, m.next()
	}
	var cmd tea.Cmd
	m.machineIn[m.machineFocus], cmd = m.machineIn[m.machineFocus].Update(k)
	return m, cmd
}

func (m model) secretErrs() (error, error) {
	return checkPassphrase(m.secretIn[0].Value(), m.secretIn[1].Value()),
		checkPassword(m.secretIn[2].Value(), m.secretIn[3].Value())
}

func (m model) keySecrets(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch k.String() {
	case "up", "shift+tab":
		if m.secretFocus > 0 {
			m.secretFocus--
		}
		return m, m.focusSecret()
	case "down", "tab":
		if m.secretFocus < len(m.secretIn)-1 {
			m.secretFocus++
		}
		return m, m.focusSecret()
	case "enter":
		if m.secretFocus < len(m.secretIn)-1 {
			m.secretFocus++
			return m, m.focusSecret()
		}
		pe, we := m.secretErrs()
		switch {
		case pe != nil || m.secretIn[1].Value() == "":
			m.secretFocus = 0
			if pe == nil {
				m.secretFocus = 1
			}
			return m, m.focusSecret()
		case we != nil || m.secretIn[3].Value() == "":
			m.secretFocus = 2
			if we == nil {
				m.secretFocus = 3
			}
			return m, m.focusSecret()
		}
		m.a.Passphrase, m.a.Password = m.secretIn[0].Value(), m.secretIn[2].Value()
		return m, m.next()
	}
	var cmd tea.Cmd
	m.secretIn[m.secretFocus], cmd = m.secretIn[m.secretFocus].Update(k)
	return m, cmd
}

type reviewLine struct {
	label, value string
	step         stepID
	danger       bool
}

func (m model) reviewLines() []reviewLine {
	var d disk
	for _, x := range m.disks {
		if x.Path == m.a.Disk {
			d = x
		}
	}
	diskVal := strings.TrimSpace(m.a.Disk + "  " + d.Model + "  " + humanSize(d.Size))
	lines := []reviewLine{{"DISK", diskVal + "  erased", stDisk, true}}
	lines = append(lines, reviewLine{"DESKTOP", m.labelFor(stDesktop), stDesktop, false})
	if m.hasGPU {
		lines = append(lines, reviewLine{"GRAPHICS", "NVIDIA, " + m.a.Nvidia + " driver", stGraphics, false})
	}
	region := m.a.Layout
	if m.a.Variant != "" {
		region += " (" + m.a.Variant + ")"
	}
	lines = append(lines, reviewLine{"REGION", region + " · " + m.a.Locale + " · " + m.a.Timezone, stRegion, false})
	if m.a.Profile == "full" {
		var opts []string
		if m.a.Extras {
			opts = append(opts, "extras")
		}
		if m.a.OpenSnitch {
			opts = append(opts, "OpenSnitch")
		}
		if m.a.PassAgent {
			opts = append(opts, "Pass SSH agent ("+m.a.Vault+")")
		}
		if m.a.ControlD {
			opts = append(opts, "Control D")
		}
		if m.a.Fingerprint {
			opts = append(opts, "fingerprint login")
		}
		v := "Full"
		if len(opts) > 0 {
			v += ", with " + strings.Join(opts, ", ")
		}
		lines = append(lines, reviewLine{"INSTALL", v, stOptions, false})
		apps := "none"
		if len(m.a.Apps) > 0 {
			var names []string
			for _, a := range m.catalog {
				if m.picked[a.ID] {
					names = append(names, a.Label)
				}
			}
			apps = strings.Join(names, ", ")
		}
		lines = append(lines, reviewLine{"APPS", apps, stApps, false})
	} else {
		lines = append(lines, reviewLine{"INSTALL", "Small; the rest later with ephemeral-setup", stType, false})
	}
	lines = append(lines,
		reviewLine{"MACHINE", m.a.Hostname + " · " + m.a.Username + " · " + m.a.Swap + " swap", stMachine, false},
		reviewLine{"SECRETS", "disk passphrase and login password set", stSecrets, false},
	)
	return lines
}

func (m model) labelFor(s stepID) string {
	cur := *m.answerFor(s)
	for _, c := range m.choices(s) {
		if c.value == cur {
			return c.label
		}
	}
	return cur
}

func (m model) keyReview(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	lines := m.reviewLines()
	switch k.String() {
	case "tab", "shift+tab":
		m.reviewFocus = 1 - m.reviewFocus
		if m.reviewFocus == 1 {
			return m, m.confirmIn.Focus()
		}
		m.confirmIn.Blur()
		return m, nil
	case "up":
		m.reviewFocus = 0
		m.confirmIn.Blur()
		if m.reviewSel > 0 {
			m.reviewSel--
		}
		return m, nil
	case "down":
		if m.reviewFocus == 0 && m.reviewSel < len(lines)-1 {
			m.reviewSel++
			return m, nil
		}
		m.reviewFocus = 1
		return m, m.confirmIn.Focus()
	case "enter":
		if m.reviewFocus == 0 {
			m.editing = true
			return m, m.goTo(lines[m.reviewSel].step)
		}
		if m.confirmIn.Value() != filepath.Base(m.a.Disk) {
			m.note = "Type " + filepath.Base(m.a.Disk) + " exactly to erase it."
			return m, nil
		}
		return m.startInstall()
	}
	if m.reviewFocus == 1 {
		var cmd tea.Cmd
		m.confirmIn, cmd = m.confirmIn.Update(k)
		return m, cmd
	}
	return m, nil
}

func (m model) startInstall() (tea.Model, tea.Cmd) {
	m.history = nil
	m.step = stInstalling
	m.curStep = 0
	m.lines = nil
	m.installErr = nil
	m.started = time.Now()
	m.note = ""
	m.installCh = startInstall(m.cfg.scriptsDir, m.a, m.cfg.dryRun)
	return m, waitFor(m.installCh)
}

func (m model) endItems() []string {
	if m.step == stDone {
		return []string{"Reboot", "Open a shell"}
	}
	return []string{"Try again", "Change an answer", "Show the full log", "Open a shell"}
}

func (m model) keyEnd(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.showLog {
		switch k.String() {
		case "l", "esc", "q":
			m.showLog = false
			return m, nil
		}
		var cmd tea.Cmd
		m.logView, cmd = m.logView.Update(k)
		return m, cmd
	}
	items := m.endItems()
	switch k.String() {
	case "up", "left":
		if m.endSel > 0 {
			m.endSel--
		}
	case "down", "right", "tab":
		if m.endSel < len(items)-1 {
			m.endSel++
		}
	case "enter":
		switch items[m.endSel] {
		case "Reboot":
			m.quitTo = "reboot"
			return m, tea.Quit
		case "Open a shell":
			m.quitTo = "shell"
			return m, tea.Quit
		case "Try again":
			return m.startInstall()
		case "Change an answer":
			m.step = stReview
			return m, m.enter()
		case "Show the full log":
			m.showLog = true
			m.logView.GotoBottom()
		}
	}
	return m, nil
}

func (m model) elapsed() time.Duration {
	end := m.finished
	if m.step == stInstalling || end.IsZero() {
		end = time.Now()
	}
	return end.Sub(m.started).Round(time.Second)
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
