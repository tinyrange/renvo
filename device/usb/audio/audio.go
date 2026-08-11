// Package audio implements a USB Audio 1.0 streaming function.
package audio

import "renvo.dev/device/usb"

const (
	requestGetCurrent    = uint8(0x81)
	requestGetMinimum    = uint8(0x82)
	requestGetMaximum    = uint8(0x83)
	requestGetResolution = uint8(0x84)
	requestSetCurrent    = uint8(0x01)

	samplingFrequencyControl = uint16(0x0100)
)

type Function struct {
	usb.DuplexPipe
	controlInterface, streamingOut, streamingIn uint8
	sampleRate                                  uint32
	packetSize                                  uint16
	outAlternate, inAlternate                   uint8
}

func New(sampleRate uint32) *Function { return &Function{sampleRate: sampleRate} }
func (f *Function) Bind(b *usb.DescriptorBuilder) error {
	// Two channels of signed 16-bit PCM are transferred once per full-speed
	// frame. Round up so non-integral kHz rates still reserve enough space.
	packet := (f.sampleRate*4 + 999) / 1000
	if packet == 0 || packet > 512 {
		return usb.ErrInvalidConfig
	}
	f.packetSize = uint16(packet)
	f.controlInterface, f.streamingOut, f.streamingIn = b.Interface(), b.Interface(), b.Interface()
	if err := f.BindEndpoints(b, usb.Isochronous, f.packetSize, 1); err != nil {
		return err
	}
	if err := b.InterfaceDescriptor(f.controlInterface, 0, 0, 1, 1, 0, 0); err != nil {
		return err
	}
	// AudioControl header plus a USB-streaming input terminal and speaker
	// output terminal, then the inverse microphone path. Keeping feature units
	// out makes the function fixed-volume and requires no mutable mixer state.
	if err := b.Append(10, 0x24, 1, 0, 1, 52, 0, 2, f.streamingOut, f.streamingIn); err != nil {
		return err
	}
	if err := b.Append(12, 0x24, 2, 1, 0x01, 0x01, 0, 2, 3, 0, 0, 0); err != nil {
		return err
	}
	if err := b.Append(9, 0x24, 3, 2, 0x01, 0x03, 0, 1, 0); err != nil {
		return err
	}
	if err := b.Append(12, 0x24, 2, 3, 0x01, 0x02, 0, 2, 3, 0, 0, 0); err != nil {
		return err
	}
	if err := b.Append(9, 0x24, 3, 4, 0x01, 0x01, 0, 3, 0); err != nil {
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
	// Adaptive OUT and asynchronous IN data endpoints, each followed by the
	// UAC1 class-specific endpoint descriptor. Sampling frequency is the only
	// advertised endpoint control.
	if err := b.EndpointDescriptorAttributes(f.Out, usb.Out, 0x09, f.packetSize, 1); err != nil {
		return err
	}
	if err := b.Append(7, 0x25, 1, 1, 0, 0, 0); err != nil {
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
	if err := b.EndpointDescriptorAttributes(f.In, usb.In, 0x05, f.packetSize, 1); err != nil {
		return err
	}
	return b.Append(7, 0x25, 1, 1, 0, 0, 0)
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
	// Audio endpoint sampling-frequency requests. This profile advertises one
	// discrete rate, so current/minimum/maximum are identical and resolution
	// is zero. Match the endpoint recipient and selector precisely.
	endpoint := uint8(setup.Index)
	if setup.RequestType&0x7f == 0x22 && setup.Value == samplingFrequencyControl &&
		(endpoint == f.Out || endpoint == f.In|0x80) {
		if setup.Request == requestGetCurrent || setup.Request == requestGetMinimum || setup.Request == requestGetMaximum {
			buffer[0], buffer[1], buffer[2] = byte(f.sampleRate), byte(f.sampleRate>>8), byte(f.sampleRate>>16)
			return 3
		}
		if setup.Request == requestGetResolution {
			buffer[0], buffer[1], buffer[2] = 0, 0, 0
			return 3
		}
		if setup.Request == requestSetCurrent && setup.Length == 3 {
			return 0
		}
	}
	return usb.ControlNotHandled
}
func (f *Function) ControlOut(setup usb.Setup, data []byte) bool {
	endpoint := uint8(setup.Index)
	if setup.RequestType&0x7f == 0x22 && setup.Request == requestSetCurrent &&
		setup.Value == samplingFrequencyControl && (endpoint == f.Out || endpoint == f.In|0x80) && len(data) == 3 {
		rate := uint32(data[0]) | uint32(data[1])<<8 | uint32(data[2])<<16
		return rate == f.sampleRate
	}
	return false
}
func (*Function) BOSDescriptor() []byte { return nil }
func (f *Function) Configured(value bool) {
	f.outAlternate, f.inAlternate = 0, 0
	f.Activate(false, false)
}
func (f *Function) Handle(event usb.Event) { f.HandleEvent(event) }
func (f *Function) WriteSamples(samples []byte) error {
	if f.inAlternate == 0 || len(samples) > int(f.packetSize) {
		return usb.ErrBusy
	}
	return f.Write(samples)
}

// ReadSamples returns one or more speaker frames queued by the host.
func (f *Function) ReadSamples(samples []byte) int {
	if f.outAlternate == 0 {
		return 0
	}
	return f.Read(samples)
}
