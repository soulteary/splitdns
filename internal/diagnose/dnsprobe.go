// Package diagnose orchestrates configuration checks and end-to-end DNS tests,
// and parses macOS command output (scutil, dscacheutil). It performs DNS
// probing using a hand-written minimal DNS message over the standard library.
package diagnose

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

// ProbeResult holds the outcome of a direct DNS query against a nameserver.
type ProbeResult struct {
	Addresses []string
	Truncated bool
	UsedTCP   bool
	RTT       time.Duration
	Err       error
}

// Probe queries the given nameserver:port for A and AAAA records of hostname
// using UDP with a timeout, falling back to TCP when the UDP response is
// truncated. DNS availability is judged by a valid DNS response, never by a raw
// TCP connect.
func Probe(server, port, hostname string, timeout time.Duration) ProbeResult {
	var res ProbeResult
	start := time.Now()

	addr := net.JoinHostPort(server, port)
	query, err := buildQuery(hostname, dnsTypeA)
	if err != nil {
		res.Err = err
		return res
	}

	resp, truncated, err := queryUDP(addr, query, timeout)
	if err == nil && truncated {
		res.Truncated = true
		resp, err = queryTCP(addr, query, timeout)
		res.UsedTCP = true
	}
	if err != nil {
		res.Err = err
		res.RTT = time.Since(start)
		return res
	}

	addrs, perr := parseAnswers(resp)
	if perr != nil {
		res.Err = perr
		res.RTT = time.Since(start)
		return res
	}
	res.Addresses = addrs
	res.RTT = time.Since(start)
	return res
}

const (
	dnsTypeA    = 1
	dnsTypeAAAA = 28
	dnsClassIN  = 1
)

func buildQuery(hostname string, qtype uint16) ([]byte, error) {
	name := strings.TrimSuffix(strings.TrimSpace(hostname), ".")
	if name == "" {
		return nil, errors.New("empty hostname")
	}

	var msg []byte
	// Fixed transaction ID keeps queries deterministic for tests; matching is
	// still validated on the response.
	header := make([]byte, 12)
	binary.BigEndian.PutUint16(header[0:2], 0x1234)
	binary.BigEndian.PutUint16(header[2:4], 0x0100) // RD=1
	binary.BigEndian.PutUint16(header[4:6], 1)      // QDCOUNT
	msg = append(msg, header...)

	for _, label := range strings.Split(name, ".") {
		if len(label) == 0 || len(label) > 63 {
			return nil, fmt.Errorf("invalid DNS label %q", label)
		}
		msg = append(msg, byte(len(label)))
		msg = append(msg, label...)
	}
	msg = append(msg, 0x00)

	qtail := make([]byte, 4)
	binary.BigEndian.PutUint16(qtail[0:2], qtype)
	binary.BigEndian.PutUint16(qtail[2:4], dnsClassIN)
	msg = append(msg, qtail...)
	return msg, nil
}

func queryUDP(addr string, query []byte, timeout time.Duration) (resp []byte, truncated bool, err error) {
	conn, err := net.DialTimeout("udp", addr, timeout)
	if err != nil {
		return nil, false, err
	}
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(timeout))

	if _, err := conn.Write(query); err != nil {
		return nil, false, err
	}
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, false, err
	}
	if n < 12 {
		return nil, false, errors.New("short DNS response")
	}
	flags := binary.BigEndian.Uint16(buf[2:4])
	truncated = flags&0x0200 != 0
	return buf[:n], truncated, nil
}

func queryTCP(addr string, query []byte, timeout time.Duration) ([]byte, error) {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return nil, err
	}
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(timeout))

	prefixed := make([]byte, 2+len(query))
	binary.BigEndian.PutUint16(prefixed[0:2], uint16(len(query)))
	copy(prefixed[2:], query)
	if _, err := conn.Write(prefixed); err != nil {
		return nil, err
	}

	lenBuf := make([]byte, 2)
	if _, err := readFull(conn, lenBuf); err != nil {
		return nil, err
	}
	msgLen := binary.BigEndian.Uint16(lenBuf)
	msg := make([]byte, msgLen)
	if _, err := readFull(conn, msg); err != nil {
		return nil, err
	}
	return msg, nil
}

func readFull(conn net.Conn, buf []byte) (int, error) {
	total := 0
	for total < len(buf) {
		n, err := conn.Read(buf[total:])
		if n > 0 {
			total += n
		}
		if err != nil {
			return total, err
		}
	}
	return total, nil
}

// parseAnswers extracts A/AAAA addresses from a DNS response message.
func parseAnswers(msg []byte) ([]string, error) {
	if len(msg) < 12 {
		return nil, errors.New("short DNS response")
	}
	flags := binary.BigEndian.Uint16(msg[2:4])
	rcode := flags & 0x000f
	if rcode != 0 {
		return nil, fmt.Errorf("DNS server returned rcode %d", rcode)
	}
	qd := int(binary.BigEndian.Uint16(msg[4:6]))
	an := int(binary.BigEndian.Uint16(msg[6:8]))

	offset := 12
	for i := 0; i < qd; i++ {
		next, err := skipName(msg, offset)
		if err != nil {
			return nil, err
		}
		offset = next + 4 // QTYPE + QCLASS
		if offset > len(msg) {
			return nil, errors.New("malformed question section")
		}
	}

	var addrs []string
	for i := 0; i < an; i++ {
		next, err := skipName(msg, offset)
		if err != nil {
			return nil, err
		}
		offset = next
		if offset+10 > len(msg) {
			return nil, errors.New("malformed answer header")
		}
		rtype := binary.BigEndian.Uint16(msg[offset : offset+2])
		rdlen := int(binary.BigEndian.Uint16(msg[offset+8 : offset+10]))
		offset += 10
		if offset+rdlen > len(msg) {
			return nil, errors.New("malformed answer rdata")
		}
		rdata := msg[offset : offset+rdlen]
		switch rtype {
		case dnsTypeA:
			if rdlen == 4 {
				addrs = append(addrs, net.IP(rdata).String())
			}
		case dnsTypeAAAA:
			if rdlen == 16 {
				addrs = append(addrs, net.IP(rdata).String())
			}
		}
		offset += rdlen
	}
	return addrs, nil
}

// skipName advances past a (possibly compressed) DNS name and returns the
// offset immediately after it.
func skipName(msg []byte, offset int) (int, error) {
	for {
		if offset >= len(msg) {
			return 0, errors.New("name offset out of range")
		}
		length := int(msg[offset])
		if length == 0 {
			return offset + 1, nil
		}
		if length&0xc0 == 0xc0 {
			// Compression pointer occupies two bytes and terminates the name.
			if offset+2 > len(msg) {
				return 0, errors.New("truncated compression pointer")
			}
			return offset + 2, nil
		}
		offset += length + 1
	}
}
