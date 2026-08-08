package config

import "strings"

// Serialize renders a Config back to file text. Output is stable: the same
// Config always serializes to identical bytes. Ordering of header notes and
// directives is preserved exactly as parsed/constructed.
func Serialize(c *Config) string {
	var b strings.Builder
	for _, d := range c.HeaderNotes {
		writeLine(&b, d)
	}
	for _, d := range c.Directives {
		writeLine(&b, d)
	}
	return b.String()
}

func writeLine(b *strings.Builder, d Directive) {
	switch d.Kind {
	case KindComment, KindBlank:
		b.WriteString(d.Raw)
	default:
		b.WriteString(d.Key)
		for _, v := range d.Values {
			b.WriteByte(' ')
			b.WriteString(v)
		}
	}
	b.WriteByte('\n')
}

// EnsureManaged prepends the managed marker to the header notes if absent and
// marks the config as managed.
func EnsureManaged(c *Config) {
	if c.Managed {
		return
	}
	marker := Directive{Kind: KindComment, Raw: ManagedMarker}
	c.HeaderNotes = append([]Directive{marker}, c.HeaderNotes...)
	c.Managed = true
}
