package domain

import "strings"

// Match reports whether the given hostname is covered by the domain suffix.
// A hostname matches when it equals the suffix or ends with "."+suffix.
func Match(hostname, suffix string) bool {
	h := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(hostname), "."))
	s := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(suffix), "."))
	if h == "" || s == "" {
		return false
	}
	if h == s {
		return true
	}
	return strings.HasSuffix(h, "."+s)
}

// LongestMatch returns the most specific (longest, by label count then length)
// suffix from suffixes that matches hostname. It returns the matched suffix and
// true, or an empty string and false when nothing matches.
func LongestMatch(hostname string, suffixes []string) (string, bool) {
	best := ""
	bestLabels := -1
	for _, s := range suffixes {
		if !Match(hostname, s) {
			continue
		}
		norm := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(s), "."))
		labels := strings.Count(norm, ".") + 1
		if labels > bestLabels || (labels == bestLabels && len(norm) > len(best)) {
			best = norm
			bestLabels = labels
		}
	}
	if bestLabels == -1 {
		return "", false
	}
	return best, true
}
