package lowspeed

import "renvo.dev/device/usb"

// SIE is the packet-level half of a low-speed software USB controller. A PHY
// decodes host waveforms into HandlePacket calls and encodes TakeReply results
// back onto the bus.
type SIE struct {
	events     [16]usb.Event
	eventRead  uint8
	eventWrite uint8

	send       [16][8]byte
	sendLength [16]uint8
	sendReady  [16]bool
	receive    [16][]byte
	stalledIn  [16]bool
	stalledOut [16]bool
	toggleIn   [16]bool
	toggleOut  [16]bool

	address       uint8
	connected     bool
	pendingPID    byte
	pendingEP     uint8
	lastInEP      uint8
	lastInPending bool

	replyPID   byte
	replyData  [8]byte
	replyCount uint8
	replyReady bool
}

// Port exposes this controller to the portable USB device stack.
func (s *SIE) Port() usb.Port { return usb.DefinePort(s) }

func (s *SIE) push(event usb.Event) {
	next := (s.eventWrite + 1) % uint8(len(s.events))
	if next == s.eventRead {
		return
	}
	s.events[s.eventWrite] = event
	s.eventWrite = next
}

// BusReset resets address and data toggles and schedules a device reset event.
func (s *SIE) BusReset() {
	s.address = 0
	s.pendingPID = 0
	s.lastInPending = false
	s.replyReady = false
	s.eventRead = 0
	s.eventWrite = 0
	for endpoint := 0; endpoint < 16; endpoint++ {
		s.sendReady[endpoint] = false
		s.receive[endpoint] = nil
		s.stalledIn[endpoint] = false
		s.stalledOut[endpoint] = false
		s.toggleIn[endpoint] = false
		s.toggleOut[endpoint] = false
	}
	s.push(usb.Event{Kind: usb.EventReset})
}

func (s *SIE) token(data []byte) (address, endpoint uint8, ok bool) {
	if len(data) != 2 {
		return 0, 0, false
	}
	address = data[0] & 0x7f
	endpoint = data[0]>>7 | data[1]<<1
	return address, endpoint & 0x0f, true
}

func (s *SIE) reply(pid byte, data []byte) {
	s.replyPID = pid
	s.replyCount = uint8(copy(s.replyData[:], data))
	s.replyReady = true
}

// HandlePacket consumes one CRC-validated host packet.
func (s *SIE) HandlePacket(pid byte, data []byte) {
	if !s.connected {
		return
	}
	if pid == PIDOut || pid == PIDSetup || pid == PIDIn {
		address, endpoint, ok := s.token(data)
		if !ok || address != s.address {
			s.pendingPID = 0
			return
		}
		if pid == PIDIn {
			if s.stalledIn[endpoint] {
				s.reply(PIDStall, nil)
			} else if !s.sendReady[endpoint] {
				s.reply(PIDNak, nil)
			} else {
				dataPID := PIDData0
				if s.toggleIn[endpoint] {
					dataPID = PIDData1
				}
				s.reply(dataPID, s.send[endpoint][:s.sendLength[endpoint]])
				s.lastInEP = endpoint
				s.lastInPending = true
			}
			s.pendingPID = 0
			return
		}
		s.pendingPID, s.pendingEP = pid, endpoint
		return
	}
	if pid == PIDAck {
		if s.lastInPending {
			endpoint := s.lastInEP
			s.lastInPending = false
			s.sendReady[endpoint] = false
			s.toggleIn[endpoint] = !s.toggleIn[endpoint]
			s.push(usb.Event{Kind: usb.EventInComplete, Endpoint: endpoint})
		}
		return
	}
	if pid != PIDData0 && pid != PIDData1 || s.pendingPID == 0 {
		return
	}
	endpoint := s.pendingEP
	pending := s.pendingPID
	s.pendingPID = 0
	if pending == PIDSetup {
		if endpoint != 0 || pid != PIDData0 || len(data) != 8 {
			return
		}
		var setup [8]byte
		copy(setup[:], data)
		s.toggleIn[0], s.toggleOut[0] = true, true
		s.stalledIn[0], s.stalledOut[0] = false, false
		s.push(usb.Event{Kind: usb.EventSetup, Setup: setup})
		s.reply(PIDAck, nil)
		return
	}
	if s.stalledOut[endpoint] {
		s.reply(PIDStall, nil)
		return
	}
	expected := PIDData0
	if s.toggleOut[endpoint] {
		expected = PIDData1
	}
	if pid != expected {
		s.reply(PIDAck, nil)
		return
	}
	buffer := s.receive[endpoint]
	if buffer == nil || len(data) > len(buffer) {
		s.reply(PIDNak, nil)
		return
	}
	length := copy(buffer, data)
	s.receive[endpoint] = nil
	s.toggleOut[endpoint] = !s.toggleOut[endpoint]
	s.push(usb.Event{Kind: usb.EventOut, Endpoint: endpoint, Data: buffer[:length]})
	s.reply(PIDAck, nil)
}

// TakeReply returns the next handshake or data response for the PHY.
func (s *SIE) TakeReply(data []byte) (pid byte, count int, ok bool) {
	if !s.replyReady {
		return 0, 0, false
	}
	s.replyReady = false
	count = copy(data, s.replyData[:s.replyCount])
	return s.replyPID, count, true
}

func (*SIE) Start() error { return nil }
func (s *SIE) Connect() error {
	s.connected = true
	return nil
}
func (s *SIE) Disconnect() { s.connected = false }
func (s *SIE) Poll(event *usb.Event) bool {
	if s.eventRead == s.eventWrite {
		return false
	}
	*event = s.events[s.eventRead]
	s.eventRead = (s.eventRead + 1) % uint8(len(s.events))
	return true
}
func (*SIE) OpenEndpoint(config usb.EndpointConfig) error {
	if config.Number == 0 || config.Number > 15 || config.Transfer != usb.Interrupt || config.MaxPacketSize == 0 || config.MaxPacketSize > 8 {
		return usb.ErrInvalidConfig
	}
	return nil
}
func (s *SIE) Send(endpoint uint8, data []byte) error {
	if endpoint > 15 || len(data) > 8 {
		return usb.ErrInvalidConfig
	}
	if s.sendReady[endpoint] {
		return usb.ErrBusy
	}
	s.sendLength[endpoint] = uint8(copy(s.send[endpoint][:], data))
	s.sendReady[endpoint] = true
	return nil
}
func (s *SIE) Receive(endpoint uint8, buffer []byte) error {
	if endpoint > 15 {
		return usb.ErrInvalidConfig
	}
	s.receive[endpoint] = buffer
	return nil
}
func (s *SIE) Stall(endpoint uint8, direction usb.Direction) {
	if endpoint > 15 {
		return
	}
	if direction == usb.In {
		s.stalledIn[endpoint] = true
	} else {
		s.stalledOut[endpoint] = true
	}
}
func (s *SIE) SetAddress(address uint8) { s.address = address }
