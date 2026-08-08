package system

import (
	"os"
	"runtime"
	"testing"
)

func TestExecRunnerSuccess(t *testing.T) {
	res := ExecRunner{}.Run("echo", "hello")
	if res.Err != nil {
		t.Fatalf("unexpected error: %v", res.Err)
	}
	if res.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", res.ExitCode)
	}
	if res.Stdout != "hello\n" {
		t.Errorf("unexpected stdout: %q", res.Stdout)
	}
}

func TestExecRunnerNonZeroExit(t *testing.T) {
	// `false` exits with a non-zero status on unix-like systems.
	res := ExecRunner{}.Run("sh", "-c", "exit 3")
	if res.Err == nil {
		t.Fatal("expected an error for non-zero exit")
	}
	if res.ExitCode != 3 {
		t.Errorf("expected exit code 3, got %d", res.ExitCode)
	}
}

func TestExecRunnerCommandNotFound(t *testing.T) {
	res := ExecRunner{}.Run("splitdns-nonexistent-command-xyz")
	if res.Err == nil {
		t.Fatal("expected an error for missing command")
	}
	if res.ExitCode != -1 {
		t.Errorf("expected exit code -1 for missing command, got %d", res.ExitCode)
	}
}

func TestEnvUseColor(t *testing.T) {
	tests := []struct {
		name  string
		mode  ColorMode
		isTTY bool
		want  bool
	}{
		{"always", ColorAlways, false, true},
		{"never", ColorNever, true, false},
		{"auto-tty", ColorAuto, true, true},
		{"auto-no-tty", ColorAuto, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := Env{ColorMode: tt.mode, IsTTY: tt.isTTY}
			if got := e.UseColor(); got != tt.want {
				t.Errorf("UseColor() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultEnv(t *testing.T) {
	e := DefaultEnv()
	if e.ResolverDir != "/etc/resolver" {
		t.Errorf("ResolverDir = %q", e.ResolverDir)
	}
	if e.HostsFile != "/etc/hosts" {
		t.Errorf("HostsFile = %q", e.HostsFile)
	}
	if e.Runner == nil {
		t.Error("Runner must not be nil")
	}
	if e.GOOS != runtime.GOOS {
		t.Errorf("GOOS = %q, want %q", e.GOOS, runtime.GOOS)
	}
	if e.Stdout != os.Stdout || e.Stderr != os.Stderr {
		t.Error("Stdout/Stderr should default to os streams")
	}
	if e.ColorMode != ColorAuto {
		t.Errorf("ColorMode = %v, want ColorAuto", e.ColorMode)
	}
}

func TestGoos(t *testing.T) {
	if goos() != runtime.GOOS {
		t.Errorf("goos() = %q, want %q", goos(), runtime.GOOS)
	}
}

func TestIsTerminal(t *testing.T) {
	// A regular temp file is not a character device.
	f, err := os.CreateTemp(t.TempDir(), "tty")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if isTerminal(f) {
		t.Error("regular file should not be reported as a terminal")
	}

	// A closed file cannot be stat-ed and should report false.
	f.Close()
	if isTerminal(f) {
		t.Error("closed file should not be reported as a terminal")
	}
}

func TestFakeRunnerKeyedResponse(t *testing.T) {
	f := NewFakeRunner()
	f.SetResponse("scutil", CommandResult{Stdout: "ok", ExitCode: 0})

	res := f.Run("scutil", "--dns")
	if res.Stdout != "ok" || res.ExitCode != 0 {
		t.Errorf("unexpected keyed result: %+v", res)
	}
	if len(f.Calls) != 1 {
		t.Fatalf("expected 1 recorded call, got %d", len(f.Calls))
	}
	if f.Calls[0].Name != "scutil" || len(f.Calls[0].Args) != 1 || f.Calls[0].Args[0] != "--dns" {
		t.Errorf("call not recorded correctly: %+v", f.Calls[0])
	}
}

func TestFakeRunnerDefaultAndMiss(t *testing.T) {
	f := NewFakeRunner()

	// No default: a miss returns an error result with ExitCode -1.
	miss := f.Run("unknown")
	if miss.Err == nil || miss.ExitCode != -1 {
		t.Errorf("expected error result on miss, got %+v", miss)
	}

	// With a default, misses return the default.
	f.HasDefault = true
	f.Default = CommandResult{Stdout: "default", ExitCode: 0}
	got := f.Run("still-unknown")
	if got.Stdout != "default" {
		t.Errorf("expected default result, got %+v", got)
	}
}

func TestFakeRunnerSetResponseInitializesMap(t *testing.T) {
	// A zero-value FakeRunner has a nil Responses map; SetResponse must init it.
	f := &FakeRunner{}
	f.SetResponse("cmd", CommandResult{Stdout: "x"})
	if res := f.Run("cmd"); res.Stdout != "x" {
		t.Errorf("SetResponse did not register response: %+v", res)
	}
}
