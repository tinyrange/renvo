package tcpip

import (
	"bytes"
	"testing"
)

var dhcpServerIP = [4]byte{192, 168, 7, 1}
var dhcpAddress = [4]byte{192, 168, 7, 42}

func newDHCP() *DHCP {
	d := &DHCP{MAC: [6]byte{2, 3, 4, 5, 6, 7}, Hostname: "test-device"}
	d.Start(0, 1234)
	return d
}
func dhcpReply(d *DHCP, kind byte) []byte {
	b := make([]byte, 342)
	for i := 0; i < 6; i++ {
		b[i] = 255
	}
	copy(b[6:12], clientMAC)
	put16(b, 12, 0x800)
	ip := b[14:34]
	ip[0] = 0x45
	ip[8] = 64
	ip[9] = 17
	put16(ip, 2, 328)
	copy(ip[12:16], dhcpServerIP[:])
	for i := 16; i < 20; i++ {
		ip[i] = 255
	}
	put16(ip, 10, checksum(ip))
	u := b[34:]
	put16(u, 0, 67)
	put16(u, 2, 68)
	put16(u, 4, 308)
	p := b[42:]
	p[0] = 2
	p[1] = 1
	p[2] = 6
	put32(p, 4, d.xid)
	copy(p[16:20], dhcpAddress[:])
	copy(p[28:34], d.MAC[:])
	put32(p, 236, 0x63825363)
	i := 240
	for _, o := range [][]byte{{53, 1, kind}, {54, 4, 192, 168, 7, 1}, {1, 4, 255, 255, 255, 0}, {3, 4, 192, 168, 7, 1}, {6, 4, 8, 8, 8, 8}, {51, 4, 0, 0, 0, 120}, {58, 4, 0, 0, 0, 60}, {59, 4, 0, 0, 0, 105}, {255}} {
		copy(p[i:], o)
		i += len(o)
	}
	return b
}
func checkedDHCP(t *testing.T, frame []byte, kind byte) []byte {
	t.Helper()
	if len(frame) < 342 {
		t.Fatalf("missing DHCP frame: %x", frame)
	}
	ip := frame[14:34]
	u := frame[34:]
	p := frame[42:]
	pseudo := append(append([]byte{}, ip[12:20]...), 0, 17, byte(len(u)>>8), byte(len(u)))
	if checksum(ip) != 0 || checksum(pseudo, u) != 0 {
		t.Fatal("invalid client checksum")
	}
	if get16(u, 0) != 68 || get16(u, 2) != 67 || get16(u, 4) != uint16(len(u)) || p[0] != 1 || get32(p, 236) != 0x63825363 {
		t.Fatal("bad BOOTP/IP/UDP encoding")
	}
	var opts dhcpOptions
	if !opts.parse(p[240:], true) || opts.kind != kind {
		t.Fatal("bad DHCP options")
	}
	return append([]byte{}, p...)
}
func offered(t *testing.T, d *DHCP) {
	t.Helper()
	checkedDHCP(t, d.Poll(1001), 1)
	request := checkedDHCP(t, d.Handle(dhcpReply(d, 2), 1010), 3)
	if get16(request, 10) != 0x8000 || get32(request, 12) != 0 {
		t.Fatal("initial REQUEST must broadcast with ciaddr zero")
	}
}
func bound(t *testing.T, d *DHCP) {
	t.Helper()
	offered(t, d)
	d.Handle(dhcpReply(d, 5), 1020)
	for now := uint32(1100); now <= 9000; now += 100 {
		d.Poll(now)
	}
	if !d.Ready() || d.Lease.IP != dhcpAddress {
		t.Fatal("lease did not become ready")
	}
}

func TestDHCPDiscoveryProbesAndRenewal(t *testing.T) {
	d := newDHCP()
	offered(t, d)
	// No ACK means no usable address, even though an offer is selected.
	if d.Ready() {
		t.Fatal("using unacknowledged offer")
	}
	d.Handle(dhcpReply(d, 5), 1020)
	probes, announcements := 0, 0
	for now := uint32(1100); now <= 10000; now += 100 {
		p := d.Poll(now)
		if len(p) == 42 {
			if get32(p, 28) == 0 {
				probes++
				if d.Ready() {
					t.Fatal("using address during probing")
				}
			} else {
				announcements++
			}
		}
	}
	if probes != 3 || announcements != 2 || !d.Ready() {
		t.Fatalf("probes %d announcements %d ready %v", probes, announcements, d.Ready())
	}
	if d.Lease.Mask != [4]byte{255, 255, 255, 0} || d.Lease.DNS != [4]byte{8, 8, 8, 8} {
		t.Fatal("lost configuration")
	}
	arp := d.Poll(61000)
	if len(arp) != 42 || !equal(arp[38:42], dhcpServerIP[:]) || !d.Ready() {
		t.Fatal("renewal did not resolve server via ARP")
	}
	response := append([]byte{}, arp...)
	copy(response[:6], d.MAC[:])
	copy(response[6:12], clientMAC)
	put16(response, 20, 2)
	copy(response[22:28], clientMAC)
	copy(response[28:32], dhcpServerIP[:])
	copy(response[32:38], d.MAC[:])
	copy(response[38:42], dhcpAddress[:])
	reqFrame := d.Handle(response, 61001)
	req := checkedDHCP(t, reqFrame, 3)
	if !equal(req[12:16], dhcpAddress[:]) || get16(req, 10) != 0 || !equal(reqFrame[:6], clientMAC) {
		t.Fatal("bad unicast renewal")
	}
	var opt dhcpOptions
	opt.parse(req[240:], true)
	if opt.seen&2 != 0 {
		t.Fatal("renewal included server identifier option")
	}
	d.Handle(dhcpReply(d, 5), 61010)
	if !d.Ready() || d.ACKs != 2 {
		t.Fatal("renewal lost lease")
	}
	d.Poll(120999)
	if d.state != dhcpBound {
		t.Fatal("renewal did not reset T1")
	}
}

func TestDHCPRebindExpiryAndNAK(t *testing.T) {
	d := newDHCP()
	bound(t, d)
	d.Poll(61000) // no ARP/renewal response
	frame := d.Poll(106000)
	req := checkedDHCP(t, frame, 3)
	if !broadcastMAC(frame[:6]) || !equal(req[12:16], dhcpAddress[:]) || !d.Ready() {
		t.Fatal("bad broadcast rebind")
	}
	d.Poll(121000)
	if d.Ready() || d.Lease.IP != [4]byte{} {
		t.Fatal("expired address remained usable")
	}
	checkedDHCP(t, d.Poll(122001), 1)
	d.Handle(dhcpReply(d, 2), 122010)
	d.Handle(dhcpReply(d, 6), 122020)
	if d.state != dhcpBackoff || d.Ready() {
		t.Fatal("NAK did not restart acquisition")
	}
	d = newDHCP()
	bound(t, d)
	d.Handle(nil, 121000)
	if d.Ready() {
		t.Fatal("input path retained an expired address until Poll")
	}
}

func TestDHCPValidationAndOverload(t *testing.T) {
	d := newDHCP()
	offered(t, d)
	valid := dhcpReply(d, 5)
	for n := 0; n < 42+240; n++ {
		d.Handle(valid[:n], 1020)
		if d.Ready() || d.state != dhcpRequesting {
			t.Fatalf("accepted truncation %d", n)
		}
	}
	for _, offset := range []int{14, 42, 43, 44, 46, 70, 278} {
		bad := append([]byte{}, valid...)
		bad[offset] ^= 1
		d.Handle(bad, 1020)
		if d.state != dhcpRequesting {
			t.Fatalf("accepted corruption at %d", offset)
		}
	}
	bad := append([]byte{}, valid...)
	bad[42+241] = 250
	d.Handle(bad, 1020)
	if d.state != dhcpRequesting {
		t.Fatal("accepted option overrun")
	}
	bad = append([]byte{}, valid...)
	put16(bad, 40, 1)
	d.Handle(bad, 1020)
	if d.state != dhcpRequesting {
		t.Fatal("accepted invalid UDP checksum")
	}
	// Put all options except overload into the BOOTP file field.
	p := valid[42:]
	copy(p[108:236], p[240:])
	for i := 240; i < len(p); i++ {
		p[i] = 0
	}
	copy(p[240:], []byte{52, 1, 1, 255})
	d.Handle(valid, 1020)
	if d.state != dhcpProbing {
		t.Fatal("option overload was not parsed")
	}
}

func TestDHCPConflictDeclineAndTimerWrap(t *testing.T) {
	d := newDHCP()
	offered(t, d)
	d.Handle(dhcpReply(d, 5), 1020)
	conflict := append([]byte{}, d.arp([4]byte{}, dhcpAddress)...)
	copy(conflict[6:12], clientMAC)
	copy(conflict[22:28], clientMAC)
	checkedDHCP(t, d.Handle(conflict, 1100), 4)
	if d.Ready() || d.Conflicts != 1 || d.Lease.IP != [4]byte{} {
		t.Fatal("conflicted address retained")
	}
	if d.Poll(10000) != nil {
		t.Fatal("DECLINE backoff too short")
	}
	d.Poll(11100)
	checkedDHCP(t, d.Poll(12101), 1)
	// Advance the millisecond counter across its 49-day wrap. Lease seconds
	// must accumulate rather than treating wrap as expiry or a stopped timer.
	d = newDHCP()
	bound(t, d)
	d.lastClock = 0xfffffff0
	before := d.seconds
	d.Poll(1000)
	if d.seconds != before+1 || !d.Ready() {
		t.Fatal("millisecond wrap broke lease age")
	}
}

func TestDHCPLongAndInfiniteLeases(t *testing.T) {
	d := newDHCP()
	offered(t, d)
	p := dhcpReply(d, 5)
	// Rebuild fixed options with a long finite lease and no explicit T1/T2.
	b := p[42:]
	copy(b[240:], []byte{53, 1, 5, 54, 4, 192, 168, 7, 1, 1, 4, 255, 255, 255, 0, 51, 4, 0x00, 0x76, 0xa7, 0x00, 255})
	d.Handle(p, 1020)
	if d.Lease.Seconds != 7776000 || d.Lease.RenewAfter != 3888000 || d.Lease.RebindAfter != 6804000 {
		t.Fatal("long lease overflow/default timers")
	}
	d = newDHCP()
	offered(t, d)
	p = dhcpReply(d, 5)
	b = p[42:]
	pattern := []byte{51, 4, 0, 0, 0, 120}
	at := bytes.Index(b, pattern)
	if at < 0 {
		t.Fatal("fixture")
	}
	put32(b, at+2, 0xffffffff)
	d.Handle(p, 1020)
	for n := uint32(2000); n < 10000; n += 100 {
		d.Poll(n)
	}
	d.seconds = 0xffffff00
	d.Poll(11000)
	if !d.Ready() || d.state != dhcpBound {
		t.Fatal("infinite lease renewed or expired")
	}
}
