package cmd

import (
	"fmt"

	"github.com/soulteary/splitdns/internal/config"
	"github.com/soulteary/splitdns/internal/domain"
	"github.com/soulteary/splitdns/internal/resolver"
	"github.com/spf13/cobra"
)

type addSetFlags struct {
	nameservers []string
	port        string
	searchOrder string
	timeout     string
	dryRun      bool
	force       bool
	noFlush     bool
	backupDir   string
}

func newAddCmd() *cobra.Command {
	f := &addSetFlags{}
	cmd := &cobra.Command{
		Use:   "add <domain>",
		Short: "Add a new resolver entry for a domain suffix",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAdd(args[0], f)
		},
	}
	bindAddSetFlags(cmd, f, true)
	return cmd
}

func bindAddSetFlags(cmd *cobra.Command, f *addSetFlags, isAdd bool) {
	fs := cmd.Flags()
	if isAdd {
		fs.StringArrayVar(&f.nameservers, "nameserver", []string{"127.0.0.1"}, "nameserver IP (repeatable)")
		fs.StringVar(&f.port, "port", "53", "DNS server port")
	} else {
		fs.StringArrayVar(&f.nameservers, "nameserver", nil, "nameserver IP (repeatable)")
		fs.StringVar(&f.port, "port", "", "DNS server port")
	}
	fs.StringVar(&f.searchOrder, "search-order", "", "search_order value")
	fs.StringVar(&f.timeout, "timeout", "", "timeout value in seconds")
	fs.BoolVar(&f.force, "force", false, "allow overwrite and relax safety warnings")
	fs.BoolVar(&f.noFlush, "no-flush", false, "do not flush DNS caches after writing")
	fs.StringVar(&f.backupDir, "backup-dir", "", "directory for pre-overwrite backups (default: system temp)")
}

func runAdd(rawDomain string, f *addSetFlags) error {
	if err := requireMacOS(); err != nil {
		return err
	}

	f.dryRun = f.dryRun || flagDryRun

	norm, err := domain.Normalize(rawDomain, f.force, f.force)
	if err != nil {
		return configErrorf("%v", err)
	}
	emitWarnings(norm.Warnings)

	for _, ns := range f.nameservers {
		if err := config.ValidateNameserver(ns); err != nil {
			return configErrorf("%v", err)
		}
	}
	if f.port != "" {
		if err := config.ValidatePort(f.port); err != nil {
			return configErrorf("%v", err)
		}
	}

	name := domain.FileName(norm.Normalized)

	if !f.dryRun {
		if err := requireRoot(fmt.Sprintf("splitdns add %s", rawDomain)); err != nil {
			return err
		}
	}

	exists, err := resolver.Exists(env.ResolverDir, name)
	if err != nil {
		return configErrorf("%v", err)
	}
	if exists && !f.force {
		return configErrorf("resolver for %q already exists; use --force to overwrite or `splitdns set` to modify", norm.Normalized)
	}

	cfg := config.Build(config.Fields{
		Domain:      norm.Normalized,
		Nameservers: f.nameservers,
		Port:        f.port,
		SearchOrder: f.searchOrder,
		Timeout:     f.timeout,
	})
	content := config.Serialize(cfg)

	return applyWrite(name, content, f, rawDomain)
}

func emitWarnings(ws []domain.Warning) {
	if flagQuiet {
		return
	}
	c := colorizer()
	for _, w := range ws {
		fmt.Fprintln(env.Stderr, c.Yellow("Warning: ")+w.Message)
	}
}
