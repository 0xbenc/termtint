package ui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

// normalizedKey maps a Bubble Tea v2 key press to a stable name.
func normalizedKey(msg tea.KeyPressMsg) string {
	key := msg.String()
	if key == "" {
		key = msg.Keystroke()
	}
	switch key {
	case "\x1b[A", "\x1bOA":
		return "up"
	case "\x1b[B", "\x1bOB":
		return "down"
	case "\x1b[C", "\x1bOC":
		return "right"
	case "\x1b[D", "\x1bOD":
		return "left"
	case "\x1b[5~":
		return "pgup"
	case "\x1b[6~":
		return "pgdown"
	case "\x1b[H", "\x1b[1~", "\x1bOH":
		return "home"
	case "\x1b[F", "\x1b[4~", "\x1bOF":
		return "end"
	}
	text := msg.Text
	switch text {
	case "\x1b[A", "\x1bOA":
		return "up"
	case "\x1b[B", "\x1bOB":
		return "down"
	case "\x1b[C", "\x1bOC":
		return "right"
	case "\x1b[D", "\x1bOD":
		return "left"
	case "\x1b[5~":
		return "pgup"
	case "\x1b[6~":
		return "pgdown"
	case "\x1b[H", "\x1b[1~", "\x1bOH":
		return "home"
	case "\x1b[F", "\x1b[4~", "\x1bOF":
		return "end"
	}
	return key
}

// safeTextInput reports whether text is a single safe printable insertion.
func safeTextInput(text string) bool {
	if text == "" {
		return false
	}
	for _, r := range text {
		if r < 0x20 || r == 0x7f || (r >= 0x80 && r <= 0x9f) {
			return false
		}
	}
	return true
}

// sanitizePaste reduces pasted text to what a single-line field can accept:
// the content up to the first line break, minus control and C1 bytes.
func sanitizePaste(text string) string {
	if i := strings.IndexAny(text, "\r\n"); i >= 0 {
		text = text[:i]
	}
	return strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f || (r >= 0x80 && r <= 0x9f) {
			return -1
		}
		return r
	}, text)
}
