package tcpip

// HTTP is a four-connection, HTTP/1.0 and HTTP/1.1 status server. Every response
// closes its TCP connection. Call Configure when the interface gains/loses an
// address, and set Status before Handle/Poll.
// It supports GET/HEAD for /, /status.json and /healthz, with no request bodies,
// TLS, keep-alive or pipelining. Requests are limited to 2048 header bytes.
type HTTP struct {
	Status   HTTPStatus
	slots    [4]httpSlot
	next     int
	Requests uint32
}

type HTTPStatus struct {
	IP, Mask, Gateway                      [4]byte
	MAC                                    [6]byte
	UptimeSeconds, LeaseSeconds, LeaseACKs uint32
}

type httpSlot struct {
	tcp Echo
	io  tcpStream
}

// Configure clears all connections and binds the server's Ethernet/IP/port.
// The HTTP value must stay at the same address after this call.
func (h *HTTP) Configure(mac [6]byte, ip [4]byte, port uint16) {
	for i := 0; i < len(h.slots); i++ {
		s := &h.slots[i]
		s.io.reset(0)
		s.tcp = Echo{MAC: mac, IP: ip, Port: port}
		s.tcp.stream = &s.io
	}
	h.next = 0
}

// Handle processes an Ethernet frame; the returned reply must be sent before
// the next Handle/Poll call. ARP and ICMP use the first slot's packet encoder.
func (h *HTTP) Handle(frame []byte, now uint32) []byte {
	if len(frame) < 34 {
		return h.slots[0].tcp.Handle(frame, now)
	}
	if get16(frame, 12) != 0x800 || frame[23] != 6 {
		return h.slots[0].tcp.Handle(frame, now)
	}
	ip := frame[14:]
	ih := int(ip[0]&15) * 4
	if ih < 20 || ih+20 > len(ip) {
		return nil
	}
	b := ip[ih:]
	port := get16(b, 0)
	chosen := -1
	for i := 0; i < len(h.slots); i++ {
		t := &h.slots[i].tcp
		if t.state != 0 && t.peerPort == port && equal(t.peerIP[:], ip[12:16]) {
			chosen = i
			break
		}
	}
	if chosen < 0 && b[13]&(syn|ack|rst|fin) == syn {
		for i := 0; i < len(h.slots); i++ {
			if h.slots[i].tcp.state == 0 || h.slots[i].tcp.state == 4 {
				chosen = i
				break
			}
		}
	}
	if chosen < 0 {
		return nil
	}
	s := &h.slots[chosen]
	reply := s.tcp.Handle(frame, now)
	h.prepare(s, now)
	if output := s.tcp.streamOutput(now); len(output) > 0 {
		return output
	}
	return reply
}

// Poll services response segments, retransmission and request timeouts fairly.
func (h *HTTP) Poll(now uint32) []byte {
	for count := 0; count < len(h.slots); count++ {
		s := &h.slots[h.next]
		h.next = (h.next + 1) % len(h.slots)
		h.prepare(s, now)
		if reply := s.tcp.Poll(now); len(reply) > 0 {
			return reply
		}
	}
	return nil
}

func (h *HTTP) prepare(slot *httpSlot, now uint32) {
	s := &slot.io
	if slot.tcp.state != 2 || s.responding {
		return
	}
	end := -1
	for i := 3; i < s.inputLen; i++ {
		if s.input[i-3] == 13 && s.input[i-2] == 10 && s.input[i-1] == 13 && s.input[i] == 10 {
			end = i + 1
			break
		}
	}
	if end < 0 {
		if s.inputLen == len(s.input) {
			h.respond(s, 431, "text/plain", "Request headers too large\n", false)
		} else if s.eof {
			h.respond(s, 400, "text/plain", "Incomplete request\n", false)
		} else if now-s.started >= 5000 {
			h.respond(s, 408, "text/plain", "Request timeout\n", false)
		}
		return
	}
	code, path, head := parseHTTPRequest(s.input[:end])
	if code != 200 {
		h.respond(s, code, "text/plain", httpReason(code)+"\n", head)
		return
	}
	switch path {
	case "/":
		h.page(s, head)
	case "/status.json":
		h.statusJSON(s, head)
	case "/healthz":
		h.respond(s, 200, "text/plain", "ok\n", head)
	case "/favicon.ico":
		h.respond(s, 204, "text/plain", "", head)
	default:
		h.respond(s, 404, "text/plain", "Not found\n", head)
	}
}

func tokenByte(b byte) bool {
	if b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9' {
		return true
	}
	const symbols = "!#$%&'*+-.^_`|~"
	for i := 0; i < len(symbols); i++ {
		if b == symbols[i] {
			return true
		}
	}
	return false
}
func lowerEqual(b []byte, s string) bool {
	if len(b) != len(s) {
		return false
	}
	for i, v := range b {
		if v >= 'A' && v <= 'Z' {
			v += 32
		}
		if v != s[i] {
			return false
		}
	}
	return true
}

// Parse only a complete header block. Strict framing avoids ambiguous request
// boundaries; bodies and transfer codings are deliberately unsupported.
func parseHTTPRequest(b []byte) (int, string, bool) {
	lineEnd := -1
	for i := 1; i < len(b); i++ {
		if b[i-1] == 13 && b[i] == 10 {
			lineEnd = i - 1
			break
		}
	}
	if lineEnd < 0 {
		return 400, "", false
	}
	line := b[:lineEnd]
	a, c := -1, -1
	for i, v := range line {
		if v == 32 {
			if a < 0 {
				a = i
			} else if c < 0 {
				c = i
			} else {
				return 400, "", false
			}
		} else if v < 33 || v > 126 {
			return 400, "", false
		}
	}
	if a <= 0 || c <= a+1 || c+1 >= len(line) {
		return 400, "", false
	}
	method := string(line[:a])
	head := method == "HEAD"
	version := string(line[c+1:])
	if version != "HTTP/1.0" && version != "HTTP/1.1" {
		return 505, "", head
	}
	if line[a+1] != '/' || c-a-1 > 256 {
		return 400, "", head
	}
	path := string(line[a+1 : c])
	for i, v := range path {
		if v == '?' {
			path = path[:i]
			break
		}
	}
	hosts, lengths := 0, 0
	for pos := lineEnd + 2; pos < len(b); {
		end := -1
		for i := pos + 1; i < len(b); i++ {
			if b[i-1] == 13 && b[i] == 10 {
				end = i - 1
				break
			}
		}
		if end < 0 {
			return 400, "", head
		}
		if end == pos {
			break
		}
		header := b[pos:end]
		colon := -1
		for i, v := range header {
			if v == ':' {
				colon = i
				break
			}
			if !tokenByte(v) {
				return 400, "", head
			}
		}
		if colon <= 0 {
			return 400, "", head
		}
		value := header[colon+1:]
		for len(value) > 0 && (value[0] == ' ' || value[0] == '\t') {
			value = value[1:]
		}
		for len(value) > 0 && (value[len(value)-1] == ' ' || value[len(value)-1] == '\t') {
			value = value[:len(value)-1]
		}
		for _, v := range value {
			if v < 32 && v != '\t' || v == 127 {
				return 400, "", head
			}
		}
		name := header[:colon]
		if lowerEqual(name, "host") {
			hosts++
			if len(value) == 0 || hosts > 1 {
				return 400, "", head
			}
		}
		if lowerEqual(name, "transfer-encoding") {
			return 400, "", head
		}
		if lowerEqual(name, "content-length") {
			lengths++
			if lengths > 1 || len(value) == 0 {
				return 400, "", head
			}
			for _, v := range value {
				if v != '0' {
					return 400, "", head
				}
			}
		}
		pos = end + 2
	}
	if version == "HTTP/1.1" && hosts != 1 {
		return 400, "", head
	}
	if method != "GET" && !head {
		return 405, "", false
	}
	return 200, path, head
}

func httpReason(code int) string {
	switch code {
	case 200:
		return "OK"
	case 204:
		return "No Content"
	case 400:
		return "Bad Request"
	case 404:
		return "Not Found"
	case 405:
		return "Method Not Allowed"
	case 408:
		return "Request Timeout"
	case 431:
		return "Request Header Fields Too Large"
	case 505:
		return "HTTP Version Not Supported"
	}
	return "Internal Server Error"
}

// Build into fixed storage. Body starts after 256 bytes of header headroom.
type httpWriter struct {
	bytes []byte
	n     int
}

func (w *httpWriter) text(s string) {
	for i := 0; i < len(s) && w.n < len(w.bytes); i++ {
		w.bytes[w.n] = s[i]
		w.n++
	}
}
func (w *httpWriter) number(v uint32) {
	var digits [10]byte
	n := 0
	for {
		digits[n] = byte(v%10) + '0'
		n++
		v /= 10
		if v == 0 {
			break
		}
	}
	for n > 0 {
		n--
		if w.n < len(w.bytes) {
			w.bytes[w.n] = digits[n]
			w.n++
		}
	}
}
func (w *httpWriter) ip(ip [4]byte) {
	for i, v := range ip {
		if i > 0 {
			w.text(".")
		}
		w.number(uint32(v))
	}
}
func (w *httpWriter) mac(mac [6]byte) {
	const hex = "0123456789abcdef"
	for i, v := range mac {
		if i > 0 {
			w.text(":")
		}
		w.text(hex[int(v>>4) : int(v>>4)+1])
		w.text(hex[int(v&15) : int(v&15)+1])
	}
}

func (h *HTTP) finishResponse(s *tcpStream, code int, kind string, body int, head bool) {
	w := httpWriter{bytes: s.output[:256]}
	w.text("HTTP/1.1 ")
	w.number(uint32(code))
	w.text(" ")
	w.text(httpReason(code))
	w.text("\r\nContent-Type: ")
	w.text(kind)
	if code != 204 {
		w.text("\r\nContent-Length: ")
		w.number(uint32(body))
	}
	w.text("\r\nConnection: close\r\nCache-Control: no-store\r\nX-Content-Type-Options: nosniff\r\n")
	if code == 405 {
		w.text("Allow: GET, HEAD\r\n")
	}
	w.text("\r\n")
	s.outputLen = w.n
	if !head {
		copy(s.output[w.n:], s.output[256:256+body])
		s.outputLen += body
	}
	s.outputPos = 0
	s.responding = true
	h.Requests++
}
func (h *HTTP) respond(s *tcpStream, code int, kind, body string, head bool) {
	w := httpWriter{bytes: s.output[256:]}
	w.text(body)
	h.finishResponse(s, code, kind, w.n, head)
}
func (h *HTTP) statusJSON(s *tcpStream, head bool) {
	w := httpWriter{bytes: s.output[256:]}
	w.text("{\"device\":\"Unit PoE-P4\",\"ip\":\"")
	w.ip(h.Status.IP)
	w.text("\",\"mac\":\"")
	w.mac(h.Status.MAC)
	w.text("\",\"mask\":\"")
	w.ip(h.Status.Mask)
	w.text("\",\"gateway\":\"")
	w.ip(h.Status.Gateway)
	w.text("\",\"uptime_seconds\":")
	w.number(h.Status.UptimeSeconds)
	w.text(",\"lease_seconds\":")
	w.number(h.Status.LeaseSeconds)
	w.text(",\"lease_acks\":")
	w.number(h.Status.LeaseACKs)
	w.text(",\"http_requests\":")
	w.number(h.Requests + 1)
	w.text("}\n")
	h.finishResponse(s, 200, "application/json", w.n, head)
}
func (h *HTTP) page(s *tcpStream, head bool) {
	w := httpWriter{bytes: s.output[256:]}
	w.text("<!doctype html><html lang=\"en\"><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width,initial-scale=1\"><title>Renvo | Unit PoE-P4</title><style>body{margin:0;background:#101820;color:#e9f1f5;font:17px system-ui,sans-serif}main{max-width:680px;margin:8vh auto;padding:32px}small{color:#8fadc0;letter-spacing:.12em}h1{font-size:44px;margin:16px 0}p{color:#adc2ce;line-height:1.6}.live{color:#75e3b4}dl{background:#192731;border:1px solid #304654;border-radius:16px;padding:24px;display:grid;grid-template-columns:1fr 1.5fr;gap:18px}dt{color:#adc2ce}dd{margin:0;overflow-wrap:anywhere}a{color:#75e3b4;display:inline-block;margin:16px 24px 0 0}footer{font-size:14px;color:#8fadc0;margin-top:40px}@media(max-width:480px){main{padding:20px}h1{font-size:34px}dl{grid-template-columns:1fr;gap:8px}dd{margin-bottom:12px}}</style><main><small>RENVO / ETHERNET</small><h1>Unit PoE-P4</h1><p class=\"live\">Online &middot; DHCP connected</p><p>This page is served directly by the ESP32-P4, running a Renvo-compiled web server over Ethernet.</p><dl><dt>IP address</dt><dd>")
	w.ip(h.Status.IP)
	w.text("</dd><dt>MAC address</dt><dd>")
	w.mac(h.Status.MAC)
	w.text("</dd><dt>Subnet mask</dt><dd>")
	w.ip(h.Status.Mask)
	w.text("</dd><dt>Gateway</dt><dd>")
	w.ip(h.Status.Gateway)
	w.text("</dd><dt>Uptime</dt><dd>")
	w.number(h.Status.UptimeSeconds)
	w.text(" seconds</dd><dt>DHCP lease</dt><dd>")
	w.number(h.Status.LeaseSeconds)
	w.text(" seconds</dd><dt>Lease acknowledgements</dt><dd>")
	w.number(h.Status.LeaseACKs)
	w.text("</dd><dt>HTTP requests</dt><dd>")
	w.number(h.Requests + 1)
	w.text("</dd></dl><a href=\"/\">Refresh status</a><a href=\"/status.json\">JSON status</a><a href=\"/healthz\">Health check</a><footer>ESP32-P4 &middot; 100 Mbps Ethernet &middot; Renvo</footer></main></html>")
	h.finishResponse(s, 200, "text/html; charset=utf-8", w.n, head)
}
