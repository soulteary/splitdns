package cmd

import (
	"fmt"
	"strings"

	"github.com/soulteary/splitdns/internal/output"
	"github.com/soulteary/splitdns/internal/resolver"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List resolver entries",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList()
		},
	}
}

func runList() error {
	entries, err := resolver.List(env.ResolverDir)
	if err != nil {
		return configErrorf("%v", err)
	}

	if flagJSON {
		if entries == nil {
			entries = []resolver.Entry{}
		}
		if err := output.JSON(env.Stdout, entries); err != nil {
			return runtimeErrorf("%v", err)
		}
		return nil
	}

	if len(entries) == 0 {
		if !flagQuiet {
			fmt.Fprintf(env.Stdout, "No resolver entries in %s\n", env.ResolverDir)
		}
		return nil
	}

	headers := []string{"NAME", "DOMAIN", "NAMESERVERS", "PORT", "MANAGED", "NOTE"}
	rows := make([][]string, 0, len(entries))
	for _, e := range entries {
		managed := "no"
		if e.Managed {
			managed = "yes"
		}
		rows = append(rows, []string{
			e.Name,
			e.Domain,
			strings.Join(e.Nameservers, ","),
			e.Port,
			managed,
			e.Warning,
		})
	}
	output.Table(env.Stdout, headers, rows)
	return nil
}
