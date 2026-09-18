package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/0xbenc/termtint/internal/termstyle"
)

// stylePicker toggles the five SGR style bits.
type stylePicker struct {
	styles [5]bool
	cursor int
}

func newStylePicker(styles [5]bool) stylePicker {
	return stylePicker{styles: styles, cursor: 0}
}

// update handles one keystroke; it returns (closed, applied).
func (sp *stylePicker) update(key string) (closed, applied bool) {
	switch key {
	case "esc":
		return true, false
	case "enter":
		return true, true
	case "x":
		sp.styles[sp.cursor] = !sp.styles[sp.cursor]
	case "up":
		if sp.cursor > 0 {
			sp.cursor--
		}
	case "down":
		if sp.cursor < len(styleNames)-1 {
			sp.cursor++
		}
	}
	return false, false
}

// view renders the picker as a shell box.
func (sp stylePicker) view(st styler, width int) []string {
	boxWidth := min(width-2, 52)
	body := make([]string, 0, len(styleNames)+2)
	for i, name := range styleNames {
		marker := "  "
		if i == sp.cursor {
			marker = st.accent(">>")
		}
		state := st.subtle("off")
		if sp.styles[i] {
			state = st.success("on")
		}
		sample := termstyle.Apply(st.theme.NoColor, styleSGR[i], name)
		row := marker + " " + termstyle.PadRight(sample, 14) + state
		body = append(body, termstyle.Truncate(row, boxWidth-4))
	}
	body = append(body, "")
	footer := termstyle.Footer([]termstyle.KeyHint{
		{Key: "x", Label: "toggle"},
		{Key: "⏎", Label: "apply"},
		{Key: "esc", Label: "cancel"},
	}, boxWidth-4)
	return renderShell(st.theme, boxWidth, shellOpts{
		Title:  "styles",
		Body:   body,
		Footer: footer,
	})
}

// updateStylePick drives the style picker overlay.
func (m Model) updateStylePick(key string) (Model, tea.Cmd) {
	sp := *m.stylePick
	closed, applied := sp.update(key)
	if !closed {
		m.stylePick = &sp
		return m, nil
	}
	m.stylePick = nil
	if applied && m.editor != nil {
		ed := *m.editor
		ed.setStyles(sp.styles)
		ed.focus = laneStyle
		m.editor = &ed
	}
	return m, nil
}
