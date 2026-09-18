package ui

import (
	"strings"

	"github.com/0xbenc/termtint/internal/termstyle"
)

// roleMeta describes one role row in the studio list.
type roleMeta struct {
	Role    termstyle.Role
	Label   string
	Desc    string
	Group   string
	Presets []string
}

var textPresets = []string{
	"", "bold", "dim", "underline", "bright-white", "bold bright-cyan", "1;38;2;96;221;255",
}
var selectionPresets = []string{
	"", "bold", "bold bright-white", "reverse", "1;38;2;255;255;255", "48;2;45;55;72",
}
var chromePresets = []string{
	"", "dim", "bold", "bright-black", "1;38;2;58;69;87",
}
var statePresets = []string{
	"", "bold", "dim", "bright-green", "bright-yellow", "bright-red",
}
var specialPresets = []string{
	"", "bold", "underline", "bold bright-white", "reverse",
}

// allRoles is the studio's role list in display order, grouped for section
// headers and section jumping.
func allRoles() []roleMeta {
	return []roleMeta{
		{Role: termstyle.RoleTitle, Label: "title", Desc: "window title and banners", Group: "Text", Presets: textPresets},
		{Role: termstyle.RolePrimary, Label: "primary", Desc: "primary content and paths", Group: "Text", Presets: textPresets},
		{Role: termstyle.RoleSecondary, Label: "secondary", Desc: "secondary text, counts", Group: "Text", Presets: textPresets},
		{Role: termstyle.RoleAccent, Label: "accent", Desc: "highlights, the >> caret", Group: "Text", Presets: textPresets},
		{Role: termstyle.RoleMuted, Label: "muted", Desc: "metadata, timestamps", Group: "Text", Presets: textPresets},
		{Role: termstyle.RoleSubtle, Label: "subtle", Desc: "faint rules and hints", Group: "Text", Presets: textPresets},
		{Role: termstyle.RoleForeground, Label: "foreground", Desc: "default text", Group: "Text", Presets: textPresets},

		{Role: termstyle.RoleSelected, Label: "selected", Desc: "selected row text", Group: "Selection", Presets: selectionPresets},
		{Role: termstyle.RoleSelectedBar, Label: "selected_bar", Desc: "selected row background", Group: "Selection", Presets: selectionPresets},

		{Role: termstyle.RoleBorder, Label: "border", Desc: "box borders and rules", Group: "Chrome", Presets: chromePresets},

		{Role: termstyle.RoleSuccess, Label: "success", Desc: "success and OK lines", Group: "State", Presets: statePresets},
		{Role: termstyle.RoleWarning, Label: "warning", Desc: "warnings", Group: "State", Presets: statePresets},
		{Role: termstyle.RoleDanger, Label: "danger", Desc: "errors and destructive", Group: "State", Presets: statePresets},
		{Role: termstyle.RoleInfo, Label: "info", Desc: "informational lines", Group: "State", Presets: statePresets},

		{Role: termstyle.RoleSearch, Label: "search", Desc: "fuzzy match highlights", Group: "Special", Presets: specialPresets},
		{Role: termstyle.RolePill, Label: "pill", Desc: "badge chips", Group: "Special", Presets: specialPresets},
	}
}

// presetLabel renders a preset spec for display ("(inherit)" for the empty one).
func presetLabel(preset string) string {
	if strings.TrimSpace(preset) == "" {
		return "(inherit)"
	}
	return preset
}
