// Package cdcethernet implements a USB CDC Ethernet Control Model function.
package cdcethernet

import "renvo.dev/device/usb"

type Function struct {
	usb.DuplexPipe
	controlInterface, dataInterface, notifyIn uint8
	io                                        usb.EndpointIO
	packetFilter                              uint16
}

func New() *Function { return &Function{} }
func (f *Function) Bind(b *usb.DescriptorBuilder) error {
	f.controlInterface, f.dataInterface = b.Interface(), b.Interface()
	var err error
	if f.notifyIn, err = b.Endpoint(usb.In, usb.Interrupt, 16, 9); err != nil {
		return err
	}
	if err = f.BindEndpoints(b, usb.Bulk, 64, 0); err != nil {
		return err
	}
	if err = b.Append(8, 11, f.controlInterface, 2, 2, 6, 0, 0); err != nil {
		return err
	}
	if err = b.InterfaceDescriptor(f.controlInterface, 0, 1, 2, 6, 0, 0); err != nil {
		return err
	}
	if err = b.Append(5, 0x24, 0, 0x10, 1); err != nil {
		return err
	}
	if err = b.Append(5, 0x24, 6, f.controlInterface, f.dataInterface); err != nil {
		return err
	}
	// Ethernet functional descriptor: string index 0, 1514-byte segment.
	if err = b.Append(13, 0x24, 15, 0, 0, 0, 0, 0, 0xea, 5, 0, 0, 0); err != nil {
		return err
	}
	if err = b.EndpointDescriptor(f.notifyIn, usb.In, usb.Interrupt, 16, 9); err != nil {
		return err
	}
	if err = b.InterfaceDescriptor(f.dataInterface, 0, 0, 0x0a, 0, 0, 0); err != nil {
		return err
	}
	if err = b.InterfaceDescriptor(f.dataInterface, 1, 2, 0x0a, 0, 0, 0); err != nil {
		return err
	}
	if err = b.EndpointDescriptor(f.Out, usb.Out, usb.Bulk, 64, 0); err != nil {
		return err
	}
	return b.EndpointDescriptor(f.In, usb.In, usb.Bulk, 64, 0)
}
func (f *Function) Attach(io usb.EndpointIO) { f.io = io; f.DuplexPipe.Attach(io) }
func (f *Function) Control(setup *usb.Setup, buffer []byte) ([]byte, bool) {
	if setup.RequestType&0x60 == 0 && setup.Request == 11 && uint8(setup.Index) == f.dataInterface {
		return buffer[:0], setup.Length == 0 && setup.Value <= 1
	}
	if uint8(setup.Index) != f.controlInterface || setup.RequestType&0x60 != 0x20 {
		return nil, false
	}
	if setup.Request == 0x43 {
		f.packetFilter = setup.Value
		return buffer[:0], true
	}
	return nil, false
}
func (*Function) ControlOut(*usb.Setup, []byte) bool { return false }
func (*Function) BOSDescriptor() []byte              { return nil }
func (f *Function) Configured(value bool) {
	f.ConfiguredState(value)
	if value {
		// NETWORK_CONNECTION notification (connected).
		n := []byte{0xa1, 0, 1, 0, f.controlInterface, 0, 0, 0}
		_ = f.io.EndpointSend(f.notifyIn, n)
	}
}
func (f *Function) Handle(event usb.Event) { f.HandleEvent(event) }
func (f *Function) WriteFrame(frame []byte) error {
	if len(frame) > 1514 {
		return usb.ErrInvalidConfig
	}
	return f.Write(frame)
}
