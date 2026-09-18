package theme

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xbenc/termtheme"
)

// testXDG points both the process environment (which termtheme's default path
// resolution reads) and the env slice at a fresh temp config dir.
func testXDG(t *testing.T) (string, []string) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	return dir, []string{"XDG_CONFIG_HOME=" + dir}
}

func TestStoreEntriesListsTheFamily(t *testing.T) {
	_, env := testXDG(t)
	store, err := NewStore(env)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	entries, err := store.Entries()
	if err != nil {
		t.Fatalf("Entries: %v", err)
	}
	var names []string
	for _, entry := range entries {
		names = append(names, entry.Name)
	}
	text := strings.Join(names, ", ")
	for _, want := range []string{"terminal", "tint", "passage (live)", "ssherpa (live)", "dangit (live)", "bitty (live)"} {
		if !strings.Contains(text, want) {
			t.Fatalf("entries = %q, want %q in the set", text, want)
		}
	}
}

func TestStoreLoadLiveParsesOverrides(t *testing.T) {
	dir, env := testXDG(t)
	path := filepath.Join(dir, "passage", "theme.conf")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte("theme = terminal\nprimary = bold red\n"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	store, err := NewStore(env)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	entry, err := store.Load("passage")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if entry.Kind != KindLive || !entry.Exists {
		t.Fatalf("entry = kind %q exists %v, want a live, existing entry", entry.Kind, entry.Exists)
	}
	if got := entry.Config.Specs[termtheme.RolePrimary]; got != "bold red" {
		t.Fatalf("primary spec = %q, want %q", got, "bold red")
	}
}

func TestStoreLoadUnknownNameFails(t *testing.T) {
	_, env := testXDG(t)
	store, err := NewStore(env)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	if _, err := store.Load("nope-nope"); err == nil {
		t.Fatal("Load(nope-nope) succeeded, want an error")
	}
}

func TestStoreSaveNamedRoundTrip(t *testing.T) {
	_, env := testXDG(t)
	store, err := NewStore(env)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	cfg := termtheme.ThemeConfig{
		BaseName: "tint",
		Codes:    map[termtheme.Role]string{termtheme.RolePrimary: "1;31"},
		Specs:    map[termtheme.Role]string{termtheme.RolePrimary: "bold red"},
	}
	path, result, err := store.SaveNamed("night", cfg, termtheme.Theme{}, "test")
	if err != nil {
		t.Fatalf("SaveNamed: %v", err)
	}
	if !result.Changed {
		t.Fatal("Changed = false, want true")
	}
	if !strings.HasSuffix(path, "night.theme") {
		t.Fatalf("path = %q, want it to end in night.theme", path)
	}

	entry, err := store.Load("night")
	if err != nil {
		t.Fatalf("Load(night): %v", err)
	}
	if entry.Kind != KindNamed {
		t.Fatalf("kind = %q, want named", entry.Kind)
	}
	if got := entry.Config.Specs[termtheme.RolePrimary]; got != "bold red" {
		t.Fatalf("round-trip primary = %q, want %q", got, "bold red")
	}
}
