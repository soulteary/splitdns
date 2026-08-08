package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestJSONStableAndPure(t *testing.T) {
	v := map[string]any{"b": 2, "a": 1, "c": []string{"x", "y"}}
	var buf1, buf2 bytes.Buffer
	if err := JSON(&buf1, v); err != nil {
		t.Fatal(err)
	}
	if err := JSON(&buf2, v); err != nil {
		t.Fatal(err)
	}
	if buf1.String() != buf2.String() {
		t.Errorf("JSON not stable:\n%s\n%s", buf1.String(), buf2.String())
	}
	// Map keys must be sorted deterministically by encoding/json.
	if strings.Index(buf1.String(), `"a"`) > strings.Index(buf1.String(), `"b"`) {
		t.Error("JSON keys not sorted")
	}
	// No ANSI color escapes leak into JSON.
	if strings.Contains(buf1.String(), "\033[") {
		t.Error("JSON contains ANSI escapes")
	}
}

func TestTableAlignment(t *testing.T) {
	var buf bytes.Buffer
	Table(&buf, []string{"NAME", "PORT"}, [][]string{{"lab.dev", "53"}, {"a", "5353"}})
	out := buf.String()
	if !strings.Contains(out, "NAME") || !strings.Contains(out, "lab.dev") {
		t.Errorf("table missing content: %s", out)
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 4 {
		t.Errorf("expected header+separator+2 rows = 4 lines, got %d: %q", len(lines), lines)
	}
}

func TestColorizerDisabled(t *testing.T) {
	c := Colorizer{Enabled: false}
	if c.Red("x") != "x" {
		t.Error("disabled colorizer must not add escapes")
	}
	c2 := Colorizer{Enabled: true}
	if !strings.Contains(c2.Red("x"), "\033[") {
		t.Error("enabled colorizer must add escapes")
	}
}

func TestColorizerAllColors(t *testing.T) {
	disabled := Colorizer{Enabled: false}
	enabled := Colorizer{Enabled: true}

	cases := []struct {
		name string
		fn   func(Colorizer, string) string
		code string
	}{
		{"Green", func(c Colorizer, s string) string { return c.Green(s) }, ansiGreen},
		{"Red", func(c Colorizer, s string) string { return c.Red(s) }, ansiRed},
		{"Yellow", func(c Colorizer, s string) string { return c.Yellow(s) }, ansiYellow},
		{"Cyan", func(c Colorizer, s string) string { return c.Cyan(s) }, ansiCyan},
		{"Bold", func(c Colorizer, s string) string { return c.Bold(s) }, ansiBold},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.fn(disabled, "x"); got != "x" {
				t.Errorf("disabled %s = %q, want %q", tc.name, got, "x")
			}
			want := tc.code + "x" + ansiReset
			if got := tc.fn(enabled, "x"); got != want {
				t.Errorf("enabled %s = %q, want %q", tc.name, got, want)
			}
		})
	}
}

func TestErrorReportFprint(t *testing.T) {
	t.Run("what only", func(t *testing.T) {
		var buf bytes.Buffer
		ErrorReport{What: "boom"}.Fprint(&buf, Colorizer{Enabled: false})
		out := buf.String()
		if !strings.Contains(out, "Error: boom") {
			t.Errorf("missing What: %q", out)
		}
		if strings.Contains(out, "Why:") || strings.Contains(out, "Next:") {
			t.Errorf("empty Why/Next should be omitted: %q", out)
		}
	})

	t.Run("full report", func(t *testing.T) {
		var buf bytes.Buffer
		ErrorReport{What: "boom", Why: "root cause", Next: "try this"}.Fprint(&buf, Colorizer{Enabled: false})
		out := buf.String()
		for _, want := range []string{"Error: boom", "Why:  root cause", "Next: try this"} {
			if !strings.Contains(out, want) {
				t.Errorf("missing %q in output: %q", want, out)
			}
		}
	})

	t.Run("colored", func(t *testing.T) {
		var buf bytes.Buffer
		ErrorReport{What: "boom", Next: "fix"}.Fprint(&buf, Colorizer{Enabled: true})
		if !strings.Contains(buf.String(), "\033[") {
			t.Errorf("expected ANSI escapes when colorizer enabled: %q", buf.String())
		}
	})
}
