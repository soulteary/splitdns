package cmd

import "fmt"

// Exit codes per the plan's exit-code mapping.
const (
	ExitSuccess     = 0
	ExitArgError    = 2
	ExitPermission  = 3
	ExitConfigError = 4
	ExitRuntime     = 5
)

// ExitError wraps an error with an explicit process exit code.
type ExitError struct {
	Code int
	Err  error
}

func (e *ExitError) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("exit code %d", e.Code)
	}
	return e.Err.Error()
}

func (e *ExitError) Unwrap() error { return e.Err }

func argErrorf(format string, args ...any) *ExitError {
	return &ExitError{Code: ExitArgError, Err: fmt.Errorf(format, args...)}
}

func permissionErrorf(format string, args ...any) *ExitError {
	return &ExitError{Code: ExitPermission, Err: fmt.Errorf(format, args...)}
}

func configErrorf(format string, args ...any) *ExitError {
	return &ExitError{Code: ExitConfigError, Err: fmt.Errorf(format, args...)}
}

func runtimeErrorf(format string, args ...any) *ExitError {
	return &ExitError{Code: ExitRuntime, Err: fmt.Errorf(format, args...)}
}
