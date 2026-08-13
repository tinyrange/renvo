// Package cdcacm implements a USB CDC ACM serial function.
package cdcacm

import "renvo.dev/device/usb"

// Function is one CDC ACM control/data interface pair.
type Function struct {
	io                     usb.EndpointIO
	controlInterface       uint8
	dataInterface          uint8
	notifyIn, dataIn       uint8
	dataOut                uint8
	configured             bool
	lineCoding             [7]byte
	controlLineState       uint16
	receive                [64]byte
	queued                 [512]byte
	queueRead, queueLength int
}

// New returns a CDC ACM function configured for 115200 8N1 by default.
func New() *Function {
	f := &Function{}
	f.lineCoding = [7]byte{0x00, 0xc2, 0x01, 0x00, 0, 0, 8}
	return f
}

func (f *Function) Bind(b *usb.DescriptorBuilder) error {
	f.controlInterface = b.Interface()
	f.dataInterface = b.Interface()
	var err error
	if f.notifyIn, err = b.Endpoint(usb.In, usb.Interrupt, 8, 16); err != nil {
		return err
	}
	if f.dataOut, err = b.Endpoint(usb.Out, usb.Bulk, 64, 0); err != nil {
		return err
	}
	if f.dataIn, err = b.Endpoint(usb.In, usb.Bulk, 64, 0); err != nil {
		return err
	}
	if err = b.Append(8, 11, f.controlInterface, 2, 2, 2, 1, 0); err != nil {
		return err
	}
	if err = b.InterfaceDescriptor(f.controlInterface, 0, 1, 2, 2, 1, 0); err != nil {
		return err
	}
	if err = b.Append(5, 0x24, 0, 0x10, 0x01); err != nil {
		return err
	}
	if err = b.Append(5, 0x24, 1, 0, f.dataInterface); err != nil {
		return err
	}
	if err = b.Append(4, 0x24, 2, 2); err != nil {
		return err
	}
	if err = b.Append(5, 0x24, 6, f.controlInterface, f.dataInterface); err != nil {
		return err
	}
	if err = b.EndpointDescriptor(f.notifyIn, usb.In, usb.Interrupt, 8, 16); err != nil {
		return err
	}
	if err = b.InterfaceDescriptor(f.dataInterface, 0, 2, 0x0a, 0, 0, 0); err != nil {
		return err
	}
	if err = b.EndpointDescriptor(f.dataOut, usb.Out, usb.Bulk, 64, 0); err != nil {
		return err
	}
	return b.EndpointDescriptor(f.dataIn, usb.In, usb.Bulk, 64, 0)
}

func (f *Function) Attach(io usb.EndpointIO) { f.io = io }

func (f *Function) Control(setup usb.Setup, buffer []byte) int {
	if uint8(setup.Index) != f.controlInterface {
		return usb.ControlNotHandled
	}
	if setup.RequestType&0x60 != 0x20 {
		return usb.ControlNotHandled
	}
	switch setup.Request {
	case 0x20: // SET_LINE_CODING; data is accepted by the EP0 core.
		if setup.Length == 7 {
			return 0
		}
	case 0x21: // GET_LINE_CODING
		copy(buffer, f.lineCoding[:])
		return 7
	case 0x22: // SET_CONTROL_LINE_STATE
		f.controlLineState = setup.Value
		if setup.Length == 0 {
			return 0
		}
	case 0x23: // SEND_BREAK
		if setup.Length == 0 {
			return 0
		}
	}
	return usb.ControlNotHandled
}

func (f *Function) ControlOut(setup usb.Setup, data []byte) bool {
	if setup.Request == 0x20 && len(data) == len(f.lineCoding) {
		copy(f.lineCoding[:], data)
		return true
	}
	return false
}
func (*Function) BOSDescriptor() []byte { return nil }

func (f *Function) Configured(configured bool) {
	f.configured = configured
	if configured && f.io != nil {
		_ = f.io.EndpointReceive(f.dataOut, f.receive[:])
	}
}

func (f *Function) Handle(event usb.Event) {
	if event.Kind != usb.EventOut || event.Endpoint != f.dataOut {
		return
	}
	for _, value := range event.Data {
		if f.queueLength == len(f.queued) {
			break
		}
		at := (f.queueRead + f.queueLength) % len(f.queued)
		f.queued[at] = value
		f.queueLength++
	}
	if f.configured {
		_ = f.io.EndpointReceive(f.dataOut, f.receive[:])
	}
}

// Write submits up to one full-speed packet.
func (f *Function) Write(data []byte) error {
	if !f.configured || len(data) > 64 {
		return usb.ErrBusy
	}
	return f.io.EndpointSend(f.dataIn, data)
}

// Read copies queued host data without blocking.
func (f *Function) Read(data []byte) int {
	count := len(data)
	if count > f.queueLength {
		count = f.queueLength
	}
	for index := 0; index < count; index++ {
		data[index] = f.queued[f.queueRead]
		f.queueRead = (f.queueRead + 1) % len(f.queued)
	}
	f.queueLength -= count
	return count
}

func (f *Function) DataEndpoints() (out, in uint8) { return f.dataOut, f.dataIn }
