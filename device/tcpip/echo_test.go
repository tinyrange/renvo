package tcpip

import (
	"bytes"
	"encoding/binary"
	"testing"
)

var clientIP = []byte{169, 254, 180, 218}
var clientMAC = []byte{0x4c, 0xea, 0x41, 0x6c, 0xf6, 0xce}

func endpoint() *Echo {
	return &Echo{MAC: [6]byte{2, 0x52, 0x4e, 0, 0, 4}, IP: [4]byte{169, 254, 180, 4}, Port: 4242}
}

// Independent test-side Internet checksum, including odd-length payloads.
func checksum(parts ...[]byte) uint16 {
	data := bytes.Join(parts, nil)
	var s uint32
	for i, v := range data {
		if i%2 == 0 {
			s += uint32(v) << 8
		} else {
			s += uint32(v)
		}
	}
	for s > 65535 {
		s = s>>16 + s&65535
	}
	return ^uint16(s)
}
func packet(e *Echo, seq, a uint32, flags byte, data []byte) []byte {
	b := make([]byte, 54+len(data))
	copy(b, e.MAC[:])
	copy(b[6:], clientMAC)
	binary.BigEndian.PutUint16(b[12:], 0x800)
	ip := b[14:34]
	ip[0] = 0x45
	binary.BigEndian.PutUint16(ip[2:], uint16(40+len(data)))
	ip[8] = 64
	ip[9] = 6
	copy(ip[12:], clientIP)
	copy(ip[16:], e.IP[:])
	binary.BigEndian.PutUint16(ip[10:], checksum(ip))
	tcp := b[34:]
	binary.BigEndian.PutUint16(tcp, 50000)
	binary.BigEndian.PutUint16(tcp[2:], e.Port)
	binary.BigEndian.PutUint32(tcp[4:], seq)
	binary.BigEndian.PutUint32(tcp[8:], a)
	tcp[12] = 0x50
	tcp[13] = flags
	binary.BigEndian.PutUint16(tcp[14:], 4096)
	copy(tcp[20:], data)
	pseudo := append(append([]byte{}, ip[12:20]...), 0, 6, byte(len(tcp)>>8), byte(len(tcp)))
	binary.BigEndian.PutUint16(tcp[16:], checksum(pseudo, tcp))
	return b
}
func checked(t *testing.T, b []byte) []byte {
	t.Helper()
	if len(b) < 54 {
		t.Fatalf("missing TCP response: %x", b)
	}
	ip := b[14:34]
	tcp := b[34:]
	pseudo := append(append([]byte{}, ip[12:20]...), 0, 6, byte(len(tcp)>>8), byte(len(tcp)))
	if checksum(ip) != 0 || checksum(pseudo, tcp) != 0 {
		t.Fatal("invalid response checksum")
	}
	if int(binary.BigEndian.Uint16(ip[2:])) != len(b)-14 {
		t.Fatal("incorrect IP length")
	}
	return append([]byte{}, tcp...)
}
func connect(t *testing.T, e *Echo, seq uint32) uint32 {
	t.Helper()
	r := checked(t, e.Handle(packet(e, seq, 0, syn, nil), 10))
	if r[13] != syn|ack || binary.BigEndian.Uint32(r[8:]) != seq+1 || !bytes.Equal(r[20:24], []byte{2, 4, 2, 0}) {
		t.Fatal("bad SYN ACK")
	}
	server := binary.BigEndian.Uint32(r[4:]) + 1
	e.Handle(packet(e, seq+1, server, ack, nil), 11)
	return server
}

func TestEchoStreamRetransmissionAndClose(t *testing.T) {
	e := endpoint()
	seq := uint32(0xfffffff0)
	server := connect(t, e, seq)
	seq++
	for _, n := range []int{1, 31, 512, 3} {
		data := make([]byte, n)
		for i := range data {
			data[i] = byte(i*73 + n)
		}
		request := packet(e, seq, server, ack|psh, data)
		r := checked(t, e.Handle(request, 100))
		if !bytes.Equal(r[20:], data) || binary.BigEndian.Uint32(r[8:]) != seq+uint32(n) {
			t.Fatal("echo changed data or ACK")
		}
		if e.Poll(599) != nil {
			t.Fatal("premature retransmit")
		}
		retry := checked(t, e.Poll(600))
		if !bytes.Equal(retry, r) {
			t.Fatal("retransmission changed bytes")
		}
		dup := checked(t, e.Handle(request, 601))
		if len(dup) != 20 || binary.BigEndian.Uint32(dup[8:]) != seq+uint32(n) {
			t.Fatal("duplicate data consumed twice")
		}
		seq += uint32(n)
		server += uint32(n)
		update := checked(t, e.Handle(packet(e, seq, server, ack, nil), 602))
		if binary.BigEndian.Uint16(update[14:]) != 512 {
			t.Fatal("receive window did not reopen")
		}
		if e.Poll(1000) != nil {
			t.Fatal("acknowledged data retransmitted")
		}
	}
	r := checked(t, e.Handle(packet(e, seq, server, ack|fin, nil), 2000))
	if r[13] != ack|fin || binary.BigEndian.Uint32(r[8:]) != seq+1 {
		t.Fatal("bad FIN response")
	}
	e.Handle(packet(e, seq+1, server+1, ack, nil), 2001)
	r = checked(t, e.Handle(packet(e, seq, server, ack|fin, nil), 2002))
	if r[13] != ack {
		t.Fatal("duplicate FIN not acknowledged")
	}
}

func TestEchoRejectsMalformedAndOutOfOrder(t *testing.T) {
	e := endpoint()
	server := connect(t, e, 99)
	valid := packet(e, 100, server, ack|psh, []byte("payload"))
	for n := 0; n < len(valid); n++ {
		if e.Handle(valid[:n], 20) != nil {
			t.Fatalf("accepted truncation %d", n)
		}
	}
	bad := append([]byte{}, valid...)
	bad[len(bad)-1] ^= 1
	if e.Handle(bad, 21) != nil {
		t.Fatal("accepted bad TCP checksum")
	}
	bad = append([]byte{}, valid...)
	bad[22] ^= 1
	if e.Handle(bad, 21) != nil {
		t.Fatal("accepted bad IPv4 checksum")
	}
	bad = append([]byte{}, valid...)
	bad[20] = 0x20
	bad[24] = 0
	bad[25] = 0
	binary.BigEndian.PutUint16(bad[24:], checksum(bad[14:34]))
	if e.Handle(bad, 21) != nil {
		t.Fatal("accepted fragmented IP")
	}
	r := checked(t, e.Handle(packet(e, 101, server, ack|psh, []byte("later")), 22))
	if binary.BigEndian.Uint32(r[8:]) != 100 || len(r) != 20 {
		t.Fatal("accepted out-of-order data")
	}
	e.Handle(packet(e, 101, server, rst, nil), 23)
	r = checked(t, e.Handle(valid, 24))
	if string(r[20:]) != "payload" {
		t.Fatal("bad RST killed connection")
	}
}

func TestEchoARPAndICMP(t *testing.T) {
	e := endpoint()
	b := make([]byte, 42)
	for i := 0; i < 6; i++ {
		b[i] = 255
	}
	copy(b[6:], clientMAC)
	binary.BigEndian.PutUint16(b[12:], 0x806)
	a := b[14:]
	copy(a, []byte{0, 1, 8, 0, 6, 4, 0, 1})
	copy(a[8:], clientMAC)
	copy(a[14:], clientIP)
	copy(a[24:], e.IP[:])
	r := e.Handle(b, 0)
	if len(r) != 42 || !bytes.Equal(r[:6], clientMAC) || get16(r, 20) != 2 || !bytes.Equal(r[28:32], e.IP[:]) {
		t.Fatal("bad ARP reply")
	}
	b = packet(e, 0, 0, 0, nil)[:43]
	ip := b[14:34]
	ip[9] = 1
	put16(ip, 2, 29)
	put16(ip, 10, 0)
	put16(ip, 10, checksum(ip))
	icmp := b[34:]
	copy(icmp, []byte{8, 0, 0, 0, 12, 34, 56, 78, 90})
	put16(icmp, 2, checksum(icmp))
	r = e.Handle(b, 0)
	if len(r) != 43 || r[34] != 0 || checksum(r[34:]) != 0 || !bytes.Equal(r[38:], icmp[4:]) {
		t.Fatal("bad ICMP reply")
	}
}

func TestEchoRetransmissionTimeoutWrap(t *testing.T) {
	e := endpoint()
	e.Handle(packet(e, 100, 0, syn, nil), 0xfffffff0)
	if e.Poll(200) != nil {
		t.Fatal("timer wrap caused premature retransmit")
	}
	if e.Poll(600) == nil {
		t.Fatal("timer wrap lost retransmit")
	}
	if e.Poll(60000) != nil || e.state != 0 {
		t.Fatal("idle connection did not expire")
	}
}

func TestEchoPartialWindowAndDataFIN(t *testing.T) {
	e := endpoint()
	server := connect(t, e, 99)
	// A peer can shrink its receive window independently of our window.
	p := packet(e, 100, server, ack|psh, []byte("abcdef"))
	put16(p, 48, 3)
	put16(p, 50, 0)
	pseudo := append(append([]byte{}, p[26:34]...), 0, 6, 0, 26)
	put16(p, 50, checksum(pseudo, p[34:]))
	r := checked(t, e.Handle(p, 20))
	if string(r[20:]) != "abc" || get32(r, 8) != 103 {
		t.Fatal("did not respect peer window")
	}
	e.Handle(packet(e, 103, server+3, ack, nil), 21)
	// Retransmission can overlap bytes already ACKed. Echo only the suffix,
	// and include FIN when the remaining data and FIN fit in this segment.
	r = checked(t, e.Handle(packet(e, 100, server+3, ack|psh|fin, []byte("abcdef")), 22))
	if string(r[20:]) != "def" || r[13] != psh|ack|fin || get32(r, 8) != 107 {
		t.Fatal("bad overlapping data/FIN handling")
	}
	e.Handle(packet(e, 107, server+7, ack, nil), 23)
	if e.state != 4 {
		t.Fatal("FIN was not acknowledged")
	}
}

func TestEchoSmallMSS(t *testing.T) {
	e := endpoint()
	p := packet(e, 100, 0, syn, []byte{2, 4, 0, 64})
	p[46] = 0x60
	put16(p, 50, 0)
	pseudo := append(append([]byte{}, p[26:34]...), 0, 6, 0, 24)
	put16(p, 50, checksum(pseudo, p[34:]))
	r := checked(t, e.Handle(p, 10))
	if get16(r, 14) != 64 {
		t.Fatal("receive window exceeds peer MSS")
	}
}
