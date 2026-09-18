package ui

import (
	"github.com/0xbenc/termtint/internal/termstyle"
)

// hintBox is the first-run getting-started overlay: what this studio is, the
// keys that matter, and the high-contrast switch — each legend word rendered
// in the mode it names, so the three states are visible at a glance. c
// cycles the chrome without dismissing; any other key dismisses.
func (m Model) hintBox(width int) []string {
	boxWidth := min(width-2, 76)
	st := styler{theme: m.chromeTheme()}
	body := []string{
		st.foreground("This studio themes " + st.search(m.entry.Name) + ":"),
		"",
		st.foreground("  every role the app can paint, with a live"),
		st.foreground("  preview of its full chrome vocabulary."),
		"",
		st.accent("  ↑↓") + st.foreground("          move · type to filter"),
		st.accent("  enter") + st.foreground("     edit the selected role"),
		st.accent("  s") + st.foreground("             save to the app's theme.conf"),
		st.accent("  ?") + st.foreground("             full key reference"),
		"",
		"  " + m.alwaysVisible("c") + m.readable("             high-contrast base: "),
		m.readable("     ") + m.contrastChips(),
	}
	footer := termstyle.Footer([]termstyle.KeyHint{{Key: "any", Label: "key to continue"}}, boxWidth-4)
	return renderShell(st.theme, boxWidth, shellOpts{
		Title:  "TERTINT — welcome",
		Body:   body,
		Footer: footer,
	})
}

// helpBox renders the grouped key reference.
func helpBox(st styler, width int) []string {
	boxWidth := min(width-2, 72)
	rows := []struct {
		section string
		hints   []string
	}{
		{"navigate", []string{"up/down move", "pgup/pgdn page", "home/end jump", "type filter", "esc clear filter"}},
		{"studio", []string{"enter edit role", "←/→ preset or base", "u undo", "d inherit", "a reset all"}},
		{"theme", []string{"s save", "n new", "x export", "i import", "p preview"}},
		{"contrast", []string{"c contrast: current/white-out/black-out", "saving in a mode rewrites all roles in it"}},
		{"editor", []string{"up/down lanes", "f fg color", "b bg color", "m style", "type raw spec", "⏎ apply", "esc cancel"}},
		{"pickers", []string{"arrows move", "x toggle", "⏎ select", "esc cancel"}},
		{"quit", []string{"esc", "ctrl+c", "ctrl+q"}},
	}
	var body []string
	for _, row := range rows {
		body = append(body, sectionHeader(st, row.section, boxWidth-4))
		for _, hint := range row.hints {
			key, label := splitHint(hint)
			body = append(body, "  "+termstyle.PadRight(st.accent(key), 18)+st.foreground(label))
		}
		body = append(body, "")
	}
	footer := termstyle.Footer([]termstyle.KeyHint{{Key: "?", Label: "close"}, {Key: "esc", Label: "close"}}, boxWidth-4)
	return renderShell(st.theme, boxWidth, shellOpts{
		Title:  "TERTINT — help",
		Body:   body,
		Footer: footer,
	})
}

func splitHint(hint string) (string, string) {
	for i, r := range hint {
		if r == ' ' {
			return hint[:i], hint[i+1:]
		}
	}
	return hint, ""
}
