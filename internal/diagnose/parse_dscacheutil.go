package diagnose

import (
	"bufio"
	"strings"
)

// DscacheEntry is a single host record block from `dscacheutil -q host`.
type DscacheEntry struct {
	Name      string
	Aliases   []string
	Addresses []string
}

// ParseDscacheHost parses `dscacheutil -q host -a name <host>` output. Each
// blank-line separated block becomes an entry; ip_address and ipv6_address
// lines accumulate into Addresses.
func ParseDscacheHost(out string) []DscacheEntry {
	var entries []DscacheEntry
	var cur *DscacheEntry

	flush := func() {
		if cur != nil && (cur.Name != "" || len(cur.Addresses) > 0) {
			entries = append(entries, *cur)
		}
		cur = nil
	}

	scanner := bufio.NewScanner(strings.NewReader(out))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			flush()
			continue
		}
		if cur == nil {
			cur = &DscacheEntry{}
		}
		key, val, ok := splitDscacheKV(line)
		if !ok {
			continue
		}
		switch key {
		case "name":
			cur.Name = val
		case "alias":
			cur.Aliases = append(cur.Aliases, strings.Fields(val)...)
		case "ip_address", "ipv6_address":
			if val != "" {
				cur.Addresses = append(cur.Addresses, val)
			}
		}
	}
	flush()
	return entries
}

func splitDscacheKV(line string) (key, val string, ok bool) {
	idx := strings.Index(line, ":")
	if idx < 0 {
		return "", "", false
	}
	return strings.TrimSpace(line[:idx]), strings.TrimSpace(line[idx+1:]), true
}

// DscacheAddresses returns the unique set of addresses across all entries, in
// first-seen order.
func DscacheAddresses(entries []DscacheEntry) []string {
	seen := make(map[string]bool)
	var out []string
	for _, e := range entries {
		for _, a := range e.Addresses {
			if !seen[a] {
				seen[a] = true
				out = append(out, a)
			}
		}
	}
	return out
}
