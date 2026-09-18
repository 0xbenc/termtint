package theme

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// WriteOptions controls AtomicWrite.
type WriteOptions struct {
	Backup       bool
	BackupPrefix string
	Mode         os.FileMode
	Now          func() time.Time
}

// WriteResult reports what AtomicWrite did.
type WriteResult struct {
	Path       string
	Changed    bool
	BackupPath string
}

// AtomicWrite writes data to path via temp file + rename + directory sync,
// taking a backup of the existing content first when opts.Backup is set. The
// same pattern the sibling apps use for every config write.
func AtomicWrite(path string, data []byte, opts WriteOptions) (WriteResult, error) {
	result := WriteResult{Path: filepath.Clean(path)}

	old, stat, exists, err := readExisting(result.Path)
	if err != nil {
		return result, err
	}
	if bytes.Equal(old, data) {
		return result, nil
	}
	result.Changed = true

	dir := filepath.Dir(result.Path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return result, fmt.Errorf("create parent directory %s: %w", dir, err)
	}

	mode := os.FileMode(0o600)
	if exists {
		mode = stat.Mode().Perm()
	}
	if opts.Mode != 0 {
		mode = opts.Mode.Perm()
	}
	if exists && opts.Backup {
		backup, err := createBackup(result.Path, old, mode, opts)
		if err != nil {
			return result, err
		}
		result.BackupPath = backup
	}

	if err := writeTempRename(result.Path, data, mode); err != nil {
		return result, err
	}
	return result, nil
}

func readExisting(path string) ([]byte, os.FileInfo, bool, error) {
	stat, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil, false, nil
		}
		return nil, nil, false, fmt.Errorf("stat %s: %w", path, err)
	}
	if stat.IsDir() {
		return nil, nil, false, fmt.Errorf("%s is a directory", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, false, fmt.Errorf("read %s: %w", path, err)
	}
	return data, stat, true, nil
}

func createBackup(path string, data []byte, mode os.FileMode, opts WriteOptions) (string, error) {
	name, err := backupPath(path, opts)
	if err != nil {
		return "", err
	}
	file, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return "", fmt.Errorf("create backup %s: %w", name, err)
	}
	if _, err := file.Write(data); err != nil {
		file.Close()
		return "", fmt.Errorf("write backup %s: %w", name, err)
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return "", fmt.Errorf("sync backup %s: %w", name, err)
	}
	if err := file.Close(); err != nil {
		return "", fmt.Errorf("close backup %s: %w", name, err)
	}
	return name, nil
}

func backupPath(path string, opts WriteOptions) (string, error) {
	now := time.Now
	if opts.Now != nil {
		now = opts.Now
	}
	prefix := opts.BackupPrefix
	if prefix == "" {
		prefix = "termtint-backup"
	}
	stamp := now().UTC().Format("20060102T150405Z")
	base := fmt.Sprintf("%s.%s.%s", path, prefix, stamp)
	for i := 0; i < 1000; i++ {
		candidate := base
		if i > 0 {
			candidate = fmt.Sprintf("%s.%d", base, i)
		}
		_, err := os.Stat(candidate)
		if errors.Is(err, os.ErrNotExist) {
			return candidate, nil
		}
		if err != nil {
			return "", fmt.Errorf("stat backup candidate %s: %w", candidate, err)
		}
	}
	return "", fmt.Errorf("could not choose unused backup name for %s", path)
}

func writeTempRename(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".termtint-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file in %s: %w", dir, err)
	}
	tmpName := tmp.Name()
	removeTmp := true
	defer func() {
		if removeTmp {
			_ = os.Remove(tmpName)
		}
	}()

	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("chmod temp file %s: %w", tmpName, err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temp file %s: %w", tmpName, err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync temp file %s: %w", tmpName, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file %s: %w", tmpName, err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("rename temp file into %s: %w", path, err)
	}
	removeTmp = false
	if err := syncDir(dir); err != nil {
		return fmt.Errorf("sync parent directory %s: %w", dir, err)
	}
	return nil
}

func syncDir(dir string) error {
	file, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer file.Close()
	if err := file.Sync(); err != nil && !isUnsupportedSync(err) {
		return err
	}
	return nil
}

func isUnsupportedSync(err error) bool {
	return errors.Is(err, syscall.EINVAL) ||
		errors.Is(err, syscall.ENOTSUP) ||
		errors.Is(err, syscall.EPERM) ||
		strings.Contains(strings.ToLower(err.Error()), "invalid argument")
}
