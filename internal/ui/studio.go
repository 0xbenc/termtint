package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/0xbenc/termnav"

	"github.com/0xbenc/termtheme"
	"github.com/0xbenc/termtint/internal/state"
	"github.com/0xbenc/termtint/internal/termstyle"
	"github.com/0xbenc/termtint/internal/theme"
)

// mode is the top-level screen.
type mode int

const (
	modeList mode = iota
	modeStudio
)

// studioRow is one flat row of the studio role list.
type studioRow struct {
	kind  int // rowBase or rowRole
	meta  roleMeta
	label string // base row: the palette name
}

const (
	rowBase = iota
	rowRole
)

// Model is the termtint studio program.
type Model struct {
	store   theme.Store
	version string
	width   int
	height  int
	glyphs  termstyle.GlyphSet

	mode mode

	// theme list screen
	entries []theme.Entry
	lFilter string
	lCursor int
	lScroll int
	lError  string

	// studio screen
	entry       theme.Entry
	origCfg     termtheme.ThemeConfig
	cfg         termtheme.ThemeConfig
	base        termstyle.Theme
	baseName    string
	unknownBase string
	filter      string
	rows        []studioRow
	cursor      int
	scroll      int
	history     []undoEntry
	message     string
	warning     string
	dirty       bool
	previewOn   bool
	noColor     bool
	hint        bool
	contrast    string

	// overlays (at most one non-nil)
	editor     *specEditor
	colorPick  *colorPicker
	stylePick  *stylePicker
	prompt     *promptModel
	saveReview *saveReview
	help       bool

	quit bool
}

type undoEntry struct {
	role termstyle.Role
	spec string
	had  bool
}

// NewModel builds the studio on top of the theme set.
func NewModel(store theme.Store, version string, env []string) Model {
	m := Model{
		store:   store,
		version: version,
		width:   100,
		height:  30,
		glyphs:  termstyle.ResolveGlyphs(env),
		mode:    modeList,
	}
	m.refreshEntries()
	return m
}

func (m *Model) refreshEntries() {
	entries, err := m.store.Entries()
	if err != nil {
		m.lError = err.Error()
		return
	}
	m.entries = entries
	if m.lCursor >= len(m.entries) {
		m.lCursor = 0
	}
}

// WithSize seeds the terminal size and preview state (snapshot rendering and
// tests).
func (m Model) WithSize(width, height int) Model {
	m.width = width
	m.height = height
	m.previewOn = width >= 100
	return m
}

// Open loads one entry into the studio screen.
func (m Model) Open(entry theme.Entry) Model {
	m.mode = modeStudio
	m.entry = entry
	m.origCfg = cloneConfig(entry.Config)
	m.cfg = cloneConfig(entry.Config)
	m.filter = ""
	m.message = ""
	m.warning = ""
	m.dirty = false
	m.history = nil
	m.cursor = 0
	m.scroll = 0

	base := termstyle.TerminalTheme()
	baseName := "terminal"
	if entry.BaseName != "" {
		if b, ok := termstyle.BuiltinTheme(entry.BaseName); ok {
			base = b
			baseName = b.Name
		} else {
			m.unknownBase = entry.BaseName
			m.warning = fmt.Sprintf("base %q is not a termtint palette; overrides are shown on the terminal base", entry.BaseName)
		}
	}
	m.base = base
	m.baseName = baseName
	m.rebuildRows()
	return m
}

func cloneConfig(cfg termtheme.ThemeConfig) termtheme.ThemeConfig {
	codes := make(map[termtheme.Role]string, len(cfg.Codes))
	for role, code := range cfg.Codes {
		codes[role] = code
	}
	specs := make(map[termtheme.Role]string, len(cfg.Specs))
	for role, spec := range cfg.Specs {
		specs[role] = spec
	}
	return termtheme.ThemeConfig{BaseName: cfg.BaseName, Codes: codes, Specs: specs, Warnings: append([]string(nil), cfg.Warnings...)}
}

// rebuildRows computes the flat studio rows for the current filter: the base
// row (only when the filter is empty), then the matching roles.
func (m *Model) rebuildRows() {
	roles := allRoles()
	rows := make([]studioRow, 0, len(roles)+1)
	if strings.TrimSpace(m.filter) == "" {
		rows = append(rows, studioRow{kind: rowBase, label: m.baseName})
	}
	for _, meta := range roles {
		if m.filter != "" {
			if _, ok := termnav.MatchFuzzy(m.filter, string(meta.Role)+" "+meta.Label+" "+meta.Group); !ok {
				continue
			}
		}
		rows = append(rows, studioRow{kind: rowRole, meta: meta})
	}
	m.rows = rows
	if m.cursor >= len(rows) {
		m.cursor = max(0, len(rows)-1)
	}
}

func (m Model) selectedRoleMeta() (roleMeta, bool) {
	if m.cursor < 0 || m.cursor >= len(m.rows) {
		return roleMeta{}, false
	}
	row := m.rows[m.cursor]
	if row.kind != rowRole {
		return roleMeta{}, false
	}
	return row.meta, true
}

// workingTheme resolves the theme the UI is currently rendering with: the base
// palette, the working overrides, and — while the spec editor is open — the
// editor's live draft on its role, so the whole screen re-skins as you pick.
func (m Model) workingTheme() termstyle.Theme {
	theme := m.cfg.Resolve(m.base.ToShared())
	if m.baseName != "terminal" || m.cfg.BaseName != "" {
		theme.Name = m.baseName
	}
	if m.editor != nil {
		if code, ok := m.editor.specCode(); ok {
			if code == "" {
				delete(theme.Codes, m.editor.role)
			} else {
				theme.Codes[m.editor.role] = code
			}
		}
	}
	t := termstyle.FromShared(theme)
	if m.noColor {
		t = t.WithNoColor(true)
	}
	return t
}

// WithNoColor forces plain output (no SGR), e.g. for --no-color or NO_COLOR.
func (m Model) WithNoColor(noColor bool) Model {
	m.noColor = noColor
	return m
}

// chromeTheme is the fixed skin the studio wears itself. The theme being
// edited styles only what it styles — role labels, spec values, the preview
// sheet — never the interface around it.
func (m Model) chromeTheme() termstyle.Theme {
	return m.chromeVariant(m.contrast)
}

// chromeVariant builds a chrome palette in a given contrast mode, honoring
// no-color.
func (m Model) chromeVariant(contrast string) termstyle.Theme {
	t := termstyle.ChromeThemeVariant(contrast)
	if m.noColor {
		t = t.WithNoColor(true)
	}
	return t
}

// applyContrastMode sets every role in the config to the contrast mode's
// color — overwriting explicit specs — so a save in this mode writes a fully
// white-out / black-out theme, the one the preview shows.
func (m *Model) applyContrastMode(mode string) {
	base := termstyle.ChromeThemeVariant(mode)
	for _, role := range termstyle.Roles() {
		code, err := termstyle.ParseStyleSpec(base.Codes[role])
		if err != nil {
			continue
		}
		m.cfg.Specs[role] = base.Codes[role]
		m.cfg.Codes[role] = code
	}
}

// contrastName is the display name of a contrast mode.
func contrastName(mode string) string {
	switch mode {
	case termstyle.ChromeContrastWhite:
		return "white-out"
	case termstyle.ChromeContrastBlack:
		return "black-out"
	default:
		return "current"
	}
}

// contrastWorkTheme is the monochrome lens the surface wears in a contrast
// mode: every role in one high-contrast color, regardless of the theme's
// current values. White-out is "current but whited out entirely"; black-out
// is the same in black. The real theme is untouched — it is what "current"
// shows and what a save writes.
func (m Model) contrastWorkTheme(mode string) termstyle.Theme {
	t := m.chromeVariant(mode)
	t.Name = contrastName(mode)
	return t
}

// contrastMode normalizes the contrast field ("" means default).
func (m Model) contrastMode() string {
	if m.contrast == "" {
		return termstyle.ChromeContrastDefault
	}
	return m.contrast
}

// alwaysVisible renders text in bold reverse video — the one style legible on
// any terminal background, light or dark. Affordances that must survive every
// contrast mode (the key that escapes black-out, the marker on the active
// mode) use it, so no mode can ever hide its own way out.
func (m Model) alwaysVisible(value string) string {
	return termstyle.Apply(m.noColor, "1;7", value)
}

// readable renders text in the terminal's default foreground — by definition
// readable on the user's own terminal, regardless of the chrome mode.
func (m Model) readable(value string) string {
	return termstyle.Apply(m.noColor, "39", value)
}

// contrastChips renders the three contrast modes as color chips: a block in
// each mode's own color, its name beside it in the always-readable default
// foreground, and a reverse-video marker on the active mode. A chip may go
// ghost on a mismatched background (that is the honest preview), but the name
// and the marker never do.
func (m Model) contrastChips() string {
	def := styler{theme: m.chromeVariant(termstyle.ChromeContrastDefault)}
	white := styler{theme: m.chromeVariant(termstyle.ChromeContrastWhite)}
	black := styler{theme: m.chromeVariant(termstyle.ChromeContrastBlack)}
	current := m.contrastMode()
	chip := func(s styler, mode, name string) string {
		label := s.foreground("█") + " " + m.readable(name)
		if mode == current {
			return m.alwaysVisible("»") + " " + label
		}
		return "  " + label
	}
	return chip(def, termstyle.ChromeContrastDefault, contrastName(termstyle.ChromeContrastDefault)) +
		chip(white, termstyle.ChromeContrastWhite, contrastName(termstyle.ChromeContrastWhite)) +
		chip(black, termstyle.ChromeContrastBlack, contrastName(termstyle.ChromeContrastBlack))
}

// contrastLegend is the main-page line for the contrast switch: the c key and
// label in reverse video (always visible), then the three mode chips.
func (m Model) contrastLegend() string {
	return m.alwaysVisible("c") + m.alwaysVisible(" contrast:") + "  " + m.contrastChips()
}

// cycleContrast walks current → white-out → black-out → current. The mode
// itself never persists, but saving while in a contrast mode rewrites every
// role in that mode's color (see openSaveReview).
func (m *Model) cycleContrast() {
	switch m.contrastMode() {
	case termstyle.ChromeContrastWhite:
		m.contrast = termstyle.ChromeContrastBlack
	case termstyle.ChromeContrastBlack:
		m.contrast = termstyle.ChromeContrastDefault
	default: // default (or unset)
		m.contrast = termstyle.ChromeContrastWhite
	}
	m.message = "contrast: " + contrastName(m.contrastMode()) + " (c cycles)"
}

// MaybeHint arms the first-run getting-started overlay when it has not yet
// been shown for this version.
func (m Model) MaybeHint(env []string) Model {
	dir, err := state.ResolveDir(env)
	if err != nil {
		return m
	}
	if state.Load(dir).StudioHintVersion == m.version {
		return m
	}
	m.hint = true
	return m
}

// dismissHint closes the getting-started overlay and records it as shown for
// this version.
func (m *Model) dismissHint() {
	m.hint = false
	dir, err := state.ResolveDir(m.store.Env)
	if err != nil {
		return
	}
	st := state.Load(dir)
	st.SetStudioHintVersion(m.version)
	_ = st.Save(dir)
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return tea.RequestWindowSize
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if msg.Width > 0 {
			m.width = msg.Width
		}
		if msg.Height > 0 {
			m.height = msg.Height
		}
		if m.previewOn == false {
			m.previewOn = widthFitsPreview(m.width)
		}
		return m, nil
	case tea.PasteMsg:
		text := sanitizePaste(msg.Content)
		switch {
		case m.prompt != nil:
			m.prompt = m.prompt.paste(text)
		case m.editor != nil:
			m.editor = m.editor.paste(text)
		default:
			m = m.updateFilter(text)
		}
		return m, nil
	case tea.KeyPressMsg:
		key := normalizedKey(msg)
		if key == "ctrl+c" || key == "ctrl+q" {
			m.quit = true
			return m, tea.Quit
		}
		if m.hint {
			if key == "c" {
				// The welcome page doubles as the contrast legend: c cycles
				// the chrome without dismissing.
				m.cycleContrast()
				return m, nil
			}
			// Any other key dismisses the getting-started overlay.
			m.dismissHint()
			return m, nil
		}
		if m.help {
			if key == "?" || key == "esc" {
				m.help = false
			}
			return m, nil
		}
		if m.saveReview != nil {
			return m.updateSaveReview(key)
		}
		if m.prompt != nil {
			return m.updatePrompt(key, msg)
		}
		if m.colorPick != nil {
			return m.updateColorPick(key)
		}
		if m.stylePick != nil {
			return m.updateStylePick(key)
		}
		if m.editor != nil {
			return m.updateEditor(key, msg)
		}
		if m.mode == modeList {
			return m.updateList(key, msg)
		}
		return m.updateStudio(key, msg)
	}
	return m, nil
}

func widthFitsPreview(width int) bool {
	return width >= 100
}

// updateFilter feeds text (typed or pasted) into the active screen's filter.
func (m Model) updateFilter(text string) Model {
	if m.mode == modeList {
		if safeTextInput(text) {
			m.lFilter += text
			m.lCursor = 0
		}
		return m
	}
	if safeTextInput(text) {
		m.filter += text
		m.rebuildRows()
		m.cursor = 0
	}
	return m
}

func (m Model) updateList(key string, msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch key {
	case "esc":
		m.quit = true
		return m, tea.Quit
	case "backspace":
		if m.lFilter != "" {
			runes := []rune(m.lFilter)
			m.lFilter = string(runes[:len(runes)-1])
			m.lCursor = 0
		}
	case "enter":
		entries, err := m.filteredEntries()
		if err != nil {
			m.lError = err.Error()
			return m, nil
		}
		if m.lCursor >= len(entries) {
			m.lCursor = 0
		}
		m = m.Open(entries[m.lCursor])
	case "up", "ctrl+p":
		m.lCursor = max(0, m.lCursor-1)
	case "down", "ctrl+n":
		if entries, _ := m.filteredEntries(); m.lCursor < len(entries)-1 {
			m.lCursor++
		}
	case "pgup":
		m.lCursor = max(0, m.lCursor-8)
	case "pgdown":
		if entries, _ := m.filteredEntries(); m.lCursor < len(entries)-1 {
			m.lCursor = min(m.lCursor+8, len(entries)-1)
		}
	case "home":
		m.lCursor = 0
	case "end":
		if entries, _ := m.filteredEntries(); len(entries) > 0 {
			m.lCursor = len(entries) - 1
		}
	case "c":
		m.cycleContrast()
	case "?":
		if m.lFilter == "" {
			m.help = true
		}
	}
	return m, nil
}

func (m Model) filteredEntries() ([]theme.Entry, error) {
	if m.lFilter == "" {
		if m.lError != "" {
			return nil, fmt.Errorf("%s", m.lError)
		}
		return m.entries, nil
	}
	var out []theme.Entry
	for _, entry := range m.entries {
		if _, ok := termnav.MatchFuzzy(m.lFilter, entry.Name+" "+string(entry.Kind)); ok {
			out = append(out, entry)
		}
	}
	return out, nil
}

func (m Model) updateStudio(key string, msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch key {
	case "esc":
		if m.filter != "" {
			m.filter = ""
			m.rebuildRows()
			m.cursor = 0
			return m, nil
		}
		m.mode = modeList
		m.message = ""
		return m, nil
	case "backspace":
		if m.filter != "" {
			runes := []rune(m.filter)
			m.filter = string(runes[:len(runes)-1])
			m.rebuildRows()
			m.cursor = 0
		}
	case "up", "ctrl+p", "shift+tab":
		// Every row is hoverable, so this is a plain clamped step;
		// termnav.Snap is for skipping reference rows and would stay put.
		m.cursor = max(0, m.cursor-1)
		m.clampScroll()
	case "down", "ctrl+n", "tab":
		m.cursor = min(max(0, len(m.rows)-1), m.cursor+1)
		m.clampScroll()
	case "pgup":
		m.cursor = max(0, m.cursor-m.listBudget())
		m.clampScroll()
	case "pgdown":
		m.cursor = min(max(0, len(m.rows)-1), m.cursor+m.listBudget())
		m.clampScroll()
	case "home":
		m.cursor = 0
		m.clampScroll()
	case "end":
		if len(m.rows) > 0 {
			m.cursor = len(m.rows) - 1
			m.clampScroll()
		}
	case "shift+up":
		m.cursor = termnav.JumpSection(len(m.rows), m.cursor, -1, m.rowGroup)
		m.clampScroll()
	case "shift+down":
		m.cursor = termnav.JumpSection(len(m.rows), m.cursor, 1, m.rowGroup)
		m.clampScroll()
	case "enter", "e":
		meta, ok := m.selectedRoleMeta()
		if ok {
			m.editor = NewSpecEditor(meta, m.cfg.Specs[meta.Role])
		}
	case "left":
		m.cycleBaseOrPreset(-1)
	case "right":
		m.cycleBaseOrPreset(1)
	case "u":
		m.undo()
	case "d":
		m.inheritCurrent()
	case "a":
		m.resetAll()
	case "s":
		m.openSaveReview()
	case "x":
		m.openExportPrompt()
	case "i":
		m.openImportPrompt()
	case "n":
		m.openNewPrompt()
	case "p":
		m.previewOn = !m.previewOn
	case "c":
		m.cycleContrast()
	case "?":
		if m.filter == "" {
			m.help = true
		}
	}
	return m, nil
}

// rowGroup reports the section of a studio row for section jumping and window
// headers; the base row belongs to no section.
func (m Model) rowGroup(i int) string {
	if i < 0 || i >= len(m.rows) {
		return ""
	}
	row := m.rows[i]
	if row.kind != rowRole {
		return ""
	}
	return row.meta.Group
}

// cycleBaseOrPreset: on the base row, left/right cycles the builtin base
// palette; on a role row, it quick-cycles that role's presets.
func (m *Model) cycleBaseOrPreset(dir int) {
	if m.cursor >= 0 && m.cursor < len(m.rows) && m.rows[m.cursor].kind == rowBase {
		names := termstyle.BuiltinThemeNames()
		idx := 0
		for i, name := range names {
			if name == m.baseName {
				idx = i
				break
			}
		}
		idx = (idx + dir + len(names)) % len(names)
		base, _ := termstyle.BuiltinTheme(names[idx])
		m.base = base
		m.baseName = base.Name
		m.message = "base " + base.Name
		return
	}
	meta, ok := m.selectedRoleMeta()
	if !ok || len(meta.Presets) == 0 {
		return
	}
	current := strings.TrimSpace(m.cfg.Specs[meta.Role])
	idx := 0
	for i, preset := range meta.Presets {
		if strings.TrimSpace(preset) == current {
			idx = i
			break
		}
	}
	next := (idx + dir + len(meta.Presets)) % len(meta.Presets)
	m.setSpec(meta.Role, meta.Presets[next], meta.Label)
}

func (m *Model) setSpec(role termstyle.Role, spec string, label string) {
	spec = strings.TrimSpace(spec)
	old, had := m.cfg.Specs[role]
	if spec == "" {
		delete(m.cfg.Specs, role)
		delete(m.cfg.Codes, role)
	} else {
		code, err := termstyle.ParseStyleSpec(spec)
		if err != nil {
			m.message = err.Error()
			return
		}
		m.cfg.Specs[role] = spec
		m.cfg.Codes[role] = code
	}
	m.history = append(m.history, undoEntry{role: role, spec: old, had: had})
	if len(m.history) > 128 {
		m.history = m.history[len(m.history)-128:]
	}
	m.dirty = true
	if spec == "" {
		m.message = label + " -> inherit"
	} else {
		m.message = label + " -> " + spec
	}
}

func (m *Model) undo() {
	if len(m.history) == 0 {
		m.message = "nothing to undo"
		return
	}
	last := m.history[len(m.history)-1]
	m.history = m.history[:len(m.history)-1]
	if last.had {
		m.cfg.Specs[last.role] = last.spec
		code, _ := termstyle.ParseStyleSpec(last.spec)
		m.cfg.Codes[last.role] = code
	} else {
		delete(m.cfg.Specs, last.role)
		delete(m.cfg.Codes, last.role)
	}
	m.dirty = true
	m.message = "undid " + string(last.role)
}

func (m *Model) inheritCurrent() {
	meta, ok := m.selectedRoleMeta()
	if !ok {
		return
	}
	m.setSpec(meta.Role, "", meta.Label)
}

func (m *Model) resetAll() {
	for _, meta := range allRoles() {
		if _, ok := m.cfg.Specs[meta.Role]; ok {
			m.setSpec(meta.Role, "", meta.Label)
		}
	}
	m.message = "reset all roles to base"
}

// listBudget is the number of logical rows (base + roles) the list may show.
// The frame must always fit the terminal: each shown row costs a line, each
// visible section a header line (≤5), the scroll indicators may cost two, and
// the contrast legend (1) and status line (1) sit below. At height 30 this
// admits all 17 rows exactly; on shorter terminals the list scrolls.
func (m Model) listBudget() int {
	budget := m.height - 13 // shell(4) + status(1) + legend(1) + headers(≤5) + indicators(≤2)
	return clampInt(budget, 8, 40)
}

func (m *Model) clampScroll() {
	if len(m.rows) == 0 {
		m.scroll = 0
		return
	}
	contains := func(start, cursor int) bool {
		return termnav.WindowContainsCursor(len(m.rows), start, cursor, m.listBudget(), func(i int) (string, bool) {
			group := m.rowGroup(i)
			return group, i >= 0 && i < len(m.rows)
		})
	}
	m.cursor, m.scroll = termnav.ClampWindow(len(m.rows), m.cursor, m.scroll, contains)
}

// View implements tea.Model. Two skins: the chrome (fixed, always readable)
// for the interface, and the working theme (what is being edited) for the
// role list values and the preview sheet.
func (m Model) View() tea.View {
	chrome := styler{theme: m.chromeTheme()}
	// The working theme is what is being edited ("current"). In a contrast
	// mode the surface wears a monochrome lens instead: white-out / black-out
	// render every role in one color, whatever the theme's current values are.
	workTheme := m.workingTheme()
	if mode := m.contrastMode(); mode != termstyle.ChromeContrastDefault {
		workTheme = m.contrastWorkTheme(mode)
	}
	work := styler{theme: workTheme}
	var lines []string
	switch {
	case m.hint:
		lines = m.hintBox(m.width)
	case m.help:
		lines = helpBox(chrome, m.width)
	case m.saveReview != nil:
		lines = m.saveReview.view(chrome, m.width)
	case m.prompt != nil:
		lines = m.prompt.view(chrome, m.width)
	case m.colorPick != nil:
		lines = m.colorPick.view(chrome, m.width)
	case m.stylePick != nil:
		lines = m.stylePick.view(chrome, m.width)
	case m.editor != nil:
		lines = m.editorView(chrome, work, m.width)
	case m.mode == modeList:
		lines = m.listView(chrome)
	default:
		lines = m.studioView(chrome, work)
	}
	view := tea.NewView(strings.Join(lines, "\n") + "\n")
	view.AltScreen = true
	return view
}

func (m Model) listView(chrome styler) []string {
	st := chrome
	entries, err := m.filteredEntries()
	body := make([]string, 0, len(entries)+2)
	if m.lError != "" && m.lFilter == "" {
		body = append(body, st.danger("theme set error: "+m.lError))
		body = append(body, "")
	}
	if err != nil {
		body = append(body, st.danger(err.Error()))
	}
	if len(entries) == 0 && err == nil {
		if m.lFilter != "" {
			body = append(body, st.muted("no themes match "+st.search(m.lFilter)))
		} else {
			body = append(body, st.muted("no themes found"))
		}
	}
	for i, entry := range entries {
		caret := "  "
		if i == m.lCursor {
			caret = st.accent(">>")
		}
		name := st.selected(entry.Name)
		if i != m.lCursor {
			name = st.primary(entry.Name)
		}
		line := caret + " " + termstyle.PadRight(name, 16) +
			termstyle.PadRight(st.secondary(string(entry.Kind)), 8)
		if entry.Kind == theme.KindBuiltin {
			line += st.muted("builtin palette")
		} else if entry.Kind == theme.KindLive {
			line += st.muted(entry.Path)
			if !entry.Exists {
				line += st.subtle("  (no config yet)")
			}
		} else {
			line += st.muted(entry.Path)
		}
		if len(entry.Warnings) > 0 {
			line += st.warning("  " + entry.Warnings[0])
		}
		body = append(body, termstyle.Truncate(line, m.width-4))
	}
	if m.lFilter != "" {
		body = append(body, "")
		body = append(body, st.muted("filter: "+st.search(m.lFilter)))
	}
	footer := termstyle.Footer([]termstyle.KeyHint{
		{Key: "type", Label: "filter"},
		{Key: "enter", Label: "open"},
		{Key: "c", Label: "contrast"},
		{Key: "?", Label: "help"},
		{Key: "esc", Label: "quit"},
	}, m.width-6)
	return renderShell(st.theme, m.width, shellOpts{
		Title:  "TERTINT — themes",
		Body:   body,
		Footer: footer,
	})
}

func (m Model) studioView(chrome, work styler) []string {
	leftWidth := m.width - 4
	rightWidth := 0
	if m.previewOn && m.width >= 100 {
		leftWidth = 56
		rightWidth = m.width - 4 - leftWidth - 3 // " " + "|" + " "
	}

	left := m.renderRoleList(chrome, work, leftWidth)
	left = append(left, termstyle.Truncate(m.contrastLegend(), leftWidth))
	var body []string
	if rightWidth > 0 {
		// The preview sheet wears the working theme: the honest answer to
		// "what will the app look like" — and in a contrast mode that
		// theme is the monochrome starting point.
		right := previewSheet(work, m.glyphs, rightWidth, m.height-5)
		rows := max(len(left), len(right))
		for i := 0; i < rows; i++ {
			l := ""
			if i < len(left) {
				l = left[i]
			}
			r := ""
			if i < len(right) {
				r = right[i]
			}
			body = append(body, termstyle.PadRight(l, leftWidth)+" "+chrome.muted("|"))
			body[i] = body[i] + " " + r
		}
	} else {
		body = left
	}

	status := m.statusLine(chrome)
	if status != "" {
		body = append(body, status)
	}

	// The footer carries only what this screen needs right now; the full
	// reference is one ? away.
	hints := []termstyle.KeyHint{
		{Key: "↑↓", Label: "move"},
		{Key: "type", Label: "filter"},
		{Key: "enter", Label: "edit"},
		{Key: "s", Label: "save"},
		{Key: "c", Label: "contrast"},
		{Key: "?", Label: "help"},
	}
	dot := ""
	if m.dirty {
		dot = " · unsaved"
	}
	return renderShell(chrome.theme, m.width, shellOpts{
		Title:  m.entry.Name + dot,
		Body:   body,
		Footer: termstyle.Footer(hints, m.width-6),
	})
}

// statusLine shows the filter, a message, or a load warning.
func (m Model) statusLine(st styler) string {
	if m.filter != "" {
		return st.muted("filter: " + st.search(m.filter))
	}
	if m.message != "" {
		return st.muted(m.message)
	}
	if m.warning != "" {
		return st.warning(m.warning)
	}
	return ""
}

// renderRoleList renders the flat role list with section headers and the
// canonical "N more" windowing markers.
func (m Model) renderRoleList(chrome, work styler, width int) []string {
	n := len(m.rows)
	if n == 0 {
		if m.filter != "" {
			return []string{chrome.muted("no roles match " + chrome.search(m.filter))}
		}
		return []string{chrome.muted("no roles")}
	}
	budget := m.listBudget()
	if budget > n {
		budget = n
	}
	if m.scroll > n {
		m.scroll = n - 1
	}
	start := m.scroll
	if n-start < budget {
		start = max(0, n-budget)
	}

	var lines []string
	if start > 0 {
		lines = append(lines, chrome.subtle(fmt.Sprintf("… %d more above", start)))
	}
	lastGroup := ""
	rendered := 0
	for i := start; i < n && rendered < budget; i++ {
		group := m.rowGroup(i)
		newGroup := group != "" && group != lastGroup
		if newGroup {
			lines = append(lines, sectionHeader(chrome, group, width))
		}
		lines = append(lines, m.renderRoleRow(chrome, work, i, width))
		if newGroup {
			lastGroup = group
		}
		rendered++
	}
	if remaining := n - (start + rendered); remaining > 0 {
		lines = append(lines, chrome.subtle(fmt.Sprintf("… %d more", remaining)))
	}
	return lines
}

func (m Model) renderRoleRow(chrome, work styler, i int, width int) string {
	row := m.rows[i]
	caret := "  "
	if i == m.cursor {
		caret = chrome.accent(">>")
	}
	if row.kind == rowBase {
		line := caret + " " +
			termstyle.PadRight(chrome.title("base"), 13) + " " +
			termstyle.PadRight(work.primary(termstyle.Truncate(row.label, 18)), 18)
		if width >= 54 {
			line += " " + chrome.muted(termstyle.Truncate("starting palette (←/→ switch)", width-termstyle.VisibleWidth(line)-1))
		}
		return termstyle.PadRight(line, width)
	}
	meta := row.meta
	value := "(inherit)"
	if spec := strings.TrimSpace(m.cfg.Specs[meta.Role]); spec != "" {
		value = spec
	}
	// The label and value wear the role's own working-theme color — an
	// honest preview of what the app will paint; the description stays in
	// chrome so it never inherits the theme's dimness.
	line := caret + " " +
		termstyle.PadRight(work.style(meta.Role, termstyle.Truncate(meta.Label, 13)), 13) + " " +
		termstyle.PadRight(work.style(meta.Role, termstyle.Truncate(value, 18)), 18)
	if width >= 54 {
		line += " " + chrome.muted(termstyle.Truncate(meta.Desc, width-termstyle.VisibleWidth(line)-1))
	}
	return termstyle.PadRight(line, width)
}
