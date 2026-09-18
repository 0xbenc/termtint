package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/0xbenc/termtheme"
	"github.com/0xbenc/termtint/internal/termstyle"
	"github.com/0xbenc/termtint/internal/theme"
)

func keyMsg(value string) tea.KeyPressMsg {
	switch value {
	case "enter":
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter})
	case "esc":
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape})
	case "tab":
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyTab})
	case "up":
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyUp})
	case "down":
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyDown})
	case "left":
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyLeft})
	case "right":
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyRight})
	case "ctrl+u":
		return tea.KeyPressMsg(tea.Key{Code: 'u', Mod: tea.ModCtrl})
	default:
		return tea.KeyPressMsg(tea.Key{Text: value})
	}
}

// update drives one message through the model and unwraps the concrete type.
func update(m Model, msg tea.Msg) Model {
	next, _ := m.Update(msg)
	return next.(Model)
}

// typeText feeds printable text into the model key by key.
func typeText(m Model, text string) Model {
	for _, ch := range text {
		m = update(m, tea.KeyPressMsg(tea.Key{Text: string(ch)}))
	}
	return m
}

// studioFixture points the model at a temp config dir whose passage
// theme.conf carries one override, and returns the opened studio model.
func studioFixture(t *testing.T) (Model, theme.Store, string) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	env := []string{"XDG_CONFIG_HOME=" + dir}

	path := filepath.Join(dir, "passage", "theme.conf")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte("theme = terminal\nprimary = bold red\n"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	store, err := theme.NewStore(env)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	entry, err := store.Load("passage")
	if err != nil {
		t.Fatalf("Load(passage): %v", err)
	}
	model := NewModel(store, "test", env)
	model = model.WithNoColor(true)
	model = model.Open(entry)
	model = model.WithSize(100, 30)
	return model, store, path
}

func viewText(m Model) string {
	return strings.ReplaceAll(m.View().Content, "\n", " ")
}

func TestStudioListShowsTheFamily(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	env := []string{"XDG_CONFIG_HOME=" + dir}
	store, err := theme.NewStore(env)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	model := NewModel(store, "test", env)
	model = model.WithNoColor(true)
	model = model.WithSize(80, 24)

	text := viewText(model)
	for _, want := range []string{"TERTINT", "terminal", "tint", "passage (live)", "ssherpa (live)", "dangit (live)", "bitty (live)"} {
		if !strings.Contains(text, want) {
			t.Fatalf("list view = %q, want %q", text, want)
		}
	}
	if strings.Contains(text, "\x1b[") {
		t.Fatal("list view contains ANSI escapes with NoColor")
	}
}

func TestStudioShowsRolesPreviewAndSections(t *testing.T) {
	model, _, _ := studioFixture(t)
	text := viewText(model)
	for _, want := range []string{
		"passage (live)", "primary", "bold red",
		"TEXT", "SELECTION", "CHROME", "STATE", "SPECIAL",
		"vocabulary", "SWATCHES", "enter edit",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("studio view = %q, want %q", text, want)
		}
	}
	if strings.Contains(text, "\x1b[") {
		t.Fatal("studio view contains ANSI escapes with NoColor")
	}
}

func TestStudioFilterNarrowsRows(t *testing.T) {
	model, _, _ := studioFixture(t)
	model = model.updateFilter("selec")
	// The base row hides while filtering; only the matching roles remain.
	if len(model.rows) != 2 {
		var roles []string
		for _, row := range model.rows {
			roles = append(roles, string(row.meta.Role))
		}
		t.Fatalf("rows = %d (%v), want 2: selected + selected_bar", len(model.rows), roles)
	}
	if model.rows[0].meta.Role != termtheme.RoleSelected || model.rows[1].meta.Role != termtheme.RoleSelectedBar {
		t.Fatalf("rows = %v %v, want selected then selected_bar", model.rows[0].meta.Role, model.rows[1].meta.Role)
	}
}

func TestStudioEditCommitsSpec(t *testing.T) {
	model, _, _ := studioFixture(t)
	// Row layout: 0 = base, then allRoles(); primary is the second role.
	model.cursor = 2
	model = update(model, keyMsg("enter"))
	if model.editor == nil {
		t.Fatal("editor did not open on enter")
	}
	if model.editor.role != termtheme.RolePrimary {
		t.Fatalf("editor role = %v, want primary", model.editor.role)
	}

	// The editor seeds the current spec ("bold red"); clear it first.
	model = update(model, keyMsg("ctrl+u"))
	if model.editor.raw != "" {
		t.Fatalf("raw = %q after ctrl+u, want empty", model.editor.raw)
	}
	model = typeText(model, "magenta")
	model = update(model, keyMsg("enter"))
	if model.editor != nil {
		t.Fatalf("editor still open after commit (raw=%q)", model.editor.raw)
	}
	if got := model.cfg.Specs[termtheme.RolePrimary]; got != "magenta" {
		t.Fatalf("primary spec = %q, want %q", got, "magenta")
	}
	if !model.dirty {
		t.Fatal("dirty = false after an edit, want true")
	}
}

func TestStudioSaveWritesConfig(t *testing.T) {
	model, _, path := studioFixture(t)
	model.cursor = 2
	model = update(model, keyMsg("enter"))
	model = update(model, keyMsg("ctrl+u"))
	model = typeText(model, "magenta")
	model = update(model, keyMsg("enter"))

	model = update(model, keyMsg("s"))
	if model.saveReview == nil {
		t.Fatal("save review did not open on s")
	}
	model = update(model, keyMsg("enter"))
	if model.saveReview == nil || !model.saveReview.done {
		t.Fatal("save review did not confirm on enter")
	}
	model = update(model, keyMsg("esc"))
	if model.saveReview != nil {
		t.Fatal("save review still open after esc")
	}
	if model.dirty {
		t.Fatal("dirty = true after a confirmed save, want false")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(data), "primary = magenta") {
		t.Fatalf("config = %q, want the primary override", data)
	}
}

func TestSaveAppliesContrastMode(t *testing.T) {
	model, _, path := studioFixture(t)
	model = update(model, keyMsg("c")) // -> white-out
	model = update(model, keyMsg("s"))
	if model.saveReview == nil {
		t.Fatal("save review did not open on s")
	}
	// The review must show the full white-out, including the hand edit.
	review := strings.Join(model.saveReview.lines, "\n")
	if !strings.Contains(review, "1;37") {
		t.Fatalf("review = %q, want white-out codes", review)
	}
	model = update(model, keyMsg("enter"))
	model = update(model, keyMsg("esc"))

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	for _, want := range []string{
		"primary = 1;37", // the hand edit (bold red) is rewritten white
		"border = 1;37",
		"selected = 1;40",
		"selected_bar = 40",
	} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("config = %q, want %q", data, want)
		}
	}
	// The model now carries the applied specs, so current mode shows the
	// same theme the file holds.
	if got := model.cfg.Specs[termtheme.RoleBorder]; got != "1;37" {
		t.Fatalf("border spec after white-out save = %q, want 1;37", got)
	}
}

func TestSaveReviewCancelRestores(t *testing.T) {
	model, _, path := studioFixture(t)
	model = update(model, keyMsg("c")) // -> white-out
	before := make(map[termtheme.Role]string, len(model.cfg.Specs))
	for role, spec := range model.cfg.Specs {
		before[role] = spec
	}
	model = update(model, keyMsg("s"))
	if model.saveReview == nil {
		t.Fatal("save review did not open on s")
	}
	if model.cfg.Specs[termtheme.RoleBorder] != "1;37" {
		t.Fatalf("contrast mode not applied to the review, border = %q", model.cfg.Specs[termtheme.RoleBorder])
	}
	model = update(model, keyMsg("esc")) // cancel
	if model.saveReview != nil {
		t.Fatal("save review still open after esc")
	}
	if len(model.cfg.Specs) != len(before) {
		t.Fatalf("specs after cancel = %v, want the pre-review set", model.cfg.Specs)
	}
	for role, spec := range before {
		if model.cfg.Specs[role] != spec {
			t.Fatalf("spec %s after cancel = %q, want %q", role, model.cfg.Specs[role], spec)
		}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if strings.Contains(string(data), "1;37") {
		t.Fatalf("cancelled save wrote the white-out theme: %q", data)
	}
}

func TestStudioRawLaneTypesHotkeyLetters(t *testing.T) {
	model, _, _ := studioFixture(t)
	model.cursor = 2
	model = update(model, keyMsg("enter"))
	model = update(model, keyMsg("ctrl+u"))
	// b/m/f/r are picker and focus hotkeys on the other lanes; on the raw
	// lane they must be plain text.
	model = typeText(model, "bold red")
	model = update(model, keyMsg("enter"))
	if model.editor != nil {
		t.Fatal("editor still open after commit")
	}
	if model.colorPick != nil || model.stylePick != nil {
		t.Fatal("a picker opened while typing in the raw lane")
	}
	if got := model.cfg.Specs[termtheme.RolePrimary]; got != "bold red" {
		t.Fatalf("primary spec = %q, want %q", got, "bold red")
	}
}

func TestStudioUndoRevertsSpec(t *testing.T) {
	model, _, _ := studioFixture(t)
	before := model.cfg.Specs[termtheme.RolePrimary]

	model.cursor = 2
	model = update(model, keyMsg("enter"))
	model = update(model, keyMsg("ctrl+u"))
	model = typeText(model, "magenta")
	model = update(model, keyMsg("enter"))
	if got := model.cfg.Specs[termtheme.RolePrimary]; got == before {
		t.Fatalf("spec still %q after edit, want it changed", before)
	}

	model = update(model, keyMsg("u"))
	if got := model.cfg.Specs[termtheme.RolePrimary]; got != before {
		t.Fatalf("spec = %q after undo, want %q", got, before)
	}
}

// dimFixture points a color-on model at a passage theme whose own subtle and
// muted roles are SGR-dim — the worst case for a studio that wears the theme
// it edits.
func dimFixture(t *testing.T) (Model, []string) {
	t.Helper()
	cfg := t.TempDir()
	stateHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfg)
	t.Setenv("XDG_STATE_HOME", stateHome)
	env := []string{
		"XDG_CONFIG_HOME=" + cfg,
		"XDG_STATE_HOME=" + stateHome,
	}
	path := filepath.Join(cfg, "passage", "theme.conf")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte("theme = terminal\nsubtle = 2\nmuted = 2\n"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	store, err := theme.NewStore(env)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	entry, err := store.Load("passage")
	if err != nil {
		t.Fatalf("Load(passage): %v", err)
	}
	model := NewModel(store, "test", env)
	model = model.Open(entry)
	model = model.WithSize(120, 30)
	return model, env
}

func TestStudioChromeStaysReadableWithDimTheme(t *testing.T) {
	model, _ := dimFixture(t)
	text := viewText(model)

	// The chrome (section headers) must wear the chrome palette's lifted
	// subtle — never the edited theme's SGR-dim subtle.
	if !strings.Contains(text, "\x1b[38;5;245mTEXT") {
		t.Fatalf("studio view = %q, want chrome section header in 38;5;245", text)
	}
	// The honest preview of the dimmed role itself is allowed to be dim.
	if !strings.Contains(text, "\x1b[2m") {
		t.Fatalf("studio view = %q, want the subtle role's own dim preview", text)
	}
}

func TestChromeThemeLiftsLowContrastRoles(t *testing.T) {
	ct := termstyle.ChromeTheme()
	if ct.Codes[termstyle.RoleSubtle] == "2" {
		t.Fatal("chrome subtle is still SGR-dim")
	}
	if ct.Codes[termstyle.RoleSelected] != "1;7" {
		t.Fatalf("chrome selected = %q, want bold-reverse", ct.Codes[termstyle.RoleSelected])
	}
}

func TestHintOverlayShowsThenDismisses(t *testing.T) {
	model, env := dimFixture(t)
	model = model.MaybeHint(env)
	text := viewText(model)
	for _, want := range []string{"TERTINT — welcome", "edit the selected role", "full key reference"} {
		if !strings.Contains(text, want) {
			t.Fatalf("hint view = %q, want %q", text, want)
		}
	}

	// Any key dismisses it and records the version in state.
	model = update(model, keyMsg("h"))
	if model.hint {
		t.Fatal("hint still up after a keypress")
	}
	if !strings.Contains(viewText(model), "primary") {
		t.Fatal("studio not visible after hint dismissed")
	}
	stateData, err := os.ReadFile(filepath.Join(env[1][len("XDG_STATE_HOME="):], "termtint", "state.json"))
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	if !strings.Contains(string(stateData), `"studio_hint_version": "test"`) {
		t.Fatalf("state = %s, want studio_hint_version recorded", stateData)
	}

	// A fresh model with the same state does not re-arm.
	again, _ := dimFixture(t)
	again = again.MaybeHint(env)
	if again.hint {
		t.Fatal("hint re-armed after it was already shown for this version")
	}
}

func TestChromeContrastCyclesWhiteBlackDefault(t *testing.T) {
	model, _ := dimFixture(t)
	if model.contrast != "" {
		t.Fatalf("initial contrast = %q, want default", model.contrast)
	}

	model = update(model, keyMsg("c"))
	if model.contrast != termstyle.ChromeContrastWhite {
		t.Fatalf("contrast after c = %q, want white", model.contrast)
	}
	text := viewText(model)
	if !strings.Contains(text, "\x1b[1;37m") {
		t.Fatalf("white mode view = %q, want bold-white chrome", text)
	}
	if !strings.Contains(text, "contrast: white-out") {
		t.Fatalf("white mode view = %q, want status message", text)
	}
	// The preview must white out too — white-out is a starting point for the
	// theme, and the preview is where the theme is seen.
	if !strings.Contains(text, "\x1b[1;37msuccess line") {
		t.Fatalf("white mode preview not white-out:\n%s", text)
	}

	model = update(model, keyMsg("c"))
	if model.contrast != termstyle.ChromeContrastBlack {
		t.Fatalf("contrast = %q, want black", model.contrast)
	}
	text = viewText(model)
	if !strings.Contains(text, "\x1b[30m") {
		t.Fatal("black mode view, want black chrome")
	}
	// And the preview blacks out.
	if !strings.Contains(text, "\x1b[30msuccess line") {
		t.Fatalf("black mode preview not black-out:\n%s", text)
	}

	model = update(model, keyMsg("c"))
	if model.contrast != termstyle.ChromeContrastDefault {
		t.Fatalf("contrast = %q, want back to default", model.contrast)
	}
	// Back to the multi-color chrome: the section header wears the lifted
	// gray again, not monochrome white.
	if !strings.Contains(viewText(model), "\x1b[38;5;245mTEXT") {
		t.Fatal("default mode view, want the multi-color chrome section header")
	}
	// And the preview is back to the working theme (terminal base success).
	if !strings.Contains(viewText(model), "\x1b[32msuccess line") {
		t.Fatalf("default mode preview not back to the working theme:\n%s", viewText(model))
	}
	// The cycle must keep spinning after a full lap: default must not be a
	// dead end.
	want := []string{
		termstyle.ChromeContrastWhite,
		termstyle.ChromeContrastBlack,
		termstyle.ChromeContrastDefault,
		termstyle.ChromeContrastWhite,
	}
	for i, w := range want {
		model = update(model, keyMsg("c"))
		if model.contrast != w {
			t.Fatalf("press %d: contrast = %q, want %q", i+4, model.contrast, w)
		}
	}
}

func TestContrastModeWhitesOutCurrentValues(t *testing.T) {
	model, _ := dimFixture(t)
	model.cfg.Specs[termtheme.RoleTitle] = "32" // title -> green
	model.cfg.Codes[termtheme.RoleTitle] = "32"
	model = update(model, keyMsg("c")) // -> white-out
	text := viewText(model)
	// White-out renders the current theme entirely in white — the green
	// title included. It adjusts every current value, not just the inherited
	// ones.
	if !strings.Contains(text, "\x1b[1;37mTERTINT") {
		t.Fatalf("white-out preview not fully white:\n%s", text)
	}
	if strings.Contains(text, "\x1b[32mTERTINT") {
		t.Fatalf("white-out preview still shows the green title:\n%s", text)
	}
	// Back in current, the green title is back.
	model = update(model, keyMsg("c")) // -> black-out
	model = update(model, keyMsg("c")) // -> current
	if !strings.Contains(viewText(model), "\x1b[32mTERTINT") {
		t.Fatalf("current mode missing the green title:\n%s", viewText(model))
	}
}

func TestHintOverlayContrastCyclesWithoutDismissing(t *testing.T) {
	model, env := dimFixture(t)
	model = model.MaybeHint(env)

	model = update(model, keyMsg("c"))
	if !model.hint {
		t.Fatal("c dismissed the welcome overlay; it should cycle contrast")
	}
	if model.contrast != termstyle.ChromeContrastWhite {
		t.Fatalf("contrast = %q, want white", model.contrast)
	}
	// The legend shows all three states, each in its own skin.
	text := viewText(model)
	for _, want := range []string{"current", "white-out", "black-out"} {
		if !strings.Contains(text, want) {
			t.Fatalf("hint view = %q, want legend word %q", text, want)
		}
	}

	model = update(model, keyMsg("x"))
	if model.hint {
		t.Fatal("welcome overlay still up after a non-c key")
	}
}

func TestChromeThemeVariantsAreMonochrome(t *testing.T) {
	white := termstyle.ChromeThemeVariant(termstyle.ChromeContrastWhite)
	for _, role := range termstyle.Roles() {
		if role == termstyle.RoleSelected || role == termstyle.RoleSelectedBar {
			continue
		}
		if white.Codes[role] != "1;37" {
			t.Fatalf("white %s = %q, want 1;37", role, white.Codes[role])
		}
	}
	if white.Codes[termstyle.RoleSelected] != "1;40" || white.Codes[termstyle.RoleSelectedBar] != "40" {
		t.Fatal("white selected roles should invert on black")
	}

	black := termstyle.ChromeThemeVariant(termstyle.ChromeContrastBlack)
	for _, role := range termstyle.Roles() {
		if role == termstyle.RoleSelected || role == termstyle.RoleSelectedBar {
			continue
		}
		if black.Codes[role] != "30" {
			t.Fatalf("black %s = %q, want 30", role, black.Codes[role])
		}
	}
	if black.Codes[termstyle.RoleSelected] != "1;47" {
		t.Fatal("black selected role should invert on white")
	}

	def := termstyle.ChromeThemeVariant(termstyle.ChromeContrastDefault)
	if def.Codes[termstyle.RoleTitle] == "1;37" || def.Codes[termstyle.RoleTitle] == "30" {
		t.Fatal("default chrome should stay multi-color")
	}
}

func TestStudioMainPageShowsContrastLegend(t *testing.T) {
	model, _ := dimFixture(t)
	text := viewText(model)
	for _, want := range []string{"contrast:", "white-out", "black-out", "»"} {
		if !strings.Contains(text, want) {
			t.Fatalf("studio view = %q, want legend %q", text, want)
		}
	}
	// Anchor at the legend: the marker must precede the active mode's word.
	markerBefore := func(text, word string) bool {
		leg := strings.Index(text, "contrast:")
		i, d := strings.Index(text[leg:], "»")+leg, strings.Index(text[leg:], word)+leg
		return i < d
	}
	if !markerBefore(text, "current") {
		t.Fatalf("active current: marker not before \"current\" in legend")
	}

	model = update(model, keyMsg("c"))
	if !markerBefore(viewText(model), "white-out") {
		t.Fatalf("after c: marker not before \"white-out\" in legend")
	}

	model = update(model, keyMsg("c"))
	if !markerBefore(viewText(model), "black-out") {
		t.Fatalf("after cc: marker not before \"black-out\" in legend")
	}

	model = update(model, keyMsg("c"))
	if !markerBefore(viewText(model), "current") {
		t.Fatalf("after cycling back: marker not before \"current\" in legend")
	}
}

func TestContrastLegendSurvivesBlackOut(t *testing.T) {
	model, _ := dimFixture(t)
	model = update(model, keyMsg("c")) // → white
	model = update(model, keyMsg("c")) // → black
	text := viewText(model)

	// The escape key and the active marker are bold reverse video — legible
	// on any background — never the black chrome color. Black-out must not
	// be able to hide the key that escapes it.
	if !strings.Contains(text, "\x1b[1;7mc\x1b[0m") {
		t.Fatalf("black-out: c key not reverse-video visible:\n%s", text)
	}
	if !strings.Contains(text, "\x1b[1;7m»\x1b[0m") {
		t.Fatalf("black-out: active-mode marker not reverse-video visible")
	}
	// The active mode is black-out: its chip is black (an honest ghost on a
	// dark terminal), but its name stays in the readable default foreground.
	if !strings.Contains(text, "\x1b[30m█\x1b[0m") {
		t.Fatalf("black-out: active chip not rendered in black:\n%s", text)
	}
	if !strings.Contains(text, "\x1b[39mblack-out\x1b[0m") {
		t.Fatalf("black-out: active mode name not readable")
	}
}

func TestStudioFrameFitsTerminal(t *testing.T) {
	model, _ := dimFixture(t)
	for _, h := range []int{24, 28, 30, 40} {
		model = model.WithSize(120, h)
		if lines := strings.Count(model.View().Content, "\n"); lines > h {
			t.Fatalf("height %d: studio frame is %d lines, want ≤ %d", h, lines, h)
		}
		// With a status message (the worst case) the frame must still fit.
		model.message = "contrast: white-out (c cycles)"
		if lines := strings.Count(model.View().Content, "\n"); lines > h {
			t.Fatalf("height %d with message: frame is %d lines, want ≤ %d", h, lines, h)
		}
		model.message = ""
	}
	model = model.WithSize(120, 24)
	model.mode = modeList
	if lines := strings.Count(model.View().Content, "\n"); lines > 24 {
		t.Fatalf("list frame is %d lines, want ≤ 24", lines)
	}
}

func TestStudioArrowsMoveCursor(t *testing.T) {
	model, _, _ := studioFixture(t)
	start := model.cursor
	model = update(model, keyMsg("down"))
	if model.cursor != start+1 {
		t.Fatalf("down: cursor = %d, want %d", model.cursor, start+1)
	}
	model = update(model, keyMsg("down"))
	model = update(model, keyMsg("up"))
	if model.cursor != start+1 {
		t.Fatalf("up: cursor = %d, want %d", model.cursor, start+1)
	}
	// up at the top stays put
	for model.cursor > 0 {
		model = update(model, keyMsg("up"))
	}
	if model.cursor != 0 {
		t.Fatalf("cursor = %d, want 0 after walking to the top", model.cursor)
	}
	// The caret actually moves in the rendered view.
	first := viewText(model)
	model = update(model, keyMsg("down"))
	second := viewText(model)
	if first == second {
		t.Fatal("view unchanged after down; the caret did not move")
	}
}

func TestListArrowsMoveCursor(t *testing.T) {
	model, _, _ := studioFixture(t)
	model.mode = modeList
	model = update(model, keyMsg("down"))
	if model.lCursor != 1 {
		t.Fatalf("down: lCursor = %d, want 1", model.lCursor)
	}
	model = update(model, keyMsg("up"))
	if model.lCursor != 0 {
		t.Fatalf("up: lCursor = %d, want 0", model.lCursor)
	}
}
