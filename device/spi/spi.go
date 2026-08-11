// Package spi defines portable synchronous-serial ports and buses.
package spi

import (
	"errors"

	"renvo.dev/device/gpio"
)

var (
	ErrInvalidConfig = errors.New("invalid spi configuration")
	ErrBusy          = errors.New("spi controller busy")
)

// Controller is the chip-specific seam used by a board-defined SPI port.
type Controller interface {
	Configure(clock, output, input gpio.Pin, frequency uint32, mode uint8) error
	Tx(write, read []byte) error
}

// Port is immutable board wiring for one external SPI connector.
type Port struct {
	controller Controller
	clock      gpio.Pin
	output     gpio.Pin
	input      gpio.Pin
	frequency  uint32
	mode       uint8
}

// DefinePort binds a controller to board wiring. Board packages use this to
// export named connectors; applications normally call New(board.HeaderSPI).
func DefinePort(controller Controller, clock, output, input gpio.Pin, frequency uint32, mode uint8) Port {
	return Port{controller: controller, clock: clock, output: output, input: input, frequency: frequency, mode: mode}
}

// Bus is one lazily configured SPI connection.
type Bus struct {
	port        Port
	initialized bool
	err         error
}

func New(port Port) *Bus { return &Bus{port: port} }

func (b *Bus) initialize() error {
	if b.initialized {
		return b.err
	}
	b.initialized = true
	if b.port.controller == nil || b.port.clock == nil || b.port.output == nil || b.port.frequency == 0 || b.port.mode > 3 {
		b.err = ErrInvalidConfig
		return b.err
	}
	b.err = b.port.controller.Configure(b.port.clock, b.port.output, b.port.input, b.port.frequency, b.port.mode)
	return b.err
}

// Tx performs one full-duplex transaction. Either direction may be nil.
func (b *Bus) Tx(write, read []byte) error {
	if err := b.initialize(); err != nil {
		return err
	}
	return b.port.controller.Tx(write, read)
}
