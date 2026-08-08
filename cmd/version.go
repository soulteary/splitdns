package cmd

import (
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// Build information, injected via -ldflags at release time.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Example: `  # Print version information
  splitdns version

  # Print version information as JSON
  splitdns version --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVersion()
		},
	}
}

func runVersion() error {
	if flagJSON {
		payload := map[string]string{
			"version":   version,
			"commit":    commit,
			"date":      date,
			"goVersion": runtime.Version(),
			"platform":  runtime.GOOS + "/" + runtime.GOARCH,
		}
		enc := json.NewEncoder(env.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(payload)
	}
	fmt.Fprintf(env.Stdout, "splitdns %s (commit %s, built %s, %s %s/%s)\n",
		version, commit, date, runtime.Version(), runtime.GOOS, runtime.GOARCH)
	return nil
}
