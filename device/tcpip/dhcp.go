package tcpip

const (
	dhcpOff = uint8(iota)
	dhcpSelecting
	dhcpRequesting
	dhcpProbing
	dhcpBound
	dhcpRenewing
	dhcpRebinding
	dhcpBackoff
)

// DHCPLease is the configuration supplied by the selected DHCP server.
// Times are seconds; Seconds == 0xffffffff denotes an infinite lease.
type DHCPLease struct {
	IP, Mask, Router, DNS, Server    [4]byte
	Seconds, RenewAfter, RebindAfter uint32
}

// DHCP is a single-interface, polling DHCPv4 client. Set MAC and Hostname,
// then Start. Handle and Poll return Ethernet frames without FCS, valid until
// the next call. Feed it ARP as well as UDP frames, including while leased.
// Call at least once per second with a wrapping millisecond clock.
// Public lease fields are read-only to the caller; use Ready before using IP.
type DHCP struct {
	MAC      [6]byte
	Hostname string
	Lease    DHCPLease
	// ACKs and Conflicts are diagnostic counters.
	ACKs, Conflicts                            uint32
	state                                      uint8
	xid, random                                uint32
	lastClock, clockRemainder, seconds         uint32
	leaseStart, requestStart, transactionStart uint32
	lastAction, delay                          uint32
	retries, probes, announcements             uint8
	routeMAC                                   [6]byte
	routeIP                                    [4]byte
	routeReady                                 bool
	out                                        [600]byte
}

// Ready reports whether the lease may currently be used, including renewal.
func (d *DHCP) Ready() bool { return d.state >= dhcpBound && d.state <= dhcpRebinding }

// Stop immediately relinquishes local use of the address (e.g. link removal).
// It sends no RELEASE, since a removed link cannot deliver it reliably.
func (d *DHCP) Stop() { d.state = dhcpOff; d.Lease = DHCPLease{} }

// Start begins a fresh acquisition. Seed should vary per boot/link event;
// it is mixed with the unique MAC, and is not cryptographic randomness.
func (d *DHCP) Start(now, seed uint32) {
	d.Stop()
	d.lastClock = now
	d.clockRemainder = 0
	d.seconds = 0
	d.random = seed ^ uint32(d.MAC[2])<<24 ^ uint32(d.MAC[3])<<16 ^ uint32(d.MAC[4])<<8 ^ uint32(d.MAC[5])
	d.begin(now)
}
func (d *DHCP) rand() uint32 { d.random = d.random*1664525 + 1013904223; return d.random }
func (d *DHCP) begin(now uint32) {
	d.Lease = DHCPLease{}
	d.state = dhcpSelecting
	d.xid = d.rand()
	d.transactionStart = d.seconds
	d.retries = 0
	d.lastAction = now
	d.delay = d.rand() % 1001
	d.probes = 0
	d.announcements = 0
	d.routeReady = false
}
func (d *DHCP) tick(now uint32) {
	delta := now - d.lastClock
	d.lastClock = now
	d.seconds += delta / 1000
	d.clockRemainder += delta % 1000
	d.seconds += d.clockRemainder / 1000
	d.clockRemainder %= 1000
}
func (d *DHCP) retry(now uint32) {
	shift := d.retries
	if shift > 4 {
		shift = 4
	}
	d.delay = (4000 << shift) - 1000 + d.rand()%2001
	if d.retries < 8 {
		d.retries++
	}
	d.lastAction = now
}

// Poll advances acquisition, conflict probes, renewal and lease expiry.
func (d *DHCP) Poll(now uint32) []byte {
	d.tick(now)
	if d.state == dhcpOff {
		return nil
	}
	if d.state >= dhcpProbing && d.state <= dhcpRebinding && d.Lease.Seconds != 0xffffffff {
		age := d.seconds - d.leaseStart
		if age >= d.Lease.Seconds {
			d.begin(now)
		} else if d.Ready() {
			if age >= d.Lease.RebindAfter && d.state != dhcpRebinding {
				d.state = dhcpRebinding
				d.lastAction = now
				d.delay = 0
			} else if age >= d.Lease.RenewAfter && d.state == dhcpBound {
				d.state = dhcpRenewing
				d.xid = d.rand()
				d.transactionStart = d.seconds
				d.routeReady = false
				d.lastAction = now
				d.delay = 0
				d.routeIP = d.Lease.Server
				for i := 0; i < 4; i++ {
					if d.Lease.IP[i]&d.Lease.Mask[i] != d.Lease.Server[i]&d.Lease.Mask[i] {
						d.routeIP = d.Lease.Router
						break
					}
				}
			}
		}
	}
	if now-d.lastAction < d.delay {
		return nil
	}
	switch d.state {
	case dhcpSelecting:
		d.retry(now)
		return d.message(1)
	case dhcpRequesting:
		if d.retries >= 5 {
			d.begin(now)
			return nil
		}
		d.retry(now)
		d.requestStart = d.seconds
		return d.message(3)
	case dhcpProbing:
		d.lastAction = now
		if d.probes < 3 {
			d.probes++
			d.delay = 1000 + d.rand()%1001
			if d.probes == 3 {
				d.delay = 2000
			}
			return d.arp([4]byte{}, d.Lease.IP)
		}
		d.state = dhcpBound
		d.announcements = 1
		d.delay = 2000
		return d.arp(d.Lease.IP, d.Lease.IP)
	case dhcpBound:
		if d.announcements == 1 {
			d.announcements = 2
			return d.arp(d.Lease.IP, d.Lease.IP)
		}
	case dhcpRenewing, dhcpRebinding:
		d.lastAction = now
		if d.state == dhcpRenewing && !d.routeReady {
			d.delay = 4000
			if !dhcpUnicast(d.routeIP) {
				return nil
			}
			return d.arp(d.Lease.IP, d.routeIP)
		}
		remaining := d.Lease.Seconds - (d.seconds - d.leaseStart)
		if d.state == dhcpRenewing {
			remaining = d.Lease.RebindAfter - (d.seconds - d.leaseStart)
		}
		wait := remaining / 2
		if wait < 60 {
			wait = 60
		}
		if wait > 86400 {
			wait = 86400
		}
		d.delay = wait * 1000
		d.requestStart = d.seconds
		return d.message(3)
	case dhcpBackoff:
		d.begin(now)
	}
	return nil
}

func dhcpUnicast(ip [4]byte) bool { return ip[0] != 0 && ip[0] != 127 && ip[0] < 224 }
func dhcpMask(mask [4]byte) bool {
	v := get32(mask[:], 0)
	inverse := ^v
	return v != 0 && inverse&(inverse+1) == 0
}
func broadcastMAC(b []byte) bool {
	if len(b) != 6 {
		return false
	}
	for _, v := range b {
		if v != 255 {
			return false
		}
	}
	return true
}

type dhcpOptions struct {
	lease          DHCPLease
	kind, overload byte
	seen           uint32
}

// The supported options have fixed lengths, except router/DNS lists. Reject
// truncated or conflicting singleton options rather than partially applying.
func (o *dhcpOptions) parse(b []byte, allowOverload bool) bool {
	for i := 0; i < len(b); {
		code := b[i]
		i++
		if code == 255 {
			return true
		}
		if code == 0 {
			continue
		}
		if i >= len(b) {
			return false
		}
		n := int(b[i])
		i++
		if n > len(b)-i {
			return false
		}
		v := b[i : i+n]
		i += n
		var bit uint32
		switch code {
		case 53:
			bit = 1
			if n != 1 {
				return false
			}
			o.kind = v[0]
		case 54:
			bit = 2
			if n != 4 {
				return false
			}
			copy(o.lease.Server[:], v)
		case 51:
			bit = 4
			if n != 4 {
				return false
			}
			o.lease.Seconds = get32(v, 0)
		case 1:
			bit = 8
			if n != 4 {
				return false
			}
			copy(o.lease.Mask[:], v)
		case 3:
			bit = 16
			if n < 4 || n%4 != 0 {
				return false
			}
			copy(o.lease.Router[:], v[:4])
		case 6:
			bit = 32
			if n < 4 || n%4 != 0 {
				return false
			}
			copy(o.lease.DNS[:], v[:4])
		case 58:
			bit = 64
			if n != 4 {
				return false
			}
			o.lease.RenewAfter = get32(v, 0)
		case 59:
			bit = 128
			if n != 4 {
				return false
			}
			o.lease.RebindAfter = get32(v, 0)
		case 52:
			bit = 256
			if !allowOverload || n != 1 || v[0] < 1 || v[0] > 3 {
				return false
			}
			o.overload = v[0]
		}
		if bit != 0 && o.seen&bit != 0 {
			return false
		}
		o.seen |= bit
	}
	return false // END is required in each options field.
}

// Handle accepts validated DHCP replies and ARP probes/responses. All offered
// configuration is copied, so input frames can be reused immediately.
func (d *DHCP) Handle(frame []byte, now uint32) []byte {
	d.tick(now)
	// The caller may process input before calling Poll. Expire here too so
	// that a frame arriving at the deadline cannot use an expired address.
	if d.state >= dhcpProbing && d.state <= dhcpRebinding && d.Lease.Seconds != 0xffffffff && d.seconds-d.leaseStart >= d.Lease.Seconds {
		d.begin(now)
		return nil
	}
	if d.state == dhcpOff || len(frame) < 14 {
		return nil
	}
	if !equal(frame[:6], d.MAC[:]) && !broadcastMAC(frame[:6]) {
		return nil
	}
	if get16(frame, 12) == 0x806 {
		return d.handleARP(frame, now)
	}
	if get16(frame, 12) != 0x800 || len(frame) < 34 {
		return nil
	}
	ip := frame[14:]
	h := int(ip[0]&15) * 4
	n := int(get16(ip, 2))
	if ip[0]>>4 != 4 || h < 20 || h > len(ip) || n < h+8 || n > len(ip) || get16(ip, 6)&0x3fff != 0 || ip[9] != 17 || finish(sum(ip[:h], 0)) != 0 {
		return nil
	}
	u := ip[h:n]
	un := int(get16(u, 4))
	if get16(u, 0) != 67 || get16(u, 2) != 68 || un < 248 || un > len(u) {
		return nil
	}
	u = u[:un]
	if get16(u, 6) != 0 && finish(sum(u, sum(ip[12:20], uint32(17+un)))) != 0 {
		return nil
	}
	b := u[8:]
	if b[0] != 2 || b[1] != 1 || b[2] != 6 || get32(b, 4) != d.xid || !equal(b[28:34], d.MAC[:]) || get32(b, 236) != 0x63825363 {
		return nil
	}
	var options dhcpOptions
	if !options.parse(b[240:], true) {
		return nil
	}
	if options.overload&1 != 0 && !options.parse(b[108:236], false) {
		return nil
	}
	if options.overload&2 != 0 && !options.parse(b[44:108], false) {
		return nil
	}
	if options.seen&3 != 3 || !dhcpUnicast(options.lease.Server) {
		return nil
	}
	if d.state != dhcpSelecting && d.state != dhcpRebinding && options.lease.Server != d.Lease.Server {
		return nil
	}
	if options.kind == 6 && (d.state == dhcpRequesting || d.state == dhcpRenewing || d.state == dhcpRebinding) {
		d.Lease = DHCPLease{}
		d.state = dhcpBackoff
		d.lastAction = now
		d.delay = 1000
		return nil
	}
	var offered [4]byte
	copy(offered[:], b[16:20])
	if !dhcpUnicast(offered) {
		return nil
	}
	if options.kind == 2 && d.state == dhcpSelecting {
		d.Lease = options.lease
		d.Lease.IP = offered
		d.state = dhcpRequesting
		d.retries = 0
		d.retry(now)
		d.requestStart = d.seconds
		return d.message(3)
	}
	if options.kind != 5 || (d.state != dhcpRequesting && d.state != dhcpRenewing && d.state != dhcpRebinding) {
		return nil
	}
	if offered != d.Lease.IP || options.seen&4 == 0 || options.lease.Seconds == 0 {
		return nil
	}
	lease := options.lease
	lease.IP = offered
	if options.seen&8 == 0 {
		lease.Mask = d.Lease.Mask
	}
	if options.seen&16 == 0 {
		lease.Router = d.Lease.Router
	}
	if options.seen&32 == 0 {
		lease.DNS = d.Lease.DNS
	}
	if !dhcpMask(lease.Mask) {
		return nil
	}
	if lease.Seconds != 0xffffffff && (lease.RenewAfter == 0 || lease.RenewAfter >= lease.RebindAfter || lease.RebindAfter >= lease.Seconds) {
		lease.RenewAfter = lease.Seconds / 2
		lease.RebindAfter = lease.Seconds - lease.Seconds/8
		if lease.RebindAfter >= lease.Seconds {
			lease.RebindAfter = lease.Seconds - 1
		}
	}
	wasReady := d.Ready()
	d.Lease = lease
	d.leaseStart = d.requestStart
	d.ACKs++
	d.lastAction = now
	d.delay = d.rand() % 1001
	if wasReady {
		d.state = dhcpBound
		d.announcements = 2
	} else {
		d.state = dhcpProbing
		d.probes = 0
	}
	return nil
}

func (d *DHCP) handleARP(frame []byte, now uint32) []byte {
	if len(frame) < 42 || d.state < dhcpProbing || d.state > dhcpRebinding {
		return nil
	}
	b := frame[14:42]
	if get16(b, 0) != 1 || get16(b, 2) != 0x800 || b[4] != 6 || b[5] != 4 || (get16(b, 6) != 1 && get16(b, 6) != 2) || equal(b[8:14], d.MAC[:]) {
		return nil
	}
	conflict := equal(b[14:18], d.Lease.IP[:])
	if d.state == dhcpProbing && get32(b, 14) == 0 && equal(b[24:28], d.Lease.IP[:]) {
		conflict = true
	}
	if conflict {
		packet := d.message(4)
		d.Conflicts++
		d.Lease = DHCPLease{}
		d.state = dhcpBackoff
		d.lastAction = now
		d.delay = 10000
		if d.Conflicts >= 10 {
			d.delay = 60000
		}
		return packet
	}
	if d.state == dhcpRenewing && get16(b, 6) == 2 && equal(b[14:18], d.routeIP[:]) && equal(b[18:24], d.MAC[:]) && equal(b[24:28], d.Lease.IP[:]) {
		copy(d.routeMAC[:], b[8:14])
		d.routeReady = true
		d.delay = 0
		return d.Poll(now)
	}
	return nil
}

func (d *DHCP) arp(source, target [4]byte) []byte {
	b := d.out[:42]
	for i := range b {
		b[i] = 0
	}
	for i := 0; i < 6; i++ {
		b[i] = 255
	}
	copy(b[6:12], d.MAC[:])
	put16(b, 12, 0x806)
	put16(b, 14, 1)
	put16(b, 16, 0x800)
	b[18] = 6
	b[19] = 4
	put16(b, 20, 1)
	copy(b[22:28], d.MAC[:])
	copy(b[28:32], source[:])
	copy(b[38:42], target[:])
	return b
}

func (d *DHCP) message(kind byte) []byte {
	for i := range d.out {
		d.out[i] = 0
	}
	frame := d.out[:]
	for i := 0; i < 6; i++ {
		frame[i] = 255
	}
	copy(frame[6:12], d.MAC[:])
	put16(frame, 12, 0x800)
	ip := frame[14:34]
	ip[0] = 0x45
	ip[8] = 64
	ip[9] = 17
	for i := 16; i < 20; i++ {
		ip[i] = 255
	}
	b := frame[42:]
	b[0] = 1
	b[1] = 1
	b[2] = 6
	put32(b, 4, d.xid)
	secs := d.seconds - d.transactionStart
	if secs > 65535 {
		secs = 65535
	}
	put16(b, 8, uint16(secs))
	put16(b, 10, 0x8000)
	if kind == 3 && (d.state == dhcpRenewing || d.state == dhcpRebinding) {
		copy(b[12:16], d.Lease.IP[:])
		copy(ip[12:16], d.Lease.IP[:])
		put16(b, 10, 0)
		if d.state == dhcpRenewing {
			copy(frame[:6], d.routeMAC[:])
			copy(ip[16:20], d.Lease.Server[:])
		}
	}
	copy(b[28:34], d.MAC[:])
	put32(b, 236, 0x63825363)
	i := 240
	b[i] = 53
	b[i+1] = 1
	b[i+2] = kind
	i += 3
	b[i] = 61
	b[i+1] = 7
	b[i+2] = 1
	copy(b[i+3:i+9], d.MAC[:])
	i += 9
	if (kind == 3 && d.state == dhcpRequesting) || kind == 4 {
		b[i] = 50
		b[i+1] = 4
		copy(b[i+2:i+6], d.Lease.IP[:])
		i += 6
		b[i] = 54
		b[i+1] = 4
		copy(b[i+2:i+6], d.Lease.Server[:])
		i += 6
	}
	if kind != 4 {
		b[i] = 55
		b[i+1] = 6
		copy(b[i+2:i+8], []byte{1, 3, 6, 51, 58, 59})
		i += 8
		b[i] = 57
		b[i+1] = 2
		put16(b, i+2, 576)
		i += 4
		n := len(d.Hostname)
		if n > 63 {
			n = 63
		}
		if n > 0 {
			b[i] = 12
			b[i+1] = byte(n)
			for j := 0; j < n; j++ {
				b[i+2+j] = d.Hostname[j]
			}
			i += 2 + n
		}
	} else {
		put16(b, 8, 0)
		put16(b, 10, 0)
	}
	b[i] = 255
	i++
	if i < 300 {
		i = 300
	}
	put16(ip, 2, uint16(28+i))
	put16(ip, 10, finish(sum(ip, 0)))
	u := frame[34 : 42+i]
	put16(u, 0, 68)
	put16(u, 2, 67)
	put16(u, 4, uint16(len(u)))
	check := finish(sum(u, sum(ip[12:20], uint32(17+len(u)))))
	if check == 0 {
		check = 65535
	}
	put16(u, 6, check)
	return frame[:42+i]
}
