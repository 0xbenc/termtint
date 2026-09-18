package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/0xbenc/termtint/internal/termstyle"
	"github.com/0xbenc/termtint/internal/theme"
)

// SchemaVersion is the state file schema.
const SchemaVersion = 1

// State is termtint's small persisted state.
type State struct {
	SchemaVersion     int    `json:"schema_version"`
	LastIntroVersion  string `json:"last_intro_version,omitempty"`
	StudioHintVersion string `json:"studio_hint_version,omitempty"`
}

// ResolveDir picks the state directory: $TERTINT_STATE_DIR, else the
// platform config/state home under "termtint".
func ResolveDir(env []string) (string, error) {
	values := termstyle.EnvMap(env)
	if dir := strings.TrimSpace(values["TERTINT_STATE_DIR"]); dir != "" {
		return termstyle.ExpandPath(filepath.Clean(dir)), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home: %w", err)
	}
	if runtime.GOOS == "darwin" {
		return filepath.Join(home, "Library", "Application Support", "termtint"), nil
	}
	if xdg := strings.TrimSpace(values["XDG_STATE_HOME"]); xdg != "" {
		return termstyle.ExpandPath(filepath.Join(xdg, "termtint")), nil
	}
	return filepath.Join(home, ".local", "state", "termtint"), nil
}

// Load reads the state file fail-open: any problem yields an empty state.
func Load(dir string) State {
	st := State{SchemaVersion: SchemaVersion}
	data, err := os.ReadFile(filepath.Join(dir, "state.json"))
	if err != nil {
		return st
	}
	if err := json.Unmarshal(data, &st); err != nil {
		return State{SchemaVersion: SchemaVersion}
	}
	if st.SchemaVersion == 0 {
		st.SchemaVersion = SchemaVersion
	}
	return st
}

// Save writes the state atomically with a backup.
func (s State) Save(dir string) error {
	if s.SchemaVersion == 0 {
		s.SchemaVersion = SchemaVersion
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("encode state: %w", err)
	}
	data = append(data, '\n')
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create state directory %s: %w", dir, err)
	}
	path := filepath.Join(dir, "state.json")
	_, err = theme.AtomicWrite(path, data, theme.WriteOptions{Mode: 0o600, Backup: false})
	if err != nil {
		return fmt.Errorf("write state %s: %w", path, err)
	}
	return nil
}

// SetLastIntroVersion records the version the intro last ran for.
func (s *State) SetLastIntroVersion(version string) {
	s.LastIntroVersion = version
}

// SetStudioHintVersion records the version the getting-started overlay last
// ran for.
func (s *State) SetStudioHintVersion(version string) {
	s.StudioHintVersion = version
}

// ErrNotSaved is returned when the state directory could not be resolved.
var ErrNotSaved = errors.New("state directory could not be resolved")
