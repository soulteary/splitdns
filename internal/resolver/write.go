package resolver

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// WriteOptions controls a write operation.
type WriteOptions struct {
	// Force allows overwriting an existing file (after backing it up).
	Force bool
	// BackupDir is where a pre-overwrite backup is written. Empty means the
	// system temp dir.
	BackupDir string
	// DryRun computes the plan without touching the filesystem.
	DryRun bool
}

// Write writes content to the resolver file for name, atomically. On overwrite
// it requires Force and takes a backup first. It returns the resolved path and
// the backup path (if any).
func Write(resolverDir, name, content string, opts WriteOptions) (path string, backup string, err error) {
	full, err := resolvePath(resolverDir, name)
	if err != nil {
		return "", "", err
	}

	existed := false
	if info, lerr := os.Lstat(full); lerr == nil {
		existed = true
		if info.Mode()&os.ModeSymlink != 0 {
			return full, "", fmt.Errorf("%q is a symlink; refusing to overwrite", full)
		}
		if !info.Mode().IsRegular() {
			return full, "", fmt.Errorf("%q is not a regular file; refusing to overwrite", full)
		}
		if !opts.Force {
			return full, "", fmt.Errorf("%q already exists; use --force to overwrite", full)
		}
	} else if !os.IsNotExist(lerr) {
		return full, "", lerr
	}

	if opts.DryRun {
		return full, "", nil
	}

	if existed && opts.Force {
		backup, err = backupFile(full, opts.BackupDir)
		if err != nil {
			return full, "", fmt.Errorf("backup before overwrite failed: %w", err)
		}
	}

	if err := atomicWrite(full, content); err != nil {
		return full, backup, err
	}
	return full, backup, nil
}

// atomicWrite writes content to a same-directory temp file, syncs, chmods to
// 0644 and renames into place, cleaning up the temp file on failure.
func atomicWrite(full, content string) error {
	dir := filepath.Dir(full)
	tmp, err := os.CreateTemp(dir, ".splitdns-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()
	cleanup := func() {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
	}

	if _, err := tmp.WriteString(content); err != nil {
		cleanup()
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		cleanup()
		return fmt.Errorf("sync temp file: %w", err)
	}
	if err := tmp.Chmod(0o644); err != nil {
		cleanup()
		return fmt.Errorf("chmod temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Rename(tmpName, full); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("rename temp file into place: %w", err)
	}
	return nil
}

// backupFile copies src to a timestamped file in backupDir (or the system temp
// dir when empty) and returns the backup path.
func backupFile(src, backupDir string) (string, error) {
	if backupDir == "" {
		backupDir = os.TempDir()
	}
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return "", err
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return "", err
	}
	base := filepath.Base(src)
	stamp := time.Now().Format("20060102-150405")
	dest := filepath.Join(backupDir, fmt.Sprintf("splitdns-%s-%s.bak", base, stamp))
	if err := os.WriteFile(dest, data, 0o644); err != nil {
		return "", err
	}
	return dest, nil
}

// Remove deletes the resolver file for name after verifying it is a regular
// file strictly inside the resolver directory. DryRun returns the target path
// without deleting.
func Remove(resolverDir, name string, dryRun bool) (string, error) {
	full, err := resolvePath(resolverDir, name)
	if err != nil {
		return "", err
	}
	info, err := os.Lstat(full)
	if err != nil {
		return full, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return full, fmt.Errorf("%q is a symlink; refusing to delete", full)
	}
	if !info.Mode().IsRegular() {
		return full, fmt.Errorf("%q is not a regular file; refusing to delete", full)
	}
	if dryRun {
		return full, nil
	}
	if err := os.Remove(full); err != nil {
		return full, err
	}
	return full, nil
}
