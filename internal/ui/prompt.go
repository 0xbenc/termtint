package ui

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/0xbenc/termtint/internal/termstyle"
)

// promptModel is a single-line input with a block cursor, used for file paths
// and theme names.
type promptModel struct {
	label    string
	hint     string
	value    string
	caret    int
	err      string
	validate func(string) bool
	onSubmit func(string)
}

func newPrompt(label, hint, initial string, validate func(string) bool, onSubmit func(string)) *promptModel {
	return &promptModel{
		label:    label,
		hint:     hint,
		value:    initial,
		caret:    len(initial),
		validate: validate,
		onSubmit: onSubmit,
	}
}

func (p *promptModel) paste(text string) *promptModel {
	p.value = p.value[:p.caret] + text + p.value[p.caret:]
	p.caret += len(text)
	return p
}

// update handles one keystroke; it returns whether the prompt closed.
func (p *promptModel) update(key string, msg tea.KeyPressMsg) bool {
	switch key {
	case "esc":
		return true
	case "enter":
		value := strings.TrimSpace(p.value)
		if p.validate != nil && !p.validate(value) {
			p.err = "invalid"
			return false
		}
		p.onSubmit(value)
		return true
	case "backspace":
		if p.caret > 0 {
			p.value = p.value[:p.caret-1] + p.value[p.caret:]
			p.caret--
		}
		p.err = ""
	case "left":
		if p.caret > 0 {
			p.caret--
		}
	case "right":
		if p.caret < len(p.value) {
			p.caret++
		}
	case "home":
		p.caret = 0
	case "end":
		p.caret = len(p.value)
	case "ctrl+u":
		p.value = ""
		p.caret = 0
		p.err = ""
	default:
		if safeTextInput(msg.Text) {
			text := msg.Text
			p.value = p.value[:p.caret] + text + p.value[p.caret:]
			p.caret += len(text)
			p.err = ""
		}
	}
	return false
}

func (p *promptModel) view(st styler, width int) []string {
	boxWidth := min(width-2, 72)
	display := p.value[:p.caret] + st.accent("▮") + p.value[p.caret:]
	body := []string{
		st.muted(p.label),
		"",
		st.foreground(termstyle.Truncate(display, boxWidth-6)),
	}
	if p.err != "" {
		body = append(body, st.danger("✗ "+p.err))
	} else if p.hint != "" {
		body = append(body, st.subtle(p.hint))
	}
	footer := termstyle.Footer([]termstyle.KeyHint{
		{Key: "⏎", Label: "submit"},
		{Key: "^u", Label: "clear"},
		{Key: "esc", Label: "cancel"},
	}, boxWidth-4)
	return renderShell(st.theme, boxWidth, shellOpts{
		Title:  "TERTINT",
		Body:   body,
		Footer: footer,
	})
}

// updatePrompt drives the prompt overlay.
func (m Model) updatePrompt(key string, msg tea.KeyPressMsg) (Model, tea.Cmd) {
	closed := m.prompt.update(key, msg)
	if closed {
		m.prompt = nil
	}
	return m, nil
}
