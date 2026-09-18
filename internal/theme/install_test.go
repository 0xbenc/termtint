package theme

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xbenc/termtheme"
)

func TestInstallWritesPortableDump(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "passage", "theme.conf")
	cfg := termtheme.ThemeConfig{
		BaseName: "terminal",
		Codes:    map[termtheme.Role]string{termtheme.RolePrimary: "1;31"},
		Specs:    map[termtheme.Role]string{termtheme.RolePrimary: "bold red"},
	}

	result, err := Install(InstallOptions{
		App:       "passage",
		ThemeFile: target,
		Config:    cfg,
		Version:   "test",
	})
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if !result.Changed {
		t.Fatal("Changed = false, want true for a new file")
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(data), "primary") {
		t.Fatalf("dump = %q, want it to carry the primary role", data)
	}

	// The dump must re-parse into the same overrides: byte-droppable.
	got, _, err := termtheme.Unmarshal(data)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.Codes[termtheme.RolePrimary] != "1;31" {
		t.Fatalf("round-trip primary code = %q, want 1;31", got.Codes[termtheme.RolePrimary])
	}
}

func TestInstallBacksUpExistingConfig(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "ssherpa", "theme.conf")
	old := []byte("format = 1\nprimary = red\n")
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(target, old, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	cfg := termtheme.ThemeConfig{
		Codes: map[termtheme.Role]string{termtheme.RolePrimary: "34"},
		Specs: map[termtheme.Role]string{termtheme.RolePrimary: "blue"},
	}

	result, err := Install(InstallOptions{
		App:       "ssherpa",
		ThemeFile: target,
		Config:    cfg,
		Version:   "test",
	})
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if !result.Changed {
		t.Fatal("Changed = false, want true")
	}
	if result.BackupPath == "" {
		t.Fatal("BackupPath is empty, want a backup of the previous config")
	}
	backup, err := os.ReadFile(result.BackupPath)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", result.BackupPath, err)
	}
	if string(backup) != string(old) {
		t.Fatalf("backup = %q, want %q", backup, old)
	}
}

func TestInstallIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "dangit", "theme.conf")
	cfg := termtheme.ThemeConfig{
		Codes: map[termtheme.Role]string{termtheme.RoleBorder: "2"},
		Specs: map[termtheme.Role]string{termtheme.RoleBorder: "dim"},
	}
	options := InstallOptions{App: "dangit", ThemeFile: target, Config: cfg, Version: "test"}

	if _, err := Install(options); err != nil {
		t.Fatalf("first Install: %v", err)
	}
	result, err := Install(options)
	if err != nil {
		t.Fatalf("second Install: %v", err)
	}
	if result.Changed {
		t.Fatal("second install changed the file, want a no-op")
	}
	if result.BackupPath != "" {
		t.Fatalf("second install took a backup %q, want none", result.BackupPath)
	}
}

func TestInstallRequiresApp(t *testing.T) {
	if _, err := Install(InstallOptions{ThemeFile: filepath.Join(t.TempDir(), "x")}); err == nil {
		t.Fatal("Install with no app: want an error")
	}
}
