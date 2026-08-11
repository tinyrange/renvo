// Package audio implements a USB Audio 1.0 streaming function.
package audio

import "renvo.dev/device/usb"

type Function struct {
	usb.DuplexPipe
	controlInterface, streamingOut, streamingIn uint8
	sampleRate                                  uint32
	outAlternate, inAlternate                   uint8
}

func New(sampleRate uint32) *Function { return &Function{sampleRate: sampleRate} }
func (f *Function) Bind(b *usb.DescriptorBuilder) error {
	f.controlInterface, f.streamingOut, f.streamingIn = b.Interface(), b.Interface(), b.Interface()
	if err := f.BindEndpoints(b, usb.Isochronous, 192, 1); err != nil {
		return err
	}
	if err := b.InterfaceDescriptor(f.controlInterface, 0, 0, 1, 1, 0, 0); err != nil {
		return err
	}
	// Compact AudioControl header referencing both streaming interfaces.
	if err := b.Append(10, 0x24, 1, 0, 1, 10, 0, 2, f.streamingOut, f.streamingIn); err != nil {
		return err
	}
	if err := b.InterfaceDescriptor(f.streamingOut, 0, 0, 1, 2, 0, 0); err != nil {
		return err
	}
	if err := b.InterfaceDescriptor(f.streamingOut, 1, 1, 1, 2, 0, 0); err != nil {
		return err
	}
	if err := b.Append(7, 0x24, 1, 1, 1, 1, 0); err != nil {
		return err
	}
	if err := b.Append(11, 0x24, 2, 1, 2, 2, 16, 1, byte(f.sampleRate), byte(f.sampleRate>>8), byte(f.sampleRate>>16)); err != nil {
		return err
	}
	if err := b.EndpointDescriptor(f.Out, usb.Out, usb.Isochronous, 192, 1); err != nil {
		return err
	}
	if err := b.InterfaceDescriptor(f.streamingIn, 0, 0, 1, 2, 0, 0); err != nil {
		return err
	}
	if err := b.InterfaceDescriptor(f.streamingIn, 1, 1, 1, 2, 0, 0); err != nil {
		return err
	}
	if err := b.Append(7, 0x24, 1, 2, 1, 1, 0); err != nil {
		return err
	}
	if err := b.Append(11, 0x24, 2, 1, 2, 2, 16, 1, byte(f.sampleRate), byte(f.sampleRate>>8), byte(f.sampleRate>>16)); err != nil {
		return err
	}
	return b.EndpointDescriptor(f.In, usb.In, usb.Isochronous, 192, 1)
}
func (f *Function) Attach(io usb.EndpointIO) { f.DuplexPipe.Attach(io) }
func (f *Function) Control(setup usb.Setup, buffer []byte) int {
	if setup.RequestType&0x60 == 0 && setup.Request == 10 {
		if uint8(setup.Index) == f.streamingOut {
			buffer[0] = f.outAlternate
			return 1
		}
		if uint8(setup.Index) == f.streamingIn {
			buffer[0] = f.inAlternate
			return 1
		}
	}
	if setup.RequestType&0x60 == 0 && setup.Request == 11 &&
		(uint8(setup.Index) == f.streamingOut || uint8(setup.Index) == f.streamingIn) {
		if setup.Length != 0 || setup.Value > 1 {
			return usb.ControlNotHandled
		}
		if uint8(setup.Index) == f.streamingOut {
			f.outAlternate = uint8(setup.Value)
		} else {
			f.inAlternate = uint8(setup.Value)
		}
		f.Activate(f.outAlternate != 0 || f.inAlternate != 0, f.outAlternate != 0)
		return 0
	}
	// Audio endpoint sampling-frequency GET_CUR/SET_CUR.
	if setup.RequestType&0x60 == 0x20 && setup.Request == 0x81 {
		buffer[0], buffer[1], buffer[2] = byte(f.sampleRate), byte(f.sampleRate>>8), byte(f.sampleRate>>16)
		return 3
	}
	if setup.RequestType&0x60 == 0x20 && setup.Request == 1 {
		return 0
	}
	return usb.ControlNotHandled
}
func (f *Function) ControlOut(setup usb.Setup, data []byte) bool {
	if setup.Request == 1 && len(data) == 3 {
		f.sampleRate = uint32(data[0]) | uint32(data[1])<<8 | uint32(data[2])<<16
		return f.sampleRate != 0
	}
	return false
}
func (*Function) BOSDescriptor() []byte { return nil }
func (f *Function) Configured(value bool) {
	f.outAlternate, f.inAlternate = 0, 0
	f.Activate(false, false)
}
func (f *Function) Handle(event usb.Event)            { f.HandleEvent(event) }
func (f *Function) WriteSamples(samples []byte) error { return f.Write(samples) }
