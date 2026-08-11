// Package usb implements a portable, polling USB device stack.
package usb

import "errors"

var (
	ErrDescriptorOverflow = errors.New("usb descriptor buffer exhausted")
	ErrEndpointOverflow   = errors.New("usb endpoint budget exhausted")
	ErrInvalidConfig      = errors.New("invalid usb configuration")
	ErrBusy               = errors.New("usb endpoint busy")
)

// ControlNotHandled is returned by Function.Control when the request belongs
// to another function or is unsupported.
const ControlNotHandled = -1

// Direction is an endpoint transfer direction.
type Direction uint8

const (
	Out Direction = iota
	In
)

// TransferType is a USB endpoint transfer type.
type TransferType uint8

const (
	Control TransferType = iota
	Isochronous
	Bulk
	Interrupt
)

// EndpointConfig describes one non-control endpoint.
type EndpointConfig struct {
	Number        uint8
	Direction     Direction
	Transfer      TransferType
	MaxPacketSize uint16
	Interval      uint8
}

// Setup is one decoded eight-byte control request.
type Setup struct {
	RequestType uint8
	Request     uint8
	Value       uint16
	Index       uint16
	Length      uint16
}

// ParseSetup decodes the USB little-endian setup packet.
func ParseSetup(packet [8]byte) Setup {
	return Setup{
		RequestType: packet[0], Request: packet[1],
		Value:  uint16(packet[2]) | uint16(packet[3])<<8,
		Index:  uint16(packet[4]) | uint16(packet[5])<<8,
		Length: uint16(packet[6]) | uint16(packet[7])<<8,
	}
}

// EventKind identifies controller work delivered to Device.Poll.
type EventKind uint8

const (
	EventNone EventKind = iota
	EventReset
	EventSuspend
	EventResume
	EventSetup
	EventInComplete
	EventOut
	EventError
)

// Event is one bounded controller event.
type Event struct {
	Kind     EventKind
	Endpoint uint8
	Setup    [8]byte
	Data     []byte
	Err      error
}

// Controller is the hardware seam consumed by the portable device core.
type Controller interface {
	Start() error
	Connect() error
	Disconnect()
	Poll(*Event) bool
	OpenEndpoint(EndpointConfig) error
	Send(endpoint uint8, data []byte) error
	Receive(endpoint uint8, buffer []byte) error
	Stall(endpoint uint8, direction Direction)
	SetAddress(address uint8)
}

// EndpointIO is the transfer surface given to a bound USB function.
type EndpointIO interface {
	EndpointSend(endpoint uint8, data []byte) error
	EndpointReceive(endpoint uint8, buffer []byte) error
	StallEndpoint(endpoint uint8, direction Direction)
}

// Port is a board-defined USB device connection.
type Port struct{ controller Controller }

// DefinePort constructs a board port around a concrete controller.
func DefinePort(controller Controller) Port { return Port{controller: controller} }

// Function is one independently reusable USB function in a composite device.
type Function interface {
	Bind(*DescriptorBuilder) error
	Attach(EndpointIO)
	Control(Setup, []byte) int
	ControlOut(Setup, []byte) bool
	BOSDescriptor() []byte
	Configured(bool)
	Handle(Event)
}

// Config identifies a USB device and the functions it exposes.
type Config struct {
	VendorID     uint16
	ProductID    uint16
	DeviceBCD    uint16
	Manufacturer string
	Product      string
	Serial       string
	Functions    []Function
}
