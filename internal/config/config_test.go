package config

import (
	"strings"
	"testing"
)

func TestParsePreservesCommentsAndUnknown(t *testing.T) {
	body := `# Managed by splitdns
# user note
domain lab.dev
nameserver 127.0.0.1
nameserver 127.0.0.1
port 53

# trailing comment
future_directive some-value
`
	cfg, err := Parse(body)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Managed {
		t.Error("expected Managed to be true")
	}
	if cfg.Domain != "lab.dev" {
		t.Errorf("domain = %q, want lab.dev", cfg.Domain)
	}
	ns := cfg.Nameservers()
	if len(ns) != 2 {
		t.Errorf("expected 2 (duplicate) nameservers, got %d: %v", len(ns), ns)
	}
	if len(cfg.Warnings) == 0 {
		t.Error("expected a warning for the unknown directive")
	}

	// Round-trip stability: serialize twice, identical output preserving order.
	out1 := Serialize(cfg)
	if !strings.Contains(out1, "future_directive some-value") {
		t.Error("unknown directive was dropped")
	}
	if !strings.Contains(out1, "# user note") {
		t.Error("user comment was dropped")
	}
	cfg2, _ := Parse(out1)
	out2 := Serialize(cfg2)
	if out1 != out2 {
		t.Errorf("serialize not stable:\n---1---\n%s\n---2---\n%s", out1, out2)
	}
}

func TestValidateNameserver(t *testing.T) {
	valid := []string{"127.0.0.1", "8.8.8.8", "::1", "2606:4700:4700::1111"}
	for _, v := range valid {
		if err := ValidateNameserver(v); err != nil {
			t.Errorf("expected %q valid: %v", v, err)
		}
	}
	invalid := []string{"", "999.1.1.1", "not-an-ip", "127.0.0.1.5"}
	for _, v := range invalid {
		if err := ValidateNameserver(v); err == nil {
			t.Errorf("expected %q invalid", v)
		}
	}
}

func TestValidatePort(t *testing.T) {
	if err := ValidatePort("53"); err != nil {
		t.Errorf("53 should be valid: %v", err)
	}
	for _, bad := range []string{"0", "65536", "-1", "abc", ""} {
		if err := ValidatePort(bad); err == nil {
			t.Errorf("expected %q invalid", bad)
		}
	}
}

func TestBuildStableSerialize(t *testing.T) {
	f := Fields{Domain: "lab.dev", Nameservers: []string{"127.0.0.1"}, Port: "53"}
	a := Serialize(Build(f))
	b := Serialize(Build(f))
	if a != b {
		t.Errorf("Build serialization not stable:\n%q\n%q", a, b)
	}
	if !strings.HasPrefix(a, ManagedMarker+"\n") {
		t.Errorf("expected managed marker header, got:\n%s", a)
	}
}

func TestFirstValue(t *testing.T) {
	cfg, _ := Parse("domain lab.dev\nnameserver 127.0.0.1\nnameserver 127.0.0.2\nport 5353\n")

	if v, ok := cfg.FirstValue("port"); !ok || v != "5353" {
		t.Errorf("FirstValue(port) = %q, %v; want 5353, true", v, ok)
	}
	// Returns the first value of the first matching directive.
	if v, ok := cfg.FirstValue("nameserver"); !ok || v != "127.0.0.1" {
		t.Errorf("FirstValue(nameserver) = %q, %v; want 127.0.0.1, true", v, ok)
	}
	// Missing key reports not-found.
	if v, ok := cfg.FirstValue("timeout"); ok || v != "" {
		t.Errorf("FirstValue(timeout) = %q, %v; want \"\", false", v, ok)
	}
}

func TestSetKeywordUpdatesInPlace(t *testing.T) {
	cfg, _ := Parse("# Managed by splitdns\ndomain lab.dev\nnameserver 127.0.0.1\nport 53\n")
	SetKeyword(cfg, "port", "5353")
	if p, _ := cfg.Port(); p != "5353" {
		t.Errorf("port = %q, want 5353", p)
	}
	// Only port changed; domain and nameserver retained.
	if cfg.Domain != "lab.dev" {
		t.Errorf("domain changed unexpectedly: %q", cfg.Domain)
	}
	if len(cfg.Nameservers()) != 1 {
		t.Errorf("nameservers changed unexpectedly: %v", cfg.Nameservers())
	}
}

func TestSetNameserversReplacesAll(t *testing.T) {
	cfg, _ := Parse("domain lab.dev\nnameserver 127.0.0.1\nnameserver 127.0.0.2\nport 53\n")
	SetNameservers(cfg, []string{"10.0.0.1"})
	ns := cfg.Nameservers()
	if len(ns) != 1 || ns[0] != "10.0.0.1" {
		t.Errorf("nameservers = %v, want [10.0.0.1]", ns)
	}
	if p, _ := cfg.Port(); p != "53" {
		t.Errorf("port lost: %q", p)
	}
}
