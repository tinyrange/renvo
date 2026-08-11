// Package esp32c6 provides reusable ESP32-C6 peripheral implementations.
package esp32c6

import (
	"renvo.dev/device/gpio"
	"renvo.dev/device/mmio"
)

const (
	ioMuxBase        = uintptr(0x60090004)
	gpioBase         = uintptr(0x60091000)
	outputSelectBase = uintptr(0x60091554)

	gpioOutSet      = uintptr(gpioBase + 0x08)
	gpioOutClear    = uintptr(gpioBase + 0x0c)
	gpioEnableSet   = uintptr(gpioBase + 0x24)
	gpioEnableClear = uintptr(gpioBase + 0x28)
	gpioInput       = uintptr(gpioBase + 0x3c)

	gpioFunctionMask = uint32(7 << 12)
	gpioFunction     = uint32(1 << 12)
	gpioInputEnable  = uint32(1 << 9)
	gpioPullUp       = uint32(1 << 8)
	gpioPullDown     = uint32(1 << 7)
	gpioOutputSignal = uint32(128)
)

// Pin identifies one ESP32-C6 GPIO.
type Pin struct {
	number uint8
}

// GPIO returns the concrete pin capability for number.
func GPIO(number uint8) *Pin { return &Pin{number: number} }

func (p *Pin) bit() uint32 { return uint32(1) << uint(p.number) }

func (p *Pin) ioMux() uintptr {
	return ioMuxBase + uintptr(p.number+1)*4
}

func (p *Pin) outputSelect() uintptr {
	return outputSelectBase + uintptr(p.number)*4
}

// Configure selects ordinary GPIO operation and applies direction and pull.
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
	if config.Direction == gpio.Output {
		mmio.Store32(gpioEnableSet, p.bit())
	} else {
		mmio.Store32(gpioEnableClear, p.bit())
	}
	return nil
}

// ConfigureOutputSignal selects an ESP GPIO-matrix output signal and enables
// the pin. It is used by chip peripheral drivers, not portable applications.
func (p *Pin) ConfigureOutputSignal(signal uint32) error {
	value := mmio.Load32(p.ioMux()) &^ (gpioFunctionMask | gpioInputEnable | gpioPullUp | gpioPullDown)
	mmio.Store32(p.ioMux(), value|gpioFunction)
	mmio.Store32(p.outputSelect(), signal)
	mmio.Store32(gpioEnableSet, p.bit())
	return nil
}

// ConfigureOpenDrain prepares a pin for software I2C. The output latch stays
// low; output-enable selects between pulling low and releasing the line.
func (p *Pin) ConfigureOpenDrain() error {
	value := mmio.Load32(p.ioMux()) &^ (gpioFunctionMask | gpioPullDown)
	mmio.Store32(p.ioMux(), value|gpioFunction|gpioInputEnable|gpioPullUp)
	mmio.Store32(p.outputSelect(), gpioOutputSignal)
	mmio.Store32(gpioOutClear, p.bit())
	mmio.Store32(gpioEnableClear, p.bit())
	return nil
}

// PullLow drives an open-drain pin low.
func (p *Pin) PullLow() { mmio.Store32(gpioEnableSet, p.bit()) }

// Release stops driving an open-drain pin.
func (p *Pin) Release() { mmio.Store32(gpioEnableClear, p.bit()) }

// High samples an open-drain pin.
func (p *Pin) High() bool { return p.Get() }

// Set changes the output latch without changing direction.
func (p *Pin) Set(high bool) {
	if high {
		mmio.Store32(gpioOutSet, p.bit())
	} else {
		mmio.Store32(gpioOutClear, p.bit())
	}
}

// Get samples the GPIO input register.
func (p *Pin) Get() bool { return mmio.Load32(gpioInput)&p.bit() != 0 }
