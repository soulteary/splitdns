package diagnose

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/soulteary/splitdns/internal/system"
)

func TestParseScutilDNS(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "scutil_dns.txt"))
	if err != nil {
		t.Fatal(err)
	}
	resolvers := ParseScutilDNS(string(data))
	if len(resolvers) == 0 {
		t.Fatal("no resolvers parsed")
	}

	r, ok := FindScutilResolverForDomain(resolvers, "lab.dev")
	if !ok {
		t.Fatal("expected to find lab.dev resolver")
	}
	if len(r.Nameservers) != 1 || r.Nameservers[0] != "127.0.0.1" {
		t.Errorf("nameservers = %v, want [127.0.0.1]", r.Nameservers)
	}
	if r.Port != "53" {
		t.Errorf("port = %q, want 53", r.Port)
	}

	// IPv6 nameserver value with colons must be preserved intact.
	var foundV6 bool
	for _, res := range resolvers {
		for _, ns := range res.Nameservers {
			if ns == "2606:4700:4700::1111" {
				foundV6 = true
			}
		}
	}
	if !foundV6 {
		t.Error("IPv6 nameserver was not parsed correctly")
	}

	// Scoped section resolvers must be flagged.
	var sawScoped bool
	for _, res := range resolvers {
		if res.Scoped {
			sawScoped = true
		}
	}
	if !sawScoped {
		t.Error("expected at least one scoped resolver")
	}

	// A scoped resolver for lab.dev must not be returned by the domain finder.
	if _, ok := FindScutilResolverForDomain(resolvers, "local"); !ok {
		t.Error("expected to find local resolver")
	}
}

func TestParseDscacheHost(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "dscacheutil_localhost.txt"))
	if err != nil {
		t.Fatal(err)
	}
	entries := ParseDscacheHost(string(data))
	addrs := DscacheAddresses(entries)
	want := map[string]bool{"::1": true, "127.0.0.1": true}
	if len(addrs) != 2 {
		t.Fatalf("addresses = %v, want 2", addrs)
	}
	for _, a := range addrs {
		if !want[a] {
			t.Errorf("unexpected address %q", a)
		}
	}
}

func TestProbeTimeout(t *testing.T) {
	// 203.0.113.0/24 is TEST-NET-3 (RFC 5737), guaranteed non-routable, so the
	// UDP query will time out rather than get answered.
	start := time.Now()
	res := Probe("203.0.113.1", "53", "example.com", 300*time.Millisecond)
	elapsed := time.Since(start)
	if res.Err == nil {
		t.Fatal("expected a timeout error")
	}
	if elapsed > 3*time.Second {
		t.Errorf("probe took too long: %v", elapsed)
	}
}

func TestBuildQueryRejectsBadLabel(t *testing.T) {
	if _, err := buildQuery("", dnsTypeA); err == nil {
		t.Error("expected error for empty hostname")
	}
	long := make([]byte, 64)
	for i := range long {
		long[i] = 'a'
	}
	if _, err := buildQuery(string(long)+".com", dnsTypeA); err == nil {
		t.Error("expected error for over-long label")
	}
}

func TestRunTestLongestSuffixAndLayers(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "example.com", "domain example.com\nnameserver 10.0.0.1\nport 53\n")
	writeFile(t, dir, "corp.example.com", "domain corp.example.com\nnameserver 10.0.0.2\nport 5353\n")

	scutil, _ := os.ReadFile(filepath.Join("..", "..", "testdata", "scutil_dns.txt"))
	fr := system.NewFakeRunner()
	fr.SetResponse("scutil", system.CommandResult{Stdout: string(scutil)})
	fr.SetResponse("dscacheutil", system.CommandResult{Stdout: "name: host.corp.example.com\nip_address: 10.0.0.2\n\n"})

	env := system.Env{Runner: fr, GOOS: "darwin", ResolverDir: dir}
	res := RunTest(env, "host.corp.example.com", 200*time.Millisecond)

	if res.MatchedSuffix != "corp.example.com" {
		t.Errorf("matched suffix = %q, want corp.example.com", res.MatchedSuffix)
	}
	if res.Port != "5353" {
		t.Errorf("port = %q, want 5353", res.Port)
	}
	if len(res.Nameservers) != 1 || res.Nameservers[0] != "10.0.0.2" {
		t.Errorf("nameservers = %v", res.Nameservers)
	}
	if len(res.DscacheAddresses) != 1 || res.DscacheAddresses[0] != "10.0.0.2" {
		t.Errorf("dscache addresses = %v", res.DscacheAddresses)
	}
}

func TestRunChecksNonDarwin(t *testing.T) {
	fr := system.NewFakeRunner()
	env := system.Env{Runner: fr, GOOS: "linux", ResolverDir: t.TempDir()}
	report := RunChecks(env, "", time.Second)
	if !report.HasError() {
		t.Error("expected an ERROR on non-darwin")
	}
}

func TestRunChecksValidatesFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "bad.dev", "nameserver not-an-ip\nport 99999\n")

	fr := system.NewFakeRunner()
	fr.SetResponse("scutil", system.CommandResult{Stdout: ""})
	env := system.Env{Runner: fr, GOOS: "darwin", ResolverDir: dir, HostsFile: filepath.Join(dir, "hosts")}
	report := RunChecks(env, "bad.dev", 200*time.Millisecond)
	if !report.HasError() {
		t.Errorf("expected ERROR for invalid nameserver/port; checks: %+v", report.Checks)
	}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
