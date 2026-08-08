// Package cmd implements the splitdns command-line interface using Cobra,
// including flag parsing, the shared execution environment and exit-code
// mapping.
package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/soulteary/splitdns/internal/system"
	"github.com/spf13/cobra"
)

var (
	flagJSON    bool
	flagQuiet   bool
	flagNoColor bool
	flagDryRun  bool
)

// env is the process-wide execution environment, constructed in
// PersistentPreRun so all subcommands share consistent dependencies.
var env system.Env

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "splitdns",
		Short: "Safely manage macOS /etc/resolver suffix-based Split DNS",
		Long:  "splitdns safely manages macOS /etc/resolver configuration for suffix-based Split DNS.",
		Example: `  # Route a domain suffix to a local DNS server
  sudo splitdns add corp.example.com --nameserver 10.0.0.53

  # List all managed resolver entries
  splitdns list

  # Inspect the resolver for a domain
  splitdns show corp.example.com

  # Test how a hostname resolves through the split-DNS layers
  splitdns test api.corp.example.com

  # Run environment and configuration diagnostics
  splitdns check

  # Remove a resolver entry
  sudo splitdns remove corp.example.com

  # Preview any change without touching the system
  splitdns add internal.dev --nameserver 127.0.0.1 --dry-run`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.PersistentFlags().BoolVar(&flagJSON, "json", false, "emit machine-readable JSON output")
	root.PersistentFlags().BoolVar(&flagQuiet, "quiet", false, "suppress non-essential output")
	root.PersistentFlags().BoolVar(&flagNoColor, "no-color", false, "disable colored output")
	root.PersistentFlags().BoolVar(&flagDryRun, "dry-run", false, "show planned changes without modifying the system")

	root.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		env = system.DefaultEnv()
		if flagNoColor {
			env.ColorMode = system.ColorNever
		}
	}

	root.AddCommand(
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
	)

	return root
}

// Execute runs the root command and maps errors to process exit codes.
func Execute() {
	root := newRootCmd()
	err := root.Execute()
	if err == nil {
		return
	}

	var exitErr *ExitError
	code := ExitArgError
	if errors.As(err, &exitErr) {
		code = exitErr.Code
	}
	fmt.Fprintln(os.Stderr, formatError(err))
	os.Exit(code)
}

func formatError(err error) string {
	return "Error: " + err.Error()
}
