// Package midi implements USB MIDI 1.0 event packet transport.
package midi

import "renvo.dev/device/usb"

// Event is one four-byte USB-MIDI 1.0 event packet.
type Event struct {
	Cable uint8
	Code  uint8
	Byte0 byte
	Byte1 byte
	Byte2 byte
}

func (e Event) Bytes(output []byte) bool {
	if len(output) < 4 || e.Cable > 15 || e.Code > 15 {
		return false
	}
	output[0] = e.Cable<<4 | e.Code
	output[1], output[2], output[3] = e.Byte0, e.Byte1, e.Byte2
	return true
}

func Parse(packet []byte, event *Event) bool {
	if len(packet) != 4 {
		return false
	}
	event.Cable, event.Code = packet[0]>>4, packet[0]&15
	event.Byte0, event.Byte1, event.Byte2 = packet[1], packet[2], packet[3]
	return true
}

type Function struct {
	usb.DuplexPipe
	controlInterface, streamingInterface uint8
}

func New() *Function { return &Function{} }
func (f *Function) Bind(b *usb.DescriptorBuilder) error {
	f.controlInterface, f.streamingInterface = b.Interface(), b.Interface()
	if err := f.BindEndpoints(b, usb.Bulk, 64, 0); err != nil {
		return err
	}
	if err := b.InterfaceDescriptor(f.controlInterface, 0, 0, 1, 1, 0, 0); err != nil {
		return err
	}
	if err := b.Append(9, 0x24, 1, 0, 1, 9, 0, 1, f.streamingInterface); err != nil {
		return err
	}
	if err := b.InterfaceDescriptor(f.streamingInterface, 0, 2, 1, 3, 0, 0); err != nil {
		return err
	}
	// MIDIStreaming header and one embedded IN/OUT jack pair.
	if err := b.Append(7, 0x24, 1, 0, 1, 47, 0); err != nil {
		return err
	}
	if err := b.Append(6, 0x24, 2, 1, 1, 0); err != nil {
		return err
	}
	if err := b.Append(9, 0x24, 3, 1, 2, 1, 1, 1, 0); err != nil {
		return err
	}
	if err := b.EndpointDescriptor(f.Out, usb.Out, usb.Bulk, 64, 0); err != nil {
		return err
	}
	if err := b.Append(5, 0x25, 1, 1, 1); err != nil {
		return err
	}
	if err := b.EndpointDescriptor(f.In, usb.In, usb.Bulk, 64, 0); err != nil {
		return err
	}
	return b.Append(5, 0x25, 1, 1, 2)
}
func (f *Function) Attach(io usb.EndpointIO)                { f.DuplexPipe.Attach(io) }
func (*Function) Control(*usb.Setup, []byte) ([]byte, bool) { return nil, false }
func (*Function) ControlOut(*usb.Setup, []byte) bool        { return false }
func (*Function) BOSDescriptor() []byte                     { return nil }
func (f *Function) Configured(value bool)                   { f.ConfiguredState(value) }
func (f *Function) Handle(event usb.Event)                  { f.HandleEvent(event) }

// WriteEvent sends one or more four-byte USB-MIDI event packets.
func (f *Function) WriteEvent(packets []byte) error {
	if len(packets)%4 != 0 {
		return usb.ErrInvalidConfig
	}
	return f.Write(packets)
}
