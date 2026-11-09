package main

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// A field with a list under it that narrows as you type. Text that matches nothing is taken
// as typed when allowFree is set, such as "us,de" for two keyboard layouts.
type picker struct {
	label     string
	items     []item
	value     string
	input     textinput.Model
	matches   []int
	sel       int
	allowFree bool
	check     func(string) error
}

const pickerRows = 6

func newPicker(label string, items []item, value string, allowFree bool, check func(string) error) *picker {
	in := textinput.New()
	in.Prompt = ""
	in.CharLimit = 64
	in.Cursor.Style = sCursor
	p := &picker{label: label, items: items, value: value, input: in, allowFree: allowFree, check: check}
	p.filter()
	return p
}

func (p *picker) setItems(items []item, value string) {
	p.items = items
	p.value = value
	p.input.SetValue("")
	p.filter()
}

func (p *picker) filter() {
	q := strings.ToLower(strings.TrimSpace(p.input.Value()))
	var prefix, rest []int
	for i, it := range p.items {
		code, desc := strings.ToLower(it.Code), strings.ToLower(it.Desc)
		switch {
		case q == "":
			prefix = append(prefix, i)
		case strings.HasPrefix(code, q):
			prefix = append(prefix, i)
		case strings.Contains(code, q) || strings.Contains(desc, q):
			rest = append(rest, i)
		}
	}
	p.matches = append(prefix, rest...)
	p.sel = 0
	if q == "" {
		for k, i := range p.matches {
			if p.items[i].Code == p.value {
				p.sel = k
				break
			}
		}
	}
}

func (p *picker) focus() tea.Cmd {
	p.input.SetValue("")
	p.filter()
	return p.input.Focus()
}

func (p *picker) blur() { p.input.Blur() }

// what enter or tab would set, and whether it's acceptable
func (p *picker) choice() (string, error) {
	if len(p.matches) > 0 {
		return p.items[p.matches[p.sel]].Code, nil
	}
	typed := strings.TrimSpace(p.input.Value())
	if !p.allowFree {
		return "", errNoMatch
	}
	if p.check != nil {
		if err := p.check(typed); err != nil {
			return "", err
		}
	}
	return typed, nil
}

func (p *picker) commit() error {
	v, err := p.choice()
	if err != nil {
		return err
	}
	p.value = v
	return nil
}

func (p *picker) update(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "up":
		if p.sel > 0 {
			p.sel--
		}
		return nil
	case "down":
		if p.sel < len(p.matches)-1 {
			p.sel++
		}
		return nil
	}
	var cmd tea.Cmd
	before := p.input.Value()
	p.input, cmd = p.input.Update(msg)
	if p.input.Value() != before {
		p.filter()
	}
	return cmd
}

func (p *picker) displayValue() string {
	for _, it := range p.items {
		if it.Code == p.value {
			if it.Code == "" {
				return it.Desc
			}
			if it.Desc != "" {
				return it.Code + "  " + sDim.Render(it.Desc)
			}
			return it.Code
		}
	}
	return p.value
}

// the open list, only while the field has focus
func (p *picker) listView(width int) []string {
	if len(p.matches) == 0 {
		if p.allowFree && strings.TrimSpace(p.input.Value()) != "" {
			if _, err := p.choice(); err != nil {
				return []string{sRed.Render(g.failed + " " + err.Error())}
			}
			return []string{sDim.Render("enter uses what you typed")}
		}
		return []string{sDim.Render("nothing matches")}
	}
	start := 0
	if p.sel >= pickerRows {
		start = p.sel - pickerRows + 1
	}
	var out []string
	for k := start; k < len(p.matches) && k < start+pickerRows; k++ {
		it := p.items[p.matches[k]]
		code := it.Code
		if code == "" {
			code = it.Desc
		}
		line := padRight(code, 14) + " " + it.Desc
		if it.Code == "" {
			line = code
		}
		line = truncate(line, width-4)
		if k == p.sel {
			out = append(out, sCursor.Render(" "+g.cursor+" "+padRight(line, width-4)))
		} else {
			out = append(out, "   "+sSub.Render(line))
		}
	}
	if n := len(p.matches) - (start + pickerRows); n > 0 {
		out = append(out, sDim.Render(spaces(3)+"…and "+itoa(n)+" more; keep typing"))
	}
	return out
}
