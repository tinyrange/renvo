package usb

// DuplexPipe is reusable bounded endpoint plumbing for class functions. Class
// packages embed it and supply their own descriptors and protocol framing.
type DuplexPipe struct {
	IO      EndpointIO
	Out, In uint8
	Active  bool
	receive [512]byte
	queued  [2048]byte
	read    int
	length  int
}

// BindEndpoints reserves one OUT and one IN endpoint.
func (p *DuplexPipe) BindEndpoints(b *DescriptorBuilder, transfer TransferType, packet uint16, interval uint8) error {
	var err error
	if p.Out, err = b.Endpoint(Out, transfer, packet, interval); err != nil {
		return err
	}
	if p.In, err = b.Endpoint(In, transfer, packet, interval); err != nil {
		return err
	}
	return nil
}

func (p *DuplexPipe) Attach(io EndpointIO) { p.IO = io }

func (p *DuplexPipe) ConfiguredState(value bool) {
	p.Activate(value, value)
}

// Activate changes the data-path state and optionally arms the OUT endpoint.
// Alternate-setting classes use this to keep their zero-bandwidth setting
// inactive until the host selects a streaming interface.
func (p *DuplexPipe) Activate(value, armReceive bool) {
	p.Active = value
	if value && armReceive && p.IO != nil {
		_ = p.IO.EndpointReceive(p.Out, p.receive[:])
	}
}

func (p *DuplexPipe) HandleEvent(event Event) {
	if event.Kind != EventOut || event.Endpoint != p.Out {
		return
	}
	for _, value := range event.Data {
		if p.length == len(p.queued) {
			break
		}
		p.queued[(p.read+p.length)%len(p.queued)] = value
		p.length++
	}
	if p.Active {
		_ = p.IO.EndpointReceive(p.Out, p.receive[:])
	}
}

// Write submits one endpoint transfer.
func (p *DuplexPipe) Write(data []byte) error {
	if !p.Active {
		return ErrBusy
	}
	return p.IO.EndpointSend(p.In, data)
}

// Read copies queued OUT data without blocking.
func (p *DuplexPipe) Read(data []byte) int {
	count := len(data)
	if count > p.length {
		count = p.length
	}
	for index := 0; index < count; index++ {
		data[index] = p.queued[p.read]
		p.read = (p.read + 1) % len(p.queued)
	}
	p.length -= count
	return count
}
