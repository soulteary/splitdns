//go:build integration

// Package integration contains read-only integration tests that observe the
// host's real DNS state via scutil/dscacheutil. They never modify
// /etc/resolver. Run with: go test -tags integration ./...
package integration

import (
	"os/exec"
	"runtime"
	"testing"
	"time"

	"github.com/soulteary/splitdns/internal/diagnose"
)

func TestScutilParsesRealOutput(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("macOS-only integration test")
	}
	out, err := exec.Command("scutil", "--dns").Output()
	if err != nil {
		t.Skipf("scutil unavailable: %v", err)
	}
	resolvers := diagnose.ParseScutilDNS(string(out))
	if len(resolvers) == 0 {
		t.Fatal("expected at least one resolver from real scutil --dns")
	}
}

func TestDscacheParsesRealOutput(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("macOS-only integration test")
	}
	out, err := exec.Command("dscacheutil", "-q", "host", "-a", "name", "localhost").Output()
	if err != nil {
		t.Skipf("dscacheutil unavailable: %v", err)
	}
	entries := diagnose.ParseDscacheHost(string(out))
	if len(diagnose.DscacheAddresses(entries)) == 0 {
		t.Fatal("expected localhost to resolve to at least one address")
	}
}

func TestProbePublicResolver(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("macOS-only integration test")
	}
	res := diagnose.Probe("1.1.1.1", "53", "one.one.one.one", 3*time.Second)
	if res.Err != nil {
		t.Skipf("network unavailable: %v", res.Err)
	}
	if len(res.Addresses) == 0 {
		t.Fatal("expected addresses from a real DNS probe")
	}
}
