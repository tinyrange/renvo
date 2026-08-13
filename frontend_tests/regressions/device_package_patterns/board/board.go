// Package board models the package-global composition pattern used by device
// boards without touching real MMIO during this frontend regression.
package board

import (
	"renvo.dev/device/clock"
	"renvo.dev/device/gpio"
	"renvo.dev/device/i2c"
)

type pin struct {
	high       bool
	configured int
}

func (p *pin) Configure(gpio.Config) error {
	p.configured++
	return nil
}
func (p *pin) Set(high bool) { p.high = high }
func (p *pin) Get() bool     { return p.high }
func (p *pin) ConfigureOpenDrain() error {
	p.high = true
	return nil
}
func (p *pin) PullLow()   { p.high = false }
func (p *pin) Release()   { p.high = true }
func (p *pin) High() bool { return p.high }

type source struct{ ticks uint32 }

func (s *source) Ticks() uint32 {
	s.ticks++
	return s.ticks
}
func (*source) TicksPerSecond() uint32 { return 1000000 }

type controller struct {
	configured int
	address    uint16
	writes     int
	reads      int
}

func (c *controller) Configure() error {
	c.configured++
	return nil
}
func (c *controller) Tx(address uint16, write, read []byte) error {
	c.address = address
	c.writes += len(write)
	c.reads += len(read)
	if len(read) != 0 {
		read[0] = 9
	}
	return nil
}

var ledPin = pin{}
var buttonPin = pin{high: true}

var BlueLED = gpio.NewLED(&ledPin, false)
var Button = gpio.NewButton(&buttonPin, gpio.PullUp, true)

var clockSource = source{}
var Clock = clock.New(&clockSource)

var groveController = controller{}
var Grove = i2c.DefinePort(&groveController, &Clock)

func PressButton() { buttonPin.high = false }

func State() (bool, int, int, uint16, int, int) {
	return ledPin.high, ledPin.configured, groveController.configured,
		groveController.address, groveController.writes, groveController.reads
}
