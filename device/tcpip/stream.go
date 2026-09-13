package tcpip

// tcpStream gives the shared packet engine bounded receive/response storage.
// The owner fills output once and sets responding. The engine copies each
// segment into its retransmission buffer before advancing outputPos.
type tcpStream struct {
	input                          [2048]byte
	output                         [4096]byte
	inputLen, outputLen, outputPos int
	responding, eof                bool
	started                        uint32
}

func (s *tcpStream) reset(now uint32) {
	s.inputLen = 0
	s.outputLen = 0
	s.outputPos = 0
	s.responding = false
	s.eof = false
	s.started = now
}

func (e *Echo) streamOutput(now uint32) []byte {
	if e.stream == nil || !e.stream.responding || e.pendingFlags != 0 || e.state != 2 || e.peerWindow == 0 {
		return nil
	}
	s := e.stream
	n := s.outputLen - s.outputPos
	if n > mss {
		n = mss
	}
	if n > e.peerMSS {
		n = e.peerMSS
	}
	if n > int(e.peerWindow) {
		n = int(e.peerWindow)
	}
	flags := ack | psh
	if s.outputPos+n == s.outputLen {
		flags |= fin
		e.state = 3
	}
	data := s.output[s.outputPos : s.outputPos+n]
	s.outputPos += n
	return e.queue(flags, data, now)
}
