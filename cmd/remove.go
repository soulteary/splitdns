package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/soulteary/splitdns/internal/domain"
	"github.com/soulteary/splitdns/internal/output"
	"github.com/soulteary/splitdns/internal/resolver"
	"github.com/spf13/cobra"
)

func newRemoveCmd() *cobra.Command {
	var (
		yes     bool
		noFlush bool
	)
	cmd := &cobra.Command{
		Use:     "remove <domain>",
		Aliases: []string{"rm"},
		Short:   "Remove a resolver entry for a domain suffix",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRemove(args[0], yes, flagDryRun, noFlush)
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "skip confirmation prompt (for automation)")
	cmd.Flags().BoolVar(&noFlush, "no-flush", false, "do not flush DNS caches after removal")
	return cmd
}

func runRemove(rawDomain string, yes, dryRun, noFlush bool) error {
	if err := requireMacOS(); err != nil {
		return err
	}

	norm, err := domain.Normalize(rawDomain, true, true)
	if err != nil {
		return configErrorf("%v", err)
	}
	name := domain.FileName(norm.Normalized)

	exists, err := resolver.Exists(env.ResolverDir, name)
	if err != nil {
		return configErrorf("%v", err)
	}
	if !exists {
		return configErrorf("resolver for %q does not exist", norm.Normalized)
	}

	_, raw, err := resolver.Read(env.ResolverDir, name)
	if err != nil {
		return configErrorf("%v", err)
	}
	path, _ := resolver.Path(env.ResolverDir, name)

	// Always show the target before deleting.
	showRemoveTarget(path, raw)

	if dryRun {
		if flagJSON {
			return output.JSON(env.Stdout, resolver.ApplyResult{Path: path, DryRun: true})
		}
		fmt.Fprintln(env.Stdout, colorizer().Bold("Dry run — file not deleted."))
		return nil
	}

	if !yes {
		if !env.IsTTY {
			return argErrorf("refusing to delete %s without confirmation in a non-interactive session; pass --yes to proceed", path)
		}
		if !confirm(fmt.Sprintf("Delete %s?", path)) {
			fmt.Fprintln(env.Stdout, "Aborted.")
			return nil
		}
	}

	if err := requireRoot(fmt.Sprintf("splitdns remove %s", rawDomain)); err != nil {
		return err
	}

	if _, err := resolver.Remove(env.ResolverDir, name, false); err != nil {
		return configErrorf("%v", err)
	}

	result := resolver.ApplyResult{Written: true, Path: path}
	if !noFlush {
		result.FlushSteps = resolver.Flush(env, false)
	}

	if flagJSON {
		if err := output.JSON(env.Stdout, result); err != nil {
			return runtimeErrorf("%v", err)
		}
		return flushFailureExit(result)
	}

	fmt.Fprintln(env.Stdout, colorizer().Green("Removed ")+path)
	reportFlushSteps(result.FlushSteps)
	return flushFailureExit(result)
}

func showRemoveTarget(path string, raw string) {
	if flagJSON || flagQuiet {
		return
	}
	fmt.Fprintf(env.Stdout, "Target: %s\n", path)
	fmt.Fprint(env.Stdout, raw)
	if !strings.HasSuffix(raw, "\n") {
		fmt.Fprintln(env.Stdout)
	}
}

func confirm(prompt string) bool {
	fmt.Fprintf(env.Stdout, "%s [y/N] ", prompt)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	line = strings.ToLower(strings.TrimSpace(line))
	return line == "y" || line == "yes"
}
