package cmd

import (
	"os"

	"github.com/soulteary/splitdns/internal/output"
)

// requireMacOS returns a runtime error when not running on macOS.
func requireMacOS() error {
	if env.GOOS != "darwin" {
		return runtimeErrorf("splitdns only supports macOS (darwin); current platform is %s", env.GOOS)
	}
	return nil
}

// rootCheckDisabled bypasses the root requirement; used only in tests.
var rootCheckDisabled bool

// requireRoot returns a permission error with a sudo suggestion when the
// process is not running as root. It never attempts to auto-elevate.
func requireRoot(suggestion string) error {
	if rootCheckDisabled || os.Geteuid() == 0 {
		return nil
	}
	return permissionErrorf("this operation modifies %s and requires root; re-run with: sudo %s",
		env.ResolverDir, suggestion)
}

// colorizer builds a Colorizer honoring the environment's color settings.
func colorizer() output.Colorizer {
	return output.Colorizer{Enabled: env.UseColor()}
}
