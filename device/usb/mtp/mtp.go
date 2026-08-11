// Package mtp implements the USB transport for Media Transfer Protocol containers.
package mtp

import (
	"errors"

	"renvo.dev/device/usb"
)

const (
	ContainerCommand  = uint16(1)
	ContainerData     = uint16(2)
	ContainerResponse = uint16(3)
	ContainerEvent    = uint16(4)
	MaxPayload        = 512
)

var ErrMalformed = errors.New("malformed mtp container")

// Container identifies one MTP command, data, response or event payload.
type Container struct {
	Type        uint16
	Code        uint16
	Transaction uint32
	Length      uint32
}

// ObjectInfo is the portable metadata exposed by an MTP object store.
type ObjectInfo struct {
	Handle uint32
	Parent uint32
	Size   uint32
	Format uint16
	Name   string
}

// ObjectStore is the bounded storage seam used by an MTP responder.
type ObjectStore interface {
	ObjectHandles(parent uint32, handles []uint32) (int, error)
	ObjectInfo(handle uint32, info *ObjectInfo) error
	ReadObject(handle, offset uint32, data []byte) (int, error)
	WriteObject(handle, offset uint32, data []byte) (int, error)
	DeleteObject(handle uint32) error
}

type Function struct {
	usb.DuplexPipe
	interfaceNumber, eventIn uint8
	eventIO                  usb.EndpointIO
	packet                   [12 + MaxPayload]byte
	packetLength             int
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

func put16(data []byte, value uint16) { data[0], data[1] = byte(value), byte(value>>8) }
func put32(data []byte, value uint32) {
	data[0], data[1], data[2], data[3] = byte(value), byte(value>>8), byte(value>>16), byte(value>>24)
}
func get16(data []byte) uint16 { return uint16(data[0]) | uint16(data[1])<<8 }
func get32(data []byte) uint32 {
	return uint32(data[0]) | uint32(data[1])<<8 | uint32(data[2])<<16 | uint32(data[3])<<24
}

// Encode writes one complete MTP container.
func Encode(container Container, payload, output []byte) (int, error) {
	if len(payload) > MaxPayload || len(output) < 12+len(payload) || container.Type < ContainerCommand || container.Type > ContainerEvent {
		return 0, ErrMalformed
	}
	put32(output[0:4], uint32(12+len(payload)))
	put16(output[4:6], container.Type)
	put16(output[6:8], container.Code)
	put32(output[8:12], container.Transaction)
	copy(output[12:], payload)
	return 12 + len(payload), nil
}

// Decode validates a complete container and copies its payload into output.
func Decode(packet []byte, container *Container, output []byte) (int, error) {
	if len(packet) < 12 || int(get32(packet[0:4])) != len(packet) || len(packet)-12 > MaxPayload || len(output) < len(packet)-12 {
		return 0, ErrMalformed
	}
	typ := get16(packet[4:6])
	if typ < ContainerCommand || typ > ContainerEvent {
		return 0, ErrMalformed
	}
	container.Type = typ
	container.Code = get16(packet[6:8])
	container.Transaction = get32(packet[8:12])
	container.Length = uint32(len(packet) - 12)
	copy(output, packet[12:])
	return len(packet) - 12, nil
}

func (f *Function) WriteContainer(container Container, payload []byte) error {
	length, err := Encode(container, payload, f.packet[:])
	if err != nil {
		return err
	}
	return f.Write(f.packet[:length])
}

func (f *Function) ReadContainer(container *Container, payload []byte) (int, error) {
	if f.packetLength < len(f.packet) {
		f.packetLength += f.Read(f.packet[f.packetLength:])
	}
	if f.packetLength < 12 {
		return 0, usb.ErrBusy
	}
	total := int(get32(f.packet[0:4]))
	if total < 12 || total > len(f.packet) {
		f.packetLength = 0
		return 0, ErrMalformed
	}
	if f.packetLength < total {
		return 0, usb.ErrBusy
	}
	count, err := Decode(f.packet[:total], container, payload)
	copy(f.packet[:], f.packet[total:f.packetLength])
	f.packetLength -= total
	return count, err
}
