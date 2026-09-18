package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// cliEnv isolates the process environment (termtheme's default path lookup)
// and returns the matching env slice for the store.
func cliEnv(t *testing.T) (string, []string) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("TERTINT_STATE_DIR", filepath.Join(t.TempDir(), "state"))
	return dir, []string{"XDG_CONFIG_HOME=" + dir}
}

func seedLive(t *testing.T, xdg, app, content string) {
	t.Helper()
	path := filepath.Join(xdg, app, "theme.conf")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func TestRunVersion(t *testing.T) {
	var out, errw bytes.Buffer
	if code := run([]string{"version"}, &out, &errw, nil); code != 0 {
		t.Fatalf("exit = %d, want 0 (stderr: %s)", code, errw.String())
	}
	if !strings.Contains(out.String(), "termtint ") {
		t.Fatalf("out = %q, want the version line", out.String())
	}
}

func TestRunListShowsTheFamily(t *testing.T) {
	xdg, env := cliEnv(t)
	seedLive(t, xdg, "passage", "theme = terminal\nprimary = bold red\n")

	var out, errw bytes.Buffer
	if code := run([]string{"list"}, &out, &errw, env); code != 0 {
		t.Fatalf("exit = %d, want 0 (stderr: %s)", code, errw.String())
	}
	text := out.String()
	for _, want := range []string{"terminal", "tint", "passage (live)", "bitty (live)"} {
		if !strings.Contains(text, want) {
			t.Fatalf("list = %q, want %q", text, want)
		}
	}
}

func TestRunShowPrintsRoleTable(t *testing.T) {
	xdg, env := cliEnv(t)
	seedLive(t, xdg, "passage", "theme = terminal\nprimary = bold red\n")

	var out, errw bytes.Buffer
	if code := run([]string{"show", "passage"}, &out, &errw, env); code != 0 {
		t.Fatalf("exit = %d, want 0 (stderr: %s)", code, errw.String())
	}
	text := out.String()
	if !strings.Contains(text, "primary") || !strings.Contains(text, "bold red") {
		t.Fatalf("show = %q, want the primary override", text)
	}
}

func TestRunExportEmitsPortableTheme(t *testing.T) {
	xdg, env := cliEnv(t)
	seedLive(t, xdg, "ssherpa", "theme = terminal\nborder = dim\n")

	var out, errw bytes.Buffer
	if code := run([]string{"export", "ssherpa"}, &out, &errw, env); code != 0 {
		t.Fatalf("exit = %d, want 0 (stderr: %s)", code, errw.String())
	}
	text := out.String()
	if !strings.Contains(text, "format = 1") || !strings.Contains(text, "border = dim") {
		t.Fatalf("export = %q, want a portable dump with the override", text)
	}
}

func TestRunInstallCopiesThemeBetweenApps(t *testing.T) {
	xdg, env := cliEnv(t)
	seedLive(t, xdg, "passage", "theme = terminal\nprimary = bold red\n")

	var out, errw bytes.Buffer
	if code := run([]string{"install", "ssherpa", "--theme", "passage"}, &out, &errw, env); code != 0 {
		t.Fatalf("exit = %d, want 0 (stderr: %s)", code, errw.String())
	}
	data, err := os.ReadFile(filepath.Join(xdg, "ssherpa", "theme.conf"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(data), "primary = bold red") {
		t.Fatalf("installed config = %q, want the source override", data)
	}
}

func TestRunImportSavesNamedTheme(t *testing.T) {
	xdg, env := cliEnv(t)
	source := filepath.Join(t.TempDir(), "dump.theme")
	if err := os.WriteFile(source, []byte("format = 1\nsource = termtint test\nprimary = bold cyan\n"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	var out, errw bytes.Buffer
	if code := run([]string{"import", source, "--name", "night"}, &out, &errw, env); code != 0 {
		t.Fatalf("exit = %d, want 0 (stderr: %s)", code, errw.String())
	}
	saved, err := os.ReadFile(filepath.Join(xdg, "termtint", "themes", "night.theme"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(saved), "primary = bold cyan") {
		t.Fatalf("saved theme = %q, want the imported override", saved)
	}
}

func TestRunUnknownVerbIsUsage(t *testing.T) {
	var out, errw bytes.Buffer
	if code := run([]string{"frobnicate"}, &out, &errw, nil); code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
	if !strings.Contains(errw.String(), "unknown verb") {
		t.Fatalf("stderr = %q, want the unknown-verb error", errw.String())
	}
}

func TestRunNoArgsWithoutTTYIsUsage(t *testing.T) {
	var out, errw bytes.Buffer
	if code := run([]string{}, &out, &errw, nil); code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
	if !strings.Contains(errw.String(), "termtint") {
		t.Fatalf("stderr = %q, want the usage", errw.String())
	}
}

func TestRunEntryNameOpensStudio(t *testing.T) {
	xdg, env := cliEnv(t)
	seedLive(t, xdg, "passage", "theme = terminal\nprimary = bold red\n")

	// A bare entry name dispatches to the studio; without a TTY it must
	// report the TTY requirement, not "unknown verb".
	var out, errw bytes.Buffer
	if code := run([]string{"passage"}, &out, &errw, env); code != 2 {
		t.Fatalf("exit = %d, want 2 (stderr: %s)", code, errw.String())
	}
	if !strings.Contains(errw.String(), "needs a TTY") {
		t.Fatalf("stderr = %q, want the TTY hint", errw.String())
	}

	// Same for a family app that has no theme.conf yet.
	errw.Reset()
	if code := run([]string{"dangit"}, &out, &errw, env); code != 2 {
		t.Fatalf("exit = %d, want 2 (stderr: %s)", code, errw.String())
	}
	if !strings.Contains(errw.String(), "needs a TTY") {
		t.Fatalf("stderr = %q, want the TTY hint for an unconfigured app", errw.String())
	}
}

func TestRunFlagInAnyPosition(t *testing.T) {
	xdg, env := cliEnv(t)
	seedLive(t, xdg, "passage", "theme = terminal\nprimary = bold red\n")

	var out, errw bytes.Buffer
	// The value flag trails the positional app name.
	if code := run([]string{"install", "ssherpa", "--theme", "passage"}, &out, &errw, env); code != 0 {
		t.Fatalf("exit = %d, want 0 (stderr: %s)", code, errw.String())
	}
	// --no-color before the verb.
	out.Reset()
	if code := run([]string{"--no-color", "show", "passage"}, &out, &errw, env); code != 0 {
		t.Fatalf("exit = %d, want 0 (stderr: %s)", code, errw.String())
	}
	if strings.Contains(out.String(), "\x1b[") {
		t.Fatal("show --no-color still emits ANSI escapes")
	}
}
