// Package system provides the execution environment (Env), a command Runner
// abstraction with real and fake implementations, and TTY detection.
package system

import (
	"io"
	"os"
	"os/exec"
	"strings"
)

// ColorMode controls colored output behavior.
type ColorMode int

const (
	// ColorAuto enables color only when writing to a TTY.
	ColorAuto ColorMode = iota
	// ColorAlways forces color output.
	ColorAlways
	// ColorNever disables color output.
	ColorNever
)

// CommandResult holds the captured output of an executed command.
type CommandResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Err      error
}

// Runner abstracts execution of external commands so tests can inject fakes.
type Runner interface {
	Run(name string, args ...string) CommandResult
}

// ExecRunner runs commands using os/exec with separated arguments.
type ExecRunner struct{}

// Run executes the command and captures stdout/stderr and exit code.
func (ExecRunner) Run(name string, args ...string) CommandResult {
	cmd := exec.Command(name, args...)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	res := CommandResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
		Err:    err,
	}
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			res.ExitCode = exitErr.ExitCode()
		} else {
			res.ExitCode = -1
		}
	}
	return res
}

// Env carries all injectable dependencies and configuration for command execution.
type Env struct {
	ResolverDir string
	HostsFile   string
	Runner      Runner
	GOOS        string
	Stdout      io.Writer
	Stderr      io.Writer
	IsTTY       bool
	ColorMode   ColorMode
}

// UseColor reports whether colored output should be emitted.
func (e Env) UseColor() bool {
	switch e.ColorMode {
	case ColorAlways:
		return true
	case ColorNever:
		return false
	default:
		return e.IsTTY
	}
}

// DefaultEnv builds an Env using the real filesystem and command runner.
func DefaultEnv() Env {
	return Env{
		ResolverDir: "/etc/resolver",
		HostsFile:   "/etc/hosts",
		Runner:      ExecRunner{},
		GOOS:        goos(),
		Stdout:      os.Stdout,
		Stderr:      os.Stderr,
		IsTTY:       isTerminal(os.Stdout),
		ColorMode:   ColorAuto,
	}
}
