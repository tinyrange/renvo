// Package vendor implements a vendor-specific bulk function with WebUSB BOS
// and landing-page descriptors.
package vendor

import "renvo.dev/device/usb"

// Function is one bidirectional vendor bulk interface.
type Function struct {
	io                       usb.EndpointIO
	interfaceNumber, out, in uint8
	requestCode              uint8
	url                      string
	receive                  [64]byte
	configured               bool
	last                     [64]byte
	lastLength               int
}

// NewWebUSB creates a WebUSB-capable vendor function.
func NewWebUSB(requestCode uint8, landingURL string) *Function {
	return &Function{requestCode: requestCode, url: landingURL}
}

func (f *Function) Bind(b *usb.DescriptorBuilder) error {
	f.interfaceNumber = b.Interface()
	var err error
	if f.out, err = b.Endpoint(usb.Out, usb.Bulk, 64, 0); err != nil {
		return err
	}
	if f.in, err = b.Endpoint(usb.In, usb.Bulk, 64, 0); err != nil {
		return err
	}
	if err = b.InterfaceDescriptor(f.interfaceNumber, 0, 2, 0xff, 0, 0, 0); err != nil {
		return err
	}
	if err = b.EndpointDescriptor(f.out, usb.Out, usb.Bulk, 64, 0); err != nil {
		return err
	}
	return b.EndpointDescriptor(f.in, usb.In, usb.Bulk, 64, 0)
}
func (f *Function) Attach(io usb.EndpointIO) { f.io = io }
func (f *Function) BOSDescriptor() []byte {
	// BOS header followed by the WebUSB platform capability UUID.
	return []byte{5, 15, 29, 0, 1, 24, 16, 5, 0, 0, 0, 0x38, 0xb6, 0x08, 0x34, 0xa9, 0x09, 0xa0, 0x47, 0x8b, 0xfd, 0xa0, 0x76, 0x88, 0x15, 0xb6, 0x65, 0, 1, f.requestCode, 1}
}
func (f *Function) Control(setup *usb.Setup, buffer []byte) ([]byte, bool) {
	if setup.RequestType != 0xc0 || setup.Request != f.requestCode || setup.Index != 2 || setup.Value != 1 {
		return nil, false
	}
	length := len(f.url)
	if length > len(buffer)-3 {
		length = len(buffer) - 3
	}
	buffer[0], buffer[1], buffer[2] = byte(length+3), 3, 1 // HTTPS
	copy(buffer[3:], f.url[:length])
	return buffer[:length+3], true
}
func (*Function) ControlOut(*usb.Setup, []byte) bool { return false }
func (f *Function) Configured(value bool) {
	f.configured = value
	if value {
		_ = f.io.EndpointReceive(f.out, f.receive[:])
	}
}
func (f *Function) Handle(event usb.Event) {
	if event.Kind == usb.EventOut && event.Endpoint == f.out {
		f.lastLength = copy(f.last[:], event.Data)
		if f.configured {
			_ = f.io.EndpointReceive(f.out, f.receive[:])
		}
	}
}
func (f *Function) Write(data []byte) error {
	if !f.configured {
		return usb.ErrBusy
	}
	return f.io.EndpointSend(f.in, data)
}
func (f *Function) Read(output []byte) int { return copy(output, f.last[:f.lastLength]) }
