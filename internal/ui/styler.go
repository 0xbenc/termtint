package ui

import (
	"strings"

	"github.com/0xbenc/termchrome"

	"github.com/0xbenc/termtint/internal/termstyle"
)

// styler applies termtint's current theme to text, one method per role.
type styler struct {
	theme termstyle.Theme
}

func (s styler) title(value string) string     { return s.theme.Style(termstyle.RoleTitle, value) }
func (s styler) primary(value string) string   { return s.theme.Style(termstyle.RolePrimary, value) }
func (s styler) secondary(value string) string { return s.theme.Style(termstyle.RoleSecondary, value) }
func (s styler) accent(value string) string    { return s.theme.Style(termstyle.RoleAccent, value) }
func (s styler) muted(value string) string     { return s.theme.Style(termstyle.RoleMuted, value) }
func (s styler) subtle(value string) string    { return s.theme.Style(termstyle.RoleSubtle, value) }
func (s styler) foreground(value string) string {
	return s.theme.Style(termstyle.RoleForeground, value)
}
func (s styler) selected(value string) string { return s.theme.Style(termstyle.RoleSelected, value) }
func (s styler) selectedBar(value string) string {
	return s.theme.Style(termstyle.RoleSelectedBar, value)
}
func (s styler) border(value string) string  { return s.theme.Style(termstyle.RoleBorder, value) }
func (s styler) success(value string) string { return s.theme.Style(termstyle.RoleSuccess, value) }
func (s styler) warning(value string) string { return s.theme.Style(termstyle.RoleWarning, value) }
func (s styler) danger(value string) string  { return s.theme.Style(termstyle.RoleDanger, value) }
func (s styler) info(value string) string    { return s.theme.Style(termstyle.RoleInfo, value) }
func (s styler) search(value string) string  { return s.theme.Style(termstyle.RoleSearch, value) }
func (s styler) pill(value string) string    { return s.theme.Style(termstyle.RolePill, value) }

func (s styler) style(role termstyle.Role, value string) string {
	return s.theme.Style(role, value)
}

type shellOpts struct {
	Title  string
	Body   []string
	Footer string
	Danger bool
}

// renderShell is termtint's local shell composition over the shared termchrome
// box geometry: the geometry is termchrome's, the composition and the
// Sanitize-on-overflow policy are termtint's.
func renderShell(theme termstyle.Theme, width int, opts shellOpts) []string {
	width = max(48, width)
	tt := theme.ToShared()
	var lines []string
	titleRole := termstyle.RoleTitle
	if opts.Danger {
		titleRole = termstyle.RoleDanger
	}
	title := strings.TrimSpace(opts.Title)
	if title == "" {
		title = "termtint"
	}
	lines = append(lines, termchrome.Top(tt, theme.Style(titleRole, title), width, truncateStyled))
	for _, line := range opts.Body {
		lines = append(lines, termchrome.Line(tt, line, width, truncateStyled))
	}
	if opts.Footer != "" {
		lines = append(lines, termchrome.Divider(tt, width))
		lines = append(lines, termchrome.Line(tt, theme.Style(termstyle.RoleMuted, opts.Footer), width, truncateStyled))
	}
	lines = append(lines, termchrome.Bottom(tt, width))
	return lines
}

// truncateStyled is termtint's Truncator seam: Sanitize on overflow so an
// oversized chrome label can never inject control bytes.
func truncateStyled(value string, width int) string {
	if width <= 0 {
		return ""
	}
	if termstyle.VisibleWidth(value) <= width {
		return value
	}
	return termstyle.Truncate(termstyle.Sanitize(value), width)
}

// sectionHeader renders a list group header row.
func sectionHeader(st styler, label string, width int) string {
	return termstyle.PadRight(st.subtle(strings.ToUpper(label)), width)
}

// padLine pads a rendered line to width cells.
func padLine(line string, width int) string {
	return termstyle.PadRight(line, width)
}

// clampInt clamps v into [lo, hi].
func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
