package theme

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAtomicWriteNewFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "theme.conf")
	data := []byte("format = 1\n")

	result, err := AtomicWrite(path, data, WriteOptions{Backup: true})
	if err != nil {
		t.Fatalf("AtomicWrite: %v", err)
	}
	if !result.Changed {
		t.Fatal("Changed = false, want true for a new file")
	}
	if result.BackupPath != "" {
		t.Fatalf("BackupPath = %q, want empty for a new file", result.BackupPath)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != string(data) {
		t.Fatalf("content = %q, want %q", got, data)
	}
}

func TestAtomicWriteBacksUpExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "theme.conf")
	old := []byte("format = 1\nprimary = red\n")
	if err := os.WriteFile(path, old, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	result, err := AtomicWrite(path, []byte("format = 1\nprimary = blue\n"), WriteOptions{Backup: true, BackupPrefix: "termtint"})
	if err != nil {
		t.Fatalf("AtomicWrite: %v", err)
	}
	if !result.Changed {
		t.Fatal("Changed = false, want true")
	}
	if result.BackupPath == "" {
		t.Fatal("BackupPath is empty, want a backup of the previous content")
	}
	backup, err := os.ReadFile(result.BackupPath)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", result.BackupPath, err)
	}
	if string(backup) != string(old) {
		t.Fatalf("backup = %q, want %q", backup, old)
	}
}

func TestAtomicWriteUnchangedIsNoop(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "theme.conf")
	data := []byte("format = 1\n")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	result, err := AtomicWrite(path, data, WriteOptions{Backup: true})
	if err != nil {
		t.Fatalf("AtomicWrite: %v", err)
	}
	if result.Changed {
		t.Fatal("Changed = true, want false for identical content")
	}
	if result.BackupPath != "" {
		t.Fatalf("BackupPath = %q, want empty when nothing changed", result.BackupPath)
	}
}
