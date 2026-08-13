// Package esp32s3 provides reusable ESP32-S3 peripheral implementations.
package esp32s3

import (
	"renvo.dev/device/gpio"
	"renvo.dev/device/mmio"
)

const (
	ioMuxBase        = uintptr(0x60009004)
	gpioBase         = uintptr(0x60004000)
	outputSelectBase = uintptr(0x60004554)

	gpioOutSet       = uintptr(gpioBase + 0x08)
	gpioOutClear     = uintptr(gpioBase + 0x0c)
	gpioOut1Set      = uintptr(gpioBase + 0x14)
	gpioOut1Clear    = uintptr(gpioBase + 0x18)
	gpioEnableSet    = uintptr(gpioBase + 0x24)
	gpioEnableClear  = uintptr(gpioBase + 0x28)
	gpioEnable1Set   = uintptr(gpioBase + 0x30)
	gpioEnable1Clear = uintptr(gpioBase + 0x34)
	gpioInput        = uintptr(gpioBase + 0x3c)
	gpioInput1       = uintptr(gpioBase + 0x40)

	gpioFunctionMask = uint32(7 << 12)
	gpioFunction     = uint32(1 << 12)
	gpioInputEnable  = uint32(1 << 9)
	gpioPullUp       = uint32(1 << 8)
	gpioPullDown     = uint32(1 << 7)
	gpioOutputSignal = uint32(256)
)

// Pin identifies one ESP32-S3 GPIO.
type Pin struct{ number uint8 }

// GPIO returns the concrete pin capability for number.
func GPIO(number uint8) *Pin { return &Pin{number: number} }

func (p *Pin) bit() uint32 {
	number := p.number
	if number >= 32 {
		number -= 32
	}
	return uint32(1) << uint(number)
}

func (p *Pin) ioMux() uintptr { return ioMuxBase + uintptr(p.number)*4 }
func (p *Pin) outputSelect() uintptr {
	return outputSelectBase + uintptr(p.number)*4
}

func (p *Pin) Configure(config gpio.Config) error {
	value := mmio.Load32(p.ioMux()) &^ (gpioFunctionMask | gpioInputEnable | gpioPullUp | gpioPullDown)
	value |= gpioFunction
	if config.Direction == gpio.Input {
		value |= gpioInputEnable
	}
	if config.Pull == gpio.PullUp {
		value |= gpioPullUp
	} else if config.Pull == gpio.PullDown {
		value |= gpioPullDown
	}
	mmio.Store32(p.ioMux(), value)
	mmio.Store32(p.outputSelect(), gpioOutputSignal)
	p.enable(config.Direction == gpio.Output)
	return nil
}

func (p *Pin) enable(enabled bool) {
	set, clear := gpioEnableSet, gpioEnableClear
	if p.number >= 32 {
		set, clear = gpioEnable1Set, gpioEnable1Clear
	}
	if enabled {
		mmio.Store32(set, p.bit())
	} else {
		mmio.Store32(clear, p.bit())
	}
}

func (p *Pin) Set(high bool) {
	set, clear := gpioOutSet, gpioOutClear
	if p.number >= 32 {
		set, clear = gpioOut1Set, gpioOut1Clear
	}
	if high {
		mmio.Store32(set, p.bit())
	} else {
		mmio.Store32(clear, p.bit())
	}
}

func (p *Pin) Get() bool {
	input := gpioInput
	if p.number >= 32 {
		input = gpioInput1
	}
	return mmio.Load32(input)&p.bit() != 0
}

// ConfigureOutputSignal connects a GPIO-matrix peripheral output.
func (p *Pin) ConfigureOutputSignal(signal uint32) error {
	value := mmio.Load32(p.ioMux()) &^ (gpioFunctionMask | gpioInputEnable | gpioPullUp | gpioPullDown)
	mmio.Store32(p.ioMux(), value|gpioFunction)
	mmio.Store32(p.outputSelect(), signal)
	p.enable(true)
	return nil
}

// ConfigureOpenDrain prepares a pin for software I2C. The output latch stays
// low; output-enable selects between pulling low and releasing the line.
func (p *Pin) ConfigureOpenDrain() error {
	value := mmio.Load32(p.ioMux()) &^ (gpioFunctionMask | gpioPullDown)
	mmio.Store32(p.ioMux(), value|gpioFunction|gpioInputEnable|gpioPullUp)
	mmio.Store32(p.outputSelect(), gpioOutputSignal)
	p.Set(false)
	p.enable(false)
	return nil
}

// PullLow drives an open-drain pin low.
func (p *Pin) PullLow() { p.enable(true) }

// Release stops driving an open-drain pin.
func (p *Pin) Release() { p.enable(false) }

// High samples an open-drain pin.
func (p *Pin) High() bool { return p.Get() }
