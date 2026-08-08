package diagnose

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/soulteary/splitdns/internal/config"
	"github.com/soulteary/splitdns/internal/domain"
	"github.com/soulteary/splitdns/internal/system"
)

// Status classifies a check outcome.
type Status string

// Diagnostic status values.
const (
	StatusOK    Status = "OK"
	StatusWarn  Status = "WARN"
	StatusError Status = "ERROR"
)

// Check is a single diagnostic result.
type Check struct {
	Name    string `json:"name"`
	Status  Status `json:"status"`
	Message string `json:"message"`
	Hint    string `json:"hint,omitempty"`
}

// CheckReport is the aggregate result of running checks.
type CheckReport struct {
	Domain string  `json:"domain,omitempty"`
	Checks []Check `json:"checks"`
}

// HasError reports whether any check ended in ERROR status.
func (r CheckReport) HasError() bool {
	for _, c := range r.Checks {
		if c.Status == StatusError {
			return true
		}
	}
	return false
}

// RunChecks performs environment and configuration checks. When domainFilter is
// non-empty, checks focus on the matching resolver; otherwise all resolver
// files are validated. DNS reachability probes use the given timeout.
func RunChecks(env system.Env, domainFilter string, timeout time.Duration) CheckReport {
	report := CheckReport{Domain: domainFilter}
	add := func(name string, status Status, msg, hint string) {
		report.Checks = append(report.Checks, Check{Name: name, Status: status, Message: msg, Hint: hint})
	}

	if env.GOOS != "darwin" {
		add("platform", StatusError, fmt.Sprintf("running on %s, not macOS", env.GOOS),
			"splitdns is macOS-only")
		return report
	}
	add("platform", StatusOK, "running on macOS", "")

	dirInfo, err := os.Stat(env.ResolverDir)
	if err != nil {
		if os.IsNotExist(err) {
			add("resolver-dir", StatusWarn, fmt.Sprintf("%s does not exist yet", env.ResolverDir),
				"it will be created when you add the first entry (requires sudo)")
			return report
		}
		add("resolver-dir", StatusError, fmt.Sprintf("cannot stat %s: %v", env.ResolverDir, err), "")
		return report
	}
	if !dirInfo.IsDir() {
		add("resolver-dir", StatusError, fmt.Sprintf("%s is not a directory", env.ResolverDir), "")
		return report
	}
	add("resolver-dir", StatusOK, fmt.Sprintf("%s exists and is readable", env.ResolverDir), "")

	files := collectResolverFiles(env.ResolverDir, domainFilter)
	if domainFilter != "" && len(files) == 0 {
		add("resolver-file", StatusError, fmt.Sprintf("no resolver file for %q", domainFilter),
			fmt.Sprintf("create it with: sudo splitdns add %s", domainFilter))
		return report
	}

	scutilResolvers := parseScutil(env)
	domainsSeen := map[string]string{}
	configFingerprints := map[string][]string{}

	for _, name := range files {
		checkOneFile(env, name, timeout, scutilResolvers, add, domainsSeen, configFingerprints)
	}

	reportOverlaps(files, add)
	reportIdenticalConfigs(configFingerprints, add)

	if domainFilter != "" {
		checkHostsFile(env, domainFilter, add)
	}

	return report
}

func collectResolverFiles(dir, domainFilter string) []string {
	if domainFilter != "" {
		norm := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(domainFilter), "."))
		full := filepath.Join(dir, norm)
		if _, err := os.Lstat(full); err == nil {
			return []string{norm}
		}
		return nil
	}
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

func checkOneFile(env system.Env, name string, timeout time.Duration, scutilResolvers []ScutilResolver,
	add func(string, Status, string, string), domainsSeen map[string]string, fingerprints map[string][]string) {

	full := filepath.Join(env.ResolverDir, name)
	label := "file:" + name

	if _, derr := domain.Normalize(name, true, true); derr != nil {
		add(label+":name", StatusWarn, fmt.Sprintf("filename %q is not a valid domain: %v", name, derr), "")
	}

	info, err := os.Lstat(full)
	if err != nil {
		add(label, StatusError, fmt.Sprintf("cannot stat %s: %v", full, err), "")
		return
	}
	if info.Mode()&os.ModeSymlink != 0 {
		add(label+":type", StatusError, fmt.Sprintf("%s is a symlink", full),
			"remove the symlink; resolver files must be regular files")
		return
	}
	if !info.Mode().IsRegular() {
		add(label+":type", StatusError, fmt.Sprintf("%s is not a regular file", full), "")
		return
	}
	add(label+":type", StatusOK, "regular file", "")

	perm := info.Mode().Perm()
	if perm&0o022 != 0 {
		add(label+":perms", StatusWarn, fmt.Sprintf("%s is group/world writable (%o)", full, perm),
			"consider: sudo chmod 0644 "+full)
	} else {
		add(label+":perms", StatusOK, fmt.Sprintf("permissions %o", perm), "")
	}

	raw, err := os.ReadFile(full)
	if err != nil {
		add(label+":read", StatusError, fmt.Sprintf("cannot read %s: %v", full, err), "")
		return
	}
	cfg, err := config.Parse(string(raw))
	if err != nil {
		add(label+":syntax", StatusError, fmt.Sprintf("parse error: %v", err), "")
		return
	}
	for _, w := range cfg.Warnings {
		add(label+":syntax", StatusWarn, w, "")
	}

	nss := cfg.Nameservers()
	if len(nss) == 0 {
		add(label+":nameserver", StatusError, "no nameserver directive", "add a nameserver line")
	}
	for _, ns := range nss {
		if net.ParseIP(ns) == nil {
			add(label+":nameserver", StatusError, fmt.Sprintf("invalid nameserver %q", ns), "")
		}
	}

	port := "53"
	if p, ok := cfg.Port(); ok {
		port = p
		if err := config.ValidatePort(p); err != nil {
			add(label+":port", StatusError, err.Error(), "")
		}
	}

	dom := cfg.Domain
	if dom == "" {
		dom = name
	}
	if strings.EqualFold(lastLabel(dom), "local") {
		add(label+":local", StatusWarn, fmt.Sprintf("%q uses the reserved .local suffix", dom),
			"this may interfere with mDNS/Bonjour discovery")
	}

	if prev, ok := domainsSeen[dom]; ok {
		add(label+":duplicate-domain", StatusWarn,
			fmt.Sprintf("domain %q also configured in %s", dom, prev), "")
	} else {
		domainsSeen[dom] = name
	}

	fp := fingerprint(nss, port)
	fingerprints[fp] = append(fingerprints[fp], name)

	// scutil load status
	if _, found := FindScutilResolverForDomain(scutilResolvers, dom); found {
		add(label+":scutil", StatusOK, fmt.Sprintf("resolver for %q is loaded by scutil", dom), "")
	} else {
		add(label+":scutil", StatusWarn, fmt.Sprintf("scutil --dns does not list a resolver for %q", dom),
			"try: sudo splitdns flush")
	}

	// DNS reachability via a real DNS probe (not a bare TCP connect).
	for _, ns := range nss {
		if net.ParseIP(ns) == nil {
			continue
		}
		res := Probe(ns, port, probeName(dom), timeout)
		if res.Err != nil {
			add(label+":reachable", StatusWarn,
				fmt.Sprintf("no DNS response from %s:%s (%v)", ns, port, res.Err),
				"ensure the DNS server is running and listening")
		} else {
			add(label+":reachable", StatusOK,
				fmt.Sprintf("%s:%s answered a DNS query", ns, port), "")
		}
	}
}

func reportOverlaps(files []string, add func(string, Status, string, string)) {
	for i := 0; i < len(files); i++ {
		for j := 0; j < len(files); j++ {
			if i == j {
				continue
			}
			if domain.Match(files[i], files[j]) && files[i] != files[j] {
				add("overlap", StatusWarn,
					fmt.Sprintf("%q is a subdomain of %q; the more specific suffix wins", files[i], files[j]), "")
			}
		}
	}
}

func reportIdenticalConfigs(fingerprints map[string][]string, add func(string, Status, string, string)) {
	for _, names := range fingerprints {
		if len(names) > 1 {
			sort.Strings(names)
			add("identical", StatusWarn,
				fmt.Sprintf("identical nameserver/port config shared by: %s", strings.Join(names, ", ")), "")
		}
	}
}

func checkHostsFile(env system.Env, domainFilter string, add func(string, Status, string, string)) {
	data, err := os.ReadFile(env.HostsFile)
	if err != nil {
		return
	}
	var matches []string
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		fields := strings.Fields(trimmed)
		for _, host := range fields[1:] {
			if domain.Match(host, domainFilter) {
				matches = append(matches, trimmed)
				break
			}
		}
	}
	if len(matches) > 0 {
		add("hosts", StatusWarn,
			fmt.Sprintf("/etc/hosts has %d entry(ies) affecting %q", len(matches), domainFilter),
			strings.Join(matches, " | "))
	} else {
		add("hosts", StatusOK, fmt.Sprintf("no /etc/hosts entries affect %q", domainFilter), "")
	}
}

func parseScutil(env system.Env) []ScutilResolver {
	res := env.Runner.Run("scutil", "--dns")
	if res.Err != nil && res.Stdout == "" {
		return nil
	}
	return ParseScutilDNS(res.Stdout)
}

func fingerprint(nss []string, port string) string {
	cp := append([]string(nil), nss...)
	sort.Strings(cp)
	return strings.Join(cp, ",") + "@" + port
}

func lastLabel(dom string) string {
	dom = strings.TrimSuffix(strings.TrimSpace(dom), ".")
	parts := strings.Split(dom, ".")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

func probeName(dom string) string {
	// Use a representative name under the suffix for reachability probing.
	if strings.Contains(dom, ".") {
		return dom
	}
	return dom + "."
}

// PortInt parses a port string, defaulting to 53 on error.
func PortInt(port string) int {
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return 53
	}
	return n
}
