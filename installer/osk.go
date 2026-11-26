package main

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// An on-screen keyboard for a handheld without a keyboard: F2 opens it on a text field, which
// is what the controller's Y sends. Each key it presses goes to the screen as if typed, so the
// screen's own checks see it. Every printable ASCII character is on one of its two layers.
type osk struct {
	open     bool
	row, col int
	shift    bool
	symbols  bool
}

var oskLetters = [][]string{
	{"1", "2", "3", "4", "5", "6", "7", "8", "9", "0"},
	{"q", "w", "e", "r", "t", "y", "u", "i", "o", "p"},
	{"a", "s", "d", "f", "g", "h", "j", "k", "l", "-"},
	{"z", "x", "c", "v", "b", "n", "m", ".", "_", "@"},
}

var oskSymbols = [][]string{
	{"!", "\"", "#", "$", "%", "&", "'", "(", ")", "*"},
	{"+", ",", "/", ":", ";", "<", "=", ">", "?", "~"},
	{"[", "\\", "]", "^", "`", "{", "|", "}", "", ""},
}

const (
	oskShift = iota
	oskLayer
	oskSpace
	oskBack
	oskDone
)

var oskActions = []string{"Shift", "!#1", "Space", "Delete", "Done"}

func (o osk) grid() [][]string {
	if o.symbols {
		return oskSymbols
	}
	return oskLetters
}

// the bottom row is the actions, one per column pair
func (o osk) rows() int { return len(o.grid()) + 1 }

func (o osk) width(row int) int {
	if row == len(o.grid()) {
		return len(oskActions)
	}
	n := 0
	for _, k := range o.grid()[row] {
		if k != "" {
			n++
		}
	}
	return n
}

func (o *osk) move(dr, dc int) {
	o.row = (o.row + dr + o.rows()) % o.rows()
	w := o.width(o.row)
	if dr != 0 {
		o.col = min(o.col, w-1)
		return
	}
	o.col = (o.col + dc + w) % w
}

// The text field the keyboard types into, if the screen has one focused.
func (m *model) focusedInput() *textinput.Model {
	var ins []*textinput.Model
	switch m.step {
	case stMachine:
		for i := range m.machineIn {
			ins = append(ins, &m.machineIn[i])
		}
	case stSecrets:
		for i := range m.secretIn {
			ins = append(ins, &m.secretIn[i])
		}
	case stServer:
		ins = append(ins, &m.keyIn)
	case stOptions:
		ins = append(ins, &m.vaultIn, &m.controldIn, &m.customDNSIn)
	case stApps:
		ins = append(ins, &m.appQuery)
	case stReview:
		ins = append(ins, &m.confirmIn)
	case stRegion:
		for _, p := range m.region {
			ins = append(ins, &p.input)
		}
	case stWifi:
		ins = append(ins, &m.wifiPass)
	}
	for _, in := range ins {
		if in.Focused() {
			return in
		}
	}
	return nil
}

// keyOSK takes the keys the open keyboard uses; the rest go to the screen.
func (m model) keyOSK(k tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	o := &m.osk
	switch k.String() {
	case "up":
		o.move(-1, 0)
	case "down":
		o.move(1, 0)
	case "left":
		o.move(0, -1)
	case "right":
		o.move(0, 1)
	case "esc", "f2":
		o.open = false
	case "enter":
		return m.pressOSK()
	default:
		return m, nil, false
	}
	return m, nil, true
}

func (m model) pressOSK() (tea.Model, tea.Cmd, bool) {
	o := &m.osk
	var send tea.KeyMsg
	if o.row < len(o.grid()) {
		c := o.grid()[o.row][o.col]
		if o.shift && !o.symbols {
			c = strings.ToUpper(c)
			o.shift = false
		}
		send = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(c)}
	} else {
		switch o.col {
		case oskShift:
			o.shift = !o.shift
			return m, nil, true
		case oskLayer:
			o.symbols = !o.symbols
			o.row = min(o.row, o.rows()-1)
			return m, nil, true
		case oskSpace:
			send = tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}}
		case oskBack:
			send = tea.KeyMsg{Type: tea.KeyBackspace}
		case oskDone:
			send = tea.KeyMsg{Type: tea.KeyEnter}
		}
	}
	next, cmd := m.screenKey(send)
	nm := next.(model)
	// Done can move to the next field, or off the screen
	if nm.focusedInput() == nil {
		nm.osk.open = false
	}
	return nm, cmd, true
}

func (m model) viewOSK(w int) []string {
	o := m.osk
	cell := func(label string, on bool, width int) string {
		label = " " + label + strings.Repeat(" ", max(width-lipgloss.Width(label)-1, 0))
		if on {
			return sCursor.Render(label)
		}
		return sTag.Render(label)
	}
	var out []string
	for r, row := range o.grid() {
		var cells []string
		for c, k := range row {
			if k == "" {
				continue
			}
			if o.shift && !o.symbols {
				k = strings.ToUpper(k)
			}
			cells = append(cells, cell(k, o.row == r && o.col == c, 3))
		}
		out = append(out, "  "+strings.Join(cells, " "))
	}
	var acts []string
	for c, a := range oskActions {
		if c == oskShift && o.shift {
			a = "SHIFT"
		}
		if c == oskLayer && o.symbols {
			a = "abc"
		}
		acts = append(acts, cell(a, o.row == len(o.grid()) && o.col == c, 8))
	}
	out = append(out, "  "+strings.Join(acts, " "))
	return out
}
