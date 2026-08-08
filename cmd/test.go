package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/soulteary/splitdns/internal/diagnose"
	"github.com/soulteary/splitdns/internal/output"
	"github.com/spf13/cobra"
)

func newTestCmd() *cobra.Command {
	var timeoutSec int
	cmd := &cobra.Command{
		Use:   "test <hostname>",
		Short: "Test resolution of a hostname through the split-DNS layers",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTest(args[0], time.Duration(timeoutSec)*time.Second)
		},
	}
	cmd.Flags().IntVar(&timeoutSec, "timeout", 2, "DNS probe timeout in seconds")
	return cmd
}

func runTest(hostname string, timeout time.Duration) error {
	if err := requireMacOS(); err != nil {
		return err
	}

	result := diagnose.RunTest(env, hostname, timeout)

	if flagJSON {
		if err := output.JSON(env.Stdout, result); err != nil {
			return runtimeErrorf("%v", err)
		}
		return nil
	}

	c := colorizer()
	fmt.Fprintf(env.Stdout, "%s %s\n", c.Bold("Hostname:"), result.Hostname)
	if result.MatchedSuffix == "" {
		fmt.Fprintln(env.Stdout, c.Yellow("No resolver suffix matches this hostname."))
	} else {
		fmt.Fprintf(env.Stdout, "%s %s\n", c.Bold("Matched suffix:"), result.MatchedSuffix)
		fmt.Fprintf(env.Stdout, "%s %s\n", c.Bold("Resolver file:"), result.ResolverFile)
		fmt.Fprintf(env.Stdout, "%s %s\n", c.Bold("Nameservers:"), strings.Join(result.Nameservers, ", "))
		if result.Port != "" {
			fmt.Fprintf(env.Stdout, "%s %s\n", c.Bold("Port:"), result.Port)
		}
		loaded := c.Red("no")
		if result.ScutilLoaded {
			loaded = c.Green("yes")
		}
		fmt.Fprintf(env.Stdout, "%s %s\n", c.Bold("scutil loaded:"), loaded)
	}

	fmt.Fprintf(env.Stdout, "%s %s\n", c.Bold("dscacheutil addresses:"), joinOrDash(result.DscacheAddresses))
	if result.DirectError != "" && len(result.DirectAddresses) == 0 {
		fmt.Fprintf(env.Stdout, "%s %s\n", c.Bold("direct DNS:"), c.Red(result.DirectError))
	} else {
		fmt.Fprintf(env.Stdout, "%s %s\n", c.Bold("direct DNS addresses:"), joinOrDash(result.DirectAddresses))
	}

	consistent := c.Yellow("no")
	if result.Consistent {
		consistent = c.Green("yes")
	}
	fmt.Fprintf(env.Stdout, "%s %s\n", c.Bold("consistent:"), consistent)

	if len(result.Troubleshooting) > 0 && !flagQuiet {
		fmt.Fprintln(env.Stdout, c.Bold("Troubleshooting:"))
		for _, t := range result.Troubleshooting {
			fmt.Fprintf(env.Stdout, "  - %s\n", t)
		}
	}
	return nil
}

func joinOrDash(items []string) string {
	if len(items) == 0 {
		return "-"
	}
	return strings.Join(items, ", ")
}
