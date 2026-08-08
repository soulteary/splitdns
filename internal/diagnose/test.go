package diagnose

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/soulteary/splitdns/internal/config"
	"github.com/soulteary/splitdns/internal/domain"
	"github.com/soulteary/splitdns/internal/system"
)

// TestResult is the machine-readable outcome of `splitdns test <hostname>`.
type TestResult struct {
	Hostname         string   `json:"hostname"`
	MatchedSuffix    string   `json:"matchedSuffix,omitempty"`
	ResolverFile     string   `json:"resolverFile,omitempty"`
	Nameservers      []string `json:"nameservers"`
	Port             string   `json:"port,omitempty"`
	ScutilLoaded     bool     `json:"scutilLoaded"`
	DscacheAddresses []string `json:"dscacheAddresses"`
	DirectAddresses  []string `json:"directAddresses"`
	Consistent       bool     `json:"consistent"`
	DirectError      string   `json:"directError,omitempty"`
	Troubleshooting  []string `json:"troubleshooting,omitempty"`
}

// RunTest executes the three-layer resolution test for hostname:
// (1) dscacheutil host lookup, (2) scutil load status, (3) direct DNS probe.
func RunTest(env system.Env, hostname string, timeout time.Duration) TestResult {
	result := TestResult{Hostname: hostname, Nameservers: []string{}, DscacheAddresses: []string{}, DirectAddresses: []string{}}

	suffixes := listResolverDomains(env.ResolverDir)
	matched, ok := domain.LongestMatch(hostname, suffixes)
	if ok {
		result.MatchedSuffix = matched
		result.ResolverFile = filepath.Join(env.ResolverDir, matched)
		if cfg, _, err := readResolver(env.ResolverDir, matched); err == nil {
			result.Nameservers = cfg.Nameservers()
			if p, has := cfg.Port(); has {
				result.Port = p
			} else {
				result.Port = "53"
			}
		}
	}

	// Layer 1: dscacheutil host lookup.
	dsRes := env.Runner.Run("dscacheutil", "-q", "host", "-a", "name", hostname)
	result.DscacheAddresses = DscacheAddresses(ParseDscacheHost(dsRes.Stdout))

	// Layer 2: scutil load status.
	scutilRes := env.Runner.Run("scutil", "--dns")
	resolvers := ParseScutilDNS(scutilRes.Stdout)
	if matched != "" {
		_, result.ScutilLoaded = FindScutilResolverForDomain(resolvers, matched)
	}

	// Layer 3: direct DNS probe against the configured nameservers.
	if len(result.Nameservers) > 0 {
		port := result.Port
		if port == "" {
			port = "53"
		}
		var lastErr string
		seen := map[string]bool{}
		for _, ns := range result.Nameservers {
			pr := Probe(ns, port, hostname, timeout)
			if pr.Err != nil {
				lastErr = pr.Err.Error()
				continue
			}
			for _, a := range pr.Addresses {
				if !seen[a] {
					seen[a] = true
					result.DirectAddresses = append(result.DirectAddresses, a)
				}
			}
		}
		if len(result.DirectAddresses) == 0 && lastErr != "" {
			result.DirectError = lastErr
		}
	}

	result.Consistent = addrSetsEqual(result.DscacheAddresses, result.DirectAddresses)
	result.Troubleshooting = buildTroubleshooting(result)
	return result
}

func buildTroubleshooting(r TestResult) []string {
	var tips []string
	if r.MatchedSuffix == "" {
		tips = append(tips, fmt.Sprintf("No resolver suffix matches %q; add one with: sudo splitdns add <suffix>", r.Hostname))
		return tips
	}
	if !r.ScutilLoaded {
		tips = append(tips, "scutil has not loaded this resolver yet; run: sudo splitdns flush")
	}
	if r.DirectError != "" && len(r.DirectAddresses) == 0 {
		tips = append(tips, fmt.Sprintf("Direct DNS query failed (%s); verify the DNS server at %v:%s is running",
			r.DirectError, r.Nameservers, r.Port))
	}
	if len(r.DirectAddresses) > 0 && len(r.DscacheAddresses) == 0 {
		tips = append(tips, "The DNS server answered directly but macOS cache did not; try: sudo splitdns flush")
	}
	if len(r.DirectAddresses) > 0 && !r.Consistent && len(r.DscacheAddresses) > 0 {
		tips = append(tips, "System resolution differs from the direct query; another resolver may take precedence (check overlapping suffixes and /etc/hosts)")
	}
	return tips
}

func listResolverDomains(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	return names
}

func readResolver(dir, name string) (*config.Config, string, error) {
	full := filepath.Join(dir, name)
	data, err := os.ReadFile(full)
	if err != nil {
		return nil, "", err
	}
	cfg, err := config.Parse(string(data))
	return cfg, string(data), err
}

func addrSetsEqual(a, b []string) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	if len(a) == 0 || len(b) == 0 {
		return false
	}
	na := normalizeAddrs(a)
	nb := normalizeAddrs(b)
	if len(na) != len(nb) {
		return false
	}
	for k := range na {
		if !nb[k] {
			return false
		}
	}
	return true
}

func normalizeAddrs(in []string) map[string]bool {
	m := make(map[string]bool)
	for _, s := range in {
		m[strings.ToLower(strings.TrimSpace(s))] = true
	}
	return m
}
