package ui

import (
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/0xbenc/termtint/internal/termstyle"
)

// colorPicker is the 256-color grid: 16 base colors, the 216-color cube
// (six rows, one per blue level), and the 24-step grayscale.
type colorPicker struct {
	target lane // laneFg or laneBg
	idx    int  // 0..255
	theme  termstyle.Theme
}

var colorRowWidths = []int{8, 8, 36, 36, 36, 36, 36, 36, 12, 12}

func newColorPicker(target lane, fg, bg string) colorPicker {
	idx := 0
	seed := fg
	if target == laneBg {
		seed = bg
	}
	if n, ok := parse256Index(seed); ok {
		idx = n
	}
	return colorPicker{target: target, idx: idx}
}

// parse256Index recovers a 0..255 index from an fg/bg token when possible.
func parse256Index(token string) (int, bool) {
	token = strings.TrimSpace(token)
	switch {
	case strings.HasPrefix(token, "38;5;"), strings.HasPrefix(token, "48;5;"):
		n, err := strconv.Atoi(strings.TrimPrefix(strings.TrimPrefix(token, "38;5;"), "48;5;"))
		if err == nil && n >= 0 && n <= 255 {
			return n, true
		}
	default:
	}
	for i, name := range baseColorNames {
		if token == name {
			return i, true
		}
		if token == "bright-"+name {
			return 8 + i, true
		}
		if token == "bg-"+name {
			return i, true
		}
		if token == "bg-bright-"+name {
			return 8 + i, true
		}
	}
	return 0, false
}

func (p colorPicker) rowCol() (int, int) {
	idx := p.idx
	for row, width := range colorRowWidths {
		if idx < width {
			return row, idx
		}
		idx -= width
	}
	return len(colorRowWidths) - 1, 0
}

func (p colorPicker) indexAt(row, col int) int {
	if row < 0 || row >= len(colorRowWidths) {
		return p.idx
	}
	col = clampInt(col, 0, colorRowWidths[row]-1)
	index := col
	for r := 0; r < row; r++ {
		index += colorRowWidths[r]
	}
	return index
}

// fragment returns the SGR fragment for the hovered index and target.
func (p colorPicker) fragment() string {
	idx := p.idx
	if target := p.target; target == laneBg {
		if idx < 8 {
			return strconv.Itoa(40 + idx)
		}
		if idx < 16 {
			return strconv.Itoa(100 + (idx - 8))
		}
		return "48;5;" + strconv.Itoa(idx)
	}
	if idx < 8 {
		return strconv.Itoa(30 + idx)
	}
	if idx < 16 {
		return strconv.Itoa(90 + (idx - 8))
	}
	return "38;5;" + strconv.Itoa(idx)
}

func (p colorPicker) swatch(idx int) string {
	frag := p.fragmentFor(idx)
	cell := " "
	if p.target == laneFg {
		cell = "█"
	}
	return termstyle.Apply(p.theme.NoColor, frag, cell)
}

func (p colorPicker) fragmentFor(idx int) string {
	if idx < 8 {
		if p.target == laneBg {
			return strconv.Itoa(40 + idx)
		}
		return strconv.Itoa(30 + idx)
	}
	if idx < 16 {
		if p.target == laneBg {
			return strconv.Itoa(100 + (idx - 8))
		}
		return strconv.Itoa(90 + (idx - 8))
	}
	if p.target == laneBg {
		return "48;5;" + strconv.Itoa(idx)
	}
	return "38;5;" + strconv.Itoa(idx)
}

// update handles one keystroke; it returns (closed, selected).
func (p *colorPicker) update(key string) (closed, selected bool) {
	row, col := p.rowCol()
	switch key {
	case "esc":
		return true, false
	case "enter":
		return true, true
	case "up":
		p.idx = p.indexAt(row-1, col)
	case "down":
		p.idx = p.indexAt(row+1, col)
	case "left":
		if col > 0 {
			p.idx = p.indexAt(row, col-1)
		}
	case "right":
		if col < colorRowWidths[row]-1 {
			p.idx = p.indexAt(row, col+1)
		}
	case "home":
		p.idx = 0
	case "end":
		p.idx = 255
	}
	return false, false
}

// view renders the picker as a shell box.
func (p colorPicker) view(st styler, width int) []string {
	boxWidth := min(width-2, 78)
	targetName := "foreground"
	if p.target == laneBg {
		targetName = "background"
	}

	row, col := p.rowCol()
	body := make([]string, 0, len(colorRowWidths)+4)
	body = append(body, st.muted("normal"))
	for r, widthCells := range colorRowWidths {
		if r == 2 {
			body = append(body, st.muted("cube"))
		}
		if r == 8 {
			body = append(body, st.muted("gray"))
		}
		var cells []string
		for c := 0; c < widthCells; c++ {
			idx := p.indexAt(r, c)
			swatch := p.swatch(idx)
			if r == row && c == col {
				swatch = termstyle.Apply(st.theme.NoColor, "7", swatch)
			}
			cells = append(cells, swatch)
		}
		body = append(body, strings.Join(cells, " "))
	}
	body = append(body, "")
	frag := p.fragment()
	name := ""
	if p.idx < 16 {
		if p.idx < 8 {
			name = baseColorNames[p.idx]
		} else {
			name = "bright-" + baseColorNames[p.idx-8]
		}
	}
	selSwatch := p.swatch(p.idx)
	body = append(body, st.muted("sel  ")+selSwatch+" "+st.foreground(frag)+" "+st.subtle(name))
	body = append(body, st.muted("exact rgb: type 38;2;r;g;b in the raw lane"))

	footer := termstyle.Footer([]termstyle.KeyHint{
		{Key: "←↑→↓", Label: "move"},
		{Key: "⏎", Label: "select"},
		{Key: "esc", Label: "cancel"},
	}, boxWidth)
	return renderShell(st.theme, boxWidth+4, shellOpts{
		Title:  "pick " + targetName,
		Body:   body,
		Footer: footer,
	})
}

// updateColorPick drives the color picker overlay.
func (m Model) updateColorPick(key string) (Model, tea.Cmd) {
	cp := *m.colorPick
	closed, selected := cp.update(key)
	if !closed {
		m.colorPick = &cp
		return m, nil
	}
	m.colorPick = nil
	if selected && m.editor != nil {
		target := cp.target
		fragment := cp.fragment()
		ed := *m.editor
		ed.applyFragment(target, fragment)
		ed.focus = target
		m.editor = &ed
	}
	return m, nil
}
