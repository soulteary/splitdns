// Package resolver provides safe, file-level CRUD over macOS /etc/resolver
// entries: atomic writes, backups before overwrite, symlink protection, and
// strict path containment inside the resolver directory.
package resolver

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/soulteary/splitdns/internal/config"
)

// ErrNotManaged indicates a file exists but lacks the splitdns managed marker.
var ErrNotManaged = errors.New("file is not managed by splitdns")

// FlushStep records the outcome of a single cache-flush command.
type FlushStep struct {
	Name    string `json:"name"`
	Command string `json:"command"`
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
}

// ApplyResult reports the outcome of a write operation, separating the config
// write from cache flushing so partial success is visible.
type ApplyResult struct {
	Written    bool        `json:"written"`
	Path       string      `json:"path"`
	BackupPath string      `json:"backupPath,omitempty"`
	DryRun     bool        `json:"dryRun"`
	Content    string      `json:"content,omitempty"`
	FlushSteps []FlushStep `json:"flushSteps,omitempty"`
}

// resolvePath validates that name is a bare filename and returns the absolute
// path strictly contained within resolverDir after cleaning.
func resolvePath(resolverDir, name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("empty resolver name")
	}
	if strings.ContainsAny(name, "/\\") {
		return "", fmt.Errorf("resolver name %q must not contain path separators", name)
	}
	if name == "." || name == ".." || strings.Contains(name, "..") {
		return "", fmt.Errorf("resolver name %q is not allowed", name)
	}

	cleanDir := filepath.Clean(resolverDir)
	full := filepath.Clean(filepath.Join(cleanDir, name))

	if filepath.Dir(full) != cleanDir {
		return "", fmt.Errorf("resolved path %q escapes resolver directory %q", full, cleanDir)
	}
	return full, nil
}

// Path returns the safe absolute path for a resolver name.
func Path(resolverDir, name string) (string, error) {
	return resolvePath(resolverDir, name)
}

// Exists reports whether a regular resolver file exists for name. It returns an
// error if the path is a dangerous symlink or other irregular file.
func Exists(resolverDir, name string) (bool, error) {
	full, err := resolvePath(resolverDir, name)
	if err != nil {
		return false, err
	}
	info, err := os.Lstat(full)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return true, fmt.Errorf("%q is a symlink; refusing to treat it as a resolver file", full)
	}
	if !info.Mode().IsRegular() {
		return true, fmt.Errorf("%q is not a regular file", full)
	}
	return true, nil
}

// Read parses the resolver file for name and returns its Config and raw body.
func Read(resolverDir, name string) (*config.Config, string, error) {
	full, err := resolvePath(resolverDir, name)
	if err != nil {
		return nil, "", err
	}
	if err := assertRegular(full); err != nil {
		return nil, "", err
	}
	raw, err := os.ReadFile(full)
	if err != nil {
		return nil, "", err
	}
	cfg, err := config.Parse(string(raw))
	if err != nil {
		return nil, "", err
	}
	return cfg, string(raw), nil
}

// assertRegular verifies that a path, if it exists, is a regular file and not a
// symlink or other special file.
func assertRegular(full string) error {
	info, err := os.Lstat(full)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%q is a symlink; refusing to operate on it", full)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%q is not a regular file", full)
	}
	return nil
}
