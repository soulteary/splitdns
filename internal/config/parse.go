package config

import (
	"bufio"
	"fmt"
	"strings"
)

// Parse reads a resolver file body and returns a Config. It never silently
// drops lines: comments, blanks, unknown directives and duplicate nameservers
// are all preserved, and unrecognized keyword lines add a warning.
func Parse(body string) (*Config, error) {
	cfg := &Config{}
	scanner := bufio.NewScanner(strings.NewReader(body))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	seenKeyword := false

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		switch {
		case trimmed == "":
			d := Directive{Kind: KindBlank, Raw: line}
			cfg.appendLine(d, seenKeyword)
		case strings.HasPrefix(trimmed, "#"):
			if strings.TrimSpace(strings.TrimPrefix(trimmed, "#")) == strings.TrimSpace(strings.TrimPrefix(ManagedMarker, "#")) {
				cfg.Managed = true
			}
			d := Directive{Kind: KindComment, Raw: line}
			cfg.appendLine(d, seenKeyword)
		default:
			seenKeyword = true
			fields := strings.Fields(trimmed)
			key := strings.ToLower(fields[0])
			values := fields[1:]
			kind := KindKeyword
			if !knownDirectives[key] {
				kind = KindUnknown
				cfg.Warnings = append(cfg.Warnings, fmt.Sprintf("unrecognized directive %q preserved as-is", key))
			}
			d := Directive{Kind: kind, Key: key, Values: append([]string(nil), values...), Raw: line}
			cfg.Directives = append(cfg.Directives, d)
			if kind == KindKeyword && key == "domain" && cfg.Domain == "" && len(values) > 0 {
				cfg.Domain = values[0]
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read resolver file: %w", err)
	}
	return cfg, nil
}

// appendLine routes comment/blank lines to HeaderNotes when they precede the
// first keyword, otherwise into the ordered Directives slice.
func (c *Config) appendLine(d Directive, seenKeyword bool) {
	if seenKeyword {
		c.Directives = append(c.Directives, d)
	} else {
		c.HeaderNotes = append(c.HeaderNotes, d)
	}
}
