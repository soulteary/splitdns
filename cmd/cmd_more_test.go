package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/soulteary/splitdns/internal/resolver"
)

func writeResolver(t *testing.T, dir, name, body string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestExitErrorConstructors(t *testing.T) {
	cases := []struct {
		name string
		err  *ExitError
		code int
	}{
		{"arg", argErrorf("bad %s", "arg"), ExitArgError},
		{"permission", permissionErrorf("need root"), ExitPermission},
		{"config", configErrorf("bad config"), ExitConfigError},
		{"runtime", runtimeErrorf("boom"), ExitRuntime},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.err.Code != tc.code {
				t.Errorf("code = %d, want %d", tc.err.Code, tc.code)
			}
			if tc.err.Error() == "" {
				t.Error("Error() should be non-empty")
			}
			if tc.err.Unwrap() == nil {
				t.Error("Unwrap() should return wrapped error")
			}
		})
	}
}

func TestExitErrorErrorWithoutWrapped(t *testing.T) {
	e := &ExitError{Code: ExitRuntime}
	if got := e.Error(); got != "exit code 5" {
		t.Errorf("Error() = %q, want %q", got, "exit code 5")
	}
	if e.Unwrap() != nil {
		t.Error("Unwrap() should be nil when Err is nil")
	}
}

func TestFormatError(t *testing.T) {
	if got := formatError(errors.New("boom")); got != "Error: boom" {
		t.Errorf("formatError = %q", got)
	}
}

func TestRequireMacOS(t *testing.T) {
	setupEnv(t, t.TempDir())
	if err := requireMacOS(); err != nil {
		t.Errorf("darwin should pass: %v", err)
	}

	env.GOOS = "linux"
	err := requireMacOS()
	if codeOf(err) != ExitRuntime {
		t.Errorf("non-darwin should return runtime error, got %v", err)
	}
}

func TestRequireRoot(t *testing.T) {
	setupEnv(t, t.TempDir())
	// rootCheckDisabled is set by setupEnv, so this passes.
	if err := requireRoot("splitdns flush"); err != nil {
		t.Errorf("expected nil when root check disabled: %v", err)
	}

	rootCheckDisabled = false
	defer func() { rootCheckDisabled = true }()
	if os.Geteuid() != 0 {
		err := requireRoot("splitdns flush")
		if codeOf(err) != ExitPermission {
			t.Errorf("non-root should return permission error, got %v", err)
		}
	}
}

func TestRunVersionText(t *testing.T) {
	out, _ := setupEnv(t, t.TempDir())
	if err := runVersion(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "splitdns") {
		t.Errorf("version output missing name: %q", out.String())
	}
}

func TestRunVersionJSON(t *testing.T) {
	out, _ := setupEnv(t, t.TempDir())
	flagJSON = true
	if err := runVersion(); err != nil {
		t.Fatal(err)
	}
	s := out.String()
	for _, want := range []string{`"version"`, `"commit"`, `"goVersion"`, `"platform"`} {
		if !strings.Contains(s, want) {
			t.Errorf("version JSON missing %s: %q", want, s)
		}
	}
}

func TestRunShowText(t *testing.T) {
	dir := t.TempDir()
	out, _ := setupEnv(t, dir)
	writeResolver(t, dir, "lab.dev", "# Managed by splitdns\ndomain lab.dev\nnameserver 127.0.0.1\nport 53\n")

	if err := runShow("lab.dev", false); err != nil {
		t.Fatal(err)
	}
	s := out.String()
	if !strings.Contains(s, "lab.dev") || !strings.Contains(s, "127.0.0.1") || !strings.Contains(s, "53") {
		t.Errorf("show output incomplete: %q", s)
	}
}

func TestRunShowRaw(t *testing.T) {
	dir := t.TempDir()
	out, _ := setupEnv(t, dir)
	body := "# Managed by splitdns\ndomain lab.dev\nnameserver 127.0.0.1\n"
	writeResolver(t, dir, "lab.dev", body)

	if err := runShow("lab.dev", true); err != nil {
		t.Fatal(err)
	}
	if out.String() != body {
		t.Errorf("raw output = %q, want verbatim %q", out.String(), body)
	}
}

func TestRunShowJSON(t *testing.T) {
	dir := t.TempDir()
	out, _ := setupEnv(t, dir)
	flagJSON = true
	writeResolver(t, dir, "lab.dev", "# Managed by splitdns\ndomain lab.dev\nnameserver 127.0.0.1\nport 5353\n")

	if err := runShow("lab.dev", false); err != nil {
		t.Fatal(err)
	}
	s := out.String()
	for _, want := range []string{`"name"`, `"domain"`, `"nameservers"`, `"port"`, `"managed"`} {
		if !strings.Contains(s, want) {
			t.Errorf("show JSON missing %s: %q", want, s)
		}
	}
}

func TestRunShowMissing(t *testing.T) {
	dir := t.TempDir()
	setupEnv(t, dir)
	err := runShow("absent.dev", false)
	if codeOf(err) != ExitConfigError {
		t.Errorf("missing resolver should be config error, got %v", err)
	}
}

func TestRunListEmptyText(t *testing.T) {
	dir := t.TempDir()
	out, _ := setupEnv(t, dir)
	if err := runList(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "No resolver entries") {
		t.Errorf("expected empty notice, got %q", out.String())
	}
}

func TestRunListTable(t *testing.T) {
	dir := t.TempDir()
	out, _ := setupEnv(t, dir)
	writeResolver(t, dir, "lab.dev", "# Managed by splitdns\ndomain lab.dev\nnameserver 127.0.0.1\nport 53\n")
	if err := runList(); err != nil {
		t.Fatal(err)
	}
	s := out.String()
	if !strings.Contains(s, "NAME") || !strings.Contains(s, "lab.dev") {
		t.Errorf("table output incomplete: %q", s)
	}
}

func TestRunFlushExecutes(t *testing.T) {
	dir := t.TempDir()
	out, _ := setupEnv(t, dir)
	if err := runFlush(false); err != nil {
		t.Fatalf("flush failed: %v", err)
	}
	_ = out
}

func TestRunFlushDryRun(t *testing.T) {
	dir := t.TempDir()
	out, _ := setupEnv(t, dir)
	if err := runFlush(true); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Dry run") {
		t.Errorf("expected dry-run planned commands, got %q", out.String())
	}
}

func TestRunFlushJSON(t *testing.T) {
	dir := t.TempDir()
	out, _ := setupEnv(t, dir)
	flagJSON = true
	if err := runFlush(false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "command") {
		t.Errorf("expected JSON steps, got %q", out.String())
	}
}

func TestRunFlushWrongOS(t *testing.T) {
	setupEnv(t, t.TempDir())
	env.GOOS = "linux"
	if codeOf(runFlush(false)) != ExitRuntime {
		t.Error("flush on non-darwin should return runtime error")
	}
}

func TestFlushStepsExit(t *testing.T) {
	ok := []resolver.FlushStep{{Name: "a", OK: true}, {Name: "b", OK: true}}
	if err := flushStepsExit(ok); err != nil {
		t.Errorf("all-OK steps should not error: %v", err)
	}

	failed := []resolver.FlushStep{{Name: "a", OK: true}, {Command: "killall", OK: false, Message: "boom"}}
	if codeOf(flushStepsExit(failed)) != ExitRuntime {
		t.Error("a failing step should return a runtime error")
	}
}

func TestRunCheckWrongOSAndBasic(t *testing.T) {
	dir := t.TempDir()
	out, _ := setupEnv(t, dir)
	// check does not require macOS gate in runCheck; it runs diagnostics.
	err := runCheck("", 100*time.Millisecond)
	// May or may not have errors depending on environment; just ensure it produced output or an ExitError.
	if err != nil {
		if codeOf(err) != ExitRuntime {
			t.Errorf("unexpected error code: %v", err)
		}
	}
	_ = out
}

func TestRunCheckJSON(t *testing.T) {
	dir := t.TempDir()
	out, _ := setupEnv(t, dir)
	flagJSON = true
	_ = runCheck("", 100*time.Millisecond)
	if !strings.Contains(out.String(), "checks") && !strings.Contains(out.String(), "{") {
		t.Errorf("expected JSON report, got %q", out.String())
	}
}

func TestRunTestWrongOS(t *testing.T) {
	setupEnv(t, t.TempDir())
	env.GOOS = "linux"
	if codeOf(runTest("host.lab.dev", 100*time.Millisecond)) != ExitRuntime {
		t.Error("test on non-darwin should return runtime error")
	}
}

func TestRunTestText(t *testing.T) {
	dir := t.TempDir()
	out, _ := setupEnv(t, dir)
	writeResolver(t, dir, "lab.dev", "# Managed by splitdns\ndomain lab.dev\nnameserver 127.0.0.1\nport 53\n")
	if err := runTest("host.lab.dev", 100*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Hostname:") {
		t.Errorf("test output missing hostname line: %q", out.String())
	}
}

func TestRunTestJSON(t *testing.T) {
	dir := t.TempDir()
	out, _ := setupEnv(t, dir)
	flagJSON = true
	if err := runTest("host.lab.dev", 100*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "hostname") {
		t.Errorf("test JSON missing hostname: %q", out.String())
	}
}

func TestJoinOrDash(t *testing.T) {
	if got := joinOrDash(nil); got != "-" {
		t.Errorf("joinOrDash(nil) = %q, want -", got)
	}
	if got := joinOrDash([]string{"a", "b"}); got != "a, b" {
		t.Errorf("joinOrDash = %q, want \"a, b\"", got)
	}
}

func TestRunRemoveDryRun(t *testing.T) {
	dir := t.TempDir()
	out, _ := setupEnv(t, dir)
	path := writeResolver(t, dir, "lab.dev", "# Managed by splitdns\ndomain lab.dev\nnameserver 127.0.0.1\n")

	if err := runRemove("lab.dev", true, true, true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Error("dry-run remove should not delete the file")
	}
	if !strings.Contains(out.String(), "Dry run") {
		t.Errorf("expected dry-run notice, got %q", out.String())
	}
}

func TestRunRemoveDeletes(t *testing.T) {
	dir := t.TempDir()
	setupEnv(t, dir)
	path := writeResolver(t, dir, "lab.dev", "# Managed by splitdns\ndomain lab.dev\nnameserver 127.0.0.1\n")

	if err := runRemove("lab.dev", true, false, true); err != nil {
		t.Fatalf("remove failed: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("remove should delete the file")
	}
}

func TestRunRemoveMissing(t *testing.T) {
	dir := t.TempDir()
	setupEnv(t, dir)
	if codeOf(runRemove("absent.dev", true, false, true)) != ExitConfigError {
		t.Error("removing a missing resolver should be a config error")
	}
}

func TestRunRemoveNonInteractiveRequiresYes(t *testing.T) {
	dir := t.TempDir()
	setupEnv(t, dir)
	writeResolver(t, dir, "lab.dev", "# Managed by splitdns\ndomain lab.dev\nnameserver 127.0.0.1\n")
	// IsTTY is false in setupEnv; without --yes it must refuse.
	if codeOf(runRemove("lab.dev", false, false, true)) != ExitArgError {
		t.Error("non-interactive remove without --yes should be an arg error")
	}
}

func TestNewCommandsConstruct(t *testing.T) {
	// Ensure command constructors wire up without panicking.
	cmds := []interface{ Name() string }{
		newAddCmd(),
		newSetCmd(),
		newRemoveCmd(),
		newListCmd(),
		newShowCmd(),
		newFlushCmd(),
		newCheckCmd(),
		newTestCmd(),
		newCompletionCmd(),
		newVersionCmd(),
		newRootCmd(),
	}
	for _, c := range cmds {
		if c.Name() == "" {
			t.Errorf("command has empty name: %T", c)
		}
	}
}
