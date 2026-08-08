package cmd

import (
	"fmt"

	"github.com/soulteary/splitdns/internal/domain"
	"github.com/soulteary/splitdns/internal/output"
	"github.com/soulteary/splitdns/internal/resolver"
	"github.com/spf13/cobra"
)

func newShowCmd() *cobra.Command {
	var raw bool
	cmd := &cobra.Command{
		Use:   "show <domain>",
		Short: "Show a resolver entry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runShow(args[0], raw)
		},
	}
	cmd.Flags().BoolVar(&raw, "raw", false, "print the original file contents verbatim")
	return cmd
}

func runShow(rawDomain string, raw bool) error {
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

	cfg, rawBody, err := resolver.Read(env.ResolverDir, name)
	if err != nil {
		return configErrorf("%v", err)
	}
	path, _ := resolver.Path(env.ResolverDir, name)

	if raw {
		fmt.Fprint(env.Stdout, rawBody)
		return nil
	}

	if flagJSON {
		payload := struct {
			Name        string   `json:"name"`
			Path        string   `json:"path"`
			Domain      string   `json:"domain"`
			Nameservers []string `json:"nameservers"`
			Port        string   `json:"port,omitempty"`
			Managed     bool     `json:"managed"`
		}{
			Name:        name,
			Path:        path,
			Domain:      cfg.Domain,
			Nameservers: cfg.Nameservers(),
			Managed:     cfg.Managed,
		}
		if port, ok := cfg.Port(); ok {
			payload.Port = port
		}
		if payload.Nameservers == nil {
			payload.Nameservers = []string{}
		}
		if err := output.JSON(env.Stdout, payload); err != nil {
			return runtimeErrorf("%v", err)
		}
		return nil
	}

	c := colorizer()
	fmt.Fprintf(env.Stdout, "%s %s\n", c.Bold("File:"), path)
	fmt.Fprintf(env.Stdout, "%s %s\n", c.Bold("Domain:"), cfg.Domain)
	fmt.Fprintf(env.Stdout, "%s %v\n", c.Bold("Nameservers:"), cfg.Nameservers())
	if port, ok := cfg.Port(); ok {
		fmt.Fprintf(env.Stdout, "%s %s\n", c.Bold("Port:"), port)
	}
	fmt.Fprintf(env.Stdout, "%s %t\n", c.Bold("Managed:"), cfg.Managed)
	return nil
}
