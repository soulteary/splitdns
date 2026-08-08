package config

import (
	"fmt"
	"net"
	"strconv"
)

// ValidateNameserver checks that ns is a valid IPv4 or IPv6 address literal.
func ValidateNameserver(ns string) error {
	if net.ParseIP(ns) == nil {
		return fmt.Errorf("nameserver %q is not a valid IPv4 or IPv6 address", ns)
	}
	return nil
}

// ValidatePort checks that a port string is an integer in [1, 65535].
func ValidatePort(port string) error {
	n, err := strconv.Atoi(port)
	if err != nil {
		return fmt.Errorf("port %q is not a valid number", port)
	}
	if n < 1 || n > 65535 {
		return fmt.Errorf("port %d is out of range 1-65535", n)
	}
	return nil
}

// Fields describes the mutable resolver settings used by add/set.
type Fields struct {
	Domain      string
	Nameservers []string
	Port        string
	SearchOrder string
	Timeout     string
}

// Build constructs a managed Config from the given fields. Only non-empty
// fields produce directives, letting callers add settings incrementally.
func Build(f Fields) *Config {
	cfg := &Config{}
	EnsureManaged(cfg)
	if f.Domain != "" {
		cfg.Directives = append(cfg.Directives, kw("domain", f.Domain))
		cfg.Domain = f.Domain
	}
	for _, ns := range f.Nameservers {
		cfg.Directives = append(cfg.Directives, kw("nameserver", ns))
	}
	if f.Port != "" {
		cfg.Directives = append(cfg.Directives, kw("port", f.Port))
	}
	if f.SearchOrder != "" {
		cfg.Directives = append(cfg.Directives, kw("search_order", f.SearchOrder))
	}
	if f.Timeout != "" {
		cfg.Directives = append(cfg.Directives, kw("timeout", f.Timeout))
	}
	return cfg
}

func kw(key string, values ...string) Directive {
	return Directive{Kind: KindKeyword, Key: key, Values: append([]string(nil), values...)}
}

// SetKeyword replaces all directives with the given key by a single directive
// carrying the provided values, preserving position of the first occurrence.
// If the key is absent, the directive is appended. When values is empty the
// key is removed entirely.
func SetKeyword(c *Config, key string, values ...string) {
	firstIdx := -1
	var kept []Directive
	for _, d := range c.Directives {
		if d.Kind == KindKeyword && d.Key == key {
			if firstIdx == -1 {
				firstIdx = len(kept)
				if len(values) > 0 {
					kept = append(kept, kw(key, values...))
				}
			}
			continue
		}
		kept = append(kept, d)
	}
	if firstIdx == -1 && len(values) > 0 {
		kept = append(kept, kw(key, values...))
	}
	c.Directives = kept
	if key == "domain" {
		if len(values) > 0 {
			c.Domain = values[0]
		} else {
			c.Domain = ""
		}
	}
}

// SetNameservers replaces all nameserver directives with one per address,
// inserted at the position of the first existing nameserver (or appended).
func SetNameservers(c *Config, nameservers []string) {
	firstIdx := -1
	var kept []Directive
	inserted := false
	for _, d := range c.Directives {
		if d.Kind == KindKeyword && d.Key == "nameserver" {
			if firstIdx == -1 {
				firstIdx = len(kept)
				for _, ns := range nameservers {
					kept = append(kept, kw("nameserver", ns))
				}
				inserted = true
			}
			continue
		}
		kept = append(kept, d)
	}
	if !inserted {
		for _, ns := range nameservers {
			kept = append(kept, kw("nameserver", ns))
		}
	}
	c.Directives = kept
}
