package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/creack/pty"
)

// buildE2EBinary compiles the real binary once per test run so the PTY test
// exercises the exact program a user would run (including the TTY gate).
var e2eOnce struct {
	bin string
	err error
}

func buildE2EBinary(t *testing.T) string {
	t.Helper()
	if e2eOnce.bin == "" {
		root, err := filepath.Abs(filepath.Join("..", ".."))
		if err != nil {
			t.Fatalf("module root: %v", err)
		}
		e2eOnce.bin = filepath.Join(t.TempDir(), "termtint-e2e")
		cmd := exec.Command("go", "build", "-o", e2eOnce.bin, "./cmd/termtint")
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			e2eOnce.err = err
			t.Fatalf("go build: %v\n%s", err, out)
		}
	} else if e2eOnce.err != nil {
		t.Fatalf("go build: %v", e2eOnce.err)
	}
	return e2eOnce.bin
}

// TestPTYInteractiveStudio is the end-to-end proof that the studio renders in
// a real terminal: it drives the built binary through a PTY with a real
// winsize, opens an app's live theme, and walks esc -> list -> esc -> quit.
func TestPTYInteractiveStudio(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("PTY test is not supported on Windows")
	}
	bin := buildE2EBinary(t)

	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	t.Setenv("TERTINT_STATE_DIR", t.TempDir())

	seed := filepath.Join(xdg, "passage", "theme.conf")
	if err := os.MkdirAll(filepath.Dir(seed), 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(seed, []byte("theme = terminal\nprimary = bold red\nselected_bar = 48;2;50;40;60\n"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cmd := exec.Command(bin, "passage", "--no-intro")
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")
	f, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: 30, Cols: 100})
	if err != nil {
		t.Fatalf("pty start: %v", err)
	}
	defer f.Close()

	var out bytes.Buffer
	done := make(chan struct{})
	go func() {
		ioCopy(&out, f)
		close(done)
	}()

	// Let the studio paint (the first-run welcome overlay is up in a fresh
	// state dir), dismiss it with any key, cycle the contrast to white-out
	// with c, then walk: esc back to the list, esc to quit.
	time.Sleep(2 * time.Second)
	f.Write([]byte{'h'})
	time.Sleep(500 * time.Millisecond)
	f.Write([]byte{'c'})
	time.Sleep(500 * time.Millisecond)
	f.Write([]byte{0x1b})
	time.Sleep(500 * time.Millisecond)
	f.Write([]byte{0x1b})

	finished := make(chan error, 1)
	go func() { finished <- cmd.Wait() }()
	select {
	case err := <-finished:
		if err != nil {
			t.Fatalf("program exited with error: %v", err)
		}
	case <-time.After(10 * time.Second):
		cmd.Process.Kill()
		t.Fatal("program did not quit within 10s of the final esc")
	}
	<-done

	text := out.String()
	for _, want := range []string{
		"TERTINT — welcome",   // first-run getting-started overlay painted
		"passage (live)",      // studio opened the app's live theme
		"selected_bar",        // role list painted
		"SWATCHES",            // preview sheet painted
		"contrast: white-out", // c cycled the contrast and the feedback is on screen
		// The preview whites out too (the renderer reorders 1;37 to 37;1 and
		// merges the same-color border into the span).
		"\x1b[37;1m│ success line",
		"TERTINT — themes", // esc reached the list screen
	} {
		if !bytes.Contains(out.Bytes(), []byte(want)) {
			t.Fatalf("PTY capture = %d bytes, want it to contain %q", len(text), want)
		}
	}
}

func ioCopy(dst *bytes.Buffer, src interface{ Read([]byte) (int, error) }) {
	buf := make([]byte, 32*1024)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			dst.Write(buf[:n])
		}
		if err != nil {
			return
		}
	}
}
