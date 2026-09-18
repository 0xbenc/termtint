package theme

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/0xbenc/termtheme"
	"github.com/0xbenc/termtint/internal/termstyle"
)

// Kind classifies a theme set entry.
type Kind string

const (
	KindBuiltin Kind = "builtin"
	KindNamed   Kind = "named"
	KindLive    Kind = "live"
)

// LiveApps are the family apps termtint reads and installs to, in display
// order.
var LiveApps = []string{"passage", "ssherpa", "dangit", "bitty"}

// Entry is one row of the theme set.
type Entry struct {
	Name     string
	Kind     Kind
	App      string // live entries: the app name
	Path     string // named/live: the file; builtin: ""
	Exists   bool
	BaseName string
	Config   termtheme.ThemeConfig
	Meta     termtheme.Meta
	Warnings []string
}

// Store is termtint's theme set: the builtin palettes, the named themes in the
// config directory, and the live theme.conf files of the family apps.
type Store struct {
	Dir string
	Env []string
}

var namePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// IsLiveApp reports whether name is a family app termtint manages.
func IsLiveApp(name string) bool { return isLiveApp(name) }

func isLiveApp(name string) bool {
	for _, app := range LiveApps {
		if name == app {
			return true
		}
	}
	return false
}

// NewStore resolves the named-theme directory: $XDG_CONFIG_HOME/termtint/themes
// (or ~/.config/termtint/themes).
func NewStore(env []string) (Store, error) {
	values := termstyle.EnvMap(env)
	configDir := strings.TrimSpace(values["XDG_CONFIG_HOME"])
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return Store{}, fmt.Errorf("resolve home: %w", err)
		}
		configDir = filepath.Join(home, ".config")
	}
	return Store{Dir: filepath.Join(termstyle.ExpandPath(configDir), "termtint", "themes"), Env: env}, nil
}

// Entries lists the theme set: builtins, then named themes (sorted), then the
// live app configs. Missing live configs are still listed (Exists=false) so
// the set is a stable map of the family.
func (s Store) Entries() ([]Entry, error) {
	out := make([]Entry, 0, len(termstyle.BuiltinThemeNames())+len(LiveApps))
	for _, name := range termstyle.BuiltinThemeNames() {
		out = append(out, Entry{Name: name, Kind: KindBuiltin})
	}

	if entries, err := os.ReadDir(s.Dir); err == nil {
		var named []string
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".theme") {
				continue
			}
			named = append(named, strings.TrimSuffix(e.Name(), ".theme"))
		}
		sort.Strings(named)
		for _, name := range named {
			entry, err := s.Load(name)
			if err != nil {
				entry = Entry{Name: name, Kind: KindNamed, Path: s.pathFor(name), Warnings: []string{err.Error()}}
			}
			out = append(out, entry)
		}
	}

	for _, app := range LiveApps {
		entry, err := s.loadLive(app)
		if err != nil {
			entry.Warnings = []string{err.Error()}
		}
		out = append(out, entry)
	}
	return out, nil
}

// loadLive reads one family app's live theme.conf (tolerating a missing file).
func (s Store) loadLive(app string) (Entry, error) {
	path, _ := termtheme.ResolveThemeFile(app, "", s.Env, false)
	entry := Entry{Name: app + " (live)", Kind: KindLive, App: app, Path: path}
	if path == "" {
		return entry, fmt.Errorf("no theme config path for %s", app)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return entry, fmt.Errorf("read %s: %w", path, err)
	}
	cfg, meta, err := termtheme.Unmarshal(data)
	if err != nil {
		return entry, err
	}
	entry.Exists = true
	entry.Config = cfg
	entry.Meta = meta
	entry.BaseName = cfg.BaseName
	entry.Warnings = meta.Warnings
	return entry, nil
}

// Load resolves one entry by display name.
func (s Store) Load(name string) (Entry, error) {
	for _, builtin := range termstyle.BuiltinThemeNames() {
		if name == builtin {
			return Entry{Name: builtin, Kind: KindBuiltin, BaseName: builtin}, nil
		}
	}
	if isLiveApp(strings.TrimSuffix(name, " (live)")) {
		return s.loadLive(strings.TrimSuffix(name, " (live)"))
	}
	if !namePattern.MatchString(name) {
		return Entry{}, fmt.Errorf("invalid theme name %q", name)
	}
	path := s.pathFor(name)
	data, err := os.ReadFile(path)
	if err != nil {
		return Entry{}, fmt.Errorf("read theme %s: %w", path, err)
	}
	cfg, meta, err := termtheme.Unmarshal(data)
	if err != nil {
		return Entry{}, err
	}
	return Entry{
		Name: name, Kind: KindNamed, Path: path, Exists: true,
		BaseName: cfg.BaseName, Config: cfg, Meta: meta, Warnings: meta.Warnings,
	}, nil
}

// SaveNamed writes a named theme as a portable .theme (full role dump, so the
// file is self-contained and interchanges with any sibling app).
func (s Store) SaveNamed(name string, cfg termtheme.ThemeConfig, base termtheme.Theme, version string) (string, WriteResult, error) {
	if !namePattern.MatchString(name) {
		return "", WriteResult{}, fmt.Errorf("invalid theme name %q", name)
	}
	path := s.pathFor(name)
	data := termtheme.Marshal(cfg, base, termtheme.MarshalOptions{
		App:        "termtint",
		AppVersion: version,
		Roles:      termtheme.Roles(),
	})
	res, err := AtomicWrite(path, data, WriteOptions{Backup: true, BackupPrefix: "termtint"})
	if err != nil {
		return "", res, err
	}
	return path, res, nil
}

func (s Store) pathFor(name string) string {
	return filepath.Join(s.Dir, name+".theme")
}
