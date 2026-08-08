package output

const (
	ansiReset  = "\033[0m"
	ansiRed    = "\033[31m"
	ansiGreen  = "\033[32m"
	ansiYellow = "\033[33m"
	ansiCyan   = "\033[36m"
	ansiBold   = "\033[1m"
)

// Colorizer applies ANSI colors when enabled, or returns text unchanged.
type Colorizer struct {
	Enabled bool
}

func (c Colorizer) wrap(code, s string) string {
	if !c.Enabled {
		return s
	}
	return code + s + ansiReset
}

// Green wraps s in green when color is enabled.
func (c Colorizer) Green(s string) string { return c.wrap(ansiGreen, s) }

// Red wraps s in red when color is enabled.
func (c Colorizer) Red(s string) string { return c.wrap(ansiRed, s) }

// Yellow wraps s in yellow when color is enabled.
func (c Colorizer) Yellow(s string) string { return c.wrap(ansiYellow, s) }

// Cyan wraps s in cyan when color is enabled.
func (c Colorizer) Cyan(s string) string { return c.wrap(ansiCyan, s) }

// Bold wraps s in bold when color is enabled.
func (c Colorizer) Bold(s string) string { return c.wrap(ansiBold, s) }
