package cmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/soulteary/splitdns/internal/system"
)

func setupEnv(t *testing.T, dir string) (*bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	var out, errBuf bytes.Buffer
	fr := system.NewFakeRunner()
	fr.HasDefault = true
	fr.Default = system.CommandResult{ExitCode: 0}
	env = system.Env{
		ResolverDir: dir,
		HostsFile:   filepath.Join(dir, "hosts"),
		Runner:      fr,
		GOOS:        "darwin",
		Stdout:      &out,
		Stderr:      &errBuf,
		IsTTY:       false,
		ColorMode:   system.ColorNever,
	}
	flagJSON = false
	flagQuiet = false
	flagNoColor = true
	flagDryRun = false
	rootCheckDisabled = true
	return &out, &errBuf
}

func codeOf(err error) int {
	if err == nil {
		return 0
	}
	var ee *ExitError
	if errors.As(err, &ee) {
		return ee.Code
	}
	return ExitArgError
}

func TestAddDoesNotOverwriteExisting(t *testing.T) {
	dir := t.TempDir()
	setupEnv(t, dir)

	existing := filepath.Join(dir, "lab.dev")
	if err := os.WriteFile(existing, []byte("original\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	f := &addSetFlags{nameservers: []string{"127.0.0.1"}, port: "53"}
	err := runAdd("lab.dev", f)
	if codeOf(err) != ExitConfigError {
		t.Fatalf("expected config error (4), got %v", err)
	}
	data, _ := os.ReadFile(existing)
	if string(data) != "original\n" {
		t.Errorf("existing file was overwritten: %q", data)
	}
}

func TestAddDryRunWritesNothing(t *testing.T) {
	dir := t.TempDir()
	out, _ := setupEnv(t, dir)

	f := &addSetFlags{nameservers: []string{"127.0.0.1"}, port: "53", dryRun: true, noFlush: true}
	if err := runAdd("lab.dev", f); err != nil {
		t.Fatalf("dry-run add failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "lab.dev")); !os.IsNotExist(err) {
		t.Error("dry-run add created a file")
	}
	if !strings.Contains(out.String(), "Dry run") {
		t.Errorf("expected dry-run notice, got: %s", out.String())
	}
}

func TestSetUpdatesOnlySpecifiedField(t *testing.T) {
	dir := t.TempDir()
	setupEnv(t, dir)

	path := filepath.Join(dir, "lab.dev")
	if err := os.WriteFile(path, []byte("# Managed by splitdns\n# keep me\ndomain lab.dev\nnameserver 127.0.0.1\nport 53\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := newSetCmd()
	cmd.SetArgs([]string{"lab.dev", "--port", "5353"})
	// Route output through our env buffers.
	if err := cmd.Execute(); err != nil {
		t.Fatalf("set failed: %v", err)
	}

	data, _ := os.ReadFile(path)
	s := string(data)
	if !strings.Contains(s, "port 5353") {
		t.Errorf("port not updated: %s", s)
	}
	if !strings.Contains(s, "nameserver 127.0.0.1") {
		t.Errorf("nameserver was lost: %s", s)
	}
	if !strings.Contains(s, "# keep me") {
		t.Errorf("user comment was lost: %s", s)
	}
}

func TestGlobalDryRunAddViaFlag(t *testing.T) {
	dir := t.TempDir()
	out, _ := setupEnv(t, dir)

	flagDryRun = true
	defer func() { flagDryRun = false }()

	f := &addSetFlags{nameservers: []string{"127.0.0.1"}, port: "53", noFlush: true}
	if err := runAdd("lab.dev", f); err != nil {
		t.Fatalf("dry-run add via global flag failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "lab.dev")); !os.IsNotExist(err) {
		t.Error("global --dry-run add created a file")
	}
	if !strings.Contains(out.String(), "Dry run") {
		t.Errorf("expected dry-run notice, got: %s", out.String())
	}
}

func TestListJSONStable(t *testing.T) {
	dir := t.TempDir()
	out, _ := setupEnv(t, dir)
	flagJSON = true

	if err := os.WriteFile(filepath.Join(dir, "a.dev"), []byte("domain a.dev\nnameserver 127.0.0.1\nport 53\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runList(); err != nil {
		t.Fatal(err)
	}
	first := out.String()

	out.Reset()
	if err := runList(); err != nil {
		t.Fatal(err)
	}
	if out.String() != first {
		t.Errorf("list JSON not stable:\n%s\n---\n%s", first, out.String())
	}
}
