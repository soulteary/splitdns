package cmd

import (
	"fmt"

	"github.com/soulteary/splitdns/internal/output"
	"github.com/soulteary/splitdns/internal/resolver"
)

// applyWrite performs the write + optional flush and reports the ApplyResult,
// clearly distinguishing a successful config write from a failed flush.
func applyWrite(name, content string, f *addSetFlags, rawDomain string) error {
	path, backup, err := resolver.Write(env.ResolverDir, name, content, resolver.WriteOptions{
		Force:     f.force,
		BackupDir: f.backupDir,
		DryRun:    f.dryRun,
	})
	if err != nil {
		return configErrorf("%v", err)
	}

	result := resolver.ApplyResult{
		Written:    !f.dryRun,
		Path:       path,
		BackupPath: backup,
		DryRun:     f.dryRun,
	}
	if f.dryRun {
		result.Content = content
	}

	if !f.dryRun && !f.noFlush {
		result.FlushSteps = resolver.Flush(env, false)
	} else if f.dryRun && !f.noFlush {
		result.FlushSteps = resolver.Flush(env, true)
	}

	return reportApply(result, content)
}

func reportApply(result resolver.ApplyResult, content string) error {
	if flagJSON {
		if err := output.JSON(env.Stdout, result); err != nil {
			return runtimeErrorf("%v", err)
		}
		return flushFailureExit(result)
	}

	c := colorizer()
	if result.DryRun {
		fmt.Fprintln(env.Stdout, c.Bold("Dry run — no changes written."))
		fmt.Fprintf(env.Stdout, "Target: %s\n", result.Path)
		fmt.Fprintln(env.Stdout, "Planned content:")
		fmt.Fprint(env.Stdout, content)
		if len(result.FlushSteps) > 0 {
			fmt.Fprintln(env.Stdout, "Planned cache flush:")
			for _, s := range result.FlushSteps {
				fmt.Fprintf(env.Stdout, "  - %s\n", s.Command)
			}
		}
		return nil
	}

	if result.Written {
		fmt.Fprintln(env.Stdout, c.Green("Wrote ")+result.Path)
	}
	if result.BackupPath != "" {
		fmt.Fprintf(env.Stdout, "Backup: %s\n", result.BackupPath)
	}
	reportFlushSteps(result.FlushSteps)
	return flushFailureExit(result)
}

func reportFlushSteps(steps []resolver.FlushStep) {
	if len(steps) == 0 {
		return
	}
	c := colorizer()
	for _, s := range steps {
		if s.OK {
			fmt.Fprintf(env.Stdout, "%s %s\n", c.Green("[ok]"), s.Command)
		} else {
			fmt.Fprintf(env.Stderr, "%s %s: %s\n", c.Red("[fail]"), s.Command, s.Message)
		}
	}
}

// flushFailureExit reports partial success: the config was written but a flush
// step failed. This is surfaced as a runtime error (exit 5) while making clear
// the file write itself succeeded.
func flushFailureExit(result resolver.ApplyResult) error {
	if !result.Written {
		return nil
	}
	for _, s := range result.FlushSteps {
		if !s.OK {
			return runtimeErrorf("configuration written to %s, but DNS cache flush failed (%s); run `sudo splitdns flush` to retry",
				result.Path, s.Command)
		}
	}
	return nil
}
