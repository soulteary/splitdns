// Package config parses and serializes macOS /etc/resolver configuration files
// while preserving ordering, comments, blank lines and unknown-but-valid
// directives.
package config

// ManagedMarker is the comment written to files managed by splitdns.
const ManagedMarker = "# Managed by splitdns"

// Directive kinds.
const (
	KindComment = "comment"
	KindBlank   = "blank"
	KindKeyword = "keyword"
	KindUnknown = "unknown"
)

// knownDirectives is the set of resolver(5) keyword directives splitdns
// recognizes. Unknown-but-plausible keywords are preserved as KindUnknown.
var knownDirectives = map[string]bool{
	"domain":       true,
	"nameserver":   true,
	"port":         true,
	"search":       true,
	"search_order": true,
	"sortlist":     true,
	"timeout":      true,
	"options":      true,
}

// Directive is a single logical line of a resolver file.
type Directive struct {
	// Kind is one of KindComment, KindBlank, KindKeyword, KindUnknown.
	Kind string
	// Key is the directive keyword for KindKeyword/KindUnknown lines.
	Key string
	// Values are the whitespace-separated arguments following the keyword.
	Values []string
	// Raw is the original line text (without trailing newline), used to
	// faithfully reproduce comments, blanks and unknown lines.
	Raw string
}

// Config is a parsed resolver file.
type Config struct {
	// Domain is the value of the first "domain" directive, if present.
	Domain string
	// HeaderNotes holds leading comment/blank lines before the first keyword.
	HeaderNotes []Directive
	// Directives holds keyword/unknown directives (and interleaved comments)
	// in original order.
	Directives []Directive
	// Managed reports whether the ManagedMarker was present.
	Managed bool
	// Warnings collects non-fatal parser advisories.
	Warnings []string
}

// Nameservers returns all nameserver values in order.
func (c *Config) Nameservers() []string {
	var out []string
	for _, d := range c.Directives {
		if d.Kind == KindKeyword && d.Key == "nameserver" {
			out = append(out, d.Values...)
		}
	}
	return out
}

// Port returns the first port directive value and whether it was present.
func (c *Config) Port() (string, bool) {
	for _, d := range c.Directives {
		if d.Kind == KindKeyword && d.Key == "port" && len(d.Values) > 0 {
			return d.Values[0], true
		}
	}
	return "", false
}

// FirstValue returns the first value of the first directive matching key.
func (c *Config) FirstValue(key string) (string, bool) {
	for _, d := range c.Directives {
		if d.Kind == KindKeyword && d.Key == key && len(d.Values) > 0 {
			return d.Values[0], true
		}
	}
	return "", false
}
