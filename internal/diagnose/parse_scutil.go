package diagnose

import (
	"bufio"
	"strings"
)

// ScutilResolver represents one resolver block from `scutil --dns`.
type ScutilResolver struct {
	Index       int
	Domain      string
	Nameservers []string
	Port        string
	Options     string
	Timeout     string
	Flags       string
	Order       string
	Scoped      bool
}

// ParseScutilDNS parses `scutil --dns` output into resolver blocks. The
// "for scoped queries" section is flagged via Scoped.
func ParseScutilDNS(out string) []ScutilResolver {
	var resolvers []ScutilResolver
	var cur *ScutilResolver
	scoped := false

	flush := func() {
		if cur != nil {
			resolvers = append(resolvers, *cur)
			cur = nil
		}
	}

	scanner := bufio.NewScanner(strings.NewReader(out))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "DNS configuration (for scoped queries)") {
			flush()
			scoped = true
			continue
		}
		if strings.HasPrefix(trimmed, "resolver #") {
			flush()
			cur = &ScutilResolver{Scoped: scoped}
			cur.Index = atoiSafe(strings.TrimPrefix(trimmed, "resolver #"))
			continue
		}
		if cur == nil {
			continue
		}

		key, val, ok := splitKeyVal(trimmed)
		if !ok {
			continue
		}
		switch {
		case key == "domain":
			cur.Domain = val
		case strings.HasPrefix(key, "nameserver"):
			cur.Nameservers = append(cur.Nameservers, val)
		case key == "port":
			cur.Port = val
		case key == "options":
			cur.Options = val
		case key == "timeout":
			cur.Timeout = val
		case key == "flags":
			cur.Flags = val
		case key == "order":
			cur.Order = val
		}
	}
	flush()
	return resolvers
}

// splitKeyVal splits a "key : value" line on the first " : " separator so
// that IPv6 values (which contain colons) are preserved intact.
func splitKeyVal(line string) (key, val string, ok bool) {
	idx := strings.Index(line, " : ")
	if idx < 0 {
		// Tolerate a trailing "key :" with an empty value.
		if strings.HasSuffix(line, " :") {
			return strings.TrimSpace(strings.TrimSuffix(line, ":")), "", true
		}
		return "", "", false
	}
	key = strings.TrimSpace(line[:idx])
	val = strings.TrimSpace(line[idx+3:])
	if key == "" {
		return "", "", false
	}
	return key, val, true
}

func atoiSafe(s string) int {
	n := 0
	for _, r := range strings.TrimSpace(s) {
		if r < '0' || r > '9' {
			break
		}
		n = n*10 + int(r-'0')
	}
	return n
}

// FindScutilResolverForDomain returns the non-scoped resolver whose domain
// exactly equals the given domain, and whether one was found.
func FindScutilResolverForDomain(resolvers []ScutilResolver, domain string) (ScutilResolver, bool) {
	target := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(domain), "."))
	for _, r := range resolvers {
		if r.Scoped {
			continue
		}
		if strings.ToLower(strings.TrimSuffix(r.Domain, ".")) == target {
			return r, true
		}
	}
	return ScutilResolver{}, false
}
