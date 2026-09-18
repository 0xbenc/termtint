package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/0xbenc/termtheme"
	"github.com/0xbenc/termtint/internal/termstyle"
	"github.com/0xbenc/termtint/internal/theme"
)

// saveReview shows the diff before a write and performs it on confirm.
type saveReview struct {
	title    string
	lines    []string
	target   string
	do       func() (string, error)
	preApply termtheme.ThemeConfig // config as it was before the contrast mode was applied
	applied  bool
	result   string
	failed   string
	done     bool
}

// openSaveReview builds the review overlay for the current working theme.
// A builtin entry is saved as a new named theme, so it goes to the name
// prompt first.
func (m *Model) openSaveReview() {
	if m.entry.Kind == theme.KindBuiltin {
		m.openNewPrompt()
		return
	}
	sr := saveReview{}
	// Saving in a contrast mode rewrites every role in that mode's color, so
	// the file becomes the fully white-out / black-out theme the preview
	// shows. The review presents the full change, and cancelling it restores
	// the config as it was.
	if mode := m.contrastMode(); mode != termstyle.ChromeContrastDefault {
		sr.preApply = cloneConfig(m.cfg)
		sr.applied = true
		m.applyContrastMode(mode)
	}
	var lines []string
	changed := 0
	for _, role := range termstyle.Roles() {
		old := strings.TrimSpace(m.origCfg.Specs[role])
		new := strings.TrimSpace(m.cfg.Specs[role])
		if old == new {
			continue
		}
		changed++
		lines = append(lines, diffLine(styler{theme: m.workingTheme()}, role, old, new))
	}
	if changed == 0 {
		if m.baseName != m.origCfg.BaseName && m.origCfg.BaseName != "" {
			lines = append(lines, fmt.Sprintf("base  %s -> %s", m.origCfg.BaseName, m.baseName))
		} else {
			lines = append(lines, "(no changes)")
		}
	}
	if len(m.cfg.Warnings) > 0 {
		lines = append(lines, termstyle.Truncate("warnings: "+strings.Join(m.cfg.Warnings, "; "), 60))
	}

	target := m.entry.Path
	if m.entry.Kind == theme.KindLive {
		target = m.entry.Path
	}
	sr.title = "save — " + m.entry.Name
	sr.lines = lines
	sr.target = target
	sr.do = func() (string, error) {
		return m.performSave()
	}
	m.saveReview = &sr
}

func (m *Model) performSave() (string, error) {
	if m.entry.Kind == theme.KindLive {
		res, err := theme.Install(theme.InstallOptions{
			App:     m.entry.App,
			Config:  m.cfg,
			Base:    m.base.ToShared(),
			Version: m.version,
			Env:     m.store.Env,
		})
		if err != nil {
			return "", err
		}
		if !res.Changed {
			return "no changes to " + res.Path, nil
		}
		return "saved " + res.Path + backupNote(res.BackupPath), nil
	}
	path, res, err := m.store.SaveNamed(m.entry.Name, m.cfg, m.base.ToShared(), m.version)
	if err != nil {
		return "", err
	}
	if !res.Changed {
		return "no changes to " + path, nil
	}
	return "saved " + path + backupNote(res.BackupPath), nil
}

func backupNote(backup string) string {
	if backup == "" {
		return ""
	}
	return " (backup " + backup + ")"
}

// diffLine renders one changed role: swatch of the old spec, arrow, swatch of
// the new, both readable.
func diffLine(st styler, role termstyle.Role, old, new string) string {
	swatch := func(spec string) string {
		if strings.TrimSpace(spec) == "" {
			return st.muted("(inherit)")
		}
		code, err := termstyle.ParseStyleSpec(spec)
		if err != nil {
			return st.danger(termstyle.Strip(spec))
		}
		return termstyle.Apply(st.theme.NoColor, code, "▮")
	}
	name := termstyle.PadRight(termstyle.Strip(string(role)), 13)
	left := swatch(old) + " " + termstyle.Truncate(termstyle.Strip(old), 16)
	right := swatch(new) + " " + termstyle.Truncate(termstyle.Strip(new), 16)
	return termstyle.PadRight(name, 13) + "  " + left + "  →  " + right
}

// view renders the review as a shell box.
func (sr saveReview) view(st styler, width int) []string {
	boxWidth := min(width-2, 78)
	body := make([]string, 0, len(sr.lines)+4)
	for _, line := range sr.lines {
		body = append(body, termstyle.Truncate(line, boxWidth-4))
	}
	body = append(body, "")
	body = append(body, st.muted("target  ")+" "+st.foreground(termstyle.Truncate(sr.target, boxWidth-13)))
	if sr.done {
		if sr.failed != "" {
			body = append(body, st.danger("✗ "+sr.failed))
		} else {
			body = append(body, st.success("✓ "+sr.result))
		}
	}
	footer := termstyle.Footer([]termstyle.KeyHint{
		{Key: "⏎", Label: "confirm"},
		{Key: "esc", Label: "cancel"},
	}, boxWidth-4)
	if sr.done {
		footer = termstyle.Footer([]termstyle.KeyHint{{Key: "esc", Label: "close"}}, boxWidth-4)
	}
	return renderShell(st.theme, boxWidth, shellOpts{
		Title:  sr.title,
		Body:   body,
		Footer: footer,
	})
}

// updateSaveReview drives the review overlay.
func (m Model) updateSaveReview(key string) (Model, tea.Cmd) {
	sr := *m.saveReview
	if sr.done {
		if key == "esc" {
			m.saveReview = nil
			if sr.failed == "" {
				m.origCfg = cloneConfig(m.cfg)
				m.dirty = false
				m.message = sr.result
			}
		}
		return m, nil
	}
	switch key {
	case "esc":
		if sr.applied {
			m.cfg = sr.preApply
		}
		m.saveReview = nil
	case "enter":
		result, err := sr.do()
		sr.done = true
		sr.result = result
		if err != nil {
			sr.failed = err.Error()
		}
		m.saveReview = &sr
	}
	return m, nil
}

// openExportPrompt asks for a path and writes the portable .theme there.
func (m *Model) openExportPrompt() {
	defaultName := m.entry.Name
	if m.entry.Kind == theme.KindBuiltin {
		defaultName = "tint"
	}
	initial := filepath.Join(m.store.Dir, defaultName+".theme")
	m.prompt = newPrompt("export to path", "portable .theme, full role dump", initial, nil, func(path string) {
		path = termstyle.ExpandPath(path)
		data := termtheme.Marshal(m.cfg, m.base.ToShared(), termtheme.MarshalOptions{
			App:        "termtint",
			AppVersion: m.version,
			Roles:      termtheme.Roles(),
		})
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			m.message = err.Error()
			return
		}
		if _, err := theme.AtomicWrite(path, data, theme.WriteOptions{Backup: false}); err != nil {
			m.message = err.Error()
			return
		}
		m.message = "exported " + path
	})
}

// openImportPrompt asks for a path and loads it into the working theme.
func (m *Model) openImportPrompt() {
	m.prompt = newPrompt("import from path", "a .theme or theme.conf file", "", func(path string) bool {
		_, err := os.Stat(termstyle.ExpandPath(path))
		return err == nil
	}, func(path string) {
		data, err := os.ReadFile(termstyle.ExpandPath(path))
		if err != nil {
			m.message = err.Error()
			return
		}
		cfg, meta, err := termtheme.Unmarshal(data)
		if err != nil {
			m.message = err.Error()
			return
		}
		m.cfg = cfg
		m.dirty = true
		m.warning = ""
		if meta.App != "" {
			m.message = fmt.Sprintf("imported from %s %s (%d roles)", meta.App, meta.AppVersion, len(cfg.Specs))
		} else {
			m.message = fmt.Sprintf("imported %s (%d roles)", path, len(cfg.Specs))
		}
		if len(cfg.Warnings) > 0 {
			m.warning = strings.Join(cfg.Warnings, "; ")
		}
	})
}

// openNewPrompt asks for a name and saves the working theme as a new named
// theme (also the save path for builtin entries).
func (m *Model) openNewPrompt() {
	valid := func(name string) bool {
		if name == "" {
			return false
		}
		for _, builtin := range termstyle.BuiltinThemeNames() {
			if name == builtin {
				return false
			}
		}
		for _, app := range theme.LiveApps {
			if name == app || name == app+" (live)" {
				return false
			}
		}
		return true
	}
	m.prompt = newPrompt("new theme name", "a-z 0-9 . _ -", "", valid, func(name string) {
		path, res, err := m.store.SaveNamed(name, m.cfg, m.base.ToShared(), m.version)
		if err != nil {
			m.message = err.Error()
			return
		}
		if !res.Changed {
			m.message = "no changes to " + path
			return
		}
		// Switch the studio to the new named theme.
		entry, err := m.store.Load(name)
		if err != nil {
			m.message = err.Error()
			return
		}
		m.entry = entry
		m.origCfg = cloneConfig(m.cfg)
		m.dirty = false
		m.message = "saved " + path
	})
}
