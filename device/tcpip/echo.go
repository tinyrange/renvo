// Package tcpip provides a bounded IPv4 TCP echo endpoint for device bring-up.
// It handles Ethernet II, ARP, ICMP echo and one TCP connection at a time.
// It is a polling diagnostic endpoint, not a general socket implementation.
package tcpip

const (
	fin = byte(1)
	syn = byte(2)
	rst = byte(4)
	psh = byte(8)
	ack = byte(16)
	mss = 512
)

// Echo uses a static IPv4 address on the attached Ethernet LAN. Configure MAC,
// IP and Port before use; call Handle for each frame and Poll even when idle.
// Returned frames exclude the FCS and remain valid until the next method call.
// One segment is kept for retransmission, bounding memory and the send window.
// No DHCP, routing, IPv6, VLANs, IP reassembly or TCP extensions are implemented.
type Echo struct {
	MAC                            [6]byte
	IP                             [4]byte
	Port                           uint16
	stream                         *tcpStream
	peerMAC                        [6]byte
	peerIP                         [4]byte
	peerPort                       uint16
	state                          uint8 // 0 listen, 1 SYN received, 2 established, 3 FIN sent, 4 TIME-WAIT, 5 FIN-WAIT-2
	recvNext, sendNext, pendingSeq uint32
	peerWindow                     uint16
	peerMSS                        int
	pendingFlags                   byte
	pending                        [mss]byte
	pendingLen                     int
	pendingEnd                     uint32
	retries                        uint8
	lastSent, lastActivity         uint32
	serial                         uint32
	out                            [14 + 20 + 24 + mss]byte
}

func get16(b []byte, i int) uint16    { return uint16(b[i])<<8 | uint16(b[i+1]) }
func get32(b []byte, i int) uint32    { return uint32(get16(b, i))<<16 | uint32(get16(b, i+2)) }
func put16(b []byte, i int, v uint16) { b[i] = byte(v >> 8); b[i+1] = byte(v) }
func put32(b []byte, i int, v uint32) { put16(b, i, uint16(v>>16)); put16(b, i+2, uint16(v)) }
func sum(b []byte, s uint32) uint32 {
	for len(b) >= 2 {
		s += uint32(get16(b, 0))
		b = b[2:]
	}
	if len(b) != 0 {
		s += uint32(b[0]) << 8
	}
	return s
}
func finish(s uint32) uint16 {
	for s>>16 != 0 {
		s = (s & 65535) + (s >> 16)
	}
	return ^uint16(s)
}
func equal(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

func (e *Echo) ethernet(dst []byte, kind uint16) {
	copy(e.out[:6], dst)
	copy(e.out[6:12], e.MAC[:])
	put16(e.out[:], 12, kind)
}
func (e *Echo) ipv4(dst []byte, proto byte, n int) {
	b := e.out[14:34]
	for i := range b {
		b[i] = 0
	}
	b[0] = 0x45
	put16(b, 2, uint16(20+n))
	b[8] = 64
	b[9] = proto
	copy(b[12:16], e.IP[:])
	copy(b[16:20], dst)
	put16(b, 10, finish(sum(b, 0)))
}

// Handle processes a received frame. now is a wrapping millisecond counter.
// Malformed, fragmented and checksum-invalid packets are ignored.
func (e *Echo) Handle(frame []byte, now uint32) []byte {
	if len(frame) < 14 {
		return nil
	}
	broadcast := true
	for _, v := range frame[:6] {
		if v != 255 {
			broadcast = false
		}
	}
	if !broadcast && !equal(frame[:6], e.MAC[:]) {
		return nil
	}
	kind := get16(frame, 12)
	if kind == 0x806 {
		if len(frame) < 42 {
			return nil
		}
		b := frame[14:42]
		if get16(b, 0) != 1 || get16(b, 2) != 0x800 || b[4] != 6 || b[5] != 4 || get16(b, 6) != 1 || !equal(b[24:28], e.IP[:]) {
			return nil
		}
		e.ethernet(b[8:14], 0x806)
		r := e.out[14:42]
		copy(r, b)
		put16(r, 6, 2)
		copy(r[18:24], b[8:14])
		copy(r[24:28], b[14:18])
		copy(r[8:14], e.MAC[:])
		copy(r[14:18], e.IP[:])
		return e.out[:42]
	}
	if kind != 0x800 || broadcast || len(frame) < 34 {
		return nil
	}
	ip := frame[14:]
	h := int(ip[0]&15) * 4
	n := int(get16(ip, 2))
	if ip[0]>>4 != 4 || h < 20 || h > len(ip) || n < h || n > len(ip) || get16(ip, 6)&0x3fff != 0 || !equal(ip[16:20], e.IP[:]) || finish(sum(ip[:h], 0)) != 0 {
		return nil
	}
	b := ip[h:n]
	if ip[9] == 1 {
		if len(b) < 8 || len(b) > len(e.out)-34 || b[0] != 8 || b[1] != 0 || finish(sum(b, 0)) != 0 {
			return nil
		}
		e.ethernet(frame[6:12], 0x800)
		e.ipv4(ip[12:16], 1, len(b))
		r := e.out[34 : 34+len(b)]
		copy(r, b)
		r[0] = 0
		put16(r, 2, 0)
		put16(r, 2, finish(sum(r, 0)))
		return e.out[:34+len(b)]
	}
	if ip[9] != 6 || len(b) < 20 || get16(b, 2) != e.Port {
		return nil
	}
	th := int(b[12]>>4) * 4
	if th < 20 || th > len(b) || finish(sum(b, sum(ip[12:20], uint32(6+len(b))))) != 0 {
		return nil
	}
	flags := b[13]
	seq := get32(b, 4)
	a := get32(b, 8)
	port := get16(b, 0)
	match := port == e.peerPort && equal(ip[12:16], e.peerIP[:])
	if flags&(syn|ack|rst|fin) == syn && (e.state == 0 || e.state == 4 && !match) {
		peerMSS := 536
		for i := 20; i < th; {
			k := b[i]
			if k == 0 {
				break
			}
			if k == 1 {
				i++
				continue
			}
			if i+1 >= th || b[i+1] < 2 || i+int(b[i+1]) > th {
				return nil
			}
			if k == 2 {
				if b[i+1] != 4 {
					return nil
				}
				peerMSS = int(get16(b, i+2))
				if peerMSS == 0 {
					return nil
				}
			}
			i += int(b[i+1])
		}
		if peerMSS > mss {
			peerMSS = mss
		}
		copy(e.peerMAC[:], frame[6:12])
		copy(e.peerIP[:], ip[12:16])
		e.peerPort = port
		e.peerMSS = peerMSS
		e.peerWindow = get16(b, 14)
		e.serial += 65537
		e.sendNext = now*64000 + e.serial
		e.recvNext = seq + 1
		e.state = 1
		e.lastActivity = now
		if e.stream != nil {
			e.stream.reset(now)
		}
		return e.queue(syn|ack, nil, now)
	}
	if e.state == 0 || !match {
		return nil
	}
	// RSTs must match the next expected sequence (RFC 5961).
	if flags&rst != 0 {
		if seq == e.recvNext {
			e.state = 0
			e.pendingFlags = 0
		}
		return nil
	}
	if flags&syn != 0 {
		if e.state == 1 && seq+1 == e.recvNext {
			return e.segment(e.pendingSeq, e.pendingFlags, e.pending[:e.pendingLen])
		}
		return nil
	}
	if flags&ack == 0 {
		return nil
	}
	if int32(a-e.sendNext) > 0 {
		return e.segment(e.sendNext, ack, nil)
	}
	if e.state == 4 {
		if flags&fin != 0 && seq+uint32(len(b)-th)+1 == e.recvNext {
			e.lastActivity = now
			return e.segment(e.sendNext, ack, nil)
		}
		return nil
	}
	if e.state == 1 && (a != e.sendNext || seq != e.recvNext) {
		return nil
	}
	payload := b[th:]
	duplicate := false
	if int32(seq-e.recvNext) < 0 {
		skip := e.recvNext - seq
		if skip <= uint32(len(payload)) {
			payload = payload[int(skip):]
			seq = e.recvNext
			duplicate = true
		}
	}
	if seq != e.recvNext {
		return e.segment(e.sendNext, ack, nil)
	}
	e.lastActivity = now
	oldWindow := e.peerWindow
	e.peerWindow = get16(b, 14)
	openedWindow := false
	if e.pendingFlags != 0 && a == e.pendingEnd {
		openedWindow = e.pendingFlags&syn == 0
		e.pendingFlags = 0
		if e.state == 1 {
			e.state = 2
		}
		if e.state == 3 {
			if e.stream != nil && !e.stream.eof {
				e.state = 5
			} else {
				e.state = 4
				return nil
			}
		}
	}
	if e.pendingFlags != 0 {
		if len(payload) != 0 || flags&fin != 0 || duplicate {
			return e.segment(e.sendNext, ack, nil)
		}
		return nil
	}
	if len(payload) > mss {
		return e.segment(e.sendNext, ack, nil)
	}
	if e.stream != nil {
		if e.state == 5 {
			if flags&fin != 0 && len(payload) == 0 {
				e.recvNext++
				e.stream.eof = true
				e.state = 4
			}
			return e.segment(e.sendNext, ack, nil)
		}
		if e.state != 2 {
			return nil
		}
		count := len(payload)
		space := len(e.stream.input) - e.stream.inputLen
		if count > space {
			count = space
		}
		if count > 0 {
			copy(e.stream.input[e.stream.inputLen:], payload[:count])
			e.stream.inputLen += count
			e.recvNext += uint32(count)
		}
		if count == len(payload) && flags&fin != 0 {
			e.recvNext++
			e.stream.eof = true
		}
		if frame := e.streamOutput(now); len(frame) > 0 {
			return frame
		}
		if count > 0 || flags&fin != 0 || openedWindow || duplicate {
			return e.segment(e.sendNext, ack, nil)
		}
		return nil
	}
	if len(payload) != 0 {
		count := len(payload)
		if count > e.peerMSS {
			count = e.peerMSS
		}
		if count > int(e.peerWindow) {
			count = int(e.peerWindow)
		}
		if count == 0 {
			return e.segment(e.sendNext, ack, nil)
		}
		complete := count == len(payload)
		payload = payload[:count]
		e.recvNext += uint32(len(payload))
		if complete && flags&fin != 0 {
			e.recvNext++
			e.state = 3
			return e.queue(psh|fin|ack, payload, now)
		}
		return e.queue(psh|ack, payload, now)
	}
	if flags&fin != 0 {
		e.recvNext++
		e.state = 3
		return e.queue(fin|ack, nil, now)
	}
	if openedWindow || duplicate || oldWindow == 0 && e.peerWindow > 0 {
		return e.segment(e.sendNext, ack, nil)
	}
	return nil
}

func (e *Echo) queue(flags byte, data []byte, now uint32) []byte {
	e.pendingSeq = e.sendNext
	e.pendingFlags = flags
	e.pendingLen = copy(e.pending[:], data)
	e.sendNext += uint32(len(data))
	if flags&(syn|fin) != 0 {
		e.sendNext++
	}
	e.pendingEnd = e.sendNext
	e.lastSent = now
	e.retries = 0
	return e.segment(e.pendingSeq, flags, e.pending[:e.pendingLen])
}

func (e *Echo) segment(seq uint32, flags byte, data []byte) []byte {
	h := 20
	if flags&syn != 0 {
		h = 24
	}
	e.ethernet(e.peerMAC[:], 0x800)
	e.ipv4(e.peerIP[:], 6, h+len(data))
	b := e.out[34 : 34+h+len(data)]
	for i := 0; i < h; i++ {
		b[i] = 0
	}
	put16(b, 0, e.Port)
	put16(b, 2, e.peerPort)
	put32(b, 4, seq)
	put32(b, 8, e.recvNext)
	b[12] = byte(h/4) << 4
	b[13] = flags
	window := uint16(mss)
	if e.stream != nil {
		space := len(e.stream.input) - e.stream.inputLen
		if int(window) > space {
			window = uint16(space)
		}
	}
	if e.peerMSS > 0 && int(window) > e.peerMSS {
		window = uint16(e.peerMSS)
	}
	if e.peerWindow == 0 {
		window = 0
	}
	if e.pendingFlags != 0 && e.pendingFlags&syn == 0 {
		window = 0
	}
	put16(b, 14, window)
	if h == 24 {
		b[20] = 2
		b[21] = 4
		put16(b, 22, mss)
	}
	copy(b[h:], data)
	put16(b, 16, finish(sum(b, sum(e.out[26:34], uint32(6+len(b))))))
	return e.out[:34+len(b)]
}

// Poll retransmits unacknowledged segments with a bounded exponential backoff
// and expires idle connections after 60 seconds. TIME-WAIT lasts 60 seconds;
// a fresh connection from a different peer port can reuse this diagnostic slot.
func (e *Echo) Poll(now uint32) []byte {
	if e.state == 0 {
		return nil
	}
	timeout := uint32(60000)
	if e.stream != nil {
		timeout = 15000
	}
	if now-e.lastActivity >= timeout {
		e.state = 0
		e.pendingFlags = 0
		return nil
	}
	if e.pendingFlags == 0 {
		return e.streamOutput(now)
	}
	shift := e.retries
	if shift > 3 {
		shift = 3
	}
	if now-e.lastSent < uint32(500)<<shift {
		return nil
	}
	if e.retries >= 6 {
		e.state = 0
		e.pendingFlags = 0
		return nil
	}
	e.retries++
	e.lastSent = now
	return e.segment(e.pendingSeq, e.pendingFlags, e.pending[:e.pendingLen])
}
