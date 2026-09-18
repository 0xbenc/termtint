package ui

import (
	"fmt"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/0xbenc/termtint/internal/termstyle"
)

// lane is a focusable lane in the spec editor.
type lane int

const (
	laneFg lane = iota
	laneBg
	laneStyle
	lanePreset
	laneRaw
)

var laneNames = []lane{laneFg, laneBg, laneStyle, lanePreset, laneRaw}

var styleNames = []string{"bold", "dim", "italic", "underline", "reverse"}
var styleSGR = []string{"1", "2", "3", "4", "7"}

// specEditor is the structured style editor for one role. The raw lane is the
// source of truth; the fg/bg/style lanes are a live decomposition of it, so
// picking a color or toggling a style rewrites the spec, and typing a spec
// re-decomposes it.
type specEditor struct {
	role    termstyle.Role
	label   string
	presets []string
	preset  int

	styles       [5]bool
	fg           string // a color name or raw SGR fragment; "" = default
	bg           string
	decomposable bool
	styleErr     string

	raw   string
	caret int
	focus lane
	err   string
}

// NewSpecEditor builds the editor seeded from the role's current spec.
func NewSpecEditor(meta roleMeta, spec string) *specEditor {
	e := &specEditor{
		role:    meta.Role,
		label:   meta.Label,
		presets: meta.Presets,
		raw:     strings.TrimSpace(spec),
		caret:   len(strings.TrimSpace(spec)),
		focus:   laneRaw,
	}
	for i, preset := range meta.Presets {
		if strings.TrimSpace(preset) == e.raw {
			e.preset = i
		}
	}
	e.refresh()
	return e
}

// refresh re-validates and re-decomposes the raw spec.
func (e *specEditor) refresh() {
	e.err = ""
	if strings.TrimSpace(e.raw) == "" {
		e.styles = [5]bool{}
		e.fg, e.bg = "", ""
		e.decomposable = true
		e.styleErr = ""
		return
	}
	code, err := termstyle.ParseStyleSpec(e.raw)
	if err != nil {
		e.err = err.Error()
		return
	}
	e.err = ""
	styles, fg, bg, sErr := decomposeSGR(code)
	if sErr != nil {
		e.decomposable = false
		e.styleErr = sErr.Error()
		return
	}
	e.decomposable = true
	e.styleErr = ""
	e.styles, e.fg, e.bg = styles, fg, bg
}

// specCode returns the parsed SGR code when the raw spec is valid.
func (e *specEditor) specCode() (string, bool) {
	if strings.TrimSpace(e.raw) == "" {
		return "", true
	}
	code, err := termstyle.ParseStyleSpec(e.raw)
	return code, err == nil
}

// formatSpec renders the structured state back to a canonical raw spec.
func (e *specEditor) formatSpec() string {
	var parts []string
	for i, name := range styleNames {
		if e.styles[i] {
			parts = append(parts, name)
		}
	}
	if e.fg != "" {
		parts = append(parts, e.fg)
	}
	if e.bg != "" {
		parts = append(parts, e.bg)
	}
	return strings.Join(parts, " ")
}

// applyFragment sets fg or bg from a color picker selection.
func (e *specEditor) applyFragment(target lane, fragment string) {
	if target == laneFg {
		e.fg = fragment
	} else {
		e.bg = fragment
	}
	e.raw = e.formatSpec()
	e.caret = len(e.raw)
	e.refresh()
	e.preset = 0
	for i, preset := range e.presets {
		if strings.TrimSpace(preset) == e.raw {
			e.preset = i
			break
		}
	}
}

// toggleStyle flips one style bit (from the style picker).
func (e *specEditor) setStyles(styles [5]bool) {
	e.styles = styles
	e.raw = e.formatSpec()
	e.caret = len(e.raw)
	e.refresh()
}

// cyclePreset moves the preset dial, which rewrites the spec.
func (e *specEditor) cyclePreset(dir int) {
	if len(e.presets) == 0 {
		return
	}
	e.preset = (e.preset + dir + len(e.presets)) % len(e.presets)
	e.raw = e.presets[e.preset]
	e.caret = len(e.raw)
	e.refresh()
}

// paste appends sanitized text to the raw lane.
func (e *specEditor) paste(text string) *specEditor {
	e.raw = e.raw[:e.caret] + text + e.raw[e.caret:]
	e.caret += len(text)
	e.refresh()
	return e
}

// decomposeSGR splits a normalized SGR string into styles plus an fg and bg
// fragment. Base colors come back as names (readable in the raw lane); 256-
// and truecolor colors come back as their raw SGR fragment.
func decomposeSGR(code string) ([5]bool, string, string, error) {
	var styles [5]bool
	fg, bg := "", ""
	parts := strings.Split(code, ";")
	for i := 0; i < len(parts); {
		part := parts[i]
		switch {
		case part == "0":
			styles = [5]bool{}
			fg, bg = "", ""
			i++
		default:
			if styleIndex := indexStyleSGR(part); styleIndex >= 0 {
				styles[styleIndex] = true
				i++
				continue
			}
			switch part {
			case "39":
				fg = "default"
				i++
				continue
			case "30", "31", "32", "33", "34", "35", "36", "37":
				fg = colorNameFor(int(part[1]-'0')+30, false)
				i++
				continue
			case "90", "91", "92", "93", "94", "95", "96", "97":
				fg = colorNameFor(int(part[1]-'0')+90, false)
				i++
				continue
			case "49":
				bg = "default"
				i++
				continue
			case "40", "41", "42", "43", "44", "45", "46", "47":
				bg = colorNameFor(int(part[1]-'0')+40, true)
				i++
				continue
			case "100", "101", "102", "103", "104", "105", "106", "107":
				bg = colorNameFor(int(part[1]-'0')+100, true)
				i++
				continue
			case "38":
				frag, consumed, err := extendedColor(parts, i, "38")
				if err != nil {
					return styles, fg, bg, err
				}
				fg = frag
				i += consumed
				continue
			case "48":
				frag, consumed, err := extendedColor(parts, i, "48")
				if err != nil {
					return styles, fg, bg, err
				}
				bg = frag
				i += consumed
				continue
			default:
				return styles, fg, bg, fmt.Errorf("cannot decompose SGR part %q", part)
			}
		}
	}
	return styles, fg, bg, nil
}

// extendedColor consumes a "38"/"48" extended-color run ("5;n" or "2;r;g;b").
func extendedColor(parts []string, i int, prefix string) (string, int, error) {
	if i+2 >= len(parts) && prefix == "38" {
		// fall through to the checks below
	}
	if i+2 < len(parts) && parts[i+1] == "5" {
		if _, err := strconv.Atoi(parts[i+2]); err != nil {
			return "", 0, fmt.Errorf("invalid 256-color index")
		}
		return strings.Join(parts[i:i+3], ";"), 3, nil
	}
	if i+4 < len(parts) && parts[i+1] == "2" {
		ok := true
		for j := 2; j <= 4; j++ {
			if _, err := strconv.Atoi(parts[i+j]); err != nil {
				ok = false
			}
		}
		if !ok {
			return "", 0, fmt.Errorf("invalid RGB color")
		}
		return strings.Join(parts[i:i+5], ";"), 5, nil
	}
	return "", 0, fmt.Errorf("cannot decompose SGR part %q", prefix)
}

func indexStyleSGR(part string) int {
	for i, code := range styleSGR {
		if code == part {
			return i
		}
	}
	return -1
}

var baseColorNames = []string{"black", "red", "green", "yellow", "blue", "magenta", "cyan", "white"}

func colorNameFor(code int, background bool) string {
	switch {
	case code >= 30 && code <= 37:
		return baseColorNames[code-30]
	case code >= 90 && code <= 97:
		return "bright-" + baseColorNames[code-90]
	case code >= 40 && code <= 47:
		return "bg-" + baseColorNames[code-40]
	case code >= 100 && code <= 107:
		return "bg-bright-" + baseColorNames[code-100]
	}
	return strconv.Itoa(code)
}

// update handles one keystroke; it returns whether the editor closed.
func (e *specEditor) update(key string, msg tea.KeyPressMsg) (closed bool) {
	switch key {
	case "esc":
		return true
	case "enter":
		if _, ok := e.specCode(); !ok {
			return false
		}
		return true
	case "tab", "shift+down":
		e.focus = nextLane(e.focus, 1)
	case "shift+tab", "shift+up":
		e.focus = nextLane(e.focus, -1)
	case "up":
		e.focus = nextLane(e.focus, -1)
	case "down":
		e.focus = nextLane(e.focus, 1)
	case "left", "right":
		if e.focus == lanePreset {
			dir := -1
			if key == "right" {
				dir = 1
			}
			e.cyclePreset(dir)
		} else if e.focus == laneRaw {
			if key == "left" && e.caret > 0 {
				e.caret--
			}
			if key == "right" && e.caret < len(e.raw) {
				e.caret++
			}
		}
	case "home":
		if e.focus == laneRaw {
			e.caret = 0
		}
	case "end":
		if e.focus == laneRaw {
			e.caret = len(e.raw)
		}
	case "backspace":
		if e.focus == laneRaw && e.caret > 0 {
			e.raw = e.raw[:e.caret-1] + e.raw[e.caret:]
			e.caret--
			e.refresh()
		}
	case "ctrl+u":
		if e.focus == laneRaw {
			e.raw = ""
			e.caret = 0
			e.refresh()
		}
	default:
		if e.focus == laneRaw && safeTextInput(msg.Text) {
			text := msg.Text
			e.raw = e.raw[:e.caret] + text + e.raw[e.caret:]
			e.caret += len(text)
			e.refresh()
		}
	}
	return false
}

func nextLane(current lane, dir int) lane {
	idx := 0
	for i, l := range laneNames {
		if l == current {
			idx = i
			break
		}
	}
	idx = (idx + dir + len(laneNames)) % len(laneNames)
	return laneNames[idx]
}

// view renders the editor as a full-screen shell.
func (m Model) editorView(chrome, work styler, width int) []string {
	e := m.editor
	if e == nil {
		return nil
	}
	laneWidth := max(48, min(width-4, 72))

	// The sample wears the working theme: it is the live answer to "what
	// will this role paint".
	sample := "The quick brown fox 0123456789"
	code, ok := e.specCode()
	var swatch string
	if ok && code != "" {
		swatch = termstyle.Apply(work.theme.NoColor, code, sample)
	} else {
		swatch = chrome.foreground(sample)
	}

	body := make([]string, 0, 12)
	body = append(body, chrome.muted("sample  ")+" "+swatch)
	body = append(body, "")

	fgText := e.fg
	if fgText == "" {
		fgText = "(default)"
	}
	bgText := e.bg
	if bgText == "" {
		bgText = "(none)"
	}
	styleText := ""
	for i, name := range styleNames {
		if e.styles[i] {
			if styleText != "" {
				styleText += ", "
			}
			styleText += name
		}
	}
	if styleText == "" {
		styleText = "(none)"
	}
	presetText := presetLabel(e.presets[e.preset])

	lanes := []struct {
		name  string
		value string
		hint  string
		l     lane
	}{
		{"fg", fgText, "[F] color", laneFg},
		{"bg", bgText, "[B] color", laneBg},
		{"style", styleText, "[M] toggle", laneStyle},
		{"preset", fmt.Sprintf("%d of %d: %s", e.preset+1, len(e.presets), presetText), "[←/→] cycle", lanePreset},
	}
	for _, ln := range lanes {
		marker := "  "
		if ln.l == e.focus {
			marker = chrome.accent(">")
		}
		row := marker + " " + termstyle.PadRight(chrome.muted(ln.name), 7)
		value := ln.value
		if ln.l == e.focus {
			value = chrome.selected(value)
		}
		row += " " + termstyle.PadRight(value, 26)
		if width >= 60 {
			row += " " + chrome.subtle(ln.hint)
		}
		body = append(body, termstyle.Truncate(row, laneWidth))
	}

	rawDisplay := e.raw
	if e.focus == laneRaw {
		head, tail := rawDisplay[:e.caret], rawDisplay[e.caret:]
		rawDisplay = head + chrome.accent("▮") + tail
	}
	marker := "  "
	if e.focus == laneRaw {
		marker = chrome.accent(">")
	}
	rawRow := marker + " " + termstyle.PadRight(chrome.muted("raw"), 7) + " " +
		work.style(termstyle.RoleForeground, termstyle.Truncate(rawDisplay, 40))
	if width >= 60 {
		rawRow += " " + chrome.subtle("[R] type here")
	}
	body = append(body, termstyle.Truncate(rawRow, laneWidth))
	body = append(body, "")

	if e.err != "" {
		body = append(body, chrome.danger("✗ "+e.err))
	} else if e.styleErr != "" {
		body = append(body, chrome.warning("⚠ "+e.styleErr+" — use the raw lane to edit"))
	}

	footer := termstyle.Footer([]termstyle.KeyHint{
		{Key: "↑↓", Label: "lanes"},
		{Key: "f/b", Label: "color"},
		{Key: "m", Label: "style"},
		{Key: "⏎", Label: "apply"},
		{Key: "esc", Label: "back"},
	}, width-6)
	return renderShell(chrome.theme, width, shellOpts{
		Title:  "TERTINT — edit: " + e.label,
		Body:   body,
		Footer: footer,
	})
}

// updateEditor drives the spec editor overlay from the Model.
func (m Model) updateEditor(key string, msg tea.KeyPressMsg) (Model, tea.Cmd) {
	e := *m.editor
	// Lane-scoped hotkeys: each picker opens from its own lane, so the same
	// letters can be typed verbatim into the raw spec (bold red, reverse...).
	switch {
	case key == "f" && e.focus == laneFg && e.decomposable:
		cp := newColorPicker(laneFg, e.fg, e.bg)
		m.colorPick = &cp
		return m, nil
	case key == "b" && e.focus == laneBg && e.decomposable:
		cp := newColorPicker(laneBg, e.fg, e.bg)
		m.colorPick = &cp
		return m, nil
	case key == "m" && e.focus == laneStyle && e.decomposable:
		sp := newStylePicker(e.styles)
		m.stylePick = &sp
		return m, nil
	case key == "r" && e.focus != laneRaw:
		e.focus = laneRaw
		m.editor = &e
		return m, nil
	}
	closed := e.update(key, msg)
	if closed {
		if key == "enter" {
			m.setSpec(e.role, e.raw, e.label)
		}
		m.editor = nil
		return m, nil
	}
	m.editor = &e
	return m, nil
}
