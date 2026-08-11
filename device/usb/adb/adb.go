// Package adb implements the USB transport framing used by Android Debug Bridge.
package adb

import (
	"errors"

	"renvo.dev/device/usb"
)

const (
	CommandSync    = uint32(0x434e5953)
	CommandConnect = uint32(0x4e584e43)
	CommandAuth    = uint32(0x48545541)
	CommandOpen    = uint32(0x4e45504f)
	CommandOkay    = uint32(0x59414b4f)
	CommandClose   = uint32(0x45534c43)
	CommandWrite   = uint32(0x45545257)
	MaxPayload     = 512
)

var (
	ErrMalformed       = errors.New("malformed adb message")
	ErrPayloadTooLarge = errors.New("adb payload exceeds bounded transport")
)

// Message is one decoded ADB transport packet.
type Message struct {
	Command   uint32
	Argument0 uint32
	Argument1 uint32
	Length    uint32
}

// Function is one ADB vendor interface (class ff/subclass 42/protocol 01).
type Function struct {
	usb.DuplexPipe
	interfaceNumber uint8
	packet          [24 + MaxPayload]byte
	packetLength    int
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

func put32(data []byte, value uint32) {
	data[0], data[1], data[2], data[3] = byte(value), byte(value>>8), byte(value>>16), byte(value>>24)
}

func get32(data []byte) uint32 {
	return uint32(data[0]) | uint32(data[1])<<8 | uint32(data[2])<<16 | uint32(data[3])<<24
}

func checksum(data []byte) uint32 {
	var sum uint32
	for _, value := range data {
		sum += uint32(value)
	}
	return sum
}

// Encode writes one complete ADB transport packet into output.
func Encode(message Message, payload, output []byte) (int, error) {
	if len(payload) > MaxPayload || len(output) < 24+len(payload) {
		return 0, ErrPayloadTooLarge
	}
	put32(output[0:4], message.Command)
	put32(output[4:8], message.Argument0)
	put32(output[8:12], message.Argument1)
	put32(output[12:16], uint32(len(payload)))
	put32(output[16:20], checksum(payload))
	put32(output[20:24], message.Command^0xffffffff)
	copy(output[24:], payload)
	return 24 + len(payload), nil
}

// Decode validates one complete packet and copies its payload into output.
func Decode(packet []byte, message *Message, output []byte) (int, error) {
	if len(packet) < 24 {
		return 0, ErrMalformed
	}
	length := int(get32(packet[12:16]))
	command := get32(packet[0:4])
	if length > MaxPayload || len(packet) != 24+length || len(output) < length ||
		get32(packet[20:24]) != command^0xffffffff || get32(packet[16:20]) != checksum(packet[24:]) {
		return 0, ErrMalformed
	}
	message.Command = command
	message.Argument0 = get32(packet[4:8])
	message.Argument1 = get32(packet[8:12])
	message.Length = uint32(length)
	copy(output, packet[24:])
	return length, nil
}

// WriteMessage frames and submits one bounded ADB message.
func (f *Function) WriteMessage(message Message, payload []byte) error {
	length, err := Encode(message, payload, f.packet[:])
	if err != nil {
		return err
	}
	return f.Write(f.packet[:length])
}

// ReadMessage consumes one complete message from the bounded receive queue.
// ErrBusy means that a complete header or payload has not arrived yet.
func (f *Function) ReadMessage(message *Message, payload []byte) (int, error) {
	if f.packetLength < len(f.packet) {
		f.packetLength += f.Read(f.packet[f.packetLength:])
	}
	if f.packetLength < 24 {
		return 0, usb.ErrBusy
	}
	length := int(get32(f.packet[12:16]))
	if length > MaxPayload {
		f.packetLength = 0
		return 0, ErrMalformed
	}
	total := 24 + length
	if f.packetLength < total {
		return 0, usb.ErrBusy
	}
	count, err := Decode(f.packet[:total], message, payload)
	copy(f.packet[:], f.packet[total:f.packetLength])
	f.packetLength -= total
	return count, err
}
