// Package adb implements the USB transport framing used by Android Debug Bridge.
package adb

import "renvo.dev/device/usb"

// Function is one ADB vendor interface (class ff/subclass 42/protocol 01).
type Function struct {
	usb.DuplexPipe
	interfaceNumber uint8
}

func New() *Function { return &Function{} }
func (f *Function) Bind(b *usb.DescriptorBuilder) error {
	f.interfaceNumber = b.Interface()
	if err := f.BindEndpoints(b, usb.Bulk, 64, 0); err != nil {
		return err
	}
	if err := b.InterfaceDescriptor(f.interfaceNumber, 0, 2, 0xff, 0x42, 1, 0); err != nil {
		return err
	}
	if err := b.EndpointDescriptor(f.Out, usb.Out, usb.Bulk, 64, 0); err != nil {
		return err
	}
	return b.EndpointDescriptor(f.In, usb.In, usb.Bulk, 64, 0)
}
func (f *Function) Attach(io usb.EndpointIO)                { f.DuplexPipe.Attach(io) }
func (*Function) Control(*usb.Setup, []byte) ([]byte, bool) { return nil, false }
func (*Function) ControlOut(*usb.Setup, []byte) bool        { return false }
func (*Function) BOSDescriptor() []byte                     { return nil }
func (f *Function) Configured(value bool)                   { f.ConfiguredState(value) }
func (f *Function) Handle(event usb.Event)                  { f.HandleEvent(event) }
