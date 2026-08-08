package system

import "fmt"

// FakeCall records a single invocation made against a FakeRunner.
type FakeCall struct {
	Name string
	Args []string
}

// FakeResponse is a canned response matched against a command invocation.
type FakeResponse struct {
	Result  CommandResult
	Matched bool
}

// FakeRunner is a Runner implementation for tests. It matches invocations by
// the command name and returns queued or keyed responses without executing
// anything on the host.
type FakeRunner struct {
	// Responses maps a command name to the result returned when it is invoked.
	Responses map[string]CommandResult
	// Calls records every invocation in order.
	Calls []FakeCall
	// Default is returned when no keyed response matches.
	Default CommandResult
	// HasDefault indicates whether Default should be used on a miss.
	HasDefault bool
}

// NewFakeRunner returns an initialized FakeRunner.
func NewFakeRunner() *FakeRunner {
	return &FakeRunner{Responses: make(map[string]CommandResult)}
}

// SetResponse registers a canned result for a command name.
func (f *FakeRunner) SetResponse(name string, res CommandResult) {
	if f.Responses == nil {
		f.Responses = make(map[string]CommandResult)
	}
	f.Responses[name] = res
}

// Run records the call and returns the matching canned response.
func (f *FakeRunner) Run(name string, args ...string) CommandResult {
	f.Calls = append(f.Calls, FakeCall{Name: name, Args: append([]string(nil), args...)})
	if res, ok := f.Responses[name]; ok {
		return res
	}
	if f.HasDefault {
		return f.Default
	}
	return CommandResult{ExitCode: -1, Err: fmt.Errorf("no fake response for %q", name)}
}
