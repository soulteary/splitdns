package resolver

import (
	"strings"

	"github.com/soulteary/splitdns/internal/system"
)

// Flush clears the macOS DNS caches: dscacheutil -flushcache then
// killall -HUP mDNSResponder. It never fails hard when killall reports no
// matching process; each step's outcome is returned individually. When dryRun
// is set, the planned commands are returned with OK=true and no execution.
func Flush(env system.Env, dryRun bool) []FlushStep {
	steps := []FlushStep{
		{Name: "flush-cache", Command: "dscacheutil -flushcache"},
		{Name: "restart-responder", Command: "killall -HUP mDNSResponder"},
	}

	if dryRun {
		for i := range steps {
			steps[i].OK = true
			steps[i].Message = "planned"
		}
		return steps
	}

	r1 := env.Runner.Run("dscacheutil", "-flushcache")
	steps[0].OK = r1.Err == nil && r1.ExitCode == 0
	if !steps[0].OK {
		steps[0].Message = commandError(r1)
	}

	r2 := env.Runner.Run("killall", "-HUP", "mDNSResponder")
	// killall exits non-zero when no matching process exists; treat that as a
	// benign no-op rather than a failure.
	if r2.Err == nil && r2.ExitCode == 0 {
		steps[1].OK = true
	} else if noSuchProcess(r2) {
		steps[1].OK = true
		steps[1].Message = "mDNSResponder was not running"
	} else {
		steps[1].OK = false
		steps[1].Message = commandError(r2)
	}
	return steps
}

func noSuchProcess(res system.CommandResult) bool {
	out := res.Stdout + res.Stderr
	return strings.Contains(out, "No matching processes") || strings.Contains(out, "no process found")
}

func commandError(res system.CommandResult) string {
	msg := res.Stderr
	if msg == "" {
		msg = res.Stdout
	}
	if msg == "" && res.Err != nil {
		msg = res.Err.Error()
	}
	if msg == "" {
		msg = "command failed"
	}
	return msg
}
