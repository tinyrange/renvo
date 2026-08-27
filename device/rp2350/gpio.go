// Package rp2350 provides reusable RP2350 peripheral implementations.
package rp2350

import (
	"renvo.dev/device/gpio"
	"renvo.dev/device/mmio"
)

const (
	ioBank0Base   = uintptr(0x40028000)
	padsBank0Base = uintptr(0x40038000)
	sioBase       = uintptr(0xd0000000)

	gpioInput     = sioBase + 0x04
	gpioOutSet    = sioBase + 0x18
	gpioOutClear  = sioBase + 0x20
	gpioEnableSet = sioBase + 0x38
	gpioEnableClr = sioBase + 0x40

	padIsolation = uint32(1 << 8)
	padOutputOff = uint32(1 << 7)
	padInput     = uint32(1 << 6)
	padPullUp    = uint32(1 << 3)
	padPullDown  = uint32(1 << 2)
	gpioFunction = uint32(5)
)

// Pin identifies one RP2350 GPIO.
type Pin struct{ number uint8 }

// GPIO returns the concrete pin capability for number.
func GPIO(number uint8) *Pin { return &Pin{number: number} }

func (p *Pin) bit() uint32 { return uint32(1) << uint(p.number) }

func (p *Pin) control() uintptr {
	return ioBank0Base + uintptr(p.number)*8 + 4
}

func (p *Pin) pad() uintptr {
	return padsBank0Base + uintptr(p.number)*4 + 4
}

// Configure selects the SIO function and applies direction and pull policy.
func (p *Pin) Configure(config gpio.Config) error {
	pad := mmio.Load32(p.pad()) &^ (padIsolation | padOutputOff | padInput | padPullUp | padPullDown)
	pad |= padInput
	if config.Pull == gpio.PullUp {
		pad |= padPullUp
	} else if config.Pull == gpio.PullDown {
		pad |= padPullDown
	}
	mmio.Store32(p.pad(), pad)
	mmio.Store32(p.control(), gpioFunction)
	if config.Direction == gpio.Output {
		mmio.Store32(gpioEnableSet, p.bit())
	} else {
		mmio.Store32(gpioEnableClr, p.bit())
	}
	return nil
}

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
