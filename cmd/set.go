package cmd

import (
	"fmt"

	"github.com/soulteary/splitdns/internal/config"
	"github.com/soulteary/splitdns/internal/domain"
	"github.com/soulteary/splitdns/internal/resolver"
	"github.com/spf13/cobra"
)

func newSetCmd() *cobra.Command {
	f := &addSetFlags{}
	cmd := &cobra.Command{
		Use:   "set <domain>",
		Short: "Update fields of an existing resolver entry",
		Example: `  # Change the nameserver for an existing entry
  sudo splitdns set corp.example.com --nameserver 10.0.0.99

  # Update several fields at once
  sudo splitdns set internal.dev --port 5353 --timeout 5

  # Preview the update without writing anything
  splitdns set lab.example.com --nameserver 127.0.0.1 --dry-run`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSet(cmd, args[0], f)
		},
	}
	bindAddSetFlags(cmd, f, false)
	return cmd
}

func runSet(cmd *cobra.Command, rawDomain string, f *addSetFlags) error {
	if err := requireMacOS(); err != nil {
		return err
	}

	f.dryRun = f.dryRun || flagDryRun

	norm, err := domain.Normalize(rawDomain, f.force, f.force)
	if err != nil {
		return configErrorf("%v", err)
	}
	emitWarnings(norm.Warnings)
	name := domain.FileName(norm.Normalized)

	exists, err := resolver.Exists(env.ResolverDir, name)
	if err != nil {
		return configErrorf("%v", err)
	}
	if !exists {
		return configErrorf("resolver for %q does not exist; use `splitdns add` to create it", norm.Normalized)
	}

	cfg, _, err := resolver.Read(env.ResolverDir, name)
	if err != nil {
		return configErrorf("%v", err)
	}
	config.EnsureManaged(cfg)

	fs := cmd.Flags()

	if fs.Changed("nameserver") {
		for _, ns := range f.nameservers {
			if err := config.ValidateNameserver(ns); err != nil {
				return configErrorf("%v", err)
			}
		}
		config.SetNameservers(cfg, f.nameservers)
	}
	if fs.Changed("port") {
		if err := config.ValidatePort(f.port); err != nil {
			return configErrorf("%v", err)
		}
		config.SetKeyword(cfg, "port", f.port)
	}
	if fs.Changed("search-order") {
		config.SetKeyword(cfg, "search_order", f.searchOrder)
	}
	if fs.Changed("timeout") {
		config.SetKeyword(cfg, "timeout", f.timeout)
	}

	if !fs.Changed("nameserver") && !fs.Changed("port") && !fs.Changed("search-order") && !fs.Changed("timeout") {
		return argErrorf("no fields specified to update; provide at least one of --nameserver/--port/--search-order/--timeout")
	}

	content := config.Serialize(cfg)

	if !f.dryRun {
		if err := requireRoot(fmt.Sprintf("splitdns set %s", rawDomain)); err != nil {
			return err
		}
	}

	// set always modifies an existing file; force the overwrite path while
	// still taking a backup.
	f.force = true
	return applyWrite(name, content, f, rawDomain)
}
