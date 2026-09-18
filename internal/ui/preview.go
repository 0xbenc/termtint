package ui

import (
	"fmt"
	"strings"

	"github.com/0xbenc/termchrome"
	"github.com/0xbenc/termnav"
	"github.com/0xbenc/termnav/render"

	"github.com/0xbenc/termtint/internal/termstyle"
)

// previewSheet renders the full chrome vocabulary with the given theme: the
// box, the selection cue, fuzzy highlight, KV rows, the state roles, the
// spinner/bar/countdown ramp, a forced footer overflow, and the role swatch
// strip. Rows are dropped from the tail (swatches first) until the sheet
// fits height, so a short terminal never overflows.
func previewSheet(st styler, glyphs termstyle.GlyphSet, width, height int) []string {
	inner := max(0, width-4)
	if inner < 20 {
		return []string{st.muted(termstyle.Truncate("(too narrow for preview)", width))}
	}

	box := make([]string, 0, 16)
	box = append(box,
		st.title("TERTINT")+" "+st.pill("LIVE"),
		st.muted(fmt.Sprintf("16 roles  ·  base %s", st.theme.Name)),
		"",
	)
	// Selection cue: the canonical ">>" caret on a full-row selected bar.
	barInner := max(2, inner-2)
	box = append(box, st.selectedBar(st.selected(">>  selected row")+strings.Repeat(" ", max(0, barInner-16))))
	box = append(box, "  "+st.primary("primary row  ")+" "+st.muted("12 / 92"))

	// Fuzzy highlight: "arch" matched in a sample row.
	res, ok := termnav.MatchFuzzy("arch", "archive | backup | token")
	if ok {
		box = append(box, "  "+render.HighlightMatches("archive | backup | token", res.Positions, inner-2,
			func(s string) string { return st.muted(s) },
			func(s string) string { return st.search(s) }))
	}

	box = append(box,
		"",
		kvRow(st, "label", "value"),
		st.success("success line"),
		st.warning("warning line"),
		st.danger("danger line"),
		st.info("info line"),
		"",
	)
	// Spinner frame + urgency bar + countdown (2s of 10: the danger end).
	bar := glyphs.Bar(2, 10, 10)
	box = append(box, glyphs.Frame(3)+" "+st.style(termstyle.RoleDanger, bar)+" "+st.danger("2s"))
	box = append(box, "")
	// Footer with more hints than fit, to show the +N overflow.
	hints := []termstyle.KeyHint{
		{Key: "a", Label: "alpha"},
		{Key: "b", Label: "bravo"},
		{Key: "c", Label: "charlie"},
		{Key: "d", Label: "delta"},
		{Key: "e", Label: "echo"},
		{Key: "f", Label: "foxtrot"},
	}
	box = append(box, termstyle.Footer(hints, inner-2))

	swatchRows := swatchStrip(st, inner)

	lines := []string{sectionHeader(st, "preview", width)}
	lines = append(lines, boxLines(st, "vocabulary", box, width)...)
	// The swatch section costs its rows plus a header line.
	fits := len(lines) + len(swatchRows)
	if len(swatchRows) > 0 {
		fits++
	}
	if fits > height {
		overflow := fits - height
		if overflow >= len(swatchRows) {
			swatchRows = nil
		} else {
			swatchRows = swatchRows[:len(swatchRows)-overflow]
		}
	}
	if len(swatchRows) > 0 {
		lines = append(lines, sectionHeader(st, "swatches", width))
		lines = append(lines, swatchRows...)
	}
	// Last-resort clamp: the sheet must never push the studio frame past the
	// terminal height.
	if len(lines) > height {
		lines = lines[:height]
	}
	for i, line := range lines {
		lines[i] = padLine(line, width)
	}
	return lines
}

// boxLines wraps body rows in a termchrome box of the given width.
func boxLines(st styler, label string, body []string, width int) []string {
	tt := st.theme.ToShared()
	var lines []string
	lines = append(lines, termchrome.Top(tt, label, width, truncateStyled))
	for _, line := range body {
		lines = append(lines, termchrome.Line(tt, line, width, truncateStyled))
	}
	lines = append(lines, termchrome.Bottom(tt, width))
	return lines
}

func kvRow(st styler, label, value string) string {
	return termstyle.PadRight(st.muted(label), 8) + st.foreground(value)
}

var swatchOrder = []struct {
	role   termstyle.Role
	sample string
}{
	{termstyle.RoleTitle, "The quick"},
	{termstyle.RolePrimary, "brown fox"},
	{termstyle.RoleSecondary, "jumps over"},
	{termstyle.RoleAccent, "the lazy"},
	{termstyle.RoleMuted, "dog 0123"},
	{termstyle.RoleSubtle, "faint 9876"},
	{termstyle.RoleForeground, "plain text"},
	{termstyle.RoleSelected, "selected"},
	{termstyle.RoleSelectedBar, "bar text"},
	{termstyle.RoleBorder, "border"},
	{termstyle.RoleSuccess, "ok"},
	{termstyle.RoleWarning, "warn"},
	{termstyle.RoleDanger, "fail"},
	{termstyle.RoleInfo, "info"},
	{termstyle.RoleSearch, "match"},
	{termstyle.RolePill, "chip"},
}

// swatchStrip renders the 16 roles as name + sample rows, each sample styled
// in its own role. Two columns when the pane is wide enough, one below.
func swatchStrip(st styler, inner int) []string {
	if inner < 46 {
		var lines []string
		for _, entry := range swatchOrder {
			lines = append(lines, swatchCell(st, entry, inner))
		}
		return lines
	}
	cell := (inner - 1) / 2
	var lines []string
	for i := 0; i < len(swatchOrder); i += 2 {
		left := swatchCell(st, swatchOrder[i], cell)
		if i+1 < len(swatchOrder) {
			right := swatchCell(st, swatchOrder[i+1], cell)
			lines = append(lines, left+" "+right)
		} else {
			lines = append(lines, left)
		}
	}
	return lines
}

func swatchCell(st styler, entry struct {
	role   termstyle.Role
	sample string
}, cell int) string {
	name := termstyle.PadRight(st.muted(string(entry.role)), 12)
	sample := st.style(entry.role, entry.sample)
	return termstyle.Truncate(name+" "+sample, cell)
}
