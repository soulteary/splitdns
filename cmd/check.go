package cmd

import (
	"fmt"
	"time"

	"github.com/soulteary/splitdns/internal/diagnose"
	"github.com/soulteary/splitdns/internal/output"
	"github.com/spf13/cobra"
)

func newCheckCmd() *cobra.Command {
	var timeoutSec int
	cmd := &cobra.Command{
		Use:   "check [domain]",
		Short: "Run configuration and environment diagnostics",
		Example: `  # Run general environment and configuration diagnostics
  splitdns check

  # Focus the diagnostics on a specific domain
  splitdns check corp.example.com

  # Use a longer DNS probe timeout
  splitdns check corp.example.com --timeout 5`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			domainArg := ""
			if len(args) == 1 {
				domainArg = args[0]
			}
			return runCheck(domainArg, time.Duration(timeoutSec)*time.Second)
		},
	}
	cmd.Flags().IntVar(&timeoutSec, "timeout", 2, "DNS probe timeout in seconds")
	return cmd
}

func runCheck(domainArg string, timeout time.Duration) error {
	report := diagnose.RunChecks(env, domainArg, timeout)

	if flagJSON {
		if err := output.JSON(env.Stdout, report); err != nil {
			return runtimeErrorf("%v", err)
		}
		if report.HasError() {
			return &ExitError{Code: ExitRuntime}
		}
		return nil
	}

	c := colorizer()
	for _, chk := range report.Checks {
		var badge string
		switch chk.Status {
		case diagnose.StatusOK:
			badge = c.Green("[OK]   ")
		case diagnose.StatusWarn:
			badge = c.Yellow("[WARN] ")
		case diagnose.StatusError:
			badge = c.Red("[ERROR]")
		}
		fmt.Fprintf(env.Stdout, "%s %s: %s\n", badge, chk.Name, chk.Message)
		if chk.Hint != "" && !flagQuiet {
			fmt.Fprintf(env.Stdout, "         hint: %s\n", chk.Hint)
		}
	}

	if report.HasError() {
		return &ExitError{Code: ExitRuntime, Err: fmt.Errorf("one or more checks reported ERROR")}
	}
	return nil
}
