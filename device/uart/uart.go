// Package uart defines portable asynchronous serial ports.
package uart

import (
	"errors"

	"renvo.dev/device/gpio"
)

var ErrInvalidConfig = errors.New("invalid uart configuration")

// Controller is the chip-specific UART capability carried by a board port.
type Controller interface {
	Configure(transmit, receive gpio.Pin, baud uint32) error
	Write([]byte) (int, error)
	Read([]byte) (int, error)
}

// Port is immutable board wiring and its default baud rate.
type Port struct {
	controller Controller
	transmit   gpio.Pin
	receive    gpio.Pin
	baud       uint32
}

func DefinePort(controller Controller, transmit, receive gpio.Pin, baud uint32) Port {
	return Port{controller: controller, transmit: transmit, receive: receive, baud: baud}
}

// Device is one lazily configured UART.
type Device struct {
	port        Port
	initialized bool
	err         error
}

func New(port Port) *Device { return &Device{port: port} }

func (d *Device) initialize() error {
	if d.initialized {
		return d.err
	}
	d.initialized = true
	if d.port.controller == nil || d.port.transmit == nil || d.port.receive == nil || d.port.baud == 0 {
		d.err = ErrInvalidConfig
		return d.err
	}
	d.err = d.port.controller.Configure(d.port.transmit, d.port.receive, d.port.baud)
	return d.err
}

func (d *Device) Write(data []byte) (int, error) {
	if err := d.initialize(); err != nil {
		return 0, err
	}
	return d.port.controller.Write(data)
}

func (d *Device) Read(data []byte) (int, error) {
	if err := d.initialize(); err != nil {
		return 0, err
	}
	return d.port.controller.Read(data)
}
