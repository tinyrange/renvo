// Package mtp implements the USB transport for Media Transfer Protocol containers.
package mtp

import "renvo.dev/device/usb"

type Function struct {
	usb.DuplexPipe
	interfaceNumber, eventIn uint8
	eventIO                  usb.EndpointIO
}

func New() *Function { return &Function{} }
func (f *Function) Bind(b *usb.DescriptorBuilder) error {
	f.interfaceNumber = b.Interface()
	if err := f.BindEndpoints(b, usb.Bulk, 64, 0); err != nil {
		return err
	}
	var err error
	if f.eventIn, err = b.Endpoint(usb.In, usb.Interrupt, 16, 6); err != nil {
		return err
	}
	if err = b.InterfaceDescriptor(f.interfaceNumber, 0, 3, 6, 1, 1, 0); err != nil {
		return err
	}
	if err = b.EndpointDescriptor(f.Out, usb.Out, usb.Bulk, 64, 0); err != nil {
		return err
	}
	if err = b.EndpointDescriptor(f.In, usb.In, usb.Bulk, 64, 0); err != nil {
		return err
	}
	return b.EndpointDescriptor(f.eventIn, usb.In, usb.Interrupt, 16, 6)
}
func (f *Function) Attach(io usb.EndpointIO) { f.eventIO = io; f.DuplexPipe.Attach(io) }
func (f *Function) Control(setup *usb.Setup, buffer []byte) ([]byte, bool) {
	if uint8(setup.Index) != f.interfaceNumber || setup.RequestType&0x60 != 0x20 {
		return nil, false
	}
	// Cancel, GetExtendedEventData, DeviceReset and GetDeviceStatus.
	if setup.Request == 0x64 || setup.Request == 0x65 || setup.Request == 0x66 {
		return buffer[:0], true
	}
	if setup.Request == 0x67 {
		buffer[0], buffer[1], buffer[2], buffer[3] = 4, 0, 1, 0x20
		return buffer[:4], true
	}
	return nil, false
}
func (f *Function) ControlOut(setup *usb.Setup, data []byte) bool {
	return setup.Request == 0x64 && len(data) == 6
}
func (*Function) BOSDescriptor() []byte    { return nil }
func (f *Function) Configured(value bool)  { f.ConfiguredState(value) }
func (f *Function) Handle(event usb.Event) { f.HandleEvent(event) }
func (f *Function) SendEvent(container []byte) error {
	if !f.Active {
		return usb.ErrBusy
	}
	return f.eventIO.EndpointSend(f.eventIn, container)
}
