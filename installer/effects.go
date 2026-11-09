package main

import "strings"

const (
	wordSpan  = 7 // ticks for the title or a word to settle from scramble
	logoFlash = 4 // ticks the snowflake draws colour-shifted before it lines up
)

var (
	glitchBlock = []rune("░▒▓▚▞▌▐█")
	glitchPlain = []rune("#%&*/\\|=")
)

func glitchChar(seed int) rune {
	set := glitchPlain
	if g.blockWord {
		set = glitchBlock
	}
	return set[seed%len(set)]
}

func revealFrac(age, span int) float64 {
	switch {
	case age >= span:
		return 1
	case age <= 0:
		return 0
	default:
		return float64(age) / float64(span)
	}
}

// scramble keeps the first frac of s and replaces the rest with flickering characters, so a line
// resolves left to right. Spaces stay spaces, so the shape holds while it settles.
func scramble(s string, frame int, frac float64) string {
	r := []rune(s)
	shown := int(float64(len(r)) * frac)
	var b strings.Builder
	for i, c := range r {
		if i < shown || c == ' ' {
			b.WriteRune(c)
		} else {
			b.WriteRune(glitchChar(frame + i*7))
		}
	}
	return b.String()
}

// The install's progress, with a bright cell travelling along it so a long step still looks alive.
func (m model) installBar(w int) string {
	filled := min(m.curStep*w/len(installSteps), w)
	if m.step == stDone {
		filled = w
	}
	glint := -1
	if m.cfg.effects && m.step == stInstalling {
		glint = m.spinFrame * 2 % w
	}
	var b strings.Builder
	for i := range w {
		switch {
		case i == glint:
			b.WriteString(sBold.Render(g.barOn))
		case i < filled:
			b.WriteString(sMauve.Render(g.barOn))
		default:
			b.WriteString(sDim.Render(g.barOff))
		}
	}
	return b.String()
}
