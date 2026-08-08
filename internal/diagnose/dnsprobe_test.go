package diagnose

import (
	"encoding/binary"
	"net"
	"testing"
	"time"
)

// buildResponse constructs a minimal DNS response for hostname with the given
// A-record addresses. When truncated is set, the TC flag is set and no answers
// are included (forcing a TCP retry by the prober).
func buildResponse(t *testing.T, hostname string, addrs []string, truncated bool) []byte {
	t.Helper()
	query, err := buildQuery(hostname, dnsTypeA)
	if err != nil {
		t.Fatal(err)
	}

	msg := make([]byte, len(query))
	copy(msg, query)

	// Turn the query into a response: set QR bit and answer count.
	flags := uint16(0x8180) // QR=1, RD=1, RA=1, rcode=0
	if truncated {
		flags |= 0x0200 // TC
	}
	binary.BigEndian.PutUint16(msg[2:4], flags)

	if truncated {
		binary.BigEndian.PutUint16(msg[6:8], 0)
		return msg
	}

	binary.BigEndian.PutUint16(msg[6:8], uint16(len(addrs)))
	for _, a := range addrs {
		ip := net.ParseIP(a).To4()
		// Name pointer to the question (offset 12).
		rr := []byte{0xc0, 0x0c}
		typeClassTTL := make([]byte, 8)
		binary.BigEndian.PutUint16(typeClassTTL[0:2], dnsTypeA)
		binary.BigEndian.PutUint16(typeClassTTL[2:4], dnsClassIN)
		binary.BigEndian.PutUint32(typeClassTTL[4:8], 60)
		rdlen := make([]byte, 2)
		binary.BigEndian.PutUint16(rdlen, 4)
		rr = append(rr, typeClassTTL...)
		rr = append(rr, rdlen...)
		rr = append(rr, ip...)
		msg = append(msg, rr...)
	}
	return msg
}

func TestParseAnswers(t *testing.T) {
	msg := buildResponse(t, "example.com", []string{"93.184.216.34"}, false)
	addrs, err := parseAnswers(msg)
	if err != nil {
		t.Fatalf("parseAnswers: %v", err)
	}
	if len(addrs) != 1 || addrs[0] != "93.184.216.34" {
		t.Errorf("addrs = %v, want [93.184.216.34]", addrs)
	}
}

func TestParseAnswersShortMessage(t *testing.T) {
	if _, err := parseAnswers([]byte{0, 1, 2}); err == nil {
		t.Error("expected error for short message")
	}
}

func TestParseAnswersRcode(t *testing.T) {
	msg := buildResponse(t, "example.com", nil, false)
	// Set rcode to 3 (NXDOMAIN).
	flags := binary.BigEndian.Uint16(msg[2:4])
	binary.BigEndian.PutUint16(msg[2:4], flags|0x0003)
	if _, err := parseAnswers(msg); err == nil {
		t.Error("expected error for non-zero rcode")
	}
}

func TestSkipName(t *testing.T) {
	// A simple name "a.bc" followed by a terminating zero.
	msg := []byte{1, 'a', 2, 'b', 'c', 0}
	next, err := skipName(msg, 0)
	if err != nil {
		t.Fatal(err)
	}
	if next != len(msg) {
		t.Errorf("next = %d, want %d", next, len(msg))
	}
}

func TestSkipNameCompressionPointer(t *testing.T) {
	// A compression pointer terminates the name and occupies two bytes.
	msg := []byte{0xc0, 0x0c}
	next, err := skipName(msg, 0)
	if err != nil {
		t.Fatal(err)
	}
	if next != 2 {
		t.Errorf("next = %d, want 2", next)
	}
}

func TestSkipNameOutOfRange(t *testing.T) {
	if _, err := skipName([]byte{}, 0); err == nil {
		t.Error("expected out-of-range error")
	}
	// A pointer at the very end with no second byte is truncated.
	if _, err := skipName([]byte{0xc0}, 0); err == nil {
		t.Error("expected truncated pointer error")
	}
}

func TestProbeUDPSuccess(t *testing.T) {
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer pc.Close()

	go func() {
		buf := make([]byte, 4096)
		n, addr, err := pc.ReadFrom(buf)
		if err != nil {
			return
		}
		_ = n
		resp := buildResponse(t, "example.com", []string{"1.2.3.4"}, false)
		_, _ = pc.WriteTo(resp, addr)
	}()

	_, port, _ := net.SplitHostPort(pc.LocalAddr().String())
	res := Probe("127.0.0.1", port, "example.com", 2*time.Second)
	if res.Err != nil {
		t.Fatalf("probe error: %v", res.Err)
	}
	if len(res.Addresses) != 1 || res.Addresses[0] != "1.2.3.4" {
		t.Errorf("addresses = %v, want [1.2.3.4]", res.Addresses)
	}
}

func TestProbeTruncatedFallsBackToTCP(t *testing.T) {
	// UDP server replies truncated; TCP server returns the full answer.
	udp, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer udp.Close()
	_, port, _ := net.SplitHostPort(udp.LocalAddr().String())

	tcp, err := net.Listen("tcp", "127.0.0.1:"+port)
	if err != nil {
		t.Fatalf("failed to bind matching TCP port: %v", err)
	}
	defer tcp.Close()

	go func() {
		buf := make([]byte, 4096)
		_, addr, err := udp.ReadFrom(buf)
		if err != nil {
			return
		}
		_, _ = udp.WriteTo(buildResponse(t, "example.com", nil, true), addr)
	}()

	go func() {
		conn, err := tcp.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		lenBuf := make([]byte, 2)
		if _, err := readFull(conn, lenBuf); err != nil {
			return
		}
		qlen := binary.BigEndian.Uint16(lenBuf)
		q := make([]byte, qlen)
		if _, err := readFull(conn, q); err != nil {
			return
		}
		resp := buildResponse(t, "example.com", []string{"5.6.7.8"}, false)
		out := make([]byte, 2+len(resp))
		binary.BigEndian.PutUint16(out[0:2], uint16(len(resp)))
		copy(out[2:], resp)
		_, _ = conn.Write(out)
	}()

	res := Probe("127.0.0.1", port, "example.com", 2*time.Second)
	if res.Err != nil {
		t.Fatalf("probe error: %v", res.Err)
	}
	if !res.UsedTCP {
		t.Error("expected TCP fallback")
	}
	if len(res.Addresses) != 1 || res.Addresses[0] != "5.6.7.8" {
		t.Errorf("addresses = %v, want [5.6.7.8]", res.Addresses)
	}
}
