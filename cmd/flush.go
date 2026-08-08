package cmd

import (
	"fmt"

	"github.com/soulteary/splitdns/internal/output"
	"github.com/soulteary/splitdns/internal/resolver"
	"github.com/spf13/cobra"
)

func newFlushCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "flush",
		Short: "Flush macOS DNS caches",
		Example: `  # Flush the DNS caches
  sudo splitdns flush

  # Preview the commands that would run
  splitdns flush --dry-run`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFlush(flagDryRun)
		},
	}
	return cmd
}

func runFlush(dryRun bool) error {
	if err := requireMacOS(); err != nil {
		return err
	}
	if !dryRun {
		if err := requireRoot("splitdns flush"); err != nil {
			return err
		}
	}

	steps := resolver.Flush(env, dryRun)

	if flagJSON {
		if err := output.JSON(env.Stdout, steps); err != nil {
			return runtimeErrorf("%v", err)
		}
		return flushStepsExit(steps)
	}

	c := colorizer()
	if dryRun {
		fmt.Fprintln(env.Stdout, c.Bold("Dry run — planned commands:"))
		for _, s := range steps {
			fmt.Fprintf(env.Stdout, "  - %s\n", s.Command)
		}
		return nil
	}
	reportFlushSteps(steps)
	return flushStepsExit(steps)
}

func flushStepsExit(steps []resolver.FlushStep) error {
	for _, s := range steps {
		if !s.OK {
			return runtimeErrorf("DNS cache flush failed at step %q: %s", s.Command, s.Message)
		}
	}
	return nil
}
