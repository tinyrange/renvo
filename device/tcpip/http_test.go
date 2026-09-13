package tcpip

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestHTTPRequestParsing(t *testing.T) {
	for _, tc := range []struct {
		request string
		code    int
	}{
		{"GET / HTTP/1.1\r\nHost: board\r\n\r\n", 200},
		{"HEAD /healthz?x=1 HTTP/1.0\r\n\r\n", 200},
		{"GET / HTTP/1.1\r\n\r\n", 400},
		{"GET / HTTP/1.1\r\nHost: a\r\nHost: b\r\n\r\n", 400},
		{"GET / HTTP/1.1\r\nHost: a\r\nContent-Length: 1\r\n\r\n", 400},
		{"GET / HTTP/1.1\r\nHost: a\r\nTransfer-Encoding: chunked\r\n\r\n", 400},
		{"GET / HTTP/1.1\r\nHost : a\r\n\r\n", 400},
		{"POST / HTTP/1.0\r\n\r\n", 405},
		{"GET / HTTP/2.0\r\n\r\n", 505},
	} {
		code, _, _ := parseHTTPRequest([]byte(tc.request))
		if code != tc.code {
			t.Fatalf("%q: got %d want %d", tc.request, code, tc.code)
		}
	}
}

func TestHTTPFragmentedRequestResponseAndClose(t *testing.T) {
	h := &HTTP{}
	e := endpoint()
	h.Configure(e.MAC, e.IP, 80)
	e = &h.slots[0].tcp
	seq := uint32(100)
	r := checked(t, h.Handle(packet(e, seq, 0, syn, nil), 10))
	server := binary.BigEndian.Uint32(r[4:]) + 1
	seq++
	h.Handle(packet(e, seq, server, ack, nil), 11)
	first := []byte("GET / HTTP/1.1\r\nHost: bo")
	r = checked(t, h.Handle(packet(e, seq, server, ack|psh, first), 20))
	seq += uint32(len(first))
	if len(r) != 20 {
		t.Fatal("responded before complete headers")
	}
	last := []byte("ard\r\n\r\n")
	r = checked(t, h.Handle(packet(e, seq, server, ack|psh, last), 30))
	seq += uint32(len(last))
	retry := checked(t, h.Poll(530))
	if !bytes.Equal(r, retry) {
		t.Fatal("retransmission changed response")
	}
	var response []byte
	for i := 0; i < 10; i++ {
		if binary.BigEndian.Uint32(r[4:]) != server {
			t.Fatal("response sequence gap")
		}
		data := r[int(r[12]>>4)*4:]
		response = append(response, data...)
		server += uint32(len(data))
		closed := r[13]&fin != 0
		if closed {
			server++
		}
		next := h.Handle(packet(e, seq, server, ack, nil), uint32(540+i))
		if closed {
			if e.state != 5 {
				t.Fatalf("expected FIN_WAIT2, got %d", e.state)
			}
			r = checked(t, h.Handle(packet(e, seq, server, ack|fin, nil), 560))
			if e.state != 4 || binary.BigEndian.Uint32(r[8:]) != seq+1 {
				t.Fatal("peer FIN not acknowledged")
			}
			break
		}
		r = checked(t, next)
	}
	res, err := http.ReadResponse(bufio.NewReader(bytes.NewReader(response)), nil)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil || res.StatusCode != 200 || !strings.Contains(string(body), "Unit PoE-P4") || int64(len(body)) != res.ContentLength || len(body) < 1024 {
		t.Fatalf("bad response: %v %v", res, err)
	}
	if h.Requests != 1 {
		t.Fatal("duplicate request counted")
	}
}

func TestHTTPResponsesAndBounds(t *testing.T) {
	for _, tc := range []struct {
		path, method string
		code         int
	}{
		{"/healthz", "GET", 200}, {"/status.json", "GET", 200}, {"/", "HEAD", 200}, {"/missing", "GET", 404}, {"/favicon.ico", "GET", 204},
	} {
		h := &HTTP{}
		s := &h.slots[0]
		s.tcp.state = 2
		s.io.inputLen = copy(s.io.input[:], tc.method+" "+tc.path+" HTTP/1.1\r\nHost: board\r\n\r\n")
		h.prepare(s, 10)
		res, err := http.ReadResponse(bufio.NewReader(bytes.NewReader(s.io.output[:s.io.outputLen])), &http.Request{Method: tc.method})
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(res.Body)
		if err != nil || res.StatusCode != tc.code {
			t.Fatalf("%s: %v %v", tc.path, res, err)
		}
		if tc.method == "HEAD" && (len(body) != 0 || res.ContentLength < 1024) {
			t.Fatal("HEAD framing")
		}
		if tc.code == 204 && res.Header.Get("Content-Length") != "" {
			t.Fatal("204 content length")
		}
	}
	for _, full := range []bool{false, true} {
		h := &HTTP{}
		s := &h.slots[0]
		s.tcp.state = 2
		want := "408"
		if full {
			s.io.inputLen = len(s.io.input)
			want = "431"
		}
		h.prepare(s, 5000)
		if !strings.HasPrefix(string(s.io.output[:s.io.outputLen]), "HTTP/1.1 "+want) {
			t.Fatal("missing bounded-request error")
		}
	}
}
