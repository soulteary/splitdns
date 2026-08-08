package domain

import "testing"

func TestNormalize(t *testing.T) {
	tests := []struct {
		name             string
		input            string
		allowSingleLabel bool
		allowLocal       bool
		want             string
		wantErr          bool
		wantWarnings     int
	}{
		{name: "simple", input: "Example.COM", want: "example.com"},
		{name: "trailing dot stripped", input: "example.com.", want: "example.com"},
		{name: "multi label", input: "a.b.example.com", want: "a.b.example.com"},
		{name: "empty", input: "", wantErr: true},
		{name: "whitespace", input: "exa mple.com", wantErr: true},
		{name: "tab", input: "example\t.com", wantErr: true},
		{name: "slash path traversal", input: "../etc/passwd", wantErr: true},
		{name: "backslash", input: "example\\com", wantErr: true},
		{name: "double dot", input: "a..b.com", wantErr: true},
		{name: "leading dot", input: ".example.com", wantErr: true},
		{name: "nul", input: "exam\x00ple.com", wantErr: true},
		{name: "control char", input: "exam\x01ple.com", wantErr: true},
		{name: "single label rejected", input: "corp", wantErr: true},
		{name: "single label allowed", input: "corp", allowSingleLabel: true, want: "corp", wantWarnings: 1},
		{name: "local rejected", input: "dev.local", wantErr: true},
		{name: "local allowed", input: "dev.local", allowLocal: true, want: "dev.local", wantWarnings: 1},
		{name: "underscore invalid", input: "ex_ample.com", wantErr: true},
		{name: "leading hyphen label", input: "-bad.com", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := Normalize(tt.input, tt.allowSingleLabel, tt.allowLocal)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q, got none (result %q)", tt.input, res.Normalized)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tt.input, err)
			}
			if res.Normalized != tt.want {
				t.Errorf("normalized = %q, want %q", res.Normalized, tt.want)
			}
			if len(res.Warnings) != tt.wantWarnings {
				t.Errorf("warnings = %d, want %d (%v)", len(res.Warnings), tt.wantWarnings, res.Warnings)
			}
		})
	}
}

func TestMatch(t *testing.T) {
	cases := []struct {
		host, suffix string
		want         bool
	}{
		{"api.example.com", "example.com", true},
		{"example.com", "example.com", true},
		{"notexample.com", "example.com", false},
		{"a.b.example.com", "example.com", true},
		{"example.com.", "example.com", true},
		{"example.com", "com", true},
		{"", "example.com", false},
	}
	for _, c := range cases {
		if got := Match(c.host, c.suffix); got != c.want {
			t.Errorf("Match(%q, %q) = %v, want %v", c.host, c.suffix, got, c.want)
		}
	}
}

func TestLongestMatch(t *testing.T) {
	suffixes := []string{"example.com", "corp.example.com", "dev.corp.example.com"}
	got, ok := LongestMatch("host.dev.corp.example.com", suffixes)
	if !ok || got != "dev.corp.example.com" {
		t.Fatalf("LongestMatch = %q,%v want dev.corp.example.com,true", got, ok)
	}

	got, ok = LongestMatch("host.example.com", suffixes)
	if !ok || got != "example.com" {
		t.Fatalf("LongestMatch = %q,%v want example.com,true", got, ok)
	}

	if _, ok := LongestMatch("host.other.net", suffixes); ok {
		t.Fatalf("expected no match for host.other.net")
	}
}

func TestFileName(t *testing.T) {
	// FileName returns the normalized domain unchanged.
	if got := FileName("lab.dev"); got != "lab.dev" {
		t.Errorf("FileName(lab.dev) = %q, want lab.dev", got)
	}
	if got := FileName(""); got != "" {
		t.Errorf("FileName(\"\") = %q, want empty", got)
	}
}
