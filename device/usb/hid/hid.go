// Package hid implements a USB Human Interface Device function.
package hid

import "renvo.dev/device/usb"

// Function is one HID interface with interrupt IN and optional OUT reports.
type Function struct {
	io                       usb.EndpointIO
	interfaceNumber, out, in uint8
	report                   []byte
	receive                  [64]byte
	lastOut                  [64]byte
	lastOutLength            int
	configured               bool
}

// New returns a HID function using reportDescriptor.
func New(reportDescriptor []byte) *Function { return &Function{report: reportDescriptor} }

func (f *Function) Bind(b *usb.DescriptorBuilder) error {
	f.interfaceNumber = b.Interface()
	var err error
	if f.out, err = b.Endpoint(usb.Out, usb.Interrupt, 64, 1); err != nil {
		return err
	}
	if f.in, err = b.Endpoint(usb.In, usb.Interrupt, 64, 1); err != nil {
		return err
	}
	if err = b.InterfaceDescriptor(f.interfaceNumber, 0, 2, 3, 0, 0, 0); err != nil {
		return err
	}
	length := len(f.report)
	if err = b.Append(9, 0x21, 0x11, 0x01, 0, 1, 0x22, byte(length), byte(length>>8)); err != nil {
		return err
	}
	if err = b.EndpointDescriptor(f.out, usb.Out, usb.Interrupt, 64, 1); err != nil {
		return err
	}
	return b.EndpointDescriptor(f.in, usb.In, usb.Interrupt, 64, 1)
}

func (f *Function) Attach(io usb.EndpointIO) { f.io = io }
func (f *Function) Control(setup *usb.Setup, buffer []byte) ([]byte, bool) {
	if uint8(setup.Index) != f.interfaceNumber {
		return nil, false
	}
	if setup.Request == 6 && byte(setup.Value>>8) == 0x22 {
		length := len(f.report)
		if int(setup.Length) < length {
			length = int(setup.Length)
		}
		copy(buffer, f.report[:length])
		return buffer[:length], true
	}
	if setup.RequestType&0x60 == 0x20 {
		switch setup.Request {
		case 1:
			return buffer[:0], true // GET_REPORT, empty default
		case 9, 10, 11:
			return buffer[:0], true // SET_REPORT/IDLE/PROTOCOL
		}
	}
	return nil, false
}
func (f *Function) ControlOut(setup *usb.Setup, data []byte) bool {
	if setup.Request == 9 {
		f.lastOutLength = copy(f.lastOut[:], data)
		return true
	}
	return false
}
func (*Function) BOSDescriptor() []byte { return nil }
func (f *Function) Configured(value bool) {
	f.configured = value
	if value {
		_ = f.io.EndpointReceive(f.out, f.receive[:])
	}
}
func (f *Function) Handle(event usb.Event) {
	if event.Kind == usb.EventOut && event.Endpoint == f.out {
		f.lastOutLength = copy(f.lastOut[:], event.Data)
		if f.configured {
			_ = f.io.EndpointReceive(f.out, f.receive[:])
		}
	}
}

// SendReport submits one interrupt report.
func (f *Function) SendReport(report []byte) error {
	if !f.configured {
		return usb.ErrBusy
	}
	return f.io.EndpointSend(f.in, report)
}

// LastOutputReport copies the most recent host report.
func (f *Function) LastOutputReport(output []byte) int {
	return copy(output, f.lastOut[:f.lastOutLength])
}
