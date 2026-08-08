package resolver

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/soulteary/splitdns/internal/system"
)

func TestWriteCreatesRegularFile(t *testing.T) {
	dir := t.TempDir()
	path, backup, err := Write(dir, "lab.dev", "domain lab.dev\nnameserver 127.0.0.1\n", WriteOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if backup != "" {
		t.Errorf("unexpected backup on fresh write: %q", backup)
	}
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o644 {
		t.Errorf("perm = %o, want 0644", info.Mode().Perm())
	}
}

func TestWriteRefusesOverwriteWithoutForce(t *testing.T) {
	dir := t.TempDir()
	if _, _, err := Write(dir, "lab.dev", "a\n", WriteOptions{}); err != nil {
		t.Fatal(err)
	}
	_, _, err := Write(dir, "lab.dev", "b\n", WriteOptions{})
	if err == nil {
		t.Fatal("expected error overwriting without force")
	}
	data, _ := os.ReadFile(filepath.Join(dir, "lab.dev"))
	if string(data) != "a\n" {
		t.Errorf("file was overwritten despite no force: %q", data)
	}
}

func TestWriteForceTakesBackup(t *testing.T) {
	dir := t.TempDir()
	backupDir := t.TempDir()
	if _, _, err := Write(dir, "lab.dev", "old\n", WriteOptions{}); err != nil {
		t.Fatal(err)
	}
	_, backup, err := Write(dir, "lab.dev", "new\n", WriteOptions{Force: true, BackupDir: backupDir})
	if err != nil {
		t.Fatal(err)
	}
	if backup == "" {
		t.Fatal("expected a backup path")
	}
	bdata, _ := os.ReadFile(backup)
	if string(bdata) != "old\n" {
		t.Errorf("backup content = %q, want old", bdata)
	}
	ndata, _ := os.ReadFile(filepath.Join(dir, "lab.dev"))
	if string(ndata) != "new\n" {
		t.Errorf("current content = %q, want new", ndata)
	}
}

func TestWriteDryRunNoFilesystemChange(t *testing.T) {
	dir := t.TempDir()
	path, _, err := Write(dir, "lab.dev", "domain lab.dev\n", WriteOptions{DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("dry-run wrote a file at %s", path)
	}
	// Ensure no temp files leaked either.
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Errorf("dry-run left files in dir: %v", entries)
	}
}

func TestPathTraversalRejected(t *testing.T) {
	dir := t.TempDir()
	for _, bad := range []string{"../evil", "sub/evil", "..", "a/../b"} {
		if _, err := Path(dir, bad); err == nil {
			t.Errorf("expected rejection for %q", bad)
		}
	}
}

func TestSymlinkProtection(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(t.TempDir(), "outside")
	if err := os.WriteFile(target, []byte("secret\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "lab.dev")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	if _, _, err := Write(dir, "lab.dev", "x\n", WriteOptions{Force: true}); err == nil {
		t.Error("Write should refuse to overwrite a symlink")
	}
	if _, err := Remove(dir, "lab.dev", false); err == nil {
		t.Error("Remove should refuse to delete a symlink")
	}
	if _, _, err := Read(dir, "lab.dev"); err == nil {
		t.Error("Read should refuse a symlink")
	}
	// The target must be untouched.
	data, _ := os.ReadFile(target)
	if string(data) != "secret\n" {
		t.Errorf("symlink target was modified: %q", data)
	}
}

func TestRemoveOnlyTarget(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, dir, "a.dev", "a\n")
	mustWrite(t, dir, "b.dev", "b\n")
	if _, err := Remove(dir, "a.dev", false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "a.dev")); !os.IsNotExist(err) {
		t.Error("a.dev should be deleted")
	}
	if _, err := os.Stat(filepath.Join(dir, "b.dev")); err != nil {
		t.Error("b.dev should be untouched")
	}
}

func TestRemoveDryRun(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, dir, "a.dev", "a\n")
	if _, err := Remove(dir, "a.dev", true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "a.dev")); err != nil {
		t.Error("dry-run remove should not delete the file")
	}
}

func TestList(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, dir, "a.dev", "# Managed by splitdns\ndomain a.dev\nnameserver 127.0.0.1\nport 53\n")
	mustWrite(t, dir, "b.dev", "domain b.dev\nnameserver 10.0.0.1\n")
	entries, err := List(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Name != "a.dev" || !entries[0].Managed {
		t.Errorf("entry 0 = %+v", entries[0])
	}
	if entries[1].Name != "b.dev" || entries[1].Managed {
		t.Errorf("entry 1 = %+v", entries[1])
	}
}

func TestFlushPartialSuccess(t *testing.T) {
	fr := system.NewFakeRunner()
	fr.SetResponse("dscacheutil", system.CommandResult{ExitCode: 0})
	fr.SetResponse("killall", system.CommandResult{ExitCode: 1, Stderr: "No matching processes belonging to you were found"})
	env := system.Env{Runner: fr, GOOS: "darwin"}

	steps := Flush(env, false)
	if len(steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(steps))
	}
	if !steps[0].OK {
		t.Errorf("flush-cache should succeed: %+v", steps[0])
	}
	if !steps[1].OK {
		t.Errorf("killall no-process should be treated as OK: %+v", steps[1])
	}
}

func TestFlushRealFailure(t *testing.T) {
	fr := system.NewFakeRunner()
	fr.SetResponse("dscacheutil", system.CommandResult{ExitCode: 1, Stderr: "permission denied"})
	fr.SetResponse("killall", system.CommandResult{ExitCode: 0})
	env := system.Env{Runner: fr, GOOS: "darwin"}

	steps := Flush(env, false)
	if steps[0].OK {
		t.Error("flush-cache failure should be reported")
	}
	if !strings.Contains(steps[0].Message, "permission denied") {
		t.Errorf("message = %q", steps[0].Message)
	}
}

func TestFlushDryRun(t *testing.T) {
	fr := system.NewFakeRunner()
	env := system.Env{Runner: fr, GOOS: "darwin"}
	steps := Flush(env, true)
	if len(fr.Calls) != 0 {
		t.Errorf("dry-run should not execute any command, got %d calls", len(fr.Calls))
	}
	for _, s := range steps {
		if !s.OK {
			t.Errorf("dry-run step should be OK: %+v", s)
		}
	}
}

func TestExists(t *testing.T) {
	dir := t.TempDir()

	// Nonexistent file: exists=false, no error.
	ok, err := Exists(dir, "absent.dev")
	if ok || err != nil {
		t.Errorf("absent: ok=%v err=%v", ok, err)
	}

	// Regular file: exists=true, no error.
	mustWrite(t, dir, "lab.dev", "x\n")
	ok, err = Exists(dir, "lab.dev")
	if !ok || err != nil {
		t.Errorf("regular: ok=%v err=%v", ok, err)
	}

	// Symlink: exists=true but reported as an error.
	target := filepath.Join(t.TempDir(), "outside")
	if err := os.WriteFile(target, []byte("s\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(dir, "link.dev")); err != nil {
		t.Fatal(err)
	}
	ok, err = Exists(dir, "link.dev")
	if !ok || err == nil {
		t.Errorf("symlink: expected ok=true with error, got ok=%v err=%v", ok, err)
	}
}

func TestListMissingDir(t *testing.T) {
	entries, err := List(filepath.Join(t.TempDir(), "does-not-exist"))
	if err != nil {
		t.Fatalf("missing dir should not error: %v", err)
	}
	if entries != nil {
		t.Errorf("expected nil entries, got %v", entries)
	}
}

func TestListReportsIrregularAsWarning(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, dir, "good.dev", "domain good.dev\nnameserver 127.0.0.1\n")

	target := filepath.Join(t.TempDir(), "outside")
	if err := os.WriteFile(target, []byte("s\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(dir, "link.dev")); err != nil {
		t.Fatal(err)
	}

	entries, err := List(dir)
	if err != nil {
		t.Fatal(err)
	}
	var sawWarning bool
	for _, e := range entries {
		if e.Name == "link.dev" && e.Warning != "" {
			sawWarning = true
		}
	}
	if !sawWarning {
		t.Errorf("expected a warning entry for the symlink; got %+v", entries)
	}
}

func TestRemoveMissingFile(t *testing.T) {
	dir := t.TempDir()
	if _, err := Remove(dir, "absent.dev", false); err == nil {
		t.Error("removing a missing file should return an error")
	}
}

func TestReadParsesConfig(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, dir, "lab.dev", "# Managed by splitdns\ndomain lab.dev\nnameserver 127.0.0.1\nport 53\n")
	cfg, raw, err := Read(dir, "lab.dev")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Domain != "lab.dev" || !cfg.Managed {
		t.Errorf("cfg = %+v", cfg)
	}
	if !strings.Contains(raw, "nameserver 127.0.0.1") {
		t.Errorf("raw body incomplete: %q", raw)
	}
}

func TestReadMissingFile(t *testing.T) {
	dir := t.TempDir()
	if _, _, err := Read(dir, "absent.dev"); err == nil {
		t.Error("reading a missing file should return an error")
	}
}

func TestBackupDefaultDir(t *testing.T) {
	dir := t.TempDir()
	if _, _, err := Write(dir, "lab.dev", "old\n", WriteOptions{}); err != nil {
		t.Fatal(err)
	}
	// Empty BackupDir must fall back to the system temp dir.
	_, backup, err := Write(dir, "lab.dev", "new\n", WriteOptions{Force: true})
	if err != nil {
		t.Fatal(err)
	}
	if backup == "" {
		t.Fatal("expected a backup path")
	}
	defer os.Remove(backup)
	if data, _ := os.ReadFile(backup); string(data) != "old\n" {
		t.Errorf("backup content = %q, want old", data)
	}
}

func mustWrite(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
